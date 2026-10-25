package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

const anthropicURL = "https://api.anthropic.com/v1/messages"

// Anthropic appelle l'API Claude. Les données envoyées quittent la machine :
// à réserver aux utilisateurs qui l'acceptent (voir Ollama pour du 100 % local).
type Anthropic struct {
	HTTPClient *http.Client
	APIKey     string
	Model      string
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

	client := providerHTTPClient(a.HTTPClient, cloudTimeout())
	resp, err := client.Do(req)
	if err != nil {
		return "", providerNetworkError(ctx, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", &ProviderError{Status: resp.StatusCode}
	}

	var cr anthropicResp
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&cr); err != nil {
		return "", &ProviderError{Kind: "response"}
	}
	if cr.Error != nil {
		return "", &ProviderError{Kind: "response"}
	}
	out := ""
	for _, c := range cr.Content {
		if c.Type == "text" {
			out += c.Text
		}
	}
	if strings.TrimSpace(out) == "" {
		return "", &ProviderError{Kind: "response"}
	}
	return out, nil
}
