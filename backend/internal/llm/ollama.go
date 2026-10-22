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
	// Think désactive les tokens de « réflexion » des modèles de raisonnement
	// (gemma3, deepseek-r1…). Sans ça, ces pensées consomment tout le budget
	// num_predict avant la vraie réponse → sortie vide. Pointeur pour n'envoyer
	// le champ que si on le fixe explicitement (certaines versions d'Ollama ne
	// le connaissent pas et l'ignorent sans erreur).
	Think *bool `json:"think,omitempty"`
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
	// Stratégie : qualité d'abord, robustesse en repli.
	//  1er essai : réflexion ACTIVÉE (think:true) — meilleure qualité sur les
	//    modèles de raisonnement.
	//  Deux cas de repli vers think:false :
	//    a) le modèle REFUSE le paramètre (« does not support thinking ») — les
	//       modèles sans raisonnement comme qwen2.5 renvoient une erreur ;
	//    b) la réponse est VIDE (les tokens de réflexion ont épuisé le budget,
	//       cas gemma3/deepseek-r1).
	//  Cas JSON : format:"json" peut aussi produire du vide → appel sans format.
	out, err := o.complete(ctx, system, user, maxTokens, wantJSON, true)
	if err != nil {
		if isThinkingUnsupported(err) {
			// Le modèle ne connaît pas le thinking : on repart proprement sans.
			markThinkingDisabled(ctx)
			out, err = o.complete(ctx, system, user, maxTokens, wantJSON, false)
			if err != nil {
				return "", err
			}
		} else {
			return "", err
		}
	} else if strings.TrimSpace(out) == "" {
		// Réponse vide malgré un appel réussi : réflexion désactivée pour
		// récupérer le budget de tokens.
		markThinkingDisabled(ctx)
		out, err = o.complete(ctx, system, user, maxTokens, wantJSON, false)
		if err != nil {
			return "", err
		}
	}
	if wantJSON && strings.TrimSpace(out) == "" {
		out, err = o.complete(ctx, system, user, maxTokens, false, false)
		if err != nil {
			return "", err
		}
	}
	return out, nil
}

// isThinkingUnsupported reconnaît l'erreur renvoyée par Ollama quand le modèle
// ne gère pas le paramètre think (ex. « ... does not support thinking »).
func isThinkingUnsupported(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "thinking") || strings.Contains(msg, "does not support think")
}

func (o *Ollama) complete(ctx context.Context, system, user string, maxTokens int, wantJSON, think bool) (string, error) {
	r := ollamaReq{
		Model: o.Model,
		Messages: []ollamaMsg{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Stream: false,
		Think:  &think, // réflexion pilotée par l'orchestrateur Complete
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
	client := &http.Client{Timeout: ollamaTimeout()}
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
	return stripThinking(or.Message.Content), nil
}

// stripThinking retire les blocs de réflexion que certains modèles émettent
// malgré think:false (vieilles versions d'Ollama, ou balises intégrées au
// texte). Couvre les formes courantes : <think>…</think>, <thinking>…</thinking>.
// Si après nettoyage il ne reste rien mais qu'il y avait du contenu, on renvoie
// l'original (mieux vaut du texte avec réflexion que du vide).
func stripThinking(s string) string {
	cleaned := s
	for _, tag := range []struct{ open, close string }{
		{"<think>", "</think>"},
		{"<thinking>", "</thinking>"},
	} {
		for {
			i := strings.Index(cleaned, tag.open)
			if i < 0 {
				break
			}
			j := strings.Index(cleaned, tag.close)
			if j < 0 || j < i {
				break
			}
			cleaned = cleaned[:i] + cleaned[j+len(tag.close):]
		}
	}
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" && strings.TrimSpace(s) != "" {
		return strings.TrimSpace(s) // ne jamais renvoyer vide si on avait du contenu
	}
	return cleaned
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
