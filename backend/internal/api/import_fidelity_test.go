package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"themis/internal/importaudit"
	"themis/internal/models"
	"themis/internal/money"
	"themis/internal/parser"
	"themis/internal/testfixture"
)

func pdfToolRequired(t *testing.T) {
	t.Helper()
	if _, e := exec.LookPath("pdftotext"); e != nil {
		if os.Getenv("THEMIS_REQUIRE_PDF_TESTS") == "1" {
			t.Fatal("Poppler requis")
		}
		t.Skip("Poppler requis pour l'intégration PDF")
	}
}
func verifySourceImport(t *testing.T, data []byte) {
	t.Helper()
	pdfToolRequired(t)
	db := statementTestDB(t)
	if _, e := db.Exec(`DELETE FROM transactions WHERE owner_id=1;DELETE FROM statements WHERE owner_id=1`); e != nil {
		t.Fatal(e)
	}
	srv := &Server{DB: db}
	res, e := parser.Parse(context.Background(), data, nil)
	if e != nil {
		t.Fatal(e)
	}
	n, skip, id, e := srv.persist("REFERENCE ANONYME TEST.pdf", res, 11, 1)
	if e != nil || n != 81 || skip != 0 {
		t.Fatalf("persist count=%d skip=%d err=%v", n, skip, e)
	}
	var count int
	var debit, credit, net, closing string
	e = db.QueryRow(`SELECT count(*),(-sum(amount) FILTER(WHERE amount<0))::text,(sum(amount) FILTER(WHERE amount>0))::text,sum(amount)::text FROM transactions WHERE statement_id=$1 AND owner_id=1`, id).Scan(&count, &debit, &credit, &net)
	if e != nil || count != 81 || debit != "4293.85" || credit != "2965.85" || net != "-1328.00" {
		t.Fatal("SQL reconciliation", count, debit, credit, net, e)
	}
	if e = db.QueryRow(`SELECT closing_balance::text FROM statements WHERE id=$1 AND owner_id=1`, id).Scan(&closing); e != nil || closing != "1164.88" {
		t.Fatal("closing", closing, e)
	}
	var first int64
	db.QueryRow(`SELECT min(id) FROM transactions WHERE statement_id=$1`, id).Scan(&first)
	// Manual metadata must survive exact-source replay.
	if _, e = db.Exec(`UPDATE transactions SET display_label='CORRECTION MANUELLE TEST' WHERE id=$1`, first); e != nil {
		t.Fatal(e)
	}
	var tagID int64
	if e = db.QueryRow(`INSERT INTO tags(name,owner_id) VALUES('MANUAL TAG TEST',1) RETURNING id`).Scan(&tagID); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(`INSERT INTO transaction_tags(transaction_id,tag_id) VALUES($1,$2)`, first, tagID); e != nil {
		t.Fatal(e)
	}
	again, e := parser.Parse(context.Background(), data, nil)
	if e != nil {
		t.Fatal(e)
	}
	n, skip, replay, e := srv.persist("NOM DIFFERENT TEST.pdf", again, 11, 1)
	if e != nil || n != 0 || skip != 81 || replay != id {
		t.Fatal("source replay not idempotent", n, skip, replay, e)
	}
	var display string
	db.QueryRow(`SELECT display_label FROM transactions WHERE id=$1`, first).Scan(&display)
	var tagCount int
	db.QueryRow(`SELECT count(*) FROM transaction_tags WHERE transaction_id=$1 AND tag_id=$2`, first, tagID).Scan(&tagCount)
	if tagCount != 1 {
		t.Fatal("manual tag lost on replay")
	}
	if display != "CORRECTION MANUELLE TEST" {
		t.Fatal("manual metadata overwritten")
	}
	req := httptest.NewRequest("GET", "/api/transactions?view=report&statement_id="+itoaTest(id)+"&limit=100", nil)
	req = req.WithContext(context.WithValue(req.Context(), ctxUserKey, &authUser{ID: 1}))
	w := httptest.NewRecorder()
	srv.listTransactions(w, req)
	if w.Code != 200 {
		t.Fatal("API report", w.Code, w.Body)
	}

	var api struct {
		Total        int `json:"total"`
		Transactions []struct {
			Amount     money.Cents `json:"amount_cents"`
			SourcePage *int        `json:"source_page"`
			SourceLine *int        `json:"source_line"`
		} `json:"transactions"`
		Summary struct {
			Net             json.Number `json:"net"`
			TransferDebits  json.Number `json:"transfer_debits"`
			TransferCredits json.Number `json:"transfer_credits"`
		} `json:"summary"`
	}
	if e = json.Unmarshal(w.Body.Bytes(), &api); e != nil || api.Total != 81 || len(api.Transactions) != 81 {
		t.Fatal("API count/pagination", api.Total, len(api.Transactions), e)
	}
	var apiDebits, apiCredits money.Cents
	for _, tx := range api.Transactions {
		if tx.Amount < 0 {
			apiDebits -= tx.Amount
		} else {
			apiCredits += tx.Amount
		}
		if tx.SourcePage == nil || tx.SourceLine == nil {
			t.Fatal("API source provenance missing")
		}
	}
	apiNet, e1 := money.Parse(api.Summary.Net.String())
	td, e2 := money.Parse(api.Summary.TransferDebits.String())
	tc, e3 := money.Parse(api.Summary.TransferCredits.String())
	if e1 != nil || e2 != nil || e3 != nil || apiDebits != 429385 || apiCredits != 296585 || apiNet+tc-td != -132800 {
		t.Fatal("API source amounts or transfer reconciliation diverge", apiDebits, apiCredits, apiNet, td, tc)
	}

	statements, e := loadCheckedStatements(context.Background(), db, 1)
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, st := range statements {
		if st.ID == id {
			found = true
			if st.TxCount != 81 || st.ImportReport == nil || st.ImportReport.Inserted != 81 || st.ImportReport.TotalsControl != "passed" {
				t.Fatal("statement import report missing or inconsistent")
			}
		}
	}
	if !found {
		t.Fatal("statement missing")
	}
	audited, e := importaudit.Run(context.Background(), db, 1)
	if e != nil {
		t.Fatal(e)
	}
	for _, a := range audited {
		if a.StatementID == id && (a.TotalsControl != "passed" || a.BalanceControl != "passed" || a.CountControl != "unverifiable" || !a.SourceTraceable) {
			t.Fatal("historical audit inconsistent", a)
		}
	}
	foreign, e := importaudit.Run(context.Background(), db, 2)
	if e != nil {
		t.Fatal(e)
	}
	for _, a := range foreign {
		if a.StatementID == id {
			t.Fatal("owner isolation broken")
		}
	}
	// The same source on another owned account must not collide with source keys.
	n, skip, _, e = srv.persist("SECOND COMPTE TEST.pdf", again, 12, 1)
	if e != nil || n != 81 || skip != 0 {
		t.Fatal("account source scoping", n, skip, e)
	}
}
func itoaTest(id int64) string { return strconv.FormatInt(id, 10) }
func TestSynthetic009ExtractionToDatabaseAndAPI(t *testing.T) {
	verifySourceImport(t, testfixture.Statement009PDF())
}
func TestReference009ImportLocal(t *testing.T) {
	path := os.Getenv("THEMIS_REFERENCE_PDF")
	if path == "" {
		t.Skip("PDF privé optionnel")
	}
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal("référence indisponible")
	}
	verifySourceImport(t, b)
}
func sourceResult(hashSeed string) *models.ParseResult {
	h := sha256.Sum256([]byte(hashSeed))
	hash := hex.EncodeToString(h[:])
	return &models.ParseResult{DocumentHash: hash, Parser: "local", PeriodStart: "2026-09-01", PeriodEnd: "2026-09-30", Transactions: []models.Transaction{{Date: "2026-09-01", Label: "IDENTICAL TEST", Amount: -10, SourcePage: 1, SourceLine: 3}, {Date: "2026-09-01", Label: "IDENTICAL TEST", Amount: -10, SourcePage: 1, SourceLine: 4}}, Report: &models.ImportReport{Detected: 2, Parsed: 2, Completeness: "unknown", Candidates: []models.ImportCandidate{{Index: 0, Page: 1, Line: 3, Status: "parsed"}, {Index: 1, Page: 1, Line: 4, Status: "parsed"}}}}
}
func TestSourceImportAtomicitySimilarityAndOwnership(t *testing.T) {
	db := statementTestDB(t)
	if _, e := db.Exec(`DELETE FROM transactions WHERE owner_id=1;DELETE FROM statements WHERE owner_id=1`); e != nil {
		t.Fatal(e)
	}
	srv := &Server{DB: db}
	res := sourceResult("identity A")
	n, _, id, e := srv.persist("SYNTHETIC TEST.pdf", res, 11, 1)
	if e != nil || n != 2 {
		t.Fatal("legitimate identical transactions lost", n, e)
	}
	var distinct int
	db.QueryRow(`SELECT count(DISTINCT hash) FROM transactions WHERE statement_id=$1`, id).Scan(&distinct)
	if distinct != 2 {
		t.Fatal("source occurrence keys collide")
	}
	// A different extraction of the same bank period is suspended even when
	// all labels have changed, so legacy LLM/OCR rewrites cannot evade hints.
	overlap := sourceResult("overlap identity")
	overlap.Transactions[0].Label = "DIFFERENT SOURCE A"
	overlap.Transactions[1].Label = "DIFFERENT SOURCE B"
	if _, _, _, e = srv.persist("OVERLAP TEST.pdf", overlap, 11, 1); e == nil {
		t.Fatal("overlapping source silently imported")
	}
	// Resemblance across different PDFs is not proof of a duplicate.
	similar := sourceResult("identity B")
	similar.PeriodStart = "2026-10-01"
	similar.PeriodEnd = "2026-10-31"
	if _, _, _, e = srv.persist("OTHER SOURCE TEST.pdf", similar, 11, 1); e == nil {
		t.Fatal("similarity silently suppressed or imported")
	}
	if _, _, _, e = srv.persist("FOREIGN ACCOUNT TEST.pdf", sourceResult("identity C"), 21, 1); e == nil {
		t.Fatal("foreign account import allowed")
	}
	altered := sourceResult("identity A")
	altered.Transactions = altered.Transactions[:1]
	if _, _, _, e = srv.persist("REPARSE TEST.pdf", altered, 11, 1); e == nil {
		t.Fatal("changed parser result silently replaced history")
	}
	failed := sourceResult("identity D")
	failed.PeriodStart = "2026-12-01"
	failed.PeriodEnd = "2026-12-31"
	failed.Transactions[0].Label = "FAILED SOURCE A"
	failed.Transactions[1].Label = "FAILED SOURCE B"
	failed.Report.Issues = []string{"conflicting_declared_totals"}
	if _, _, _, e = srv.persist("BAD TEST.pdf", failed, 11, 1); e == nil || !strings.Contains(e.Error(), "contrôles d'import en échec") {
		t.Fatal("failed extraction was not rejected by its report", e)
	}
	// A SQL error on the second entry must roll back the first and its statement.
	if _, e = db.Exec(`CREATE FUNCTION fail_second_test() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.source_index=1 THEN RAISE EXCEPTION 'synthetic insert failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER fail_second BEFORE INSERT ON transactions FOR EACH ROW EXECUTE FUNCTION fail_second_test()`); e != nil {
		t.Fatal(e)
	}
	sqlFailure := sourceResult("identity E")
	sqlFailure.PeriodStart = "2026-11-01"
	sqlFailure.PeriodEnd = "2026-11-30"
	sqlFailure.Transactions[0].Label = "NEW A"
	sqlFailure.Transactions[1].Label = "NEW B"
	if _, _, _, e = srv.persist("SQL FAILURE TEST.pdf", sqlFailure, 11, 1); e == nil || !strings.Contains(e.Error(), "échec d'insertion") {
		t.Fatal("SQL error ignored")
	}
	var partial int
	db.QueryRow(`SELECT count(*) FROM statements WHERE filename='SQL FAILURE TEST.pdf'`).Scan(&partial)
	if partial != 0 || sqlFailure.Report.Inserted != 0 || sqlFailure.Report.Candidates[0].Status != "parsed" {
		t.Fatal("partial import or misleading report survived rollback")
	}
	rollback, e := os.ReadFile("../db/rollbacks/026_import_fidelity.sql")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(string(rollback)); e == nil {
		t.Fatal("rollback lost source provenance")
	}
	var raw []byte
	db.QueryRow(`SELECT import_report FROM statements WHERE id=$1`, id).Scan(&raw)
	if strings.Contains(string(raw), "IDENTICAL TEST") {
		t.Fatal("raw bank labels duplicated in report")
	}
}

