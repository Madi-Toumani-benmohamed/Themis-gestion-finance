// Package memory fournit un client HTTP léger vers le service Python Graphiti.
// Il est appelé par le handler chat pour enrichir le contexte et par
// l'import de transactions pour apprendre les patterns financiers.
package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// Config regroupe les paramètres du service mémoire.
type Config struct {
	URL           string // ex: http://memory:8100
	Enabled       bool
	RequestTimeout time.Duration
}

// Load lit la config depuis les variables d'environnement.
func Load() Config {
	url := os.Getenv("MEMORY_SERVICE_URL")
	if url == "" {
		url = "http://localhost:8100"
	}
	enabled := os.Getenv("MEMORY_ENABLED") != "false"
	return Config{
		URL:            url,
		Enabled:        enabled,
		RequestTimeout: 10 * time.Second,
	}
}

// Client est l'interface vers le service mémoire.
type Client struct {
	cfg    Config
	http   *http.Client
}

// New crée un client prêt à l'emploi.
func New(cfg Config) *Client {
	return &Client{
		cfg: cfg,
		http: &http.Client{
			Timeout: cfg.RequestTimeout,
		},
	}
}

// IsReady vérifie que le service est joignable.
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

// ────────────────────────── Episode ──────────────────────────

type episodeIn struct {
	Name                string `json:"name"`
	Body                string `json:"body"`
	SourceDescription   string `json:"source_description"`
	GroupID             string `json:"group_id"`
	ReferenceTime       string `json:"reference_time,omitempty"`
	EpisodeType         string `json:"episode_type,omitempty"`
}

type episodeOut struct {
	EpisodeUUID     string `json:"episode_uuid"`
	EntitiesAdded   int    `json:"entities_added"`
	EdgesAdded      int    `json:"edges_added"`
	CommunitiesUpd  int    `json:"communities_updated"`
}

// AddEpisode ajoute un épisode au graphe et retourne les counts d'entités.
func (c *Client) AddEpisode(ctx context.Context, groupID, name, body, srcDesc string) (entities, edges int, err error) {
	if !c.cfg.Enabled || c.cfg.URL == "" {
		return 0, 0, nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	reqBody := episodeIn{
		Name:              name,
		Body:              body,
		SourceDescription: srcDesc,
		GroupID:           groupID,
		ReferenceTime:     now,
		EpisodeType:       "message",
	}
	b, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.URL+"/memory/episodes", bytes.NewReader(b))
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		log.Printf("memory AddEpisode failed: %v", err)
		return 0, 0, nil // non bloquant
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("memory AddEpisode status=%d: %s", resp.StatusCode, string(body))
		return 0, 0, nil
	}
	var out episodeOut
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, 0, err
	}
	return out.EntitiesAdded, out.EdgesAdded, nil
}

// ────────────────────────── Search ──────────────────────────

type searchIn struct {
	Query      string `json:"query"`
	GroupID    string `json:"group_id"`
	NumResults int    `json:"num_results"`
}

type searchResult struct {
	Fact           string  `json:"fact"`
	SourceEntity   string  `json:"source_entity"`
	TargetEntity   string  `json:"target_entity"`
	Relationship   string  `json:"relationship"`
	RelevanceScore float64 `json:"relevance_score"`
}

type searchOut struct {
	Results []searchResult `json:"results"`
	Count   int            `json:"count"`
}

// Search fait une requête hybride sur le graphe.
func (c *Client) Search(ctx context.Context, groupID, query string, n int) ([]searchResult, error) {
	if !c.cfg.Enabled || c.cfg.URL == "" {
		return nil, nil
	}
	reqBody := searchIn{Query: query, GroupID: groupID, NumResults: n}
	b, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.URL+"/memory/search", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		log.Printf("memory Search failed: %v", err)
		return nil, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}
	var out searchOut
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Results, nil
}

// ────────────────────────── Triples ──────────────────────────

type tripletIn struct {
	Source   string `json:"source"`
	Relation string `json:"relation"`
	Target   string `json:"target"`
	GroupID  string `json:"group_id"`
}

// AddTriple insère une relation explicite (merchant → catégorie, etc.).
func (c *Client) AddTriple(ctx context.Context, groupID, source, relation, target string) error {
	if !c.cfg.Enabled || c.cfg.URL == "" {
		return nil
	}
	reqBody := tripletIn{Source: source, Relation: relation, Target: target, GroupID: groupID}
	b, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.URL+"/memory/triplets", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		log.Printf("memory AddTriple failed: %v", err)
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("memory AddTriple status=%d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// ────────────────────────── Cleanup ──────────────────────────

// ClearGroup supprime toutes les données d'un groupe utilisateur.
func (c *Client) ClearGroup(ctx context.Context, groupID string) error {
	if !c.cfg.Enabled || c.cfg.URL == "" {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/memory/clear/%s", c.cfg.URL, groupID), nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		log.Printf("memory ClearGroup failed: %v", err)
		return nil
	}
	defer resp.Body.Close()
	_ = resp.Body.Close()
	return nil
}

// FactsToString formatte les résultats de recherche pour les injecter dans un prompt.
func FactsToString(results []searchResult) string {
	if len(results) == 0 {
		return ""
	}
	var b bytes.Buffer
	b.WriteString("\n## Faits mémorisés pertinents :\n")
	for _, r := range results {
		b.WriteString(fmt.Sprintf("- %s\n", r.Fact))
	}
	return b.String()
}

// BuildMemoryPrompt enrichit le prompt système avec les faits du graphe.
func (c *Client) BuildMemoryContext(ctx context.Context, groupID, userQuestion string) string {
	results, err := c.Search(ctx, groupID, userQuestion, 5)
	if err != nil || len(results) == 0 {
		return ""
	}
	return FactsToString(results)
}
