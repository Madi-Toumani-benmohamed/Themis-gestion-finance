package parser

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"themis/internal/models"
	"themis/internal/money"
)

// Optional private regression: the real PDF is never part of the repository.
// No banking labels or individual transactions are printed to the test logs.
func TestReferenceStatement009Local(t *testing.T) {
	path := os.Getenv("THEMIS_REFERENCE_PDF")
	if path == "" {
		t.Skip("THEMIS_REFERENCE_PDF non défini")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal("lecture du PDF de référence impossible")
	}
	rows, err := ExtractRowsContext(context.Background(), data)
	if err != nil {
		t.Fatal("extraction native impossible")
	}
	counts := map[int]int{}
	for _, r := range rows {
		counts[r.Page]++
	}
	t.Logf("native rows=%d pages=%v useful=%v", len(rows), counts, HasUsefulText(rows))
	private := os.Getenv("THEMIS_REFERENCE_AUDIT_DIR")
	save := func(name string, v any) {
		if private == "" {
			return
		}
		raw, e := json.MarshalIndent(v, "", "  ")
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(private, name), raw, 0600); e != nil {
			t.Fatal(e)
		}
	}
	summarize := func(name string, res *models.ParseResult) {
		if res == nil {
			return
		}
		var debit, credit money.Cents
		hashes := map[string]int{}
		for _, tx := range res.Transactions {
			value, e := money.FromLegacy(tx.Amount)
			if e != nil {
				t.Fatal("montant non exact")
			}
			if value < 0 {
				debit -= value
			} else {
				credit += value
			}
			hashes[tx.Hash]++
		}
		t.Logf("%s count=%d unique_hashes=%d debit=%s credit=%s net=%s validated=%v period=%s/%s", name, len(res.Transactions), len(hashes), debit, credit, credit-debit, res.Validated, res.PeriodStart, res.PeriodEnd)
		save(name+"-result.json", res)
	}
	save("native-rows.json", rows)
	local, err := ParseLocal(rows)
	if err == nil {
		summarize("native", local)
	} else {
		t.Log("native parsing unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	if OCRAvailable() && os.Getenv("THEMIS_REFERENCE_COMPARE_OCR") == "1" {
		ocrRows, e := ExtractRowsOCR(ctx, data)
		if e != nil {
			t.Fatal("OCR de référence impossible")
		}
		save("ocr-rows.json", ocrRows)
		ocr, e := ParseLocal(ocrRows)
		if e == nil {
			summarize("ocr", ocr)
		} else {
			t.Log("OCR parsing unavailable")
		}
	}
	result, err := Parse(ctx, data, nil)
	if err != nil {
		t.Fatal("parsing déterministe de référence impossible")
	}
	summarize("pipeline", result)
	var debit, credit money.Cents
	for _, tx := range result.Transactions {
		value, e := money.FromLegacy(tx.Amount)
		if e != nil {
			t.Fatal(e)
		}
		if value < 0 {
			debit -= value
		} else {
			credit += value
		}
	}
	if len(result.Transactions) != 81 || debit != 429385 || credit != 296585 || credit-debit != -132800 {
		t.Fatalf("référence non réconciliée : count=%d debit=%s credit=%s net=%s", len(result.Transactions), debit, credit, credit-debit)
	}
	if result.OpeningBalance == nil || result.ClosingBalance == nil {
		t.Fatal("soldes absents")
	}
	opening, e1 := money.FromLegacy(*result.OpeningBalance)
	closing, e2 := money.FromLegacy(*result.ClosingBalance)
	if e1 != nil || e2 != nil || opening != 249288 || closing != 116488 || opening+credit-debit != closing {
		t.Fatal("soldes de référence incohérents")
	}
	first, last := "", ""
	for _, tx := range result.Transactions {
		if first == "" || tx.Date < first {
			first = tx.Date
		}
		if last == "" || tx.Date > last {
			last = tx.Date
		}
	}
	if first != "2026-08-25" || last != "2026-09-21" {
		t.Fatal("dates d’opération non fidèles à la référence")
	}
	if result.Report == nil || result.Report.Detected != 81 || result.Report.Parsed != 81 || result.Report.Rejected != 0 || result.Report.TotalsControl != "passed" || result.Report.CountControl != "not_applicable" || result.Report.Completeness != "unknown" {
		t.Fatal("rapport de référence incohérent ou nombre attendu fabriqué")
	}
	if result.PeriodStart != "2026-08-21" || result.PeriodEnd != "2026-09-21" {
		t.Fatal("période bancaire confondue avec les dates des opérations")
	}
}