func TestConcurrentSourceReplay(t *testing.T) {
	db := statementTestDB(t)
	if _, e := db.Exec(`DELETE FROM transactions WHERE owner_id=1;DELETE FROM statements WHERE owner_id=1`); e != nil {
		t.Fatal(e)
	}
	srv := &Server{DB: db}
	type outcome struct {
		inserted, skipped int
		id                int64
		e                 error
	}
	results := make(chan outcome, 2)
	for i := 0; i < 2; i++ {
		go func() {
			a, b, id, e := srv.persist("CONCURRENT SOURCE TEST.pdf", sourceResult("concurrent same source"), 11, 1)
			results <- outcome{a, b, id, e}
		}()
	}
	a, b := <-results, <-results
	if a.e != nil || b.e != nil || a.id != b.id || a.inserted+b.inserted != 2 || a.skipped+b.skipped != 2 {
		t.Fatal("concurrent source replay", a, b)
	}
}

func TestDeclaredTotalsPreventManualValidationOfBalancedMissingEntries(t *testing.T) {
	db := statementTestDB(t)
	// Seed the audit evidence of a missing equal debit/credit pair. The balance
	// still reconciles, but independent gross totals must prohibit validation.
	if _, e := db.Exec(`UPDATE statements SET import_report='{"parsed":1,"detected":1,"rejected":0,"declared_debits_cents":11000,"declared_credits_cents":1000,"expected_count":null,"issues":[],"candidates":[]}' WHERE id=101`); e != nil {
		t.Fatal(e)
	}
	srv := &Server{DB: db}
	w := statementCall(srv, srv.validateStatement, "PATCH", "/api/statements/101/validation", `{"action":"validate","note":"Comparaison originale TEST"}`, 1)
	if w.Code < 400 {
		t.Fatal("balanced missing debit/credit pair allowed manual validation")
	}
}
