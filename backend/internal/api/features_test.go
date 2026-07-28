package api

import "testing"

func TestNormalizeRecLabel(t *testing.T) {
	cases := [][2]string{
		{"PRLV FREE TELECOM ABONNEMENT", "FREE TELECOM"},
		{"Prlv Free Telecom Free Hautdebit 147851605447939382 Core", "FREE TELECOM"},
		{"Carte X4519 Burger King Les Ulis 21/05", "BURGER KING"},
		{"CB CARREFOUR MARKET", "CARREFOUR MARKET"},
		{"Virement Vir Inst vers Ben-mohamed Madi T", "BEN-MOHAMED MADI"},
		{"Ret DAB X4519 Les Ulis 06/06 09H28", "LES ULIS"},
	}
	for _, c := range cases {
		if got := normalizeRecLabel(c[0]); got != c[1] {
			t.Errorf("normalizeRecLabel(%q) = %q, attendu %q", c[0], got, c[1])
		}
	}
	// deux relevés différents du même marchand → même clé
	if normalizeRecLabel("PRLV BOUYGUES TELECOM FORFAIT") != normalizeRecLabel("Prlv Bouygues Telecom 06xxxxx951BT1150PNIAM30 Core") {
		t.Errorf("les deux libellés Bouygues devraient donner la même clé")
	}
}
