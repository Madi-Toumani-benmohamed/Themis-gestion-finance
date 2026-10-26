package parser

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
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
	if res.Report.CountControl != "not_applicable" || res.Report.ExpectedCount != nil || res.Report.Completeness != "unknown" {
		t.Fatal("le compteur attendu ne doit pas être fabriqué")
	}
	if !res.Validated || res.Report.TotalsControl != "passed" {
		t.Fatal("totaux indépendants non vérifiés")
	}
}

func TestMissingTransactionFailsReconciliationEvenWithoutDeclaredCount(t *testing.T) {
	rows := sampleRows()
	rows = append(rows[:2], rows[3:]...) // omit the card debit; leave the source closing balance intact
	res, err := ParseLocal(rows)
	if err != nil {
		t.Fatal(err)
	}
	if res.Report.CountControl != "not_applicable" || res.Report.BalanceControl != "failed" || res.Validated {
		t.Fatalf("missing transaction must fail the balance proof: %+v", res.Report)
	}
}

func TestMissingSourceReferencePreventsAutomaticValidation(t *testing.T) {
	res, err := ParseLocal(sampleRows())
	if err != nil {
		t.Fatal(err)
	}
	res.Transactions[0].SourceLine = 0
	RefreshImportReport(res)
	if res.Validated || res.Report.TraceabilityControl != "failed" || res.Report.Completeness != "failed" {
		t.Fatalf("untraceable operation was automatically validated: %+v", res.Report)
	}
}

func TestLegitimateIdenticalPaymentsRemainSeparate(t *testing.T) {
	rows := sampleRows()
	rows = append(append(append([]Row{}, rows[:3]...), row(88, w("22.05", 50, 35), w("22.05", 90, 35), w("Carte", 130, 35), w("Burger", 170, 42), w("23,10", 405, 38))), rows[3:]...)
	rows[5].Words[1].S = "46,20"                     // independent debit total
	rows[6].Words[len(rows[6].Words)-1].S = "560,26" // closing balance reflects both payments
	res, err := ParseLocal(rows)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Transactions) != 3 || res.Transactions[0].Date != res.Transactions[1].Date || res.Transactions[0].Amount != res.Transactions[1].Amount || res.Transactions[0].Label != res.Transactions[1].Label {
		t.Fatalf("identical source occurrences were not preserved: %+v", res.Transactions)
	}
	if !res.Validated || res.Report.TraceabilityControl != "passed" || res.Report.Completeness != "unknown" {
		t.Fatalf("similarity or absent count incorrectly blocked validation/completeness: %+v", res.Report)
	}
}

func TestRepeatedTwentyTwoEuroTransfersAndCreditReversalRemainDistinct(t *testing.T) {
	rows := []Row{
		row(100, w("Date", 50, 30), w("Libellé", 110, 50), w("Débit", 400, 40), w("Crédit", 480, 45)),
		row(95, w("Ancien", 50, 45), w("solde", 100, 38), w("au", 142, 15), w("21.05.2026", 160, 70), w("306,46", 490, 40)),
		row(90, w("22.05", 50, 35), w("Virement", 110, 60), w("22,00", 405, 38)),
		row(88, w("23.05", 50, 35), w("Virement", 110, 60), w("22,00", 405, 38)),
		row(86, w("24.05", 50, 35), w("Annulation virement", 110, 120), w("22,00", 488, 42)),
		row(81, w("Total des opérations", 50, 160), w("44,00", 405, 38), w("22,00", 488, 42)),
		row(80, w("Nouveau", 50, 55), w("solde", 108, 38), w("22.06.2026", 168, 70), w("284,46", 490, 40)),
	}
	res, err := ParseLocal(rows)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Transactions) != 3 || res.Transactions[0].Amount != -22 || res.Transactions[1].Amount != -22 || res.Transactions[2].Amount != 22 {
		t.Fatalf("virements ou annulation confondus : %+v", res.Transactions)
	}
	if !res.Validated || res.Report.TotalsControl != "passed" || res.Report.BalanceControl != "passed" {
		t.Fatalf("les écritures distinctes ne se rapprochent pas : %+v", res.Report)
	}
}

