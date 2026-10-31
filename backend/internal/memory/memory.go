// Package memory fournit un client HTTP vers le service Knowledge Graph Python.
// Ce service stocke des faits financiers persistants (habitudes, marchands récurrents,
// préférences de dépenses) qui enrichissent le raisonnement de l'agent IA.
package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"themis/internal/models"
)

// Config regroupe les paramètres du service KG.
type Config struct {
	URL            string // ex: http://knowledge-graph:8100
	Enabled        bool
	RequestTimeout time.Duration
}

// Load lit la config depuis les variables d'environnement.
func Load() Config {
	url := os.Getenv("MEMORY_SERVICE_URL")
	if url == "" {
		url = "http://localhost:8100"
	}
	return Config{
		URL:            url,
		Enabled:        os.Getenv("MEMORY_ENABLED") != "false",
		RequestTimeout: 10 * time.Second,
	}
}

// Client est l'interface vers le service de connaissances.
type Client struct {
	cfg  Config
	http *http.Client
}

// New crée un client prêt à l'emploi.
func New(cfg Config) *Client {
	cfg.URL = strings.TrimRight(cfg.URL, "/")
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.RequestTimeout},
	}
}

// Enabled and Configured report configuration without performing network IO.
func (c *Client) Enabled() bool    { return c.cfg.Enabled }
func (c *Client) Configured() bool { return c.cfg.URL != "" }

// IsReady checks only the health endpoint; no graph facts are read or changed.
func (c *Client) IsReady(ctx context.Context) bool {
	if !c.cfg.Enabled || c.cfg.URL == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.URL+"/health", nil)
	if err != nil {
		return false
	}
	client := *c.http
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := client.Do(req)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return false
	}
	var body struct {
		Status string `json:"status"`
	}
	return json.NewDecoder(io.LimitReader(response.Body, 4096)).Decode(&body) == nil && body.Status == "ok"
}

// ────────────────────────── Learn ────────────────────────────

// Result is a scoped Graphiti relation, not financial evidence or a model score.
type Result struct {
	Sources    []Source `json:"sources"`
	ID         string   `json:"id"`
	GroupID    string   `json:"group_id"`
	Fact       string   `json:"fact"`
	Source     string   `json:"source"`
	Target     string   `json:"target"`
	Type       string   `json:"type"`
	Score      *float64 `json:"score"`
	CreatedAt  *string  `json:"created_at"`
	ValidAt    *string  `json:"valid_at"`
	InvalidAt  *string  `json:"invalid_at"`
	ExpiredAt  *string  `json:"expired_at"`
	EpisodeIDs []string `json:"episode_ids"`
	Truncated  bool     `json:"truncated"`
}

func (c *Client) Learn(ctx context.Context, groupID, source string, facts []string) (int, int, error) {
	return c.LearnSource(ctx, groupID, source, "manual", facts)
}
func (c *Client) Query(ctx context.Context, groupID, query string, topK int) ([]Result, error) {
	return c.query(ctx, groupID, query, topK)
}

// ────────────────────────── Formatage prompt ─────────────────

// FactsToString formatte les résultats pour injection dans un prompt.
func FactsToString(results []Result) string {
	if len(results) == 0 {
		return ""
	}
	var b bytes.Buffer
	b.WriteString("\n## Connaissances financières de l'utilisateur :\n")
	for _, r := range results {
		b.WriteString(fmt.Sprintf("- %s\n", r.Fact))
	}
	return b.String()
}

// SearchFacts fait une recherche et retourne le texte formaté.
func (c *Client) SearchFacts(ctx context.Context, groupID, query string, topK int) string {
	results, err := c.Query(ctx, groupID, query, topK)
	if err != nil || len(results) == 0 {
		return ""
	}
	return FactsToString(results)
}

// ────────────────────────── Fact extraction ─────────────────

// BuildFinancialFacts extrait des faits d'un import de transactions.
// Chaque fait est une phrase simple que Graphiti ingestera dans le graphe.
func BuildFinancialFacts(res *models.ParseResult, ownerID int64) []string {
	if res == nil || len(res.Transactions) == 0 {
		return nil
	}

	var facts []string
	for _, t := range res.Transactions {
		// Un fait par transaction : le marchand, le montant, le type (débit/crédit)
		dir := "débit"
		if t.Amount > 0 {
			dir = "crédit"
		}
		facts = append(facts,
			fmt.Sprintf("Transaction le %s : %s (%s) de %.2f€ (%s)",
				t.Date, t.Label, t.Merchant, t.Amount, dir),
		)
	}

	// Synthèse globale
	var total float64
	for _, t := range res.Transactions {
		total += t.Amount
	}
	facts = append(facts,
		fmt.Sprintf("Total des %d transactions de cet import : %.2f€", len(res.Transactions), total),
	)

	return facts
}
