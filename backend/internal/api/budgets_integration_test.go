package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"themis/internal/models"
	"themis/internal/money"
)

func TestBudgetsPostgres(t *testing.T) {
	dsn := os.Getenv("THEMIS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("THEMIS_TEST_DATABASE_URL non défini")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`
 CREATE TEMP TABLE domains(id bigint PRIMARY KEY,owner_id bigint,parent_id bigint,name text,is_transfer bool DEFAULT false);
 CREATE TEMP TABLE accounts(id bigint PRIMARY KEY,owner_id bigint,name text);
 CREATE TEMP TABLE statements(id bigserial PRIMARY KEY,account_id bigint,owner_id bigint,filename text,period_start date,period_end date,validated bool,uploaded_at timestamptz DEFAULT now());
 CREATE TEMP TABLE budgets(id bigserial PRIMARY KEY,domain_id bigint REFERENCES domains(id),monthly_limit numeric(12,2) CHECK(monthly_limit>0),owner_id bigint,UNIQUE(owner_id,domain_id));
 CREATE TEMP TABLE transactions(id bigserial PRIMARY KEY,statement_id bigint,owner_id bigint,tx_date date,label text DEFAULT 'TEST',display_label text DEFAULT '',merchant text DEFAULT '',label_kind text DEFAULT '',amount numeric(12,2),domain_id bigint);
 CREATE TEMP TABLE tags(id bigint PRIMARY KEY,name text);
 CREATE TEMP TABLE transaction_tags(transaction_id bigint,tag_id bigint);
 INSERT INTO domains VALUES(1,1,null,'Personnel',false),(2,1,1,'Courses',false),(3,1,1,'Télécom',false),(4,1,null,'Virement',true),(5,1,4,'Livret',false),(6,1,null,'Loisirs',false),(7,2,null,'SECRET CATEGORY',false),(8,1,null,'Atteint',false),(9,1,null,'Vide',false);
 INSERT INTO accounts VALUES(1,1,'TEST');
 INSERT INTO statements(account_id,owner_id,filename,period_start,period_end,validated) VALUES(1,1,'TEST','2020-01-01','2020-01-31',true);
 INSERT INTO budgets(domain_id,monthly_limit,owner_id) VALUES(1,100,1),(2,50,1),(4,500,1),(8,20,1),(9,30,1),(7,40,2);
 INSERT INTO transactions(statement_id,owner_id,tx_date,amount,domain_id,label_kind) VALUES
 (1,1,'2020-01-10',-10,1,''),(1,1,'2020-01-10',-20,2,''),(1,1,'2020-01-10',-20,2,''),(1,1,'2020-01-10',-20,2,''),
 (1,1,'2020-01-10',-15,3,''),(1,1,'2020-01-11',10,2,''),(1,1,'2020-01-11',-100,3,'savings'),
 (1,1,'2020-01-12',-200,4,''),(1,1,'2020-01-12',-50,5,'savings'),(1,1,'2020-01-12',-7,null,''),
 (1,1,'2020-01-13',-4,7,''),(1,1,'2020-01-13',-20,8,''),(1,1,'2020-02-01',-999,1,''),(1,2,'2020-01-10',-999,7,'');
 INSERT INTO tags VALUES(1,'tag a'),(2,'tag b');INSERT INTO transaction_tags VALUES(2,1),(2,2);`)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{DB: db}
	request := func(method, path, body string, owner int64) *http.Request {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		return req.WithContext(context.WithValue(req.Context(), ctxUserKey, &authUser{ID: owner}))
	}
	report, err := server.loadBudgetReport(context.Background(), 1, "2020-01", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if report.Month != "2020-01" || report.QueryTo != "2020-01-31" || report.Period.Completeness != "complete" || len(report.Budgets) != 5 {
		t.Fatalf("period/scope: %+v", report)
	}
	summary := report.Summary
	for key, pair := range map[string][2]json.Number{"allocated": {summary.Allocated, "200.00"}, "spent": {summary.Spent, "105.00"}, "remaining": {summary.Remaining, "95.00"}, "observed": {summary.ObservedExpenses, "116.00"}, "covered": {summary.CoveredExpenses, "105.00"}, "uncategorized": {summary.UncategorizedSpent, "11.00"}, "credits": {summary.CreditsObserved, "10.00"}, "savings": {summary.SavingsOutflows, "150.00"}, "transfers": {summary.TransferOutflows, "200.00"}} {
		if pair[0] != pair[1] {
			t.Errorf("%s %s != %s", key, pair[0], pair[1])
		}
	}
	if summary.OverCount != 1 || summary.UncategorizedCount != 2 || summary.CoverageRatio == nil {
		t.Fatalf("summary counts: %+v", summary)
	}
	byDomain := map[int64]budgetStatus{}
	for _, b := range report.Budgets {
		byDomain[b.DomainID] = b
	}
	if byDomain[1].Spent != "25.00" || byDomain[2].Spent != "60.00" || byDomain[2].CreditTotal != "10.00" || byDomain[2].TransactionCount != 4 {
		t.Fatalf("nearest assignment or credits: %+v", byDomain)
	}
	if byDomain[4].Scope != "transfer" || byDomain[4].Spent != "250.00" || byDomain[8].Status != "reached" || byDomain[8].Exceeded || byDomain[9].Spent != "0.00" {
		t.Fatalf("transfer/exact/empty: %+v", byDomain)
	}
	// Each drilldown returns precisely the contributing source set, despite tags.
	for _, domain := range []int64{1, 2, 4, 8, 9} {
		b := byDomain[domain]
		w := httptest.NewRecorder()
		server.listTransactions(w, request("GET", fmt.Sprintf("/api/transactions?budget_id=%d&month=2020-01", b.ID), "", 1))
		if w.Code != 200 {
			t.Fatalf("sources: %d %s", w.Code, w.Body)
		}
		var sources []models.Transaction
		if err = json.Unmarshal(w.Body.Bytes(), &sources); err != nil {
			t.Fatal(err)
		}
		var debits, credits money.Cents
		for _, source := range sources {
			amount, err := money.FromLegacy(source.Amount)
			if err != nil {
				t.Fatal(err)
			}
			if amount < 0 {
				debits -= amount
			} else {
				credits += amount
			}
		}
		if budgetNumber(debits) != b.Spent || budgetNumber(credits) != b.CreditTotal || len(sources) != b.TransactionCount {
			t.Fatalf("sources mismatch domain %d: %s %+v", domain, w.Body, b)
		}
	}
	// Source whitelist is exact even with tags, and can never expose another owner.
	sourceResponse := httptest.NewRecorder()
	server.listTransactions(sourceResponse, request("GET", "/api/transactions?ids=2,3,14&month=2020-01", "", 1))
	var exactSources []models.Transaction
	if err = json.Unmarshal(sourceResponse.Body.Bytes(), &exactSources); sourceResponse.Code != 200 || err != nil || len(exactSources) != 2 {
		t.Fatalf("source whitelist: %d %s", sourceResponse.Code, sourceResponse.Body)
	}
	for _, source := range exactSources {
		if source.ID != 2 && source.ID != 3 {
			t.Fatal("non-source or foreign transaction returned")
		}
	}
	badSourceResponse := httptest.NewRecorder()
	server.listTransactions(badSourceResponse, request("GET", "/api/transactions?ids=1,not-an-id", "", 1))
	if badSourceResponse.Code != 400 {
		t.Fatal("malformed source filter silently ignored")
	}
	// Report/legacy API retain their respective envelope shape and avoid leakage.
	w := httptest.NewRecorder()
	server.budgetReportHandler(w, request("GET", "/api/budgets/report?month=2020-01", "", 1))
	if w.Code != 200 || strings.Contains(w.Body.String(), "SECRET") {
		t.Fatalf("report leak: %d %s", w.Code, w.Body)
	}
	w = httptest.NewRecorder()
	server.listBudgets(w, request("GET", "/api/budgets?month=2020-01", "", 1))
	if w.Code != 200 || !strings.HasPrefix(w.Body.String(), "[") {
		t.Fatalf("legacy contract: %d %s", w.Code, w.Body)
	}
	// Invalid domains and precision never reach PostgreSQL's silent rounding.
	for _, body := range []string{`{"domain_id":7,"monthly_limit":50}`, `{"domain_id":999,"monthly_limit":50}`, `{"domain_id":6,"monthly_limit":1.005}`, `{"domain_id":6,"monthly_limit":0}`, `{"domain_id":6,"monthly_limit":-1}`} {
		w = httptest.NewRecorder()
		server.upsertBudget(w, request("POST", "/api/budgets", body, 1))
		if w.Code < 400 {
			t.Fatalf("accepted invalid budget %s: %s", body, w.Body)
		}
	}
	// Modification changes the exact threshold; no update touches another owner.
	w = httptest.NewRecorder()
	server.upsertBudget(w, request("POST", "/api/budgets", `{"domain_id":2,"monthly_limit":60.01}`, 1))
	if w.Code != 200 {
		t.Fatalf("update: %s", w.Body)
	}
	report, err = server.loadBudgetReport(context.Background(), 1, "2020-01", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range report.Budgets {
		if b.DomainID == 2 && (b.Exceeded || b.Remaining != "0.01") {
			t.Fatalf("update precision: %+v", b)
		}
	}
	// Deletion preserves transactions; a second or foreign deletion is not success.
	childID := byDomain[2].ID
	for _, owner := range []int64{2, 1, 1} {
		req := request("DELETE", fmt.Sprintf("/api/budgets/%d", childID), "", owner)
		req.SetPathValue("id", fmt.Sprint(childID))
		w = httptest.NewRecorder()
		server.deleteBudget(w, req)
		want := 404
		if owner == 1 && w.Code == 200 {
			want = 200
		}
		if w.Code != want {
			t.Fatalf("delete owner %d: %d", owner, w.Code)
		}
	}
	report, err = server.loadBudgetReport(context.Background(), 1, "2020-01", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range report.Budgets {
		if b.DomainID == 1 && b.Spent != "85.00" {
			t.Fatalf("sources not reassigned after child deletion: %+v", b)
		}
	}
	var rowsCount int
	if err = db.QueryRow(`SELECT count(*) FROM transactions WHERE owner_id=1`).Scan(&rowsCount); err != nil || rowsCount != 13 {
		t.Fatalf("delete lost sources %d %v", rowsCount, err)
	}
	// A gap is not filled by transaction dates or by the calendar.
	if _, err = db.Exec(`UPDATE statements SET validated=false`); err != nil {
		t.Fatal(err)
	}
	report, err = server.loadBudgetReport(context.Background(), 1, "2020-01", time.Now())
	if err != nil || report.Period.Completeness == "complete" {
		t.Fatalf("coverage gap: %+v %v", report.Period, err)
	}
	// No salary in the current month is not a budget error; future rows excluded.
	if _, err = db.Exec(`INSERT INTO transactions(owner_id,tx_date,amount,domain_id) VALUES(1,CURRENT_DATE,-1,1),(1,CURRENT_DATE+interval '1 month',-200,1)`); err != nil {
		t.Fatal(err)
	}
	report, err = server.loadBudgetReport(context.Background(), 1, time.Now().Format("2006-01"), time.Now())
	if err != nil || !report.Period.IsCurrent || report.Summary.Spent != "1.00" {
		t.Fatalf("current without income: %+v %v", report, err)
	}
	for _, month := range []string{"bad", "2020-99", "9999-12"} {
		w = httptest.NewRecorder()
		server.listBudgets(w, request("GET", "/api/budgets?month="+month, "", 1))
		if w.Code != 400 {
			t.Fatalf("month %s accepted", month)
		}
	}
	// Empty owner and documented zero have no manufactured coverage percentage.
	report, err = server.loadBudgetReport(context.Background(), 3, "2020-01", time.Now())
	if err != nil || len(report.Budgets) != 0 || report.Summary.CoverageRatio != nil || report.Period.Completeness != "unknown" {
		t.Fatalf("empty owner: %+v %v", report, err)
	}
	// A SQL failure cannot masquerade as zero expense.
	if _, err = db.Exec(`ALTER TABLE budgets RENAME COLUMN monthly_limit TO broken_limit`); err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	server.listBudgets(w, request("GET", "/api/budgets?month=2020-01", "", 1))
	if w.Code != 500 {
		t.Fatalf("SQL failure hidden: %d %s", w.Code, w.Body)
	}
}
