package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const anthropicURL = "https://api.anthropic.com/v1/messages"

// Anthropic appelle l'API Claude. Les données envoyées quittent la machine :
// à réserver aux utilisateurs qui l'acceptent (voir Ollama pour du 100 % local).
type Anthropic struct {
	APIKey string
	Model  string
}

func (a *Anthropic) Name() string { return "anthropic (" + a.Model + ")" }

type anthropicReq struct {
	Model     string         `json:"model"`
	MaxTokens int            `json:"max_tokens"`
	System    string         `json:"system"`
	Messages  []anthropicMsg `json:"messages"`
}
type anthropicMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type anthropicResp struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (a *Anthropic) Complete(ctx context.Context, system, user string, maxTokens int, _ bool) (string, error) {
	body, _ := json.Marshal(anthropicReq{
		Model:     a.Model,
		MaxTokens: maxTokens,
		System:    system,
		Messages:  []anthropicMsg{{Role: "user", Content: user}},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: cloudTimeout()}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var cr anthropicResp
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return "", err
	}
	if cr.Error != nil {
		return "", fmt.Errorf("API Anthropic: %s", cr.Error.Message)
	}
	out := ""
	for _, c := range cr.Content {
		if c.Type == "text" {
			out += c.Text
		}
	}
	return out, nil
}
