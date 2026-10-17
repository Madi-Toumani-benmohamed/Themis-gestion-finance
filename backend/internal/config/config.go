package config

import "os"

type Config struct {
	Port        string
	DatabaseURL string

// IA — plusieurs fournisseurs possibles :
//  - "anthropic"    : API Claude (cloud)
//  - "ollama"       : modèle local, aucune donnée ne quitte la machine
//  - "mistral"      : API Mistral AI (cloud)
//  - "freellm"      : agrégateur OpenRouter/FreeLLM (compatible OpenAI)
//  - "none"         : fonctionnalités IA désactivées
// AIProvider est résolu automatiquement si AI_PROVIDER n'est pas défini :
// ollama si OLLAMA_URL est défini, sinon mistral si MISTRAL_API_KEY,
// sinon freellm si FREELLM_API_KEY, sinon anthropic si ANTHROPIC_API_KEY,
// sinon none. L'ordre privilégie le local : des données bancaires ne doivent
// partir dans le cloud que par choix explicite.
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
	// Catégorisation IA automatique des transactions restées sans catégorie
	// après les règles, à chaque import (nécessite un fournisseur d'IA).
	AIAutoCategorize bool

	// Cookie de session marqué Secure (à activer derrière HTTPS en production).
	CookieSecure bool
}

func get(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func Load() Config {
	cfg := Config{
		Port:           get("PORT", "8080"),
		DatabaseURL:    get("DATABASE_URL", "postgres://themis:themis@localhost:5432/themis?sslmode=disable"),
		AnthropicKey:   os.Getenv("ANTHROPIC_API_KEY"),
		AnthropicModel: get("ANTHROPIC_MODEL", "claude-sonnet-4-6"),
		OllamaURL:      os.Getenv("OLLAMA_URL"),
		OllamaModel:    get("OLLAMA_MODEL", "qwen2.5:14b"),
		MistralKey:     os.Getenv("MISTRAL_API_KEY"),
		MistralModel:   get("MISTRAL_MODEL", "mistral-small-latest"),
		FreeLLMKey:     os.Getenv("FREELLM_API_KEY"),
		FreeLLMURL:     get("FREELLM_URL", "http://freellm.local:3001/v1"),
		FreeLLMModel:   get("FREELLM_MODEL", "auto:fast"),
	}
	cfg.AIAutoCategorize = get("AI_AUTO_CATEGORIZE", "true") != "false"
	cfg.CookieSecure = os.Getenv("COOKIE_SECURE") == "true"
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
		default:
			cfg.AIProvider = "none"
		}
	}
	return cfg
}
