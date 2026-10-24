package parser

import (
	"testing"
	"themis/internal/models"
)

func TestCSVRejectsSilentLoss(t *testing.T) {
	for _, line := range []string{"31/02/2026;ACHAT;-10.00", "2026-13-01;ACHAT;-10", "2026-01-01;ACHAT;EUR dix", "2026-01-01;ACHAT;1.234", "2026-01-01;;-10"} {
		if _, err := ParseCSV([]byte("Date;Libellé;Montant\n2026-01-01;OK;-1\n" + line)); err == nil {
			t.Fatalf("ligne invalide ignorée: %s", line)
		}
	}
	if _, err := ParseCSV([]byte("Date;Libellé;Débit;Crédit\n2026-01-01;AMBIGU;10;20")); err == nil {
		t.Fatal("débit/crédit ambigus acceptés")
	}
}
func TestValidateExactReconciliation(t *testing.T) {
	opening, closing := 0.0, 0.3
	res := &models.ParseResult{OpeningBalance: &opening, ClosingBalance: &closing, Transactions: []models.Transaction{{Amount: 0.1}, {Amount: 0.2}}}
	if !Validate(res) {
		t.Fatal("0.10+0.20 doit être 0.30")
	}
	closing = 0.31
	if Validate(res) {
		t.Fatal("écart de un centime accepté")
	}
	res.ClosingBalance = nil
	if Validate(res) {
		t.Fatal("solde manquant accepté")
	}
}
func TestCSVPreservesDistinctSources(t *testing.T) {
	data := []byte("Date;Libellé;Montant\n2026-01-01;ACHAT REF1;-10\n2026-01-01;ACHAT REF2;-10\n2026-01-02;REMBOURSEMENT;10")
	first, e := ParseCSV(data)
	if e != nil {
		t.Fatal(e)
	}
	second, e := ParseCSV(data)
	if e != nil {
		t.Fatal(e)
	}
	if first.Validated {
		t.Fatal("CSV sans soldes présenté comme validé")
	}
	if first.Transactions[0].Hash == first.Transactions[1].Hash {
		t.Fatal("opérations distinctes fusionnées")
	}
	for i := range first.Transactions {
		if first.Transactions[i].Hash != second.Transactions[i].Hash {
			t.Fatal("empreinte instable")
		}
	}
	if first.Transactions[2].Amount != 10 {
		t.Fatal("signe remboursement perdu")
	}
}

func TestCSVDifferentBookingDatesAreNotSilentlyLost(t *testing.T) {
	data := []byte("Date opération;Date valeur;Libellé;Montant\n2026-01-31;2026-02-02;ACHAT;-10")
	if _, err := ParseCSV(data); err == nil {
		t.Fatal("date comptable distincte ignorée")
	}
}

func TestCSVAmountNeverRepairsMalformedNumbers(t *testing.T) {
	for _, s := range []string{"1,2,3", "1 2 3", "12+3", "1.234", "NaN", "(12.00)", "10000000000"} {
		if _, ok := parseCSVAmount(s); ok {
			t.Fatalf("malformed amount repaired: %s", s)
		}
	}
	for _, s := range []string{"1 234,56 €", "1,234.56", "1.234,56", "1234.56"} {
		if v, ok := parseCSVAmount(s); !ok || v != 1234.56 {
			t.Fatalf("recognized format rejected: %s", s)
		}
	}
}
