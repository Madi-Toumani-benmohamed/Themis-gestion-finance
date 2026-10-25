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

const mistralURL = "https://api.mistral.ai/v1/chat/completions"

// Mistral appelle l'API Mistral AI (cloud). Les données envoyées quittent la machine.
type Mistral struct {
	HTTPClient *http.Client
	APIKey     string
	Model      string
}

func (m *Mistral) Name() string { return "mistral (" + m.Model + ")" }

type mistralMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type mistralReq struct {
	Model     string       `json:"model"`
	Messages  []mistralMsg `json:"messages"`
	MaxTokens int          `json:"max_tokens"`
}

type mistralResp struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (m *Mistral) Complete(ctx context.Context, system, user string, maxTokens int, wantJSON bool) (string, error) {
	messages := []mistralMsg{{Role: "system", Content: system}, {Role: "user", Content: user}}
	body, _ := json.Marshal(mistralReq{
		Model:     m.Model,
		Messages:  messages,
		MaxTokens: maxTokens,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, mistralURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.APIKey)

	client := providerHTTPClient(m.HTTPClient, cloudTimeout())
	resp, err := client.Do(req)
	if err != nil {
		return "", providerNetworkError(ctx, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", &ProviderError{Status: resp.StatusCode}
	}

	var mr mistralResp
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&mr); err != nil {
		return "", &ProviderError{Kind: "response"}
	}
	if mr.Error != nil {
		return "", &ProviderError{Kind: "response"}
	}
	if len(mr.Choices) == 0 || strings.TrimSpace(mr.Choices[0].Message.Content) == "" {
		return "", fmt.Errorf("réponse vide de Mistral")
	}
	return mr.Choices[0].Message.Content, nil
}
