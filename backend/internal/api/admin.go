package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"themis/internal/config"
	"themis/internal/llm"
	"themis/internal/memory"
)

// ---------- Configuration IA modifiable à chaud ----------

// AIConfig est la configuration IA effective, modifiable depuis la page
// d'admin sans redémarrage. Persistée dans settings (clé "ai_config").
type AIConfig struct {
	OpenAIKey      string `json:"openai_key,omitempty"`
	OpenAIModel    string `json:"openai_model"`
	Provider       string `json:"provider"` // "none" | "ollama" | "anthropic" | "mistral" | "freellm" | "openai"
	OllamaURL      string `json:"ollama_url"`
	OllamaModel    string `json:"ollama_model"`
	AnthropicModel string `json:"anthropic_model"`
	// AnthropicKey vide = utiliser ANTHROPIC_API_KEY de l'environnement.
	// Jamais renvoyée en clair par l'API (masquée).
	AnthropicKey string `json:"anthropic_key,omitempty"`
	// Mistral AI
	MistralKey   string `json:"mistral_key,omitempty"`
	MistralModel string `json:"mistral_model"`
	// FreeLLM
	FreeLLMKey     string `json:"freellm_key,omitempty"`
	FreeLLMURL     string `json:"freellm_url"`
	FreeLLMModel   string `json:"freellm_model"`
	AutoCategorize bool   `json:"auto_categorize"`
	// Context : instructions libres fournies par l'utilisateur pour éclairer les
	// commentaires de l'IA (ex. « mon loyer est trimestriel », « le virement à
	// Paul est un remboursement, pas une dépense »). Injecté dans les prompts
	// d'ANALYSE et de synthèse — jamais dans l'extraction technique des relevés.
	Context string `json:"context,omitempty"`
}

func aiConfigFromEnv(cfg config.Config) AIConfig {
	return AIConfig{
		Provider:       cfg.AIProvider,
		OpenAIModel:    cfg.OpenAIModel,
		OllamaURL:      cfg.OllamaURL,
		OllamaModel:    cfg.OllamaModel,
		AnthropicModel: cfg.AnthropicModel,
		MistralKey:     cfg.MistralKey,
		MistralModel:   cfg.MistralModel,
		FreeLLMKey:     cfg.FreeLLMKey,
		FreeLLMURL:     cfg.FreeLLMURL,
		FreeLLMModel:   cfg.FreeLLMModel,
		AutoCategorize: cfg.AIAutoCategorize,
	}
}

// buildClient construit le client LLM correspondant à la config.
func (s *Server) buildClient(c AIConfig) llm.Client {
	switch c.Provider {
	case "openai":
		c = s.withOpenAIDefaults(c)
		key := c.OpenAIKey
		if key == "" {
			key = s.Cfg.OpenAIKey
		}
		if key == "" {
			return nil
		}
		return &llm.OpenAI{APIKey: key, Model: c.OpenAIModel}
	case "ollama":
		if c.OllamaURL == "" {
			return nil
		}
		return &llm.Ollama{URL: c.OllamaURL, Model: c.OllamaModel}
	case "anthropic":
		key := c.AnthropicKey
		if key == "" {
			key = s.Cfg.AnthropicKey // repli sur l'environnement
		}
		if key == "" {
			return nil
		}
		return &llm.Anthropic{APIKey: key, Model: c.AnthropicModel}
	case "mistral":
		key := c.MistralKey
		if key == "" {
			key = s.Cfg.MistralKey
		}
		if key == "" {
			return nil
		}
		return &llm.Mistral{APIKey: key, Model: c.MistralModel}
	case "freellm":
		key := c.FreeLLMKey
		if key == "" {
			key = s.Cfg.FreeLLMKey
		}
		if key == "" {
			return nil
		}
		baseURL := c.FreeLLMURL
		if baseURL == "" {
			baseURL = s.Cfg.FreeLLMURL
		}
		return &llm.FreeLLM{APIKey: key, BaseURL: baseURL, Model: c.FreeLLMModel}
	default:
		return nil
	}
}

