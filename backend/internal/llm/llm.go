// Package llm fournit une abstraction minimale au-dessus des fournisseurs
// d'IA : API Anthropic (cloud) ou Ollama (100 % local). Toutes les
// fonctionnalités IA de solde passent par l'interface Client, ce qui permet
// aux utilisateurs de garder leurs données bancaires sur leur machine.
package llm

import (
	"context"
	"time"

	"themis/internal/config"
)

// Client est un fournisseur de complétion de texte.
type Client interface {
	// Complete envoie un prompt système + utilisateur et retourne le texte.
	// wantJSON demande au fournisseur de forcer une sortie JSON quand il le
	// supporte (Ollama) ; sinon c'est au prompt de l'exiger.
	Complete(ctx context.Context, system, user string, maxTokens int, wantJSON bool) (string, error)
	// Name identifie le fournisseur pour les logs et /api/health.
	Name() string
}

// ctxKey est le type des clés de contexte de ce package (évite les collisions).
type ctxKey int

const thinkingFlagKey ctxKey = 0

// WithThinkingFlag attache au contexte un drapeau que le client renseigne à true
// s'il a dû DÉSACTIVER le raisonnement (modèle qui ne le supporte pas, ou repli
// sur réponse vide). Permet à l'appelant d'informer l'utilisateur.
func WithThinkingFlag(ctx context.Context, flag *bool) context.Context {
	return context.WithValue(ctx, thinkingFlagKey, flag)
}

// markThinkingDisabled positionne le drapeau du contexte, s'il existe.
func markThinkingDisabled(ctx context.Context) {
	if f, ok := ctx.Value(thinkingFlagKey).(*bool); ok && f != nil {
		*f = true
	}
}

// ConfigTimeouts applique les timeouts depuis la config globale.
func ConfigTimeouts(cfg config.Config) {
	_ollamaTimeout = cfg.AIOllamaTimeout
	_cloudTimeout = cfg.AICloudTimeout
	_chatTimeout = cfg.AIChatTimeout
	_categorizeTimeout = cfg.AICategorizeTimeout
	_narrativeTimeout = cfg.AINarrativeTimeout
	_testTimeout = cfg.AITestTimeout
}

var (
	_ollamaTimeout     int
	_cloudTimeout      int
	_chatTimeout       int
	_categorizeTimeout int
	_narrativeTimeout  int
	_testTimeout       int
)

func ollamaTimeout() time.Duration {
	if _ollamaTimeout > 0 {
		return time.Duration(_ollamaTimeout) * time.Second
	}
	return 600 * time.Second
}
func cloudTimeout() time.Duration {
	if _cloudTimeout > 0 {
		return time.Duration(_cloudTimeout) * time.Second
	}
	return 120 * time.Second
}
func ChatTimeout() time.Duration {
	if _chatTimeout > 0 {
		return time.Duration(_chatTimeout) * time.Second
	}
	return 480 * time.Second
}
func CategorizeTimeout() time.Duration {
	if _categorizeTimeout > 0 {
		return time.Duration(_categorizeTimeout) * time.Second
	}
	return 300 * time.Second
}
func NarrativeTimeout() time.Duration {
	if _narrativeTimeout > 0 {
		return time.Duration(_narrativeTimeout) * time.Second
	}
	return 120 * time.Second
}
func TestTimeout() time.Duration {
	if _testTimeout > 0 {
		return time.Duration(_testTimeout) * time.Second
	}
	return 30 * time.Second
}

// FromConfig construit le client selon la configuration.
// Retourne nil si aucune IA n'est configurée (l'appli reste fonctionnelle,
// seuls le fallback de parsing et les conseils mensuels sont désactivés).
func FromConfig(cfg config.Config) Client {
	switch cfg.AIProvider {
	case "openai":
		if cfg.OpenAIKey == "" {
			return nil
		}
		return &OpenAI{APIKey: cfg.OpenAIKey, Model: cfg.OpenAIModel}
	case "anthropic":
		return &Anthropic{APIKey: cfg.AnthropicKey, Model: cfg.AnthropicModel}
	case "ollama":
		return &Ollama{URL: cfg.OllamaURL, Model: cfg.OllamaModel}
	default:
		return nil
	}
}
