package api

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"themis/internal/llm"
)

var aiModelPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/@+\-]{0,199}$`)
var aiFields = map[string]bool{"provider": true, "openai_key": true, "openai_model": true, "anthropic_key": true, "anthropic_model": true, "mistral_key": true, "mistral_model": true, "freellm_key": true, "freellm_url": true, "freellm_model": true, "ollama_url": true, "ollama_model": true, "context": true, "auto_categorize": true}
var aiKeyFields = []string{"openai_key", "anthropic_key", "mistral_key", "freellm_key"}

func (s *Server) aiEndpointPolicy() llm.EndpointPolicy {
	trusted := append([]string{}, s.Cfg.AITrustedEndpoints...)
	trusted = append(trusted, s.Cfg.OllamaURL, s.Cfg.FreeLLMURL)
	return llm.EndpointPolicy{TrustedEndpoints: trusted}
}
func (s *Server) aiAdminOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := r.Context().Value(ctxUserKey).(*authUser)
		if !ok || user == nil || user.ID <= 0 {
			writeErr(w, 401, fmt.Errorf("authentification requise"))
			return
		}
		if user.Role != "admin" {
			writeErr(w, 403, fmt.Errorf("réservé aux administrateurs"))
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		next(w, r)
	}
}
func aiRevision() string {
	var token [16]byte
	if _, err := rand.Read(token[:]); err != nil {
		return fmt.Sprint(time.Now().UnixNano())
	}
	return hex.EncodeToString(token[:])
}
func (s *Server) aiBaseConfig() AIConfig {
	c := s.getAIConfig()
	if c.Provider != "" {
		return c
	}
	c = aiConfigFromEnv(s.Cfg)
	if c.Provider == "" {
		c.Provider = "none"
	}
	if c.OpenAIModel == "" {
		c.OpenAIModel = llm.DefaultOpenAIModel
	}
	return c
}
func (s *Server) aiKeyValues(c AIConfig) map[string]string {
	return map[string]string{"openai_key": c.OpenAIKey, "anthropic_key": c.AnthropicKey, "mistral_key": c.MistralKey, "freellm_key": c.FreeLLMKey}
}
func (s *Server) aiEnvKeys() map[string]string {
	return map[string]string{"openai_key": s.Cfg.OpenAIKey, "anthropic_key": s.Cfg.AnthropicKey, "mistral_key": s.Cfg.MistralKey, "freellm_key": s.Cfg.FreeLLMKey}
}
func (s *Server) aiConnection(c AIConfig) (model, endpoint, key string) {
	keys := s.aiKeyValues(c)
	key = keys[c.Provider+"_key"]
	if key == "" {
		key = s.aiEnvKeys()[c.Provider+"_key"]
	}
	switch c.Provider {
	case "ollama":
		model, endpoint = c.OllamaModel, c.OllamaURL
	case "openai":
		model = c.OpenAIModel
	case "anthropic":
		model = c.AnthropicModel
	case "mistral":
		model = c.MistralModel
	case "freellm":
		model, endpoint = c.FreeLLMModel, c.FreeLLMURL
	}
	return
}
func (s *Server) aiConnectionDigest(c AIConfig) [32]byte {
	model, endpoint, key := s.aiConnection(c)
	raw, _ := json.Marshal([]string{c.Provider, model, strings.TrimRight(endpoint, "/"), key})
	return sha256.Sum256(raw)
}
func (s *Server) validateAIConfig(c AIConfig, behavior bool) error {
	switch c.Provider {
	case "none", "ollama", "openai", "anthropic", "mistral", "freellm":
	default:
		return fmt.Errorf("fournisseur IA inconnu")
	}
	if behavior && utf8.RuneCountInString(c.Context) > maxContextLen {
		return fmt.Errorf("contexte personnel limité à %d caractères", maxContextLen)
	}
	for _, key := range s.aiKeyValues(c) {
		if len(key) > 8192 || strings.ContainsAny(key, "\r\n\x00") {
			return fmt.Errorf("clé API invalide")
		}
	}
	if c.Provider == "none" {
		return nil
	}
	model, endpoint, key := s.aiConnection(c)
	if !aiModelPattern.MatchString(model) {
		return fmt.Errorf("identifiant de modèle requis, sans espaces, 200 caractères maximum")
	}
	if c.Provider != "ollama" && strings.TrimSpace(key) == "" {
		return fmt.Errorf("clé API requise pour ce fournisseur")
	}
	if c.Provider == "ollama" || c.Provider == "freellm" {
		if err := s.aiEndpointPolicy().Validate(endpoint); err != nil {
			return err
		}
	}
	return nil
}
func decodeAIPatch(w http.ResponseWriter, r *http.Request) (map[string]json.RawMessage, error) {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	var patch map[string]json.RawMessage
	if err := decoder.Decode(&patch); err != nil || patch == nil {
		return nil, fmt.Errorf("objet JSON de configuration requis")
	}
	if decoder.Decode(new(any)) != io.EOF {
		return nil, fmt.Errorf("un seul objet JSON est attendu")
	}
	for key, value := range patch {
		if !aiFields[key] && key != "clear_keys" && key != "expected_revision" {
			return nil, fmt.Errorf("champ de configuration inconnu")
		}
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return nil, fmt.Errorf("valeur nulle interdite")
		}
	}
	return patch, nil
}
func (s *Server) mergeAIPatch(base AIConfig, patch map[string]json.RawMessage) (AIConfig, string, error) {
	var revision string
	if raw, ok := patch["expected_revision"]; ok {
		if json.Unmarshal(raw, &revision) != nil {
			return base, "", fmt.Errorf("révision invalide")
		}
	}
	var cleared []string
	if raw, ok := patch["clear_keys"]; ok {
		if json.Unmarshal(raw, &cleared) != nil {
			return base, "", fmt.Errorf("liste de clés à retirer invalide")
		}
	}
	fields := make(map[string]json.RawMessage)
	for key, raw := range patch {
		if aiFields[key] {
			fields[key] = raw
		}
	}
	// A blank secret keeps the stored one. Removal is always explicit.
	for _, field := range aiKeyFields {
		if raw, ok := fields[field]; ok {
			var key string
			if json.Unmarshal(raw, &key) != nil {
				return base, "", fmt.Errorf("clé API invalide")
			}
			if strings.TrimSpace(key) == "" {
				delete(fields, field)
			}
		}
	}
	raw, _ := json.Marshal(fields)
	c := base
	if err := json.Unmarshal(raw, &c); err != nil {
		return base, "", fmt.Errorf("types de paramètres invalides")
	}
	for _, field := range cleared {
		if _, present := fields[field]; present {
			return base, "", fmt.Errorf("une clé ne peut pas être remplacée et retirée simultanément")
		}
		switch field {
		case "openai_key":
			c.OpenAIKey = ""
		case "anthropic_key":
			c.AnthropicKey = ""
		case "mistral_key":
			c.MistralKey = ""
		case "freellm_key":
			c.FreeLLMKey = ""
		default:
			return base, "", fmt.Errorf("clé à retirer inconnue")
		}
	}
	values := []*string{&c.Provider, &c.OpenAIModel, &c.OpenAIKey, &c.OllamaURL, &c.OllamaModel, &c.AnthropicKey, &c.AnthropicModel, &c.MistralKey, &c.MistralModel, &c.FreeLLMKey, &c.FreeLLMModel, &c.FreeLLMURL, &c.Context}
	for _, value := range values {
		*value = strings.TrimSpace(*value)
	}
	return c, revision, nil
}
func (s *Server) ApplyAIConfig(c AIConfig) {
	if c.OpenAIModel == "" {
		c.OpenAIModel = s.Cfg.OpenAIModel
		if c.OpenAIModel == "" {
			c.OpenAIModel = llm.DefaultOpenAIModel
		}
	}
	if c.Provider == "" {
		c.Provider = "none"
	}
	var client llm.Client
	problem := ""
	if err := s.validateAIConfig(c, false); err != nil {
		problem = err.Error()
	} else {
		client = s.buildClient(c)
	}
	digest := s.aiConnectionDigest(c)
	s.ai.mu.Lock()
	s.ai.config = c
	s.ai.client = client
	s.ai.revision = aiRevision()
	s.ai.configError = problem
	var record *aiTestRecord
	if s.ai.lastDraft != nil && s.ai.lastDraft.digest == digest {
		record = s.ai.lastDraft
	} else if s.ai.lastSaved != nil && s.ai.lastSaved.digest == digest {
		record = s.ai.lastSaved
	}
	s.ai.lastSaved = nil
	if record != nil {
		copy := *record
		copy.Result.Scope = "saved"
		copy.Result.Revision = s.ai.revision
		s.ai.lastSaved = &copy
	}
	s.ai.mu.Unlock()
	// Model names, URLs, keys and user context are intentionally absent from logs.
	log.Printf("configuration IA appliquée (fournisseur=%s)", safeAIProvider(c.Provider))
}
func safeAIProvider(provider string) string {
	switch provider {
	case "none", "ollama", "openai", "anthropic", "mistral", "freellm":
		return provider
	}
	return "unknown"
}
func (s *Server) InitAI() {
	c := aiConfigFromEnv(s.Cfg)
	source := "environment"
	var raw []byte
	err := s.DB.QueryRow(`SELECT value FROM settings WHERE key='ai_config' AND owner_id IS NULL`).Scan(&raw)
	if err == nil {
		var saved AIConfig
		if json.Unmarshal(raw, &saved) == nil && saved.Provider != "" {
			c = saved
			source = "database"
		} else {
			log.Print("configuration IA enregistrée illisible ; environnement conservé")
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		log.Print("lecture de la configuration IA indisponible ; environnement conservé")
	}
	s.ApplyAIConfig(c)
	s.ai.mu.Lock()
	s.ai.source = source
	s.ai.mu.Unlock()
}
func publicAIURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	u.User = nil
	u.RawQuery = ""
	u.Fragment = ""
	u.ForceQuery = false
	return u.String()
}
func (s *Server) maskedConfig() map[string]any {
	s.ai.mu.RLock()
	c, client, revision, source, problem := s.ai.config, s.ai.client, s.ai.revision, s.ai.source, s.ai.configError
	var saved *aiDiagnostic
	if s.ai.lastSaved != nil {
		copy := s.ai.lastSaved.Result
		saved = &copy
	}
	var memoryTest *aiDiagnostic
	if s.ai.lastMemory != nil {
		copy := *s.ai.lastMemory
		memoryTest = &copy
	}
	s.ai.mu.RUnlock()
	if c.Provider == "" {
		c = s.aiBaseConfig()
	}
	active := "none"
	if client != nil {
		active = client.Name()
	}
	out := map[string]any{"provider": c.Provider, "active": active, "enabled": client != nil, "revision": revision, "source": source, "configuration_error": problem, "openai_model": c.OpenAIModel, "ollama_url": publicAIURL(c.OllamaURL), "ollama_model": c.OllamaModel, "anthropic_model": c.AnthropicModel, "mistral_model": c.MistralModel, "freellm_model": c.FreeLLMModel, "freellm_url": publicAIURL(c.FreeLLMURL), "auto_categorize": c.AutoCategorize, "context": c.Context}
	sources := map[string]string{}
	for field, key := range s.aiKeyValues(c) {
		kind := "none"
		if key != "" {
			kind = "stored"
		} else if s.aiEnvKeys()[field] != "" {
			kind = "environment"
		}
		sources[strings.TrimSuffix(field, "_key")] = kind
		out[strings.TrimSuffix(field, "_key")+"_key_set"] = kind != "none"
	}
	out["key_sources"] = sources
	if saved == nil {
		model, _, _ := s.aiConnection(c)
		status := "not_tested"
		if c.Provider == "none" || client == nil {
			status = "not_configured"
		}
		saved = &aiDiagnostic{Status: status, Provider: safeAIProvider(c.Provider), Model: model, Scope: "saved", Revision: revision}
	}
	out["diagnostic"] = saved
	out["memory"] = s.aiMemoryInfo(memoryTest)
	return out
}
func (s *Server) adminGetConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, s.maskedConfig())
}
func (s *Server) adminPutConfig(w http.ResponseWriter, r *http.Request) {
	patch, err := decodeAIPatch(w, r)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	s.ai.saveMu.Lock()
	defer s.ai.saveMu.Unlock()
	c, expected, err := s.mergeAIPatch(s.aiBaseConfig(), patch)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	s.ai.mu.RLock()
	revision := s.ai.revision
	s.ai.mu.RUnlock()
	if expected != "" && expected != revision {
		writeErr(w, 409, fmt.Errorf("configuration modifiée ailleurs ; recharger avant de réessayer"))
		return
	}
	if err = s.validateAIConfig(c, true); err != nil {
		writeErr(w, 400, err)
		return
	}
	raw, err := json.Marshal(c)
	if err != nil {
		writeErr(w, 500, fmt.Errorf("configuration non enregistrée"))
		return
	}
	_, err = s.DB.ExecContext(r.Context(), `INSERT INTO settings(key,value,owner_id) VALUES('ai_config',$1,NULL) ON CONFLICT(COALESCE(owner_id,0),key) DO UPDATE SET value=EXCLUDED.value,updated_at=now()`, raw)
	if err != nil {
		writeErr(w, 500, fmt.Errorf("configuration non enregistrée ; les réglages précédents sont conservés"))
		return
	}
	s.ApplyAIConfig(c)
	s.ai.mu.Lock()
	s.ai.source = "database"
	s.ai.mu.Unlock()
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, s.maskedConfig())
}