// withOpenAIDefaults conserve le secret lors d'un enregistrement ou test
// avec champ masqué vide. Le repli environnement reste dans buildClient.
func (s *Server) withOpenAIDefaults(c AIConfig) AIConfig {
	c.OpenAIKey = strings.TrimSpace(c.OpenAIKey)
	c.OpenAIModel = strings.TrimSpace(c.OpenAIModel)
	if c.OpenAIKey == "" {
		c.OpenAIKey = s.getAIConfig().OpenAIKey
	}
	if c.OpenAIModel == "" {
		c.OpenAIModel = s.Cfg.OpenAIModel
	}
	if c.OpenAIModel == "" {
		c.OpenAIModel = llm.DefaultOpenAIModel
	}
	return c
}

// aiState protège la config et le client actifs.
type aiState struct {
	mu     sync.RWMutex
	config AIConfig
	client llm.Client
}

func (s *Server) getLLM() llm.Client {
	s.ai.mu.RLock()
	defer s.ai.mu.RUnlock()
	return s.ai.client
}

// getMemory retourne le client knowledge graph, ou nil s'il n'est pas configuré.
func (s *Server) getMemory() *memory.Client { return s.Memory }

// aiContext renvoie les instructions personnalisées de l'utilisateur (peut être
// vide), APRÈS sanitisation contre l'injection de prompt. Utilisé pour enrichir
// les prompts d'analyse et de synthèse.
func (s *Server) aiContext() string {
	s.ai.mu.RLock()
	raw := s.ai.config.Context
	s.ai.mu.RUnlock()
	return sanitizeUserContext(raw)
}

func (s *Server) getAIConfig() AIConfig {
	s.ai.mu.RLock()
	defer s.ai.mu.RUnlock()
	return s.ai.config
}

func (s *Server) autoCategorizeEnabled() bool {
	s.ai.mu.RLock()
	defer s.ai.mu.RUnlock()
	return s.ai.config.AutoCategorize
}

// ApplyAIConfig active une configuration (client reconstruit) sans la persister.
func (s *Server) ApplyAIConfig(c AIConfig) {
	if c.Provider == "openai" {
		c = s.withOpenAIDefaults(c)
	}
	client := s.buildClient(c)
	s.ai.mu.Lock()
	s.ai.config = c
	s.ai.client = client
	s.ai.mu.Unlock()
	if client != nil {
		log.Printf("IA: %s", client.Name())
	} else {
		log.Printf("IA désactivée")
	}
}

// InitAI charge la config IA : celle persistée en base si présente,
// sinon celle de l'environnement.
func (s *Server) InitAI() {
	c := aiConfigFromEnv(s.Cfg)
	var raw []byte
	err := s.DB.QueryRow(`SELECT value FROM settings WHERE key = 'ai_config'`).Scan(&raw)
	if err == nil {
		var saved AIConfig
		if json.Unmarshal(raw, &saved) == nil && saved.Provider != "" {
			c = saved
			log.Printf("configuration IA chargée depuis la base (admin)")
		}
	} else if err != sql.ErrNoRows {
		log.Printf("lecture ai_config: %v", err)
	}
	s.ApplyAIConfig(c)
}

// ---------- Handlers admin ----------

// masked retourne la config sans secret, avec un indicateur de présence de clé.
func (s *Server) maskedConfig() map[string]any {
	c := s.getAIConfig()
	client := s.getLLM()
	provider := "none"
	if client != nil {
		provider = client.Name()
	}
	return map[string]any{
		"provider":          c.Provider,
		"openai_model":      c.OpenAIModel,
		"openai_key_set":    c.OpenAIKey != "" || s.Cfg.OpenAIKey != "",
		"ollama_url":        c.OllamaURL,
		"ollama_model":      c.OllamaModel,
		"anthropic_model":   c.AnthropicModel,
		"anthropic_key_set": c.AnthropicKey != "" || s.Cfg.AnthropicKey != "",
		"mistral_key_set":   c.MistralKey != "" || s.Cfg.MistralKey != "",
		"mistral_model":     c.MistralModel,
		"freellm_key_set":   c.FreeLLMKey != "" || s.Cfg.FreeLLMKey != "",
		"freellm_url":       c.FreeLLMURL,
		"freellm_model":     c.FreeLLMModel,
		"auto_categorize":   c.AutoCategorize,
		"context":           c.Context,
		"active":            provider,
	}
}

func (s *Server) adminGetConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.maskedConfig())
}

