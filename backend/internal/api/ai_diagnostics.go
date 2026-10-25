package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"themis/internal/llm"
)

type aiDiagnostic struct {
	OK        bool    `json:"ok"`
	Status    string  `json:"status"`
	Provider  string  `json:"provider,omitempty"`
	Model     string  `json:"model,omitempty"`
	Scope     string  `json:"scope,omitempty"`
	Revision  string  `json:"revision,omitempty"`
	CheckedAt *string `json:"checked_at"`
	LatencyMS *int64  `json:"latency_ms"`
	Detail    string  `json:"detail"`
	Code      string  `json:"code,omitempty"`
}
type aiTestRecord struct {
	digest [32]byte
	Result aiDiagnostic
}

func safeAIDiagnosticError(err error) (string, string) {
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout", "Délai dépassé. Vérifier le serveur, le réseau ou la disponibilité du modèle."
	}
	if errors.Is(err, context.Canceled) {
		return "cancelled", "Contrôle annulé."
	}
	var p *llm.ProviderError
	if errors.As(err, &p) {
		if p.Kind == "model_missing" {
			return "model_missing", "Modèle absent du serveur Ollama. Installer ce modèle ou en sélectionner un autre."
		}
		switch p.Status {
		case 401:
			return "authentication", "Clé API invalide ou révoquée (HTTP 401)."
		case 403:
			return "access_denied", "Accès au modèle refusé (HTTP 403)."
		case 429:
			return "quota", "Quota ou limite de requêtes atteint (HTTP 429)."
		}
		if p.Kind == "network" {
			return "network", "Connexion impossible. Vérifier l’URL, l’autorisation réseau du serveur et TLS."
		}
		if p.Status != 0 {
			return "provider_http", fmt.Sprintf("Le fournisseur a refusé la requête (HTTP %d).", p.Status)
		}
	}
	return "provider_unavailable", "Réponse indisponible ou inexploitable. Vérifier le modèle, le serveur et ses autorisations."
}
func (s *Server) adminTestAI(w http.ResponseWriter, r *http.Request) {
	patch, err := decodeAIPatch(w, r)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	candidate, _, err := s.mergeAIPatch(s.aiBaseConfig(), patch)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	if err = s.validateAIConfig(candidate, false); err != nil {
		writeErr(w, 400, err)
		return
	}
	model, _, _ := s.aiConnection(candidate)
	result := aiDiagnostic{Provider: candidate.Provider, Model: model, Scope: "draft", Status: "not_configured", Detail: "Aucun fournisseur activé."}
	client := s.buildClient(candidate)
	if client == nil {
		writeJSON(w, 200, result)
		return
	}
	digest := s.aiConnectionDigest(candidate)
	s.ai.mu.Lock()
	s.ai.testSeq++
	seq := s.ai.testSeq
	result.Revision = s.ai.revision
	s.ai.mu.Unlock()
	timeout := llm.TestTimeout()
	if s.Cfg.AITestTimeout > 0 {
		timeout = time.Duration(s.Cfg.AITestTimeout) * time.Second
	}
	if timeout > 30*time.Second {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()
	started := time.Now()
	if ollama, ok := client.(*llm.Ollama); ok {
		err = ollama.Ping(ctx)
		result.Detail = "Instance joignable et modèle présent ; la génération n’est pas testée par ce contrôle."
	} else {
		tokens := 4
		if candidate.Provider == "openai" {
			tokens = 256
		}
		var text string
		text, err = client.Complete(ctx, "Réponds uniquement OK.", "ping", tokens, false)
		if err == nil && strings.TrimSpace(text) == "" {
			err = &llm.ProviderError{Kind: "response"}
		}
		result.Detail = "API joignable ; réponse obtenue à une requête synthétique."
	}
	elapsed := time.Since(started).Milliseconds()
	stamp := time.Now().UTC().Format(time.RFC3339Nano)
	result.LatencyMS = &elapsed
	result.CheckedAt = &stamp
	result.Status = "available"
	result.OK = err == nil
	if err != nil {
		result.Status = "unavailable"
		result.Code, result.Detail = safeAIDiagnosticError(err)
	}
	s.ai.mu.Lock()
	result.Revision = s.ai.revision
	if s.aiConnectionDigest(s.ai.config) == digest {
		result.Scope = "saved"
		result.Revision = s.ai.revision
	}
	if seq == s.ai.testSeq && !errors.Is(err, context.Canceled) {
		record := &aiTestRecord{digest: digest, Result: result}
		if result.Scope == "saved" {
			s.ai.lastSaved = record
		} else {
			s.ai.lastDraft = record
		}
	}
	s.ai.mu.Unlock()
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, result)
}
func (s *Server) adminOllamaModels(w http.ResponseWriter, r *http.Request) {
	endpoint := r.URL.Query().Get("url")
	if endpoint == "" {
		endpoint = s.aiBaseConfig().OllamaURL
	}
	client, err := s.aiEndpointPolicy().Client(endpoint)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	models, err := llm.ListOllamaModels(ctx, endpoint, client)
	if err != nil {
		_, detail := safeAIDiagnosticError(err)
		writeErr(w, 502, fmt.Errorf("%s", detail))
		return
	}
	filtered := []string{}
	seen := map[string]bool{}
	for _, model := range models {
		if aiModelPattern.MatchString(model) && !seen[model] && len(filtered) < 1000 {
			filtered = append(filtered, model)
			seen[model] = true
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, map[string]any{"models": filtered})
}
func (s *Server) aiMemoryInfo(last *aiDiagnostic) map[string]any {
	enabled, configured := false, false
	if memory := s.getMemory(); memory != nil {
		enabled, configured = memory.Enabled(), memory.Configured()
	} else {
		enabled = s.Cfg.MemoryEnabled
	}
	result := aiDiagnostic{Status: "disabled", Detail: "Mémoire désactivée côté serveur."}
	if enabled {
		result.Status = "not_configured"
		result.Detail = "Service de mémoire non configuré."
		if configured {
			result.Status = "not_tested"
			result.Detail = "Contrôle de santé non effectué depuis le démarrage."
			if last != nil {
				result = *last
			}
		}
	}
	return map[string]any{"enabled": enabled, "configured": configured, "required": false, "diagnostic": result}
}
func (s *Server) adminTestMemory(w http.ResponseWriter, r *http.Request) {
	info := s.aiMemoryInfo(nil)
	if info["enabled"] != true || info["configured"] != true {
		writeJSON(w, 200, info)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	start := time.Now()
	ready := s.getMemory().IsReady(ctx)
	elapsed := time.Since(start).Milliseconds()
	checked := time.Now().UTC().Format(time.RFC3339Nano)
	result := aiDiagnostic{OK: ready, Status: "available", CheckedAt: &checked, LatencyMS: &elapsed, Detail: "Service de mémoire joignable. Son contenu n’a pas été consulté."}
	if !ready {
		result.Status = "unavailable"
		result.Code = "memory_unavailable"
		result.Detail = "Mémoire indisponible. Vérifier le service Knowledge Graph et ses dépendances. L’assistant peut fonctionner sans mémoire."
	}
	s.ai.mu.Lock()
	s.ai.lastMemory = &result
	s.ai.mu.Unlock()
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, s.aiMemoryInfo(&result))
}
