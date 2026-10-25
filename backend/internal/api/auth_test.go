package api

import (
	"testing"
)

func TestPasswordHashRoundtrip(t *testing.T) {
	pw := "themis2026secret"
	h := hashPassword(pw)
	if !verifyPassword(pw, h) {
		t.Fatal("le mot de passe correct devrait être vérifié")
	}
	if verifyPassword("mauvais", h) {
		t.Fatal("un mauvais mot de passe ne doit pas passer")
	}
	// deux hachages du même mot de passe diffèrent (sel aléatoire)
	if h == hashPassword(pw) {
		t.Fatal("le sel devrait rendre chaque hachage unique")
	}
	// format corrompu → refus, pas de panique
	if verifyPassword(pw, "nimportequoi") {
		t.Fatal("un encodage invalide ne doit pas être accepté")
	}
}

func TestValidPassword(t *testing.T) {
	bad := []string{"court1", "sansdigits", "1234567890", ""}
	for _, p := range bad {
		if validPassword(p) == nil {
			t.Errorf("validPassword(%q) aurait dû échouer", p)
		}
	}
	if validPassword("themis2026") != nil {
		t.Error("un mot de passe valide a été rejeté")
	}
}

func TestLoanPow(t *testing.T) {
	// pow gère les exposants négatifs (actualisation des mensualités)
	if got := pow(2, -2); got != 0.25 {
		t.Errorf("pow(2,-2) = %v, attendu 0.25", got)
	}
	if got := pow(1.5, 0); got != 1 {
		t.Errorf("pow(x,0) = %v, attendu 1", got)
	}
}
