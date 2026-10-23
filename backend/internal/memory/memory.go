// Package memory fournit un client HTTP vers le service Knowledge Graph Python.
// Ce service stocke des faits financiers persistants (habitudes, marchands récurrents,
// préférences de dépenses) qui enrichissent le raisonnement de l'agent IA.
package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
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
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.RequestTimeout},
	}
}

// IsReady vérifie que le service KG est joignable.
func (c *Client) IsReady(ctx context.Context) bool {
	if !c.cfg.Enabled || c.cfg.URL == "" {
		return false
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.URL+"/health", nil)
	if err != nil {
		return false
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// ────────────────────────── Learn ────────────────────────────

type learnIn struct {
	GroupID  string   `json:"group_id"`
	Facts    []factIn `json:"facts"`
	Source   string   `json:"source"`
}

type factIn struct {
	Fact  string `json:"fact"`
	Type_ string `json:"type"` // merchant | habit | preference
}

type learnOut struct {
	EntitiesAdded int `json:"entities_added"`
	EdgesAdded    int `json:"edges_added"`
}

// Learn ingère des faits financiers dans le graphe.
// Les faits sont formatés par l'appelant avant d'être envoyés.
func (c *Client) Learn(ctx context.Context, groupID, source string, facts []string) (int, int, error) {
	if !c.cfg.Enabled || c.cfg.URL == "" {
		return 0, 0, nil
	}
	factList := make([]factIn, 0, len(facts))
	for _, f := range facts {
		factList = append(factList, factIn{Fact: f, Type_: "habit"})
	}
	body := learnIn{GroupID: groupID, Facts: factList, Source: source}
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.URL+"/kg/learn", bytes.NewReader(b))
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		log.Printf("KG Learn failed: %v", err)
		return 0, 0, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, 0, nil
	}
	var out learnOut
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, 0, err
	}
	return out.EntitiesAdded, out.EdgesAdded, nil
}

// ────────────────────────── Query ────────────────────────────

type queryIn struct {
	Query   string `json:"query"`
	GroupID string `json:"group_id"`
	TopK    int    `json:"top_k"`
}

type kgResult struct {
	Fact   string  `json:"fact"`
	Source string  `json:"source"`
	Target string  `json:"target"`
	Type_  string  `json:"type"`
	Score  float64 `json:"score"`
}

type queryOut struct {
	Results []kgResult `json:"results"`
	Count   int        `json:"count"`
}

// Query cherche des faits pertinents dans le graphe.
func (c *Client) Query(ctx context.Context, groupID, query string, topK int) ([]kgResult, error) {
	if !c.cfg.Enabled || c.cfg.URL == "" {
		return nil, nil
	}
	body := queryIn{Query: query, GroupID: groupID, TopK: topK}
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.URL+"/kg/query", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		log.Printf("KG Query failed: %v", err)
		return nil, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}
	var out queryOut
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Results, nil
}

// ────────────────────────── Formatage prompt ─────────────────

// FactsToString formatte les résultats pour injection dans un prompt.
func FactsToString(results []kgResult) string {
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
		fmt.Sprintf("Total des %.0f transactions de cet import : %.2f€", len(res.Transactions), total),
	)

	return facts
}
