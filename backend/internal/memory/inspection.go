package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Failure struct{ Code string }

func (e *Failure) Error() string {
	return map[string]string{
		"model_missing": "Modèle ou endpoint introuvable. Vérifier les modèles de génération et d’embedding du Knowledge Graph.",
		"model_access":  "Accès au modèle refusé par le fournisseur du Knowledge Graph.",
		"model_limited": "Limite ou quota du modèle du Knowledge Graph atteint.",
		"disabled":      "Mémoire désactivée.", "not_configured": "Service mémoire non configuré.",
		"timeout": "Délai de réponse de la mémoire dépassé.", "cancelled": "Requête mémoire annulée.",
		"network": "Connexion au Knowledge Graph impossible.", "upstream": "Le Knowledge Graph a signalé un échec.",
		"protocol": "Réponse du Knowledge Graph incompatible ou incomplète.", "scope": "Réponse mémoire hors du périmètre utilisateur refusée.",
	}[e.Code]
}
func FailureCode(err error) string {
	var f *Failure
	if errors.As(err, &f) {
		return f.Code
	}
	return "upstream"
}
func validGroup(group string) bool {
	n, e := strconv.ParseInt(group, 10, 64)
	return e == nil && n > 0 && strconv.FormatInt(n, 10) == group
}
func (c *Client) request(ctx context.Context, method, path string, input, output any) error {
	if !c.Enabled() {
		return &Failure{"disabled"}
	}
	if !c.Configured() {
		return &Failure{"not_configured"}
	}
	var body io.Reader
	if input != nil {
		data, e := json.Marshal(input)
		if e != nil {
			return &Failure{"protocol"}
		}
		body = bytes.NewReader(data)
	}
	req, e := http.NewRequestWithContext(ctx, method, c.cfg.URL+path, body)
	if e != nil {
		return &Failure{"network"}
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := *c.http
	// The caller owns a bounded operation context. An LLM-backed ingestion can
	// legitimately exceed the generic 10s transport timeout used for health.
	if method == http.MethodPost && path == "/kg/learn" {
		client.Timeout = 0
	}
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	res, e := client.Do(req)
	if e != nil {
		if errors.Is(e, context.Canceled) {
			return &Failure{"cancelled"}
		}
		var netErr net.Error
		if errors.Is(e, context.DeadlineExceeded) || errors.As(e, &netErr) && netErr.Timeout() {
			return &Failure{"timeout"}
		}
		return &Failure{"network"}
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		if res.StatusCode == 408 || res.StatusCode == 504 {
			return &Failure{"timeout"}
		}
		var remote struct {
			Detail struct {
				Code string `json:"code"`
			} `json:"detail"`
		}
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		if json.Unmarshal(body, &remote) == nil {
			switch remote.Detail.Code {
			case "timeout", "model_missing", "model_access", "model_limited":
				return &Failure{remote.Detail.Code}
			}
		}
		return &Failure{"upstream"}
	}
	data, e := io.ReadAll(io.LimitReader(res.Body, 512*1024+1))
	if e != nil {
		if errors.Is(e, context.Canceled) {
			return &Failure{"cancelled"}
		}
		var ne net.Error
		if errors.Is(e, context.DeadlineExceeded) || errors.As(e, &ne) && ne.Timeout() {
			return &Failure{"timeout"}
		}
		return &Failure{"upstream"}
	}
	if len(data) > 512*1024 {
		return &Failure{"protocol"}
	}
	if json.Unmarshal(data, output) != nil {
		return &Failure{"protocol"}
	}
	return nil
}
func (c *Client) LearnSource(ctx context.Context, group, source, sourceRef string, facts []string) (int, int, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if !validGroup(group) || len(facts) == 0 || len(facts) > 1000 {
		return 0, 0, &Failure{"protocol"}
	}
	items := make([]map[string]string, 0, len(facts))
	for _, f := range facts {
		if len([]rune(f)) == 0 || len([]rune(f)) > 4096 {
			return 0, 0, &Failure{"protocol"}
		}
		items = append(items, map[string]string{"fact": f, "type": "habit"})
	}
	input := map[string]any{"group_id": group, "facts": items, "source": source, "source_ref": sourceRef}
	var out struct {
		Entities *int `json:"entities_added"`
		Edges    *int `json:"edges_added"`
	}
	if e := c.request(ctx, http.MethodPost, "/kg/learn", input, &out); e != nil {
		return 0, 0, e
	}
	if out.Entities == nil || out.Edges == nil || *out.Entities < 0 || *out.Edges < 0 {
		return 0, 0, &Failure{"protocol"}
	}
	return *out.Entities, *out.Edges, nil
}
func validateResults(items []Result, group string, limit int) error {
	if len(items) > limit {
		return &Failure{"protocol"}
	}
	for _, r := range items {
		if len(r.ID) > 128 || len(r.Source) > 128 || len(r.Target) > 128 || len(r.Type) > 128 || len(r.Sources) > 5 {
			return &Failure{"protocol"}
		}
		for _, source := range r.Sources {
			if source.GroupID != group {
				return &Failure{"scope"}
			}
		}
		if r.GroupID != group {
			return &Failure{"scope"}
		}
		if r.ID == "" || r.Source == "" || r.Target == "" || strings.TrimSpace(r.Fact) == "" || len([]rune(r.Fact)) > 4096 || len(r.EpisodeIDs) > 5 {
			return &Failure{"protocol"}
		}
	}
	return nil
}
func (c *Client) query(ctx context.Context, group, query string, topK int) ([]Result, error) {
	if !validGroup(group) || strings.TrimSpace(query) == "" || len([]rune(query)) > 500 || topK < 1 || topK > 5 {
		return nil, &Failure{"protocol"}
	}
	var out struct {
		Results []Result `json:"results"`
		Count   *int     `json:"count"`
	}
	if e := c.request(ctx, http.MethodPost, "/kg/query", map[string]any{"group_id": group, "query": query, "top_k": topK}, &out); e != nil {
		return nil, e
	}
	if out.Results == nil || out.Count == nil || *out.Count != len(out.Results) {
		return nil, &Failure{"protocol"}
	}
	if e := validateResults(out.Results, group, topK); e != nil {
		return nil, e
	}
	return out.Results, nil
}

type Node struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	GroupID   string  `json:"group_id"`
	CreatedAt *string `json:"created_at"`
}
type Source struct {
	GroupID   string  `json:"group_id"`
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Source    string  `json:"source"`
	CreatedAt *string `json:"created_at"`
}
type Graph struct {
	GroupID      string   `json:"group_id"`
	NodeCount    *int64   `json:"node_count"`
	EdgeCount    *int64   `json:"edge_count"`
	EpisodeCount *int64   `json:"episode_count"`
	Nodes        []Node   `json:"nodes"`
	Edges        []Result `json:"edges"`
	Sources      []Source `json:"sources"`
	Truncated    bool     `json:"truncated"`
}

