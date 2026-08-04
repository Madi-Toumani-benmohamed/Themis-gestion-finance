package llm

import "testing"

func TestStripThinking(t *testing.T) {
	cases := []struct{ in, want string }{
		{"<think>je réfléchis…</think>Voici la réponse.", "Voici la réponse."},
		{"<thinking>hmm</thinking>\n\nSynthèse claire.", "Synthèse claire."},
		{"Pas de réflexion ici.", "Pas de réflexion ici."},
		{"<think>tout est réflexion</think>", "<think>tout est réflexion</think>"}, // ne pas renvoyer vide
		{"Avant <think>milieu</think> après", "Avant  après"},
		{"", ""},
	}
	for i, c := range cases {
		got := stripThinking(c.in)
		if got != c.want {
			t.Errorf("cas %d: stripThinking(%q) = %q, want %q", i, c.in, got, c.want)
		}
	}
}
