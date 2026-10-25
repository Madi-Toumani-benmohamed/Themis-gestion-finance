package parser

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"themis/internal/money"
	"themis/internal/testfixture"
)

func requirePoppler(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("pdftotext"); err != nil {
		if os.Getenv("THEMIS_REQUIRE_PDF_TESTS") == "1" {
			t.Fatal("Poppler requis pour les tests PDF CI")
		}
		t.Skip("Poppler indisponible ; activer THEMIS_REQUIRE_PDF_TESTS=1 en CI")
	}
}
func TestSynthetic009CustomFontExtraction(t *testing.T) {
	requirePoppler(t)
	pdf := testfixture.Statement009PDF()
	// This is the actual failing layer, not a mocked list of transactions.
	broken, e := ExtractRows(pdf)
	if e == nil {
		if r, e := ParseLocal(broken); e == nil && len(r.Transactions) == 81 {
			t.Fatal("fixture no longer reproduces the legacy custom-font decoding failure")
		}
	}
	res, e := Parse(context.Background(), pdf, nil)
	if e != nil {
		t.Fatal(e)
	}
	var debit, credit money.Cents
	for _, tx := range res.Transactions {
		c, e := money.FromLegacy(tx.Amount)
		if e != nil {
			t.Fatal(e)
		}
		if c < 0 {
			debit -= c
		} else {
			credit += c
		}
		if tx.SourcePage < 1 || tx.SourceLine < 1 {
			t.Fatal("provenance absente")
		}
	}
	if len(res.Transactions) != 81 || debit != 429385 || credit != 296585 || credit-debit != -132800 || res.Report.BalanceDifference == nil || *res.Report.BalanceDifference != 0 || res.ClosingBalance == nil || *res.ClosingBalance != 1164.88 {
		t.Fatalf("référence synthétique non réconciliée: count=%d debit=%s credit=%s report=%+v", len(res.Transactions), debit, credit, res.Report)
	}
	if res.Report.CountControl != "unverifiable" || res.Report.ExpectedCount != nil || res.Report.Completeness != "unknown" {
		t.Fatal("le compteur attendu ne doit pas être fabriqué")
	}
	if !res.Validated || res.Report.TotalsControl != "passed" {
		t.Fatal("totaux indépendants non vérifiés")
	}
}
func TestMarginReferenceNeverHidesAnOperation(t *testing.T) {
	rows := sampleRows()
	rows[2].Words = append([]Word{w("099410", 0, 25)}, rows[2].Words...)
	res, e := ParseLocal(rows)
	if e != nil {
		t.Fatal(e)
	}
	if len(res.Transactions) != 2 || res.Report.Detected != 2 || !res.Validated {
		t.Fatal("référence de marge masque une opération", res.Report)
	}
}
func TestMultilineAndPageBoundary(t *testing.T) {
	rows := sampleRows()
	rows[2].Words = rows[2].Words[:len(rows[2].Words)-1]
	continuation := row(89, w("COMPLEMENT", 130, 80), w("23,10", 405, 38))
	continuation.Page = 2
	// A repeated header must not flush a pending operation.
	header := rows[0]
	header.Page = 2
	rows = append(rows[:3], append([]Row{header, continuation}, rows[3:]...)...)
	res, e := ParseLocal(rows)
	if e != nil {
		t.Fatal(e)
	}
	if len(res.Transactions) != 2 || res.Transactions[0].Amount != -23.1 || res.Report.Rejected != 0 {
		t.Fatal("opération multiligne perdue", res.Report)
	}
}
func TestIndependentTotalsDetectBalancedOmissions(t *testing.T) {
	rows := sampleRows()
	// Remove equal debit and credit: the net is unchanged, but independent
	// document totals must still fail. No alteration of the source balances.
	rows[3].Words[len(rows[3].Words)-1].S = "23,10"
	rows[len(rows)-1].Words[len(rows[len(rows)-1].Words)-1].S = "306,46"
	rows = append(rows[:2], rows[4:]...)
	// Keep one zero operation so that the parser can produce a diagnostic.
	rows = append(rows[:2], append([]Row{row(88, w("01.06", 50, 35), w("TEST", 130, 40), w("0,00", 405, 38))}, rows[2:]...)...)
	res, e := ParseLocal(rows)
	if e != nil {
		t.Fatal(e)
	}
	if !Validate(res) || res.Validated || res.Report.TotalsControl != "failed" {
		t.Fatal("équilibre net considéré à tort comme preuve de complétude")
	}
}
func TestInvalidMissingAndNegativeAmounts(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func([]Row)
		issue  bool
	}{
		{"missing", func(rs []Row) { rs[2].Words = rs[2].Words[:len(rs[2].Words)-1] }, true},
		{"invalid_date", func(rs []Row) { rs[2].Words[0].S = "31.02" }, true},
		{"two_amounts", func(rs []Row) { rs[2].Words = append(rs[2].Words, w("1,00", 480, 30)) }, true},
		{"negative", func(rs []Row) { rs[2].Words[len(rs[2].Words)-1].S = "-23,10" }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res, e := ParseLocal(func() []Row { rs := sampleRows(); tc.change(rs); return rs }())
			if e != nil {
				t.Fatal(e)
			}
			if res.Validated || res.Report.Completeness != "failed" {
				t.Fatal("lecture incorrecte validée", res.Report)
			}
			if tc.name == "negative" && res.Transactions[0].Amount != 23.1 {
				t.Fatal("montant négatif source mal interprété")
			}
		})
	}
}

func TestAllRejectedCandidatesStillProduceAnImportReport(t *testing.T) {
	rows := sampleRows()
	rows = append(rows[:3], rows[len(rows)-1])
	rows[2].Words = rows[2].Words[:len(rows[2].Words)-1]
	res, e := ParseLocal(rows)
	if e != nil || res == nil || len(res.Transactions) != 0 || res.Report.Detected != 1 || res.Report.Rejected != 1 || res.Report.Completeness != "failed" {
		t.Fatal("rejected candidates disappeared with the parser error", e)
	}
}
