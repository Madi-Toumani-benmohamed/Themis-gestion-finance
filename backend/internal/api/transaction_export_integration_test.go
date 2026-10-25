package api

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"themis/internal/money"
)

func TestTransactionExportPostgres(t *testing.T) {
	dsn := os.Getenv("THEMIS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("THEMIS_TEST_DATABASE_URL non défini")
	}
	stagingDir := t.TempDir()
	t.Setenv("TMPDIR", stagingDir)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`CREATE TEMP TABLE domains(id bigint PRIMARY KEY,owner_id bigint,parent_id bigint,is_transfer bool DEFAULT false,name text);
 CREATE TEMP TABLE accounts(id bigint PRIMARY KEY,owner_id bigint,name text,tracking_mode text NOT NULL DEFAULT 'statements');
 CREATE TEMP TABLE statements(id bigint PRIMARY KEY,owner_id bigint,account_id bigint,filename text,validated bool,period_start date,period_end date);
 CREATE TEMP TABLE transactions(id bigserial PRIMARY KEY,statement_id bigint,owner_id bigint,tx_date date,label text,display_label text DEFAULT '',merchant text DEFAULT '',label_kind text DEFAULT '',amount numeric(12,2),domain_id bigint,classification_source text DEFAULT 'unknown');
 CREATE TEMP TABLE tags(id bigint PRIMARY KEY,owner_id bigint,name text);
 CREATE TEMP TABLE transaction_tags(transaction_id bigint,tag_id bigint,PRIMARY KEY(transaction_id,tag_id));
 CREATE TEMP TABLE transaction_suggestions(transaction_id bigint PRIMARY KEY,owner_id bigint,domain_id bigint,tags text[],source text);
 CREATE TEMP TABLE budgets(id bigint PRIMARY KEY,owner_id bigint,domain_id bigint);
 INSERT INTO domains VALUES(1,1,null,false,'=TEST CATEGORY'),(2,1,null,true,'TEST TRANSFER'),(3,1,1,false,'TEST CHILD'),(10,2,null,false,'SECRET CATEGORY');
 INSERT INTO accounts(id,owner_id,name) VALUES(1,1,'TEST CURRENT'),(2,1,'TEST SAVINGS'),(10,2,'SECRET ACCOUNT');
 INSERT INTO statements VALUES(1,1,1,'TEST SOURCE',true,'2020-01-01','2020-01-31'),(2,1,2,'TEST SAVINGS',false,null,null),(10,2,10,'SECRET SOURCE',true,'2020-01-01','2020-01-31');
 INSERT INTO transactions(statement_id,owner_id,tx_date,label,merchant,label_kind,amount,domain_id) VALUES
 (1,1,'2020-01-01','=RAW FORMULA','TEST MERCHANT','',-0.10,1),
 (1,1,'2020-01-02','TEST RAW TWO','TEST MERCHANT','',-0.20,3),
 (1,1,'2020-01-03','TEST CREDIT','OTHER','',0.30,1),
 (1,1,'2020-01-03','TEST TRANSFER','OTHER','',-22,2),
 (1,1,'2020-01-04','TEST INVALID CATEGORY','OTHER','',-999,10),
 (10,2,'2020-01-01','SECRET RAW','SECRET','',-333,10),
 (2,1,'2020-01-04','TEST SAVINGS','OTHER','savings',-10,1),
 (1,1,'2020-02-01','TEST NEXT','OTHER','',-99,1);
 INSERT INTO tags VALUES(1,1,'food'),(2,2,'secret');
 INSERT INTO transaction_tags VALUES(1,1),(2,1),(1,2);
 INSERT INTO budgets VALUES(1,1,1),(2,2,10);`)
	if err != nil {
		t.Fatal(err)
	}
	seedDocumentedStatementBalances(t, db)
	s := &Server{DB: db}
	call := func(path string, export bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", path, nil)
		req = req.WithContext(context.WithValue(req.Context(), ctxUserKey, &authUser{ID: 1}))
		w := httptest.NewRecorder()
		if export {
			s.exportTransactions(w, req)
		} else {
			s.listTransactions(w, req)
		}
		return w
	}
	check := func(query string, want int) [][]string {
		t.Helper()
		w := call("/api/transactions/export?view=report&"+query, true)
		if w.Code != 200 {
			t.Fatalf("export %s: %d %s", query, w.Code, w.Body)
		}
		if strings.Contains(w.Body.String(), "SECRET") {
			t.Fatal("export owner leak")
		}
		length, err := strconv.Atoi(w.Header().Get("Content-Length"))
		if err != nil || length != w.Body.Len() {
			t.Fatal("invalid export length")
		}
		reader := csv.NewReader(strings.NewReader(w.Body.String()))
		reader.Comma = ';'
		records, err := reader.ReadAll()
		if err != nil || len(records) != want+1 {
			t.Fatalf("CSV %s: %d records %v", query, len(records), err)
		}
		var debits, credits money.Cents
		for _, record := range records[1:] {
			amount, err := money.Parse(strings.ReplaceAll(record[2], ",", "."))
			if err != nil {
				t.Fatal(err)
			}
			if amount < 0 {
				debits -= amount
			} else {
				credits += amount
			}
		}
		// Export and report share every predicate; row pagination never affects
		// the CSV or report's full-filter aggregate, including transfer scopes.
		reportResponse := call("/api/transactions?view=report&"+query, false)
		var report transactionReport
		if reportResponse.Code != 200 || json.Unmarshal(reportResponse.Body.Bytes(), &report) != nil {
			t.Fatalf("report %s", reportResponse.Body)
		}
		if report.Total != int64(want) {
			t.Fatalf("report/export count mismatch: %d != %d", report.Total, want)
		}
		expense, err := parseTransactionAggregate(report.Summary.Expenses.String())
		if err != nil {
			t.Fatal(err)
		}
		transferDebit, err := parseTransactionAggregate(report.Summary.TransferDebits.String())
		if err != nil {
			t.Fatal(err)
		}
		income, err := parseTransactionAggregate(report.Summary.Income.String())
		if err != nil {
			t.Fatal(err)
		}
		transferCredit, err := parseTransactionAggregate(report.Summary.TransferCredits.String())
		if err != nil {
			t.Fatal(err)
		}
		if debits != expense+transferDebit || credits != income+transferCredit {
			t.Fatalf("export money diverges %s: %s %s %+v", query, debits, credits, report.Summary)
		}
		return records
	}
	records := check("month=2020-01&limit=1&offset=1&account=1&type=debit&domain=1&tag=food", 2)
	if records[1][1] != "'=RAW FORMULA" || records[1][2] != "-0,10" || records[1][3] != "'=TEST CATEGORY" {
		t.Fatalf("raw labels, exact money or formula escaping: %+v", records)
	}
	for query, want := range map[string]int{"month=2020-01": 6, "month=2020-01&type=transfer": 2, "month=2020-01&domain=none": 1, "month=2020-01&account=2": 1, "month=2020-01&account=10": 0, "month=2020-01&tag=secret": 0, "month=2020-01&q=TEST+MERCHANT": 2, "month=2020-01&q=-0%2C20": 1, "month=2020-01&ids=1,2,6": 2, "month=2020-01&budget_id=1": 3, "from=2020-01-02&to=2020-01-03": 3, "month=2019-01": 0} {
		check(query, want)
	}
	for _, query := range []string{"month=2020-99", "from=2020-02-30&to=2020-03-01", "type=refund", "limit=0", "domain=invalid"} {
		w := call("/api/transactions/export?view=report&"+query, true)
		if w.Code != 400 {
			t.Fatalf("invalid filter %s: %d", query, w.Code)
		}
	}
	if w := call("/api/transactions/export?view=report&month=2020-01&budget_id=2", true); w.Code != 404 {
		t.Fatalf("foreign budget export: %d", w.Code)
	}
	// A legacy caller keeps its historical all-months default and CSV columns.
	legacy := call("/api/transactions/export", true)
	if legacy.Code != 200 {
		t.Fatal(legacy.Body.String())
	}
	reader := csv.NewReader(legacy.Body)
	reader.Comma = ';'
	legacyRows, err := reader.ReadAll()
	if err != nil || len(legacyRows) != 8 {
		t.Fatalf("legacy export contract: %d %v", len(legacyRows), err)
	}
	// An SQL failure cannot produce a successful, partially populated CSV.
	if _, err = db.Exec(`ALTER TABLE transactions RENAME COLUMN label TO broken_label`); err != nil {
		t.Fatal(err)
	}
	w := call("/api/transactions/export?month=2020-01", true)
	if w.Code != 500 || strings.Contains(w.Header().Get("Content-Type"), "text/csv") {
		t.Fatal("failed export passed as CSV")
	}
	if _, err = db.Exec(`ALTER TABLE transactions RENAME COLUMN broken_label TO label; UPDATE transactions SET label=NULL WHERE id=2`); err != nil {
		t.Fatal(err)
	}
	w = call("/api/transactions/export?month=2020-01", true)
	if w.Code != 500 || strings.Contains(w.Header().Get("Content-Type"), "text/csv") || strings.Contains(w.Body.String(), "=RAW FORMULA") {
		t.Fatal("mid-export scan failure returned a truncated financial CSV")
	}
	stagedFiles, err := os.ReadDir(stagingDir)
	if err != nil || len(stagedFiles) != 0 {
		t.Fatal("temporary banking export not removed")
	}
}
