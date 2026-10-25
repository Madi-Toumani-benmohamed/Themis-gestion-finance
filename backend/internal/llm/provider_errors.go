package llm

import (
	"context"
	"errors"
	"fmt"
	"net"
)

// ProviderError deliberately excludes response bodies, URLs, tokens and prompts.
type ProviderError struct {
	Kind   string
	Status int
}

func (e *ProviderError) Error() string {
	switch e.Kind {
	case "model_missing":
		return "modèle configuré absent du serveur Ollama"
	case "response":
		return "réponse du fournisseur invalide ou vide"
	case "network":
		return "connexion au fournisseur impossible (réseau ou TLS)"
	}
	switch e.Status {
	case 401:
		return "clé API invalide ou révoquée (HTTP 401)"
	case 403:
		return "accès au modèle refusé (HTTP 403)"
	case 429:
		return "quota ou limite de requêtes atteint (HTTP 429)"
	default:
		return fmt.Sprintf("requête fournisseur refusée (HTTP %d)", e.Status)
	}
}
func providerNetworkError(ctx context.Context, err error) error {
	var n net.Error
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) || (errors.As(err, &n) && n.Timeout()) {
		return fmt.Errorf("délai du fournisseur dépassé : %w", context.DeadlineExceeded)
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return context.Canceled
	}
	return &ProviderError{Kind: "network"}
}
