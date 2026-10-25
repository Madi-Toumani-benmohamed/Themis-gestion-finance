package normalize

import "testing"

// Les cas de référence, dont les exemples exacts du cahier des charges.
func TestLabel(t *testing.T) {
	cases := []struct {
		raw     string
		amount  float64
		display string
		kind    string
	}{
		// Exemples du cahier des charges
		{"X4519 Urssaf Marseille 09/01", -2291, "Cotisation URSSAF", "tax"},
		{"Mens.pel 44690429796", -200, "Versement PEL", "savings"},
		{"X4519 Amazon Payments Payl 10/01", -34.90, "Amazon", ""},
		{"CB CARREFOUR CITY PARIS 15", -23.45, "Carrefour City", ""},
		{"PRLV SEPA FREE MOBILE", -19.99, "Free Mobile", ""},
		{"VIR PAYPAL EUROPE", -50, "PayPal", ""},

		// Interprétation selon le sens
		{"VIR LIVRET A", -300, "Versement Livret A", "savings"},
		{"VIR LIVRET A", 300, "Retrait Livret A", "savings"},
		{"PRLV CAF DU NORD", 450, "Prestation CAF", "benefit"},

		// Marchands courants
		{"Prlv Netflix", -13.49, "Netflix", ""},
		{"CB MCDONALD'S PARIS 11", -12.50, "McDonald's", ""},
		{"PRLV SEPA BOUYGUES TELECOM", -19.99, "Bouygues Telecom", ""},
		{"Carte X4519 Mpcarrefour Les Uli 10/06", -45.20, "Carrefour", ""},
		{"Carte X4519 Naval Group Oll Olli 21/09", -12.00, "Naval Group Oll Olli", ""},
		{"Carte X4519 Naval Group Oll Olli 22/09", -12.00, "Naval Group Oll Olli", ""},
		{"Prlv Bouygues Telecom", -19.99, "Bouygues Telecom", ""},
		{"Carte X4519 Carrefour Mayol Toul 17/10", -42.00, "Carrefour", ""},

		// Faux positifs à éviter : mots courts protégés par frontières
		{"CB LIBRE APPELLATION VINS", -30, "Libre Appellation Vins", ""},
		{"CB CAFETERIA DU PARC", -8, "Cafeteria Du Parc", ""},

		// Inconnu : nettoyé et mis en forme, jamais vide
		{"X4519 Ulis Viandes Les Uli 12/06", -28, "Ulis Viandes Les Uli", ""},
		{"CB SUMUP  *COIFFURE LEA", -25, "Coiffure Lea", ""},
		{"CB ZETTLE_*GARAGE MARTIN", -60, "Garage Martin", ""},
		{"CB SARL DUPONT BOULANGERIE 33", -8, "Sarl Dupont Boulangerie", ""},
		{"CB 4756 TABAC PRESSE", -12, "Tabac Presse", ""},
		// références opaques variables → même marchand après nettoyage
		{"Acheel France Ref-927b2800281923c168gfgw bh4 Core", -80, "Acheel France Core", ""},
		{"Acheel France Ref-1cbbb98877f24695gfgw bh4 Fr457z787f911 Core", -80, "Acheel France Core", ""},
		{"CB 1234567890", -10, "CB 1234567890", ""},          // tout est bruit → brut conservé
		{"CB Épicerie du Port", -10, "Épicerie Du Port", ""}, // Title Case doit préserver l'UTF-8
	}
	for _, c := range cases {
		got := Label(c.raw, c.amount)
		if got.Display != c.display {
			t.Errorf("Label(%q).Display = %q, attendu %q", c.raw, got.Display, c.display)
		}
		if got.Kind != c.kind {
			t.Errorf("Label(%q).Kind = %q, attendu %q", c.raw, got.Kind, c.kind)
		}
		if got.Merchant == "" {
			t.Errorf("Label(%q).Merchant vide", c.raw)
		}
	}
}

func TestCashWithdrawalsAreOperationsNotMerchants(t *testing.T) {
	cases := []string{
		"Ret Distrib 14h23",
		"Ret Distrib 17h29",
		"Ret DAB X4519 Toulon Liberté 03/10 10H21",
		"Retrait DAB X4519 Toulon Liberté 03/10 10H21",
	}
	for _, raw := range cases {
		got := Label(raw, -40)
		if got.Display != "Retrait DAB" || got.Kind != "cash_withdrawal" || got.Merchant != "" {
			t.Errorf("Label(%q) = %#v, attendu un type d'opération sans marchand", raw, got)
		}
	}
}

func TestClockTimeIsRemovedWithoutChangingMerchantContext(t *testing.T) {
	got := Label("Carte X4519 Naval Group Oll Olli 21/09 10H21", -12)
	if got.Display != "Naval Group Oll Olli" {
		t.Fatalf("Display = %q", got.Display)
	}
	other := Label("Carte X4519 Naval Group Oll Olli 21/09 17h29", -12)
	if other.Display != got.Display || other.Merchant != got.Merchant {
		t.Fatalf("heures variables ont différencié le marchand: %#v / %#v", got, other)
	}
}
