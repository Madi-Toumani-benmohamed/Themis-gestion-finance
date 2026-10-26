package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

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
		MistralModel:   cfg.MistralModel,
		FreeLLMURL:     cfg.FreeLLMURL,
		FreeLLMModel:   cfg.FreeLLMModel,
		AutoCategorize: cfg.AIAutoCategorize,
	}
}

// buildClient construit le client LLM correspondant à la config.
func (s *Server) buildClient(c AIConfig) llm.Client {
	switch c.Provider {
	case "openai":
		c.OpenAIModel = s.withOpenAIDefaults(c).OpenAIModel
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
		client, err := s.aiEndpointPolicy().Client(c.OllamaURL)
		if err != nil {
			return nil
		}
		return &llm.Ollama{URL: c.OllamaURL, Model: c.OllamaModel, HTTPClient: client}
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
		client, err := s.aiEndpointPolicy().Client(baseURL)
		if err != nil {
			return nil
		}
		return &llm.FreeLLM{APIKey: key, BaseURL: baseURL, Model: c.FreeLLMModel, HTTPClient: client}
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
	saveMu      sync.Mutex
	revision    string
	source      string
	configError string
	testSeq     uint64
	lastSaved   *aiTestRecord
	lastDraft   *aiTestRecord
	lastMemory  *aiDiagnostic
	mu          sync.RWMutex
	config      AIConfig
	client      llm.Client
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

// ---------- Statistiques ----------

func (s *Server) adminStats(w http.ResponseWriter, r *http.Request) {
	owner := s.ownerID(r)
	stats := map[string]any{}
	queries := map[string]string{
		"statements":   `SELECT count(*) FROM statements WHERE owner_id = $1`,
		"transactions": `SELECT count(*) FROM transactions WHERE owner_id = $1`,
		"categorized":  `SELECT count(*) FROM transactions WHERE domain_id IS NOT NULL AND owner_id = $1`,
		// Keep the established frontend/API field name. Domains replaced the
		// legacy category table, but the admin panel still consumes categories.
		"categories": `SELECT count(*) FROM domains WHERE parent_id IS NULL AND owner_id = $1`,
		"rules":      `SELECT count(*) FROM rules WHERE owner_id = $1`,
		"analyses":   `SELECT count(*) FROM analyses WHERE owner_id = $1`,
		"budgets":    `SELECT count(*) FROM budgets WHERE owner_id = $1`,
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
	return s.applyAllRulesForStatement(ownerID, 0)
}

func (s *Server) applyAllRulesForStatement(ownerID, statementID int64) (int, error) {
	// On parcourt les règles de l'utilisateur (avec leurs tags) et on applique
	// chacune : catégorie sur les transactions non classées + tags sur toutes
	// celles qui correspondent. Une seule logique, partagée avec la création.
	rows, err := s.DB.Query(`SELECT r.pattern,r.domain_id,r.tags FROM rules r JOIN domains d ON d.id=r.domain_id AND d.owner_id=r.owner_id WHERE r.owner_id=$1 ORDER BY length(r.pattern) DESC,r.id`, ownerID)
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
	err = rows.Err()
	rows.Close()
	if err != nil {
		return 0, err
	}
	total := 0
	for _, rl := range rules {
		n, err := s.applyRuleToTransactionsScoped(ownerID, rl.pattern, rl.dom, rl.tags, statementID)
		if err != nil {
			return total, err
		}
		total += int(n)
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
