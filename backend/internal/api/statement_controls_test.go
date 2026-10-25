package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/lib/pq"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	database "themis/internal/db"
	"themis/internal/models"
	"time"
)

func statementTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("THEMIS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("THEMIS_TEST_DATABASE_URL non défini")
	}
	admin, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("statement_controls_%d", time.Now().UnixNano())
	if _, err = admin.Exec(`CREATE SCHEMA ` + pq.QuoteIdentifier(schema)); err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(dsn)
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	db, err := sql.Open("postgres", u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Close()
		admin.Exec(`DROP SCHEMA ` + pq.QuoteIdentifier(schema) + ` CASCADE`)
		admin.Close()
	})
	if err = database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO users(id,username,password_hash,role) VALUES(1,'DOCUMENT TEST','TEST','admin'),(2,'OTHER TEST','TEST','user');
 INSERT INTO accounts(id,owner_id,name,kind,tracking_mode) VALUES(11,1,'Document TEST','courant','statements'),(12,1,'Livret TEST','livret_a','manual'),(13,1,'PEA TEST','bourse','manual'),(21,2,'SECRET ACCOUNT','courant','statements');
 INSERT INTO statements(id,owner_id,account_id,filename,period_start,period_end,opening_balance,closing_balance,validated) VALUES
 (101,1,11,'A TEST.pdf','2026-08-22','2026-09-21',1000,900,true),
 (102,1,11,'B TEST.pdf','2026-09-22','2026-10-21',900,950,true),
 (201,2,21,'SECRET.pdf','2026-09-01','2026-09-30',100,101,true);
 INSERT INTO transactions(statement_id,owner_id,tx_date,label,amount,hash) VALUES(101,1,'2026-09-01','TEST DEBIT',-100,'a'),(102,1,'2026-09-23','TEST CREDIT',50,'b'),(201,2,'2026-09-03','SECRET LABEL',1,'secret');`)
	if err != nil {
		t.Fatal(err)
	}
	return db
}
func statementCall(s *Server, handler http.HandlerFunc, method, path, body string, owner int64) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	parts := strings.Split(path, "/")
	req.SetPathValue("id", parts[3])
	req = req.WithContext(context.WithValue(req.Context(), ctxUserKey, &authUser{ID: owner, Role: "user"}))
	w := httptest.NewRecorder()
	handler(w, req)
	return w
}
func TestStatementDocumentaryModesAndExactCoveragePostgres(t *testing.T) {
	db := statementTestDB(t)
	s := &Server{DB: db}
	now := reliabilityTestDate(t, "2026-10-25")
	r, err := s.loadReliability(context.Background(), 1, "2026-09", now)
	if err != nil {
		t.Fatal(err)
	}
	p := periodFromReliability(r, "2026-09-23", 30, false, 25)
	if p.Completeness != "complete" || len(r.Accounts) != 3 || r.Accounts[1].Status != "not_required" {
		t.Fatalf("manual accounts degraded proof: %+v", r)
	}
	if _, err = db.Exec(`INSERT INTO accounts(id,owner_id,name,kind,tracking_mode) VALUES(14,1,'Second documentary','courant','statements')`); err != nil {
		t.Fatal(err)
	}
	if s.heroMonthCovered(1, "2026-09") {
		t.Fatal("second documentary account without source ignored")
	}
	w := statementCall(s, s.updateAccountTracking, "PATCH", "/api/accounts/14", `{"tracking_mode":"manual"}`, 2)
	if w.Code != 404 {
		t.Fatal("cross-user mode change", w.Code)
	}
	w = statementCall(s, s.updateAccountTracking, "PATCH", "/api/accounts/14", `{"tracking_mode":"manual"}`, 1)
	if w.Code != 204 || !s.heroMonthCovered(1, "2026-09") {
		t.Fatal("manual mode did not exclude account", w.Code, w.Body)
	}
	if _, err = db.Exec(`UPDATE statements SET closing_balance=950.01 WHERE id=102`); err != nil {
		t.Fatal(err)
	}
	if s.heroMonthCovered(1, "2026-09") {
		t.Fatal("one cent mismatch accepted as coverage")
	}
	all, err := loadCheckedStatements(context.Background(), db, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, st := range all {
		if st.ID == 102 && (st.DifferenceCents == nil || *st.DifferenceCents != 1 || st.EffectiveValidated || st.CanValidate) {
			t.Fatal("inexact reconciliation", st)
		}
	}
	raw, _ := json.Marshal(all)
	if strings.Contains(string(raw), "SECRET") {
		t.Fatal("foreign statements leaked")
	}
	if _, err = db.Exec(`UPDATE accounts SET tracking_mode='manual' WHERE owner_id=1`); err != nil {
		t.Fatal(err)
	}
	if s.heroMonthCovered(1, "2026-09") {
		t.Fatal("no documentary account became complete")
	}
}
func TestStatementValidationHistoryAndDeletionPostgres(t *testing.T) {
	db := statementTestDB(t)
	s := &Server{DB: db}
	call := func(method, id, body string, owner int64) *httptest.ResponseRecorder {
		handler := s.validateStatement
		if method == "DELETE" {
			handler = s.deleteStatement
		}
		return statementCall(s, handler, method, "/api/statements/"+id+"/validation", body, owner)
	}
	if w := call("PATCH", "102", `{"action":"reopen","note":"Review original TEST"}`, 2); w.Code != 404 {
		t.Fatal("IDOR", w.Code)
	}
	if w := call("DELETE", "102", "", 2); w.Code != 404 {
		t.Fatal("delete IDOR", w.Code)
	}
	if w := call("PATCH", "102", `{"action":"reopen","note":"Review original TEST"}`, 1); w.Code != 204 {
		t.Fatal(w.Code, w.Body)
	}
	if w := call("PATCH", "102", `{"action":"reopen","note":"Review original TEST"}`, 1); w.Code != 409 {
		t.Fatal("duplicate transition accepted")
	}
	if w := call("PATCH", "102", `{"action":"validate","note":"short"}`, 1); w.Code != 400 {
		t.Fatal("untraced review accepted")
	}
	if _, err := db.Exec(`CREATE FUNCTION fail_review() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.note='FAIL REVIEW TEST' THEN RAISE EXCEPTION 'injected'; END IF;RETURN NEW;END $$;CREATE TRIGGER fail_review BEFORE INSERT ON statement_validation_history FOR EACH ROW EXECUTE FUNCTION fail_review()`); err != nil {
		t.Fatal(err)
	}
	if w := call("PATCH", "102", `{"action":"validate","note":"FAIL REVIEW TEST"}`, 1); w.Code != 500 {
		t.Fatal("history failure hidden")
	}
	var valid bool
	if err := db.QueryRow(`SELECT validated FROM statements WHERE id=102`).Scan(&valid); err != nil || valid {
		t.Fatal("history failure did not roll back")
	}
	if _, err := db.Exec(`ALTER TABLE analyses RENAME TO unavailable_analyses`); err != nil {
		t.Fatal(err)
	}
	if w := call("PATCH", "102", `{"action":"validate","note":"Late SQL failure TEST"}`, 1); w.Code != 500 {
		t.Fatal("late SQL error hidden")
	}
	var historyCount int
	if err := db.QueryRow(`SELECT validated,(SELECT count(*) FROM statement_validation_history WHERE resource_id=102) FROM statements WHERE id=102`).Scan(&valid, &historyCount); err != nil || valid || historyCount != 1 {
		t.Fatal("late SQL error did not roll back state and history", valid, historyCount, err)
	}
	if _, err := db.Exec(`ALTER TABLE unavailable_analyses RENAME TO analyses`); err != nil {
		t.Fatal(err)
	}
	if w := call("PATCH", "102", `{"action":"validate","note":"Checked original TEST"}`, 1); w.Code != 204 {
		t.Fatal(w.Code, w.Body)
	}
	rows, err := loadCheckedStatements(context.Background(), db, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, st := range rows {
		if st.ID == 102 && (st.ValidationSource != "manual" || len(st.History) != 2) {
			t.Fatal("review not traced", st)
		}
	}
	if _, err = db.Exec(`INSERT INTO analyses(owner_id,month,content) VALUES(1,'2026-09','STALE'),(2,'2026-09','SECRET');INSERT INTO transactions(statement_id,owner_id,tx_date,label,amount,hash) VALUES(102,2,'2026-09-24','FOREIGN SOURCE',1,'bad-link')`); err != nil {
		t.Fatal(err)
	}
	if w := call("DELETE", "102", "", 1); w.Code != 409 {
		t.Fatal("foreign cascade not blocked", w.Code)
	}
	if _, err = db.Exec(`DELETE FROM transactions WHERE hash='bad-link'`); err != nil {
		t.Fatal(err)
	}
	if w := call("DELETE", "102", "", 1); w.Code != 204 {
		t.Fatal(w.Code, w.Body)
	}
	var txs, history, analyses int
	if err = db.QueryRow(`SELECT (SELECT count(*) FROM transactions WHERE statement_id=102),(SELECT count(*) FROM statement_validation_history WHERE resource_id=102 AND statement_id IS NULL AND owner_id=1),(SELECT count(*) FROM analyses WHERE owner_id=1)`).Scan(&txs, &history, &analyses); err != nil || txs != 0 || history != 3 || analyses != 0 {
		t.Fatal("cascade/history/cache", txs, history, analyses, err)
	}
	if err = db.QueryRow(`SELECT count(*) FROM analyses WHERE owner_id=2`).Scan(&analyses); err != nil || analyses != 1 {
		t.Fatal("foreign cache deleted")
	}
}
func TestStatementPeriodControlsAndUnknownAmounts(t *testing.T) {
	str := func(s string) *string { return &s }
	cent := func(n int64) *int64 { return &n }
	acct := int64(1)
	makeSt := func(id int64, a, b string, opening, closing int64) checkedStatement {
		st := checkedStatement{Statement: models.Statement{ID: id, PeriodStart: str(a), PeriodEnd: str(b), Validated: true}, AccountID: &acct, OpeningCents: cent(opening), ClosingCents: cent(closing), DifferenceCents: cent(0), Controls: []statementControl{}}
		assessStatement(&st, 0, 0)
		return st
	}
	rows := []checkedStatement{makeSt(1, "2026-08-21", "2026-09-21", 10000, 9000), makeSt(2, "2026-09-21", "2026-10-21", 9000, 8000)}
	compareStatementPeriods(rows)
	for _, st := range rows {
		for _, c := range st.Controls {
			if c.Code == "overlap" || c.Status == "failed" {
				t.Fatal("shared boundary became false anomaly")
			}
		}
	}
	rows[1].OpeningCents = cent(9001)
	rows[1].Controls = nil
	compareStatementPeriods(rows)
	if rows[1].CanValidate {
		t.Fatal("confirmed balance discontinuity can be validated")
	}
	rows = []checkedStatement{makeSt(1, "2026-08-21", "2026-09-21", 10000, 9000), makeSt(2, "2026-09-20", "2026-10-21", 9000, 8000)}
	compareStatementPeriods(rows)
	if rows[0].Status != "to_review" || rows[1].Status != "to_review" {
		t.Fatal("overlap not disclosed")
	}
	rows = []checkedStatement{makeSt(1, "2026-08-21", "2026-09-21", 10000, 9000), makeSt(2, "2026-09-23", "2026-10-21", 9000, 8000)}
	compareStatementPeriods(rows)
	found := false
	for _, c := range rows[1].Controls {
		if c.Code == "continuity" && c.Status == "unknown" {
			found = true
		}
	}
	if !found {
		t.Fatal("gap not disclosed")
	}
	unknown := checkedStatement{Statement: models.Statement{ID: 3, Validated: true}}
	assessStatement(&unknown, 0, 0)
	if unknown.EffectiveValidated || unknown.DifferenceCents != nil || unknown.CanValidate {
		t.Fatal("missing evidence became successful zero")
	}
}

func TestSimilarOccurrencesAreInformationalNotBlocking(t *testing.T) {
	start, end := "2026-10-01", "2026-10-31"
	opening, closing, zero := int64(10000), int64(10000), int64(0)
	account := int64(1)
	st := checkedStatement{Statement: models.Statement{ID: 44, PeriodStart: &start, PeriodEnd: &end, Validated: true}, AccountID: &account, OpeningCents: &opening, ClosingCents: &closing, DifferenceCents: &zero, Controls: []statementControl{}}
	assessStatement(&st, 0, 4)
	if !st.CanValidate || !st.EffectiveValidated || st.Status != "validated" {
		t.Fatalf("similar occurrences blocked a reconciled statement: %+v", st)
	}
	for _, control := range st.Controls {
		if control.Code == "similar_transactions" && control.Status != "informational" {
			t.Fatalf("similarity must remain informational: %+v", control)
		}
	}
}

func TestMissingOfficialCountIsNotApplicableAndDoesNotBlock(t *testing.T) {
	start, end := "2026-10-01", "2026-10-31"
	opening, closing, zero := int64(10000), int64(10000), int64(0)
	account := int64(1)
	st := checkedStatement{ImportReport: &models.ImportReport{}, Statement: models.Statement{ID: 45, PeriodStart: &start, PeriodEnd: &end, Validated: true}, AccountID: &account, OpeningCents: &opening, ClosingCents: &closing, DifferenceCents: &zero, Controls: []statementControl{}}
	assessStatement(&st, 0, 0)
	if !st.CanValidate || !st.EffectiveValidated || st.Status != "validated" {
		t.Fatalf("absent official count blocked a reconciled statement: %+v", st)
	}
	for _, control := range st.Controls {
		if control.Code == "declared_count" && control.Status != "not_applicable" {
			t.Fatalf("absent official count was not marked non-applicable: %+v", control)
		}
	}
}

func TestStatementZeroOperationsAndPartialDuplicateImportPostgres(t *testing.T) {
	db := statementTestDB(t)
	s := &Server{DB: db}
	balance := 50.0
	empty := &models.ParseResult{OpeningBalance: &balance, ClosingBalance: &balance, PeriodStart: "2026-07-01", PeriodEnd: "2026-07-31", Validated: true, Parser: "local"}
	r := httptest.NewRequest("POST", "/api/statements", nil).WithContext(context.WithValue(context.Background(), ctxUserKey, &authUser{ID: 1}))
	w := httptest.NewRecorder()
	s.finishStatementImport(w, r, "EMPTY TEST.pdf", empty, 11, 1, "")
	if w.Code != 201 {
		t.Fatal("empty source not preserved", w.Code, w.Body)
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM statements WHERE filename='EMPTY TEST.pdf' AND owner_id=1`).Scan(&count); err != nil || count != 1 {
		t.Fatal("legitimate empty statement deleted", count, err)
	}
	opening, closing := 0.0, 10.0
	partial := &models.ParseResult{OpeningBalance: &opening, ClosingBalance: &closing, PeriodStart: "2026-09-01", PeriodEnd: "2026-09-30", Validated: true, Parser: "local", Transactions: []models.Transaction{{Date: "2026-09-01", Label: "TEST DEBIT", Amount: -100, Hash: "a"}, {Date: "2026-09-03", Label: "NEW SOURCE TEST", Amount: 110, Hash: "new"}}}
	inserted, skipped, id, err := s.persist("PARTIAL TEST.pdf", partial, 11, 1)
	if err != nil || inserted != 1 || skipped != 1 || partial.Validated {
		t.Fatal("partial dedup not disclosed", inserted, skipped, err)
	}
	var valid bool
	if err = db.QueryRow(`SELECT validated FROM statements WHERE id=$1`, id).Scan(&valid); err != nil || valid {
		t.Fatal("parsing-only validation retained")
	}
	all, err := loadCheckedStatements(context.Background(), db, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, st := range all {
		if st.ID == id && (st.CanValidate || st.DifferenceCents == nil || *st.DifferenceCents != -10000) {
			t.Fatal("dedup reconciliation inconsistency hidden", st)
		}
	}
	w = statementCall(s, s.validateStatement, "PATCH", fmt.Sprintf("/api/statements/%d/validation", id), `{"action":"validate","note":"Review original TEST"}`, 1)
	if w.Code != 409 {
		t.Fatal("manual review bypassed arithmetic", w.Code)
	}
}

