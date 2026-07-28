package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Ollama appelle une instance Ollama (https://ollama.com) : le modèle tourne
// sur la machine ou le cluster de l'utilisateur, aucune donnée bancaire ne
// sort du réseau. Modèles conseillés : qwen2.5:14b ou mistral-small pour le
// parsing (bonne extraction JSON), llama3.1:8b suffit pour les conseils.
type Ollama struct {
	URL   string // ex: http://localhost:11434
	Model string // ex: qwen2.5:14b
}

func (o *Ollama) Name() string { return "ollama (" + o.Model + ")" }

type ollamaReq struct {
	Model    string         `json:"model"`
	Messages []ollamaMsg    `json:"messages"`
	Stream   bool           `json:"stream"`
	Format   string         `json:"format,omitempty"` // "json" force une sortie JSON valide
	Options  map[string]any `json:"options,omitempty"`
}
type ollamaMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type ollamaResp struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	Error string `json:"error"`
}

func (o *Ollama) Complete(ctx context.Context, system, user string, maxTokens int, wantJSON bool) (string, error) {
	r := ollamaReq{
		Model: o.Model,
		Messages: []ollamaMsg{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Stream: false,
		Options: map[string]any{
			"num_predict": maxTokens,
			"temperature": 0, // extraction : déterminisme avant tout
		},
	}
	if wantJSON {
		r.Format = "json" // Ollama contraint le décodage : JSON garanti
	}
	body, _ := json.Marshal(r)

	url := strings.TrimRight(o.URL, "/") + "/api/chat"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	// Les modèles locaux sur CPU peuvent être lents sur un relevé de 3 pages.
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Ollama injoignable sur %s: %w", o.URL, err)
	}
	defer resp.Body.Close()

	var or ollamaResp
	if err := json.NewDecoder(resp.Body).Decode(&or); err != nil {
		return "", fmt.Errorf("réponse Ollama invalide: %w", err)
	}
	if or.Error != "" {
		return "", fmt.Errorf("Ollama: %s", or.Error)
	}
	return or.Message.Content, nil
}

// ListOllamaModels retourne les modèles disponibles sur une instance.
func ListOllamaModels(ctx context.Context, baseURL string) ([]string, error) {
	url := strings.TrimRight(baseURL, "/") + "/api/tags"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Ollama injoignable sur %s: %w", baseURL, err)
	}
	defer resp.Body.Close()
	var tags struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, fmt.Errorf("réponse Ollama invalide: %w", err)
	}
	out := make([]string, 0, len(tags.Models))
	for _, m := range tags.Models {
		out = append(out, m.Name)
	}
	return out, nil
}

// Ping vérifie que l'instance répond et que le modèle est disponible.
func (o *Ollama) Ping(ctx context.Context) error {
	url := strings.TrimRight(o.URL, "/") + "/api/tags"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Ollama injoignable sur %s: %w", o.URL, err)
	}
	defer resp.Body.Close()
	var tags struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return err
	}
	for _, m := range tags.Models {
		if m.Name == o.Model || strings.HasPrefix(m.Name, o.Model+":") {
			return nil
		}
	}
	return fmt.Errorf("modèle %q absent d'Ollama (faire : ollama pull %s)", o.Model, o.Model)
}