func (c *Client) Inspect(ctx context.Context, group string) (*Graph, error) {
	if !validGroup(group) {
		return nil, &Failure{"protocol"}
	}
	var out Graph
	if e := c.request(ctx, http.MethodGet, "/kg/inspect/"+group, nil, &out); e != nil {
		return nil, e
	}
	if out.GroupID != group {
		return nil, &Failure{"scope"}
	}
	if out.NodeCount == nil || out.EdgeCount == nil || out.EpisodeCount == nil || *out.NodeCount < 0 || *out.EdgeCount < 0 || *out.EpisodeCount < 0 || out.Nodes == nil || out.Edges == nil || out.Sources == nil || len(out.Nodes) > 110 || len(out.Sources) > 200 || int64(len(out.Nodes)) > *out.NodeCount || int64(len(out.Edges)) > *out.EdgeCount {
		return nil, &Failure{"protocol"}
	}
	if e := validateResults(out.Edges, group, 40); e != nil {
		return nil, e
	}
	seen := map[string]bool{}
	for _, n := range out.Nodes {
		if n.GroupID != group {
			return nil, &Failure{"scope"}
		}
		if n.ID == "" || seen[n.ID] || len([]rune(n.Name)) > 256 {
			return nil, &Failure{"protocol"}
		}
		seen[n.ID] = true
	}
	for _, r := range out.Edges {
		if !seen[r.Source] || !seen[r.Target] {
			return nil, &Failure{"protocol"}
		}
	}
	for _, source := range out.Sources {
		if source.GroupID != group {
			return nil, &Failure{"scope"}
		}
		if len(source.ID) > 128 || len([]rune(source.Name)) > 256 || len([]rune(source.Source)) > 256 {
			return nil, &Failure{"protocol"}
		}
	}
	return &out, nil
}
func Group(owner int64) string { return fmt.Sprint(owner) }