func TestStatementSourcesContinuityAndUnknownPeriodPostgres(t *testing.T) {
	db := statementTestDB(t)
	s := &Server{DB: db}
	if _, err := db.Exec(`INSERT INTO accounts(id,owner_id,name,kind,tracking_mode) VALUES(14,1,'Second current TEST','courant','statements');
 INSERT INTO statements(id,owner_id,account_id,filename,period_start,period_end,opening_balance,closing_balance,validated) VALUES(301,1,14,'SECOND TEST','2026-09-01','2026-09-30',0,0,true);
 INSERT INTO account_snapshots(account_id,snap_date,balance) VALUES(12,'2020-01-01',999.99)`); err != nil {
		t.Fatal(err)
	}
	if !s.heroMonthCovered(1, "2026-09") {
		t.Fatal("two documentary accounts and an old manual snapshot should be covered")
	}
	if _, err := db.Exec(`UPDATE statements SET opening_balance=900.01,closing_balance=950.01 WHERE id=102`); err != nil {
		t.Fatal(err)
	}
	if s.heroMonthCovered(1, "2026-09") {
		t.Fatal("arithmetically balanced discontinuity accepted")
	}
	all, err := loadCheckedStatements(context.Background(), db, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, st := range all {
		if st.ID == 102 && (st.CanValidate || st.EffectiveValidated) {
			t.Fatal("confirmed discontinuity appears validated")
		}
	}
	for _, source := range []string{"102", "201"} {
		req := httptest.NewRequest("GET", "/api/transactions?view=report&statement_id="+source, nil).WithContext(context.WithValue(context.Background(), ctxUserKey, &authUser{ID: 1}))
		w := httptest.NewRecorder()
		s.listTransactions(w, req)
		var report transactionReport
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &report) != nil {
			t.Fatal("source query", w.Code, w.Body)
		}
		if source == "102" && (report.Total != 1 || report.Summary.Income.String() != "50.00") {
			t.Fatal("source window or cents lost", w.Body)
		}
		if source == "102" && (report.Transactions[0].StatementValidated == nil || *report.Transactions[0].StatementValidated) {
			t.Fatal("transaction provenance trusted an invalid legacy flag")
		}
		if source == "201" && report.Total != 0 {
			t.Fatal("foreign source disclosed")
		}
	}
	if _, err = db.Exec(`UPDATE statements SET opening_balance=900,closing_balance=950 WHERE id=102;
 INSERT INTO statements(id,owner_id,account_id,filename,period_start,period_end,opening_balance,closing_balance,validated) VALUES(302,1,11,'UNKNOWN END TEST','2026-09-01',NULL,0,0,true)`); err != nil {
		t.Fatal(err)
	}
	r, err := s.loadReliability(context.Background(), 1, "2026-09", reliabilityTestDate(t, "2026-10-25"))
	if err != nil || r.Accounts[0].UnknownPeriods != 1 {
		t.Fatal("missing end converted to SQL error or coverage", r, err)
	}
	request := httptest.NewRequest("GET", "/api/export", nil).WithContext(context.WithValue(context.Background(), ctxUserKey, &authUser{ID: 1}))
	w := httptest.NewRecorder()
	s.exportAll(w, request)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"tracking_mode": "manual"`) || strings.Contains(w.Body.String(), "SECRET") {
		t.Fatal("export lost tracking or leaked foreign resources", w.Code)
	}
}
