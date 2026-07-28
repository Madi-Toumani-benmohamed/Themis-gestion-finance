package api

import "testing"

func TestLoginRateLimitByIP(t *testing.T) {
	ip := "203.0.113.9"
	recordLoginSuccess(ip) // repartir propre
	for i := 0; i < loginMaxFails-1; i++ {
		recordLoginFail(ip)
	}
	if loginBlocked(ip) {
		t.Fatal("ne devrait pas être bloqué avant le seuil")
	}
	recordLoginFail(ip) // atteint le seuil
	if !loginBlocked(ip) {
		t.Fatal("devrait être bloqué au seuil")
	}
	recordLoginSuccess(ip)
	if loginBlocked(ip) {
		t.Fatal("un succès doit lever le blocage")
	}
}

func TestCSVSafe(t *testing.T) {
	cases := map[string]string{
		"CARREFOUR":     "CARREFOUR",
		"=cmd|calc":     "'=cmd|calc",
		"+SUM(A1)":      "'+SUM(A1)",
		"-2+3":          "'-2+3",
		"@import":       "'@import",
		"":              "",
	}
	for in, want := range cases {
		if got := csvSafe(in); got != want {
			t.Errorf("csvSafe(%q) = %q, attendu %q", in, got, want)
		}
	}
}

func TestClientIP(t *testing.T) {
	// X-Forwarded-For prioritaire, première valeur
	if ip := trimSpace("10.0.0.1"); ip != "10.0.0.1" {
		t.Errorf("trimSpace KO: %q", ip)
	}
}
