package parser

import (
	"testing"

	"themis/internal/models"
)

func row(y float64, words ...Word) Row { return Row{Words: words, Y: y, Page: 1} }
func w(s string, x, wd float64) Word   { return Word{S: s, X: x, W: wd, FontSize: 10} }

// relevé synthétique minimal : en-têtes, soldes, deux opérations
func sampleRows() []Row {
	return []Row{
		row(100, w("Date", 50, 30), w("Libellé", 110, 50), w("Débit", 400, 40), w("Crédit", 480, 45)),
		row(95, w("Ancien", 50, 45), w("solde", 100, 38), w("au", 142, 15), w("21.05.2026", 160, 70), w("306,46", 490, 40)),
		row(90, w("22.05", 50, 35), w("22.05", 90, 35), w("Carte", 130, 35), w("Burger", 170, 42), w("23,10", 405, 38)),
		row(85, w("03.06", 50, 35), w("Virement", 90, 60), w("reçu", 155, 30), w("300,00", 488, 42)),
		row(80, w("Nouveau", 50, 55), w("solde", 108, 38), w("au", 150, 15), w("22.06.2026", 168, 70), w("583,36", 490, 40)),
	}
}

func TestParseLocalNominal(t *testing.T) {
	res, err := ParseLocal(sampleRows())
	if err != nil {
		t.Fatalf("ParseLocal: %v", err)
	}
	if len(res.Transactions) != 2 {
		t.Fatalf("attendu 2 transactions, obtenu %d", len(res.Transactions))
	}
	if res.Transactions[0].Amount != -23.10 {
		t.Errorf("débit attendu -23.10, obtenu %v", res.Transactions[0].Amount)
	}
	if res.Transactions[1].Amount != 300.00 {
		t.Errorf("crédit attendu +300.00, obtenu %v", res.Transactions[1].Amount)
	}
	if res.Transactions[0].Date != "2026-05-22" || res.Transactions[1].Date != "2026-06-03" {
		t.Errorf("dates incorrectes: %s / %s", res.Transactions[0].Date, res.Transactions[1].Date)
	}
	if !res.Validated {
		t.Errorf("le relevé équilibré devrait être validé (306.46 - 23.10 + 300 = 583.36)")
	}
	if res.PeriodStart != "2026-05-21" || res.PeriodEnd != "2026-06-22" {
		t.Errorf("période incorrecte: %s → %s", res.PeriodStart, res.PeriodEnd)
	}
}

func TestChooseYearCrossYear(t *testing.T) {
	// relevé décembre → janvier : les opérations de décembre prennent l'année de début
	rows := []Row{
		row(100, w("Date", 50, 30), w("Libellé", 110, 50), w("Débit", 400, 40), w("Crédit", 480, 45)),
		row(95, w("Ancien", 50, 45), w("solde", 100, 38), w("au", 142, 15), w("20.12.2025", 160, 70), w("100,00", 490, 40)),
		row(90, w("28.12", 50, 35), w("Achat", 90, 40), w("10,00", 405, 38)),
		row(85, w("05.01", 50, 35), w("Achat", 90, 40), w("20,00", 405, 38)),
		row(80, w("Nouveau", 50, 55), w("solde", 108, 38), w("au", 150, 15), w("19.01.2026", 168, 70), w("70,00", 490, 40)),
	}
	res, err := ParseLocal(rows)
	if err != nil {
		t.Fatalf("ParseLocal: %v", err)
	}
	if res.Transactions[0].Date != "2025-12-28" {
		t.Errorf("décembre devrait être 2025, obtenu %s", res.Transactions[0].Date)
	}
	if res.Transactions[1].Date != "2026-01-05" {
		t.Errorf("janvier devrait être 2026, obtenu %s", res.Transactions[1].Date)
	}
	if !res.Validated {
		t.Errorf("devrait valider: 100 - 10 - 20 = 70")
	}
}

func TestDateNotAnAmount(t *testing.T) {
	// une date jj.mm ne doit jamais être prise pour un montant
	if reAmount.MatchString("22.05") {
		t.Errorf("22.05 (date) ne doit pas matcher reAmount")
	}
	for _, ok := range []string{"23,10", "1 520,00", "1.234,56", "-7,50"} {
		if !reAmount.MatchString(ok) {
			t.Errorf("%s devrait matcher reAmount", ok)
		}
	}
}

func TestBankNameNotAHeader(t *testing.T) {
	// « CREDIT AGRICOLE » seul ne doit pas définir la colonne Crédit
	rows := []Row{
		row(110, w("CREDIT", 50, 50), w("AGRICOLE", 105, 65)),
		row(100, w("Date", 50, 30), w("Débit", 400, 40), w("Crédit", 480, 45)),
	}
	d, c, _ := findColumns(rows)
	if d < 0 || c < 0 {
		t.Fatalf("colonnes non trouvées")
	}
	if c < 450 {
		t.Errorf("creditX=%v : capturé sur le nom de banque au lieu de l'en-tête", c)
	}
}

func TestReconcileArbitratesOCR(t *testing.T) {
	// deux lectures ambiguës : seule la combinaison (560 ; 7,50) équilibre
	alt1, alt2 := 560.00, -7.90
	res := &models.ParseResult{
		Transactions: []models.Transaction{
			{Date: "2026-06-05", Label: "A", Amount: 960.00, AltAmount: &alt1},
			{Date: "2026-06-08", Label: "B", Amount: -7.50, AltAmount: &alt2},
		},
	}
	open, close := 100.00, 652.50 // 100 + 560 - 7.50
	res.OpeningBalance, res.ClosingBalance = &open, &close
	if Validate(res) {
		t.Fatalf("ne devrait pas valider avant réconciliation")
	}
	reconcile(res, nil, nil)
	if !res.Validated {
		t.Fatalf("la réconciliation devrait trouver la combinaison équilibrée")
	}
	if res.Transactions[0].Amount != 560.00 || res.Transactions[1].Amount != -7.50 {
		t.Errorf("mauvaise combinaison retenue: %v / %v", res.Transactions[0].Amount, res.Transactions[1].Amount)
	}
}

func TestTxHashStable(t *testing.T) {
	a := models.Transaction{Date: "2026-06-05", Label: "Carte  Burger King", Amount: -23.10}
	b := models.Transaction{Date: "2026-06-05", Label: "carte burger king", Amount: -23.10}
	if TxHash(a) != TxHash(b) {
		t.Errorf("le hash doit être insensible à la casse et aux espaces multiples")
	}
	c := models.Transaction{Date: "2026-06-05", Label: "Carte Burger King", Amount: -23.11}
	if TxHash(a) == TxHash(c) {
		t.Errorf("un montant différent doit changer le hash")
	}
}

func TestSanitizeRowsKeepsAmountsBeyondCredit(t *testing.T) {
	// un montant à droite du bord Crédit doit survivre (largeurs natives peu fiables),
	// mais pas un glyphe de case à cocher
	rows := []Row{
		row(90, w("22.05", 50, 35), w("Achat", 90, 40), w("23,10", 505, 38), w("O", 560, 10)),
	}
	out := sanitizeRows(rows, 500)
	if len(out[0].Words) != 3 {
		t.Fatalf("attendu 3 mots (coche filtrée), obtenu %d: %v", len(out[0].Words), out[0].Words)
	}
	last := out[0].Words[len(out[0].Words)-1].S
	if last != "23,10" {
		t.Errorf("le montant devrait être conservé, dernier mot: %s", last)
	}
}
