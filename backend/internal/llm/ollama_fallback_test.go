package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Vérifie la stratégie : think:true d'abord, repli think:false si vide.
func TestCompleteThinkFallback(t *testing.T) {
	var calls []bool // valeur de "think" à chaque appel reçu
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		json.Unmarshal(body, &req)
		think := true
		if v, ok := req["think"].(bool); ok {
			think = v
		}
		calls = append(calls, think)
		// 1er appel (think:true) → vide ; 2e appel (think:false) → réponse
		content := ""
		if !think {
			content = "Voici la synthèse."
		}
		json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]string{"content": content},
		})
	}))
	defer srv.Close()

	o := &Ollama{URL: srv.URL, Model: "gemma3:12b"}
	out, err := o.Complete(context.Background(), "sys", "user", 1500, false)
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	if out != "Voici la synthèse." {
		t.Errorf("réponse attendue après repli, got %q", out)
	}
	// On doit avoir tenté think:true PUIS think:false
	if len(calls) != 2 || calls[0] != true || calls[1] != false {
		t.Errorf("séquence d'appels attendue [true,false], got %v", calls)
	}
}

// Si le 1er essai (think:true) réussit, pas de repli.
func TestCompleteNoFallbackWhenOK(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]string{"content": "Réponse directe."},
		})
	}))
	defer srv.Close()

	o := &Ollama{URL: srv.URL, Model: "qwen2.5:14b"}
	out, _ := o.Complete(context.Background(), "sys", "user", 1500, false)
	if out != "Réponse directe." {
		t.Errorf("got %q", out)
	}
	if calls != 1 {
		t.Errorf("un seul appel attendu si succès direct, got %d", calls)
	}
}

// Reproduit le bug qwen2.5 : Ollama renvoie une ERREUR sur think:true.
// On doit basculer sur think:false et réussir, pas planter.
func TestCompleteThinkingUnsupported(t *testing.T) {
	var calls []bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		json.Unmarshal(body, &req)
		think, _ := req["think"].(bool)
		calls = append(calls, think)
		if think {
			// le modèle refuse le paramètre think
			json.NewEncoder(w).Encode(map[string]any{
				"error": `"qwen2.5:14b" does not support thinking`,
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]string{"content": "Synthèse générée."},
		})
	}))
	defer srv.Close()

	o := &Ollama{URL: srv.URL, Model: "qwen2.5:14b"}
	flag := false
	ctx := WithThinkingFlag(context.Background(), &flag)
	out, err := o.Complete(ctx, "sys", "user", 2500, false)
	if err != nil {
		t.Fatalf("ne doit PAS échouer, doit basculer sur think:false : %v", err)
	}
	if out != "Synthèse générée." {
		t.Errorf("got %q", out)
	}
	if len(calls) != 2 || calls[0] != true || calls[1] != false {
		t.Errorf("séquence attendue [true,false], got %v", calls)
	}
	if !flag {
		t.Error("le drapeau thinking_disabled devrait être positionné")
	}
}
