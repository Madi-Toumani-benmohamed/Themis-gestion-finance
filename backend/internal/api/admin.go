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
)

// ---------- Configuration IA modifiable à chaud ----------

// AIConfig est la configuration IA effective, modifiable depuis la page
// d'admin sans redémarrage. Persistée dans settings (clé "ai_config").
type AIConfig struct {
	Provider       string `json:"provider"` // "none" | "ollama" | "anthropic"
	OllamaURL      string `json:"ollama_url"`
	OllamaModel    string `json:"ollama_model"`
	AnthropicModel string `json:"anthropic_model"`
	// AnthropicKey vide = utiliser ANTHROPIC_API_KEY de l'environnement.
	// Jamais renvoyée en clair par l'API (masquée).
	AnthropicKey   string `json:"anthropic_key,omitempty"`
	AutoCategorize bool   `json:"auto_categorize"`
}

func aiConfigFromEnv(cfg config.Config) AIConfig {
	return AIConfig{
		Provider:       cfg.AIProvider,
		OllamaURL:      cfg.OllamaURL,
		OllamaModel:    cfg.OllamaModel,
		AnthropicModel: cfg.AnthropicModel,
		AutoCategorize: cfg.AIAutoCategorize,
	}
}

// buildClient construit le client LLM correspondant à la config.
func (s *Server) buildClient(c AIConfig) llm.Client {
	switch c.Provider {
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
	default:
		return nil
	}
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
		"ollama_url":        c.OllamaURL,
		"ollama_model":      c.OllamaModel,
		"anthropic_model":   c.AnthropicModel,
		"anthropic_key_set": c.AnthropicKey != "" || s.Cfg.AnthropicKey != "",
		"auto_categorize":   c.AutoCategorize,
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
	switch in.Provider {
	case "none", "ollama", "anthropic":
	default:
		writeErr(w, http.StatusBadRequest, fmt.Errorf(`provider doit être "none", "ollama" ou "anthropic"`))
		return
	}
	if in.Provider == "ollama" && in.OllamaURL == "" {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("ollama_url requis"))
		return
	}
	// clé non fournie → conserver l'existante (le champ masqué n'est jamais renvoyé)
	if in.AnthropicKey == "" {
		in.AnthropicKey = s.getAIConfig().AnthropicKey
	}
	if in.OllamaModel == "" {
		in.OllamaModel = s.Cfg.OllamaModel
	}
	if in.AnthropicModel == "" {
		in.AnthropicModel = s.Cfg.AnthropicModel
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
		client = s.buildClient(in)
	}
	if client == nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "detail": "aucun fournisseur configuré"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if o, ok := client.(*llm.Ollama); ok {
		if err := o.Ping(ctx); err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"ok": false, "provider": client.Name(), "detail": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "provider": client.Name(), "detail": "instance joignable, modèle présent"})
		return
	}
	// Anthropic : mini-complétion (coût négligeable)
	if _, err := client.Complete(ctx, "Réponds uniquement OK.", "ping", 4, false); err != nil {
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
		"statements":    `SELECT count(*) FROM statements WHERE owner_id = $1`,
		"transactions":  `SELECT count(*) FROM transactions WHERE owner_id = $1`,
		"categorized":   `SELECT count(*) FROM transactions WHERE domain_id IS NOT NULL AND owner_id = $1`,
		"domains":       `SELECT count(*) FROM domains WHERE parent_id IS NULL AND owner_id = $1`,
		"rules":         `SELECT count(*) FROM rules WHERE owner_id = $1`,
		"analyses":      `SELECT count(*) FROM analyses WHERE owner_id = $1`,
		"budgets":       `SELECT count(*) FROM budgets WHERE owner_id = $1`,
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
