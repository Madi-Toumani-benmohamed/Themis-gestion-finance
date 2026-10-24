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

const DefaultOpenAIModel = "gpt-4.1-mini"
const openAIResponsesURL = "https://api.openai.com/v1/responses"

// OpenAI utilise Responses, sans historique distant conservé par Thémis.
// L'URL est fixe ; endpoint sert uniquement aux tests locaux du package.
type OpenAI struct {
	APIKey   string
	Model    string
	endpoint string
}

func (o *OpenAI) model() string {
	if strings.TrimSpace(o.Model) == "" {
		return DefaultOpenAIModel
	}
	return o.Model
}
func (o *OpenAI) Name() string { return "openai (" + o.model() + ")" }

func (o *OpenAI) Complete(ctx context.Context, system, user string, maxTokens int, wantJSON bool) (string, error) {
	if strings.TrimSpace(o.APIKey) == "" {
		return "", fmt.Errorf("clé API OpenAI manquante")
	}
	payload := map[string]any{
		"model": o.model(), "instructions": system, "input": user, "store": false,
	}
	if maxTokens > 0 {
		if maxTokens < 16 {
			maxTokens = 16
		}
		payload["max_output_tokens"] = maxTokens
	}
	// L'interface partagée fournit des schémas différents dans ses prompts.
	// Le mode objet JSON conserve ce contrat sans inventer de schéma concurrent.
	if wantJSON {
		payload["instructions"] = system + "\nRéponds uniquement avec un objet JSON valide."
		payload["text"] = map[string]any{"format": map[string]string{"type": "json_object"}}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	endpoint := openAIResponsesURL
	if o.endpoint != "" {
		endpoint = o.endpoint
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.APIKey)
	client := &http.Client{Timeout: cloudTimeout(), CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("connexion OpenAI impossible : %w", err)
	}
	defer resp.Body.Close()
	// Ne pas renvoyer le corps d'erreur : il peut contenir un extrait de clé
	// invalide ou de requête financière. Le statut suffit au diagnostic admin.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		switch resp.StatusCode {
		case 401:
			return "", fmt.Errorf("OpenAI : clé API invalide ou révoquée (HTTP 401)")
		case 403:
			return "", fmt.Errorf("OpenAI : accès au modèle refusé (HTTP 403)")
		case 429:
			return "", fmt.Errorf("OpenAI : quota ou limite de requêtes atteint (HTTP 429)")
		default:
			return "", fmt.Errorf("OpenAI : requête refusée (HTTP %d)", resp.StatusCode)
		}
	}
	var result struct {
		Status string          `json:"status"`
		Error  json.RawMessage `json:"error"`
		Output []struct {
			Type    string `json:"type"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&result); err != nil {
		return "", fmt.Errorf("OpenAI : réponse illisible")
	}
	if result.Status != "completed" || (len(result.Error) > 0 && string(result.Error) != "null") {
		return "", fmt.Errorf("OpenAI : réponse interrompue ou incomplète ; vérifier le modèle et la limite de jetons")
	}
	var text strings.Builder
	for _, item := range result.Output {
		if item.Type != "message" {
			continue
		}
		for _, part := range item.Content {
			if part.Type == "refusal" {
				return "", fmt.Errorf("OpenAI : demande refusée par le modèle")
			}
			if part.Type == "output_text" {
				text.WriteString(part.Text)
			}
		}
	}
	out := strings.TrimSpace(text.String())
	if out == "" {
		return "", fmt.Errorf("OpenAI : réponse vide")
	}
	if wantJSON && !json.Valid([]byte(out)) {
		return "", fmt.Errorf("OpenAI : réponse JSON invalide")
	}
	return out, nil
}
