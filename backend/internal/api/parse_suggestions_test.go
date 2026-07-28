package api

import "testing"

func TestParseSuggestions(t *testing.T) {
	cases := []string{
		`[{"key":"SC RESIDENCE","name":"Loyer"}]`,                                  // pur
		"```json\n[{\"key\":\"SC RESIDENCE\",\"name\":\"Loyer\"}]\n```",             // bloc markdown
		`Voici mes suggestions : [{"key":"SC RESIDENCE","name":"Loyer"}] Voilà.`,   // texte autour
		`{"suggestions":[{"key":"SC RESIDENCE","name":"Loyer"}]}`,                  // objet englobant
		`{"SC RESIDENCE":"Loyer"}`,                                                 // objet plat clé→nom
	}
	for i, c := range cases {
		out, err := parseSuggestions(c)
		if err != nil {
			t.Errorf("cas %d: erreur inattendue: %v", i, err)
			continue
		}
		if len(out) != 1 || out[0]["name"] != "Loyer" {
			t.Errorf("cas %d: parsing incorrect: %v", i, out)
		}
	}
	// cas vraiment illisible
	if _, err := parseSuggestions("désolé je ne peux pas"); err == nil {
		t.Error("une réponse sans JSON devrait échouer proprement")
	}
}
