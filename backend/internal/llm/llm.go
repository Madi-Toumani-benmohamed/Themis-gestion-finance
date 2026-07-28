// Package llm fournit une abstraction minimale au-dessus des fournisseurs
// d'IA : API Anthropic (cloud) ou Ollama (100 % local). Toutes les
// fonctionnalités IA de solde passent par l'interface Client, ce qui permet
// aux utilisateurs de garder leurs données bancaires sur leur machine.
package llm

import (
	"context"

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

// FromConfig construit le client selon la configuration.
// Retourne nil si aucune IA n'est configurée (l'appli reste fonctionnelle,
// seuls le fallback de parsing et les conseils mensuels sont désactivés).
func FromConfig(cfg config.Config) Client {
	switch cfg.AIProvider {
	case "anthropic":
		return &Anthropic{APIKey: cfg.AnthropicKey, Model: cfg.AnthropicModel}
	case "ollama":
		return &Ollama{URL: cfg.OllamaURL, Model: cfg.OllamaModel}
	default:
		return nil
	}
}