func (s *Server) adminPutConfig(w http.ResponseWriter, r *http.Request) {
	var in AIConfig
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("corps JSON invalide"))
		return
	}
	in = s.withOpenAIDefaults(in)
	switch in.Provider {
	case "none", "ollama", "anthropic", "mistral", "freellm", "openai":
	default:
		writeErr(w, http.StatusBadRequest, fmt.Errorf(`provider doit être "none", "ollama", "anthropic", "mistral", "freellm" ou "openai"`))
		return
	}
	if in.Provider == "openai" && in.OpenAIKey == "" && s.Cfg.OpenAIKey == "" {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("openai_key requis"))
		return
	}
	if in.Provider == "ollama" && in.OllamaURL == "" {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("ollama_url requis"))
		return
	}
	if in.Provider == "mistral" && in.MistralKey == "" && s.Cfg.MistralKey == "" {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("mistral_key requis"))
		return
	}
	if in.Provider == "freellm" && in.FreeLLMKey == "" && s.Cfg.FreeLLMKey == "" {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("freellm_key requis"))
		return
	}
	// clé non fournie → conserver l'existante (le champ masqué n'est jamais renvoyé)
	if in.AnthropicKey == "" {
		in.AnthropicKey = s.getAIConfig().AnthropicKey
	}
	if in.MistralKey == "" {
		in.MistralKey = s.getAIConfig().MistralKey
	}
	if in.FreeLLMKey == "" {
		in.FreeLLMKey = s.getAIConfig().FreeLLMKey
	}
	if in.OllamaModel == "" {
		in.OllamaModel = s.Cfg.OllamaModel
	}
	if in.AnthropicModel == "" {
		in.AnthropicModel = s.Cfg.AnthropicModel
	}
	if in.MistralModel == "" {
		in.MistralModel = s.Cfg.MistralModel
	}
	if in.FreeLLMModel == "" {
		in.FreeLLMModel = s.Cfg.FreeLLMModel
	}
	if in.FreeLLMURL == "" {
		in.FreeLLMURL = s.Cfg.FreeLLMURL
	}

	raw, _ := json.Marshal(in)
	if _, err := s.DB.Exec(`
		INSERT INTO settings (key, value, owner_id) VALUES ('ai_config', $1, NULL)
		ON CONFLICT (COALESCE(owner_id, 0), key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`, raw); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	s.ApplyAIConfig(in)
	writeJSON(w, http.StatusOK, s.maskedConfig())
}

// adminTestAI vérifie la connexion au fournisseur configuré (ou à celui du
// corps de requête, pour tester avant d'enregistrer).
func (s *Server) adminTestAI(w http.ResponseWriter, r *http.Request) {
	var in AIConfig
	client := s.getLLM()
	if err := json.NewDecoder(r.Body).Decode(&in); err == nil && in.Provider != "" {
		if in.AnthropicKey == "" {
			in.AnthropicKey = s.getAIConfig().AnthropicKey
		}
		in = s.withOpenAIDefaults(in)
		client = s.buildClient(in)
	}
	if client == nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "detail": "aucun fournisseur configuré"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), llm.TestTimeout())
	defer cancel()

	if o, ok := client.(*llm.Ollama); ok {
		if err := o.Ping(ctx); err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"ok": false, "provider": client.Name(), "detail": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "provider": client.Name(), "detail": "instance joignable, modèle présent"})
		return
	}
	// Test synthétique uniquement ; aucun flux bancaire envoyé.
	maxTokens := 4
	if _, ok := client.(*llm.OpenAI); ok {
		maxTokens = 256
	}
	if _, err := client.Complete(ctx, "Réponds uniquement OK.", "ping", maxTokens, false); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "provider": client.Name(), "detail": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "provider": client.Name(), "detail": "API joignable"})
}

// adminOllamaModels liste les modèles disponibles sur une instance Ollama
// (?url= optionnel, sinon celle configurée) pour alimenter le sélecteur.
func (s *Server) adminOllamaModels(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		url = s.getAIConfig().OllamaURL
	}
	if url == "" {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("aucune URL Ollama"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	models, err := llm.ListOllamaModels(ctx, url)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"models": models})
}

// ---------- Statistiques ----------

