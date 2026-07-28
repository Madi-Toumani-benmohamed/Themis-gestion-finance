package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

type scriptedLLM struct{ reply string }

func (s scriptedLLM) Complete(ctx context.Context, sys, u string, m int, j bool) (string, error) {
	return s.reply, nil
}
func (s scriptedLLM) Name() string { return "scripted" }

// Vérifie le comportement complet de l'endpoint selon ce que « répond » le modèle.
func TestSuggestEndpoint_Fallback(t *testing.T) {
	srv := &Server{}
	srv.ai.client = scriptedLLM{reply: "- CB ACHEEL FR : Acheel\n- CB PHARM CENTRE : Pharmacie du Centre"}

	mkReq := func(body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/api/merchants/suggest", strings.NewReader(body))
		u := &authUser{ID: 1}
		r = r.WithContext(context.WithValue(r.Context(), ctxUserKey, u))
		w := httptest.NewRecorder()
		srv.suggestMerchantNames(w, r)
		return w
	}

	// cas 1 : liste markdown → rattrapée
	w := mkReq(`{"keys":["CB ACHEEL FR","CB PHARM CENTRE"]}`)
	var resp struct {
		Suggestions []map[string]string `json:"suggestions"`
		AIError     string              `json:"ai_error"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp.Suggestions) != 2 {
		t.Errorf("liste markdown : attendu 2, obtenu %d (err=%q)", len(resp.Suggestions), resp.AIError)
	}

	// cas 2 : refus en langage naturel → repli déterministe + diagnostic
	srv.ai.client = scriptedLLM{reply: "Désolé, je ne peux pas identifier ces commerçants."}
	w2 := mkReq(`{"keys":["CB CARREFOUR CITY PARIS"]}`)
	var resp2 struct {
		Suggestions []map[string]string `json:"suggestions"`
		AIError     string              `json:"ai_error"`
		AIRaw       string              `json:"ai_raw"`
	}
	json.NewDecoder(w2.Body).Decode(&resp2)
	if resp2.AIError == "" {
		t.Error("cas refus : ai_error devrait être renseigné")
	}
	if resp2.AIRaw == "" {
		t.Error("cas refus : ai_raw (diagnostic) devrait contenir un aperçu")
	}
	if len(resp2.Suggestions) == 0 {
		t.Error("cas refus : le repli devrait proposer un nom nettoyé")
	} else if !strings.Contains(resp2.Suggestions[0]["name"], "Carrefour") {
		t.Errorf("repli attendu ~Carrefour, obtenu %q", resp2.Suggestions[0]["name"])
	}
}
