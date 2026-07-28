package api

import (
	"testing"
	"time"
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

func TestComputeCredit(t *testing.T) {
	now := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	c := creditOut{MonthlyAmount: 250, MonthsTotal: 48, StartMonth: "2025-01"}
	computeCredit(&c, now)
	// janv 2025 → juil 2026 = 19 mensualités
	if c.MonthsPaid != 19 {
		t.Errorf("MonthsPaid = %d, attendu 19", c.MonthsPaid)
	}
	if c.MonthsRemaining != 29 {
		t.Errorf("MonthsRemaining = %d, attendu 29", c.MonthsRemaining)
	}
	if c.Remaining != 29*250 {
		t.Errorf("Remaining = %.2f, attendu %.2f", c.Remaining, 29*250.0)
	}
	if c.EndMonth != "2028-12" {
		t.Errorf("EndMonth = %q, attendu 2028-12", c.EndMonth)
	}

	// crédit déjà soldé : ne dépasse jamais le total
	done := creditOut{MonthlyAmount: 100, MonthsTotal: 12, StartMonth: "2020-01"}
	computeCredit(&done, now)
	if done.MonthsRemaining != 0 || done.Remaining != 0 {
		t.Errorf("crédit soldé mal calculé: reste %d mois / %.2f", done.MonthsRemaining, done.Remaining)
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
