package api

import (
	"context"
	"testing"
)

// fakeLLM simule les réponses réelles que produisent les modèles locaux.
type fakeLLM struct{ reply string; err error }

func (f fakeLLM) Complete(ctx context.Context, system, user string, maxTokens int, wantJSON bool) (string, error) {
	return f.reply, f.err
}

// On teste directement parseSuggestions avec les formes réelles qui font
// échouer les modèles locaux — c'est le cœur du bug rapporté.
func TestParseSuggestions_RealWorldFailures(t *testing.T) {
	cases := []struct {
		name    string
		reply   string
		wantOK  bool
		wantLen int
	}{
		{"tableau pur", `[{"key":"CB ACHEEL","name":"Acheel"}]`, true, 1},
		{"bloc markdown", "```json\n[{\"key\":\"CB ACHEEL\",\"name\":\"Acheel\"}]\n```", true, 1},
		{"texte avant", `Voici le résultat :\n[{"key":"CB ACHEEL","name":"Acheel"}]`, true, 1},
		{"texte avant et après", `Bien sûr ! [{"key":"CB ACHEEL","name":"Acheel"}] J'espère que ça aide.`, true, 1},
		{"objet suggestions", `{"suggestions":[{"key":"CB ACHEEL","name":"Acheel"}]}`, true, 1},
		{"objet plat", `{"CB ACHEEL":"Acheel","CB CARREFOUR":"Carrefour"}`, true, 2},
		{"objet plat avec texte", `Voici : {"CB ACHEEL":"Acheel"} voilà.`, true, 1},
		// cas qui DOIVENT échouer proprement (déclenchent le repli)
		{"refus en langage naturel", `Je ne peux pas déterminer ces noms avec certitude.`, false, 0},
		{"vide", ``, false, 0},
		{"markdown liste (rattrapé)", "- CB ACHEEL : Acheel\n- CB CARREFOUR : Carrefour", true, 2},
	}
	for _, c := range cases {
		out, err := parseSuggestions(c.reply)
		if c.wantOK && err != nil {
			t.Errorf("%s : échec inattendu (%v) pour %q", c.name, err, c.reply)
		}
		if !c.wantOK && err == nil {
			t.Errorf("%s : aurait dû échouer, mais a parsé %v", c.name, out)
		}
		if c.wantOK && len(out) != c.wantLen {
			t.Errorf("%s : %d résultats, attendu %d", c.name, len(out), c.wantLen)
		}
	}
}
