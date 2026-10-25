package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AITrustedEndpoints []string
	Port               string
	DatabaseURL        string

	// IA — plusieurs fournisseurs possibles :
	//  - "anthropic"    : API Claude (cloud)
	//  - "ollama"       : modèle local, aucune donnée ne quitte la machine
	//  - "mistral"      : API Mistral AI (cloud)
	//  - "freellm"      : agrégateur OpenRouter/FreeLLM (compatible OpenAI)
	//  - "openai"       : API OpenAI (cloud)
	//  - "none"         : fonctionnalités IA désactivées
	// AIProvider est résolu automatiquement si AI_PROVIDER n'est pas défini :
	// ollama si OLLAMA_URL est défini, sinon mistral si MISTRAL_API_KEY,
	// sinon freellm si FREELLM_API_KEY, sinon anthropic si ANTHROPIC_API_KEY,
	// sinon openai si OPENAI_API_KEY, sinon none. L'ordre privilégie le local : des données bancaires ne doivent
	// partir dans le cloud que par choix explicite.
	OpenAIKey      string
	OpenAIModel    string
	AIProvider     string
	AnthropicKey   string
	AnthropicModel string
	OllamaURL      string
	OllamaModel    string
	// Mistral AI
	MistralKey   string
	MistralModel string
	// FreeLLM (service local agrégateur de modèles)
	// Compatible OpenAI, supporte "auto" et "auto:fast" comme modèles
	FreeLLMKey   string
	FreeLLMURL   string
	FreeLLMModel string
	// Propositions IA séparées pour les transactions restées sans catégorie
	// après les règles à l'import. Leur application reste une action manuelle.
	AIAutoCategorize bool

	// Timeout des requêtes IA (en secondes). 0 = pas de timeout.
	AIOllamaTimeout     int // timeout HTTP pour Ollama (défaut: 600s)
	AICloudTimeout      int // timeout HTTP pour API cloud (défaut: 120s)
	AIChatTimeout       int // timeout contexte orchestrateur chat (défaut: 480s)
	AICategorizeTimeout int // timeout catégorisation auto (défaut: 300s)
	AINarrativeTimeout  int // timeout synthèse narrative (défaut: 120s)
	AITestTimeout       int // timeout test connexion admin (défaut: 30s)

	// Memory Service (Graphiti)
	MemoryServiceURL string // URL du service mémoire Python (défaut: http://memory:8100)
	MemoryEnabled    bool   // true si MEMORY_ENABLED != "false"

	// Cookie de session marqué Secure (à activer derrière HTTPS en production).
	CookieSecure bool
}

func get(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func parseInt(s string) (int, error) {
	return strconv.Atoi(s)
}

func Load() Config {
	cfg := Config{
		Port:               get("PORT", "8080"),
		AITrustedEndpoints: strings.Split(os.Getenv("AI_TRUSTED_ENDPOINTS"), ","),
		DatabaseURL:        get("DATABASE_URL", "postgres://themis:themis@localhost:5432/themis?sslmode=disable"),
		OpenAIKey:          os.Getenv("OPENAI_API_KEY"),
		OpenAIModel:        get("OPENAI_MODEL", "gpt-4.1-mini"),
		AnthropicKey:       os.Getenv("ANTHROPIC_API_KEY"),
		AnthropicModel:     get("ANTHROPIC_MODEL", "claude-sonnet-4-6"),
		OllamaURL:          os.Getenv("OLLAMA_URL"),
		OllamaModel:        get("OLLAMA_MODEL", "qwen2.5:14b"),
		MistralKey:         os.Getenv("MISTRAL_API_KEY"),
		MistralModel:       get("MISTRAL_MODEL", "mistral-small-latest"),
		FreeLLMKey:         os.Getenv("FREELLM_API_KEY"),
		FreeLLMURL:         get("FREELLM_URL", "http://freellm.local:3001/v1"),
		FreeLLMModel:       get("FREELLM_MODEL", "auto:fast"),
	}
	cfg.AIAutoCategorize = get("AI_AUTO_CATEGORIZE", "true") != "false"
	cfg.CookieSecure = os.Getenv("COOKIE_SECURE") == "true"

	// Timeouts IA en secondes (0 = pas de timeout)
	if v := get("AI_OLLAMA_TIMEOUT", ""); v != "" {
		if n, err := parseInt(v); err == nil && n > 0 {
			cfg.AIOllamaTimeout = n
		}
	}
	if v := get("AI_CLOUD_TIMEOUT", ""); v != "" {
		if n, err := parseInt(v); err == nil && n > 0 {
			cfg.AICloudTimeout = n
		}
	}
	if v := get("AI_CHAT_TIMEOUT", ""); v != "" {
		if n, err := parseInt(v); err == nil && n > 0 {
			cfg.AIChatTimeout = n
		}
	}
	if v := get("AI_CATEGORIZE_TIMEOUT", ""); v != "" {
		if n, err := parseInt(v); err == nil && n > 0 {
			cfg.AICategorizeTimeout = n
		}
	}
	if v := get("AI_NARRATIVE_TIMEOUT", ""); v != "" {
		if n, err := parseInt(v); err == nil && n > 0 {
			cfg.AINarrativeTimeout = n
		}
	}
	if v := get("AI_TEST_TIMEOUT", ""); v != "" {
		if n, err := parseInt(v); err == nil && n > 0 {
			cfg.AITestTimeout = n
		}
	}

	cfg.MemoryServiceURL = os.Getenv("MEMORY_SERVICE_URL")
	if cfg.MemoryServiceURL == "" {
		cfg.MemoryServiceURL = "http://localhost:8100"
	}
	cfg.MemoryEnabled = os.Getenv("MEMORY_ENABLED") != "false"

	cfg.AIProvider = os.Getenv("AI_PROVIDER")
	if cfg.AIProvider == "" {
		switch {
		case cfg.OllamaURL != "":
			cfg.AIProvider = "ollama"
		case cfg.MistralKey != "":
			cfg.AIProvider = "mistral"
		case cfg.FreeLLMKey != "":
			cfg.AIProvider = "freellm"
		case cfg.AnthropicKey != "":
			cfg.AIProvider = "anthropic"
		case cfg.OpenAIKey != "":
			cfg.AIProvider = "openai"
		default:
			cfg.AIProvider = "none"
		}
	}
	return cfg
}