func TestFXContinuationsDoNotBecomeOperationsAndPreserveExactReport(t *testing.T) {
	rows := []Row{
		{Page: 1, Words: []Word{w("Date", 50, 30), w("Libellé", 110, 50), w("Débit", 400, 40), w("Crédit", 480, 45)}},
		{Page: 1, Words: []Word{w("Ancien", 50, 45), w("solde", 100, 38), w("au", 142, 15), w("01.02.2026", 160, 70), w("611,81", 490, 40)}},
	}
	addOperation := func(page, line int, date, label, amount string, credit bool) {
		x := 405.0
		if credit {
			x = 488
		}
		rows = append(rows, Row{Page: page, Line: line, Words: []Word{w(date, 50, 35), w(label, 130, 220), w(amount, x, 42)}})
	}
	addOperation(2, 74, "03.02", "Carte Royal Hotel Monrovia", "46,49", false)
	fxRow := func(line int, amount, rate string) Row {
		words := []string{"Mt", "initial", ":", amount, "USD", "-", "taux", "de", "change", ":", "1", "USD", "=", rate, "Eur"}
		fragments := make([]Word, 0, len(words))
		for i, word := range words {
			fragments = append(fragments, w(word, float64(130+i*25), 20))
		}
		return Row{Page: 2, Line: line, Words: fragments}
	}
	rows = append(rows, fxRow(75, "55,00", "0,84527"))
	addOperation(2, 78, "05.02", "Carte Royal Hotel Monrovia", "27,70", false)
	rows = append(rows, fxRow(79, "32,48", "0,85283"))
	for i := 0; i < 65; i++ {
		addOperation(2, 80+i, "06.02", fmt.Sprintf("Carte Synthetic Shop %02d", i), "195,57", false)
	}
	addOperation(2, 145, "18.02", "Carte Synthetic Final", "195,61", false)
	addOperation(2, 146, "19.02", "Virement salaire synthétique", "12 482,50", true)
	rows = append(rows,
		Row{Page: 2, Line: 147, Words: []Word{w("Total des opérations", 50, 160), w("12 981,85", 405, 42), w("12 482,50", 488, 42)}},
		Row{Page: 2, Line: 148, Words: []Word{w("Nouveau", 50, 55), w("solde", 108, 38), w("au", 150, 15), w("20.02.2026", 168, 70), w("112,46", 490, 40)}},
	)
	res, err := ParseLocal(rows)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Transactions) != 69 || res.Report.Parsed != 69 || res.Report.Detected != 69 || res.Report.Rejected != 0 {
		t.Fatalf("les métadonnées FX ont modifié le registre des opérations: tx=%d report=%+v", len(res.Transactions), res.Report)
	}
	if res.Report.Debits != 1_298_185 || res.Report.Credits != 1_248_250 || res.Report.BalanceDifference == nil || *res.Report.BalanceDifference != 0 || !res.Validated {
		t.Fatalf("rapprochement synthétique attendu débits=12981.85 crédits=12482.50 écart=0 validation=true; report=%+v", res.Report)
	}
	if res.OpeningBalance == nil || *res.OpeningBalance != 611.81 || res.ClosingBalance == nil || *res.ClosingBalance != 112.46 {
		t.Fatalf("soldes synthétiques incorrects: opening=%v closing=%v", res.OpeningBalance, res.ClosingBalance)
	}
	if len(res.Report.Continuations) != 2 {
		t.Fatalf("2 conversions doivent rester attachées et traçables: %+v", res.Report.Continuations)
	}
	want := []struct {
		line  int
		cents int64
		rate  string
	}{{75, 5500, "0.84527"}, {79, 3248, "0.85283"}}
	for i, expected := range want {
		got := res.Report.Continuations[i]
		if got.Kind != "currency_conversion" || got.Page != 2 || got.Line != expected.line || got.AmountCents != expected.cents || got.Rate != expected.rate || got.ParentIndex != i {
			t.Fatalf("provenance/conversion incorrecte: %+v", got)
		}
	}
}

func TestSEPAReferenceAndAmountContinuationStayWithParent(t *testing.T) {
	rows := sampleRows()
	ref := row(89, w("Référence", 130, 65), w("SEPA", 200, 35), w("123456", 240, 40), w("8,00", 285, 38))
	ref.Page, ref.Line = 2, 17
	rows = append(rows[:3], append([]Row{ref}, rows[3:]...)...)
	res, err := ParseLocal(rows)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Transactions) != 2 || res.Report.Rejected != 0 || !strings.Contains(res.Transactions[0].Label, "8,00") {
		t.Fatalf("référence SEPA contenant un montant séparée à tort: %+v", res)
	}
	found := false
	for _, continuation := range res.Report.Continuations {
		if continuation.Kind == "sepa_reference" && continuation.Page == 2 && continuation.Line == 17 && continuation.ParentIndex == 0 {
			found = true
		}
	}
	if !found {
		t.Fatal("la ligne SEPA de continuation n'est pas traçable")
	}
}

func TestUnattachedOrMismatchedFXLineRemainsBlocking(t *testing.T) {
	rows := sampleRows()
	bad := row(89, w("Mt initial : 55,00 USD-taux de change : 1 USD=0,84527 Eur", 130, 400))
	bad.Page, bad.Line = 2, 9
	rows = append(rows[:3], append([]Row{bad}, rows[3:]...)...)
	res, err := ParseLocal(rows)
	if err != nil {
		t.Fatal(err)
	}
	unattached := false
	for _, candidate := range res.Report.Candidates {
		if candidate.Reason == "unattached_currency_conversion" {
			unattached = true
		}
	}
	if res.Validated || res.Report.Rejected != 1 || !unattached {
		t.Fatalf("conversion non rapprochée absorbée silencieusement: rejected=%d unattached=%v validated=%v", res.Report.Rejected, unattached, res.Validated)
	}
}

func TestSEPALabeledAmountInFinancialColumnRemainsBlocking(t *testing.T) {
	rows := sampleRows()
	operation := row(89, w("Reference", 130, 65), w("SEPA", 200, 35), w("8,00", 405, 38))
	operation.Page, operation.Line = 2, 18
	header := rows[0]
	header.Page, header.Line = 2, 17
	d, c, _ := findColumns(sampleRows())
	if !amountInFinancialColumn(operation, d, c) {
		t.Fatalf("la fixture de contrôle doit aligner son montant sur une colonne bancaire: debit=%v credit=%v", d, c)
	}
	rows = append(rows[:3], append([]Row{header, operation}, rows[3:]...)...)
	res, err := ParseLocal(rows)
	if err != nil {
		t.Fatal(err)
	}
	if res.Validated || res.Report.Rejected != 1 {
		t.Fatalf("un montant aligné sur une colonne bancaire ne doit pas être absorbé comme référence SEPA: rejected=%d detected=%d parsed=%d continuations=%+v candidates=%+v", res.Report.Rejected, res.Report.Detected, res.Report.Parsed, res.Report.Continuations, res.Report.Candidates)
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
