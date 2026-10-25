package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// FreeLLM est un service agrégateur d'APIs IA (OpenRouter, etc.) avec une interface
// compatible OpenAI. Le modèle est spécifié par son identifiant FreeLLM (ex: "openai/gpt-3.5-turbo",
// "anthropic/claude-3-haiku", etc.).
type FreeLLM struct {
	HTTPClient *http.Client
	APIKey     string
	BaseURL    string // ex: https://api.freellm.ai/v1 ou autre provider compatible
	Model      string
}

func (f *FreeLLM) Name() string { return "freellm (" + f.Model + ")" }

type freellmMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type freellmReq struct {
	Model     string       `json:"model"`
	Messages  []freellmMsg `json:"messages"`
	MaxTokens int          `json:"max_tokens,omitempty"`
}

type freellmResp struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (f *FreeLLM) Complete(ctx context.Context, system, user string, maxTokens int, wantJSON bool) (string, error) {
	url := strings.TrimRight(f.BaseURL, "/") + "/chat/completions"
	messages := []freellmMsg{{Role: "system", Content: system}, {Role: "user", Content: user}}
	body, _ := json.Marshal(freellmReq{
		Model:     f.Model,
		Messages:  messages,
		MaxTokens: maxTokens,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+f.APIKey)

	client := providerHTTPClient(f.HTTPClient, cloudTimeout())
	resp, err := client.Do(req)
	if err != nil {
		return "", providerNetworkError(ctx, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", &ProviderError{Status: resp.StatusCode}
	}

	var fr freellmResp
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&fr); err != nil {
		return "", &ProviderError{Kind: "response"}
	}
	if fr.Error != nil {
		return "", &ProviderError{Kind: "response"}
	}
	if len(fr.Choices) == 0 || strings.TrimSpace(fr.Choices[0].Message.Content) == "" {
		return "", fmt.Errorf("réponse vide de FreeLLM")
	}
	return fr.Choices[0].Message.Content, nil
}
