package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const mistralURL = "https://api.mistral.ai/v1/chat/completions"

// Mistral appelle l'API Mistral AI (cloud). Les données envoyées quittent la machine.
type Mistral struct {
	APIKey string
	Model  string
}

func (m *Mistral) Name() string { return "mistral (" + m.Model + ")" }

type mistralMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type mistralReq struct {
	Model    string      `json:"model"`
	Messages []mistralMsg `json:"messages"`
	MaxTokens int        `json:"max_tokens"`
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
		Model:    m.Model,
		Messages: messages,
		MaxTokens: maxTokens,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, mistralURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.APIKey)

	client := &http.Client{Timeout: cloudTimeout()}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var mr mistralResp
	if err := json.NewDecoder(resp.Body).Decode(&mr); err != nil {
		return "", err
	}
	if mr.Error != nil {
		return "", fmt.Errorf("API Mistral: %s", mr.Error.Message)
	}
	if len(mr.Choices) == 0 {
		return "", fmt.Errorf("réponse vide de Mistral")
	}
	return mr.Choices[0].Message.Content, nil
}
