package api

import "testing"

func TestLooksLikeNoise(t *testing.T) {
	cases := []struct {
		label  string
		amount float64
		noise  bool
		why    string
	}{
		{"Machine à café Paris", 0.70, true, "micro-montant"},
		{"Frais Bancaire", 6, true, "sous le seuil de 10 €"},
		{"Retrait Distributeur Les Ulis", 20, true, "faux marchand (retrait)"},
		{"Retrait DAB Société Générale", 40, true, "faux marchand (DAB)"},
		{"Virement interne", 500, true, "mouvement de trésorerie interne"},
		{"Netflix", 13.49, false, "vrai abonnement, montant significatif"},
		{"Loyer", 950, false, "dépense majeure"},
		{"Spotify", 9.99, true, "sous le seuil (bordure basse)"},
		{"EDF", 89.90, false, "vrai marchand récurrent"},
	}
	for _, c := range cases {
		got := looksLikeNoise(c.label, c.amount)
		if got != c.noise {
			t.Errorf("looksLikeNoise(%q, %.2f) = %v, attendu %v (%s)",
				c.label, c.amount, got, c.noise, c.why)
		}
	}
}