func (s *Server) adminStats(w http.ResponseWriter, r *http.Request) {
	owner := s.ownerID(r)
	stats := map[string]any{}
	queries := map[string]string{
		"statements":   `SELECT count(*) FROM statements WHERE owner_id = $1`,
		"transactions": `SELECT count(*) FROM transactions WHERE owner_id = $1`,
		"categorized":  `SELECT count(*) FROM transactions WHERE domain_id IS NOT NULL AND owner_id = $1`,
		"domains":      `SELECT count(*) FROM domains WHERE parent_id IS NULL AND owner_id = $1`,
		"rules":        `SELECT count(*) FROM rules WHERE owner_id = $1`,
		"analyses":     `SELECT count(*) FROM analyses WHERE owner_id = $1`,
		"budgets":      `SELECT count(*) FROM budgets WHERE owner_id = $1`,
	}
	for k, q := range queries {
		var n int64
		if err := s.DB.QueryRow(q, owner).Scan(&n); err == nil {
			stats[k] = n
		}
	}
	var size string
	if err := s.DB.QueryRow(`SELECT pg_size_pretty(pg_database_size(current_database()))`).Scan(&size); err == nil {
		stats["db_size"] = size
	}
	var first, last sql.NullString
	_ = s.DB.QueryRow(`SELECT min(tx_date)::text, max(tx_date)::text FROM transactions WHERE owner_id = $1`, owner).Scan(&first, &last)
	stats["first_tx"] = first.String
	stats["last_tx"] = last.String
	writeJSON(w, http.StatusOK, stats)
}

// ---------- Maintenance ----------

// adminReapplyRules réapplique toutes les règles aux transactions sans domaine.
func (s *Server) adminReapplyRules(w http.ResponseWriter, r *http.Request) {
	n, err := s.applyAllRules(s.ownerID(r))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"categorized": n})
}

// applyAllRules réapplique les règles d'un utilisateur à ses transactions sans
// domaine. Partagé entre l'action admin et le bouton « Appliquer les règles ».
func (s *Server) applyAllRules(ownerID int64) (int, error) {
	// On parcourt les règles de l'utilisateur (avec leurs tags) et on applique
	// chacune : catégorie sur les transactions non classées + tags sur toutes
	// celles qui correspondent. Une seule logique, partagée avec la création.
	rows, err := s.DB.Query(`SELECT pattern, domain_id, tags FROM rules WHERE owner_id = $1 ORDER BY length(pattern) DESC`, ownerID)
	if err != nil {
		return 0, err
	}
	type rule struct {
		pattern string
		dom     int64
		tags    string
	}
	var rules []rule
	for rows.Next() {
		var rl rule
		if err := rows.Scan(&rl.pattern, &rl.dom, &rl.tags); err != nil {
			rows.Close()
			return 0, err
		}
		rules = append(rules, rl)
	}
	rows.Close()

	total := 0
	for _, rl := range rules {
		total += int(s.applyRuleToTransactions(ownerID, rl.pattern, rl.dom, rl.tags))
	}
	return total, nil
}

// applyRules : version accessible à tout utilisateur connecté (bouton UI).
func (s *Server) applyRules(w http.ResponseWriter, r *http.Request) {
	n, err := s.applyAllRules(s.ownerID(r))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"applied": n})
}

// adminPurge efface TOUTES les données (relevés, transactions, analyses).
// Les catégories, règles, budgets et réglages sont conservés.
// Exige la confirmation exacte {"confirm": "TOUT EFFACER"}.
func (s *Server) adminPurge(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Confirm string `json:"confirm"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || strings.TrimSpace(in.Confirm) != "TOUT EFFACER" {
		writeErr(w, http.StatusBadRequest, fmt.Errorf(`confirmation requise : {"confirm": "TOUT EFFACER"}`))
		return
	}
	owner := s.ownerID(r)
	// Supprimer UNIQUEMENT les données de l'utilisateur courant (les
	// transactions et tags associés partent en cascade via statements).
	tx, err := s.DB.Begin()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	defer tx.Rollback()
	for _, q := range []string{
		`DELETE FROM transactions WHERE owner_id = $1`,
		`DELETE FROM statements WHERE owner_id = $1`,
		`DELETE FROM analyses WHERE owner_id = $1`,
	} {
		if _, err := tx.Exec(q, owner); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	log.Printf("ADMIN — purge des données de l'utilisateur %d effectuée", owner)
	writeJSON(w, http.StatusOK, map[string]string{"status": "purgé"})
}

// GetLLMForCheck expose le client actif pour la vérification de démarrage.
func (s *Server) GetLLMForCheck() llm.Client { return s.getLLM() }
