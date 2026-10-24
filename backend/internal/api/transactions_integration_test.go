package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// All fixtures are temporary and shadow public tables even when testing errors.
func TestTransactionsPostgres(t *testing.T) {
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
	_, err = db.Exec(`CREATE TEMP TABLE domains(id bigint PRIMARY KEY,owner_id bigint,parent_id bigint,is_transfer bool DEFAULT false);
 CREATE TEMP TABLE accounts(id bigint PRIMARY KEY,owner_id bigint,name text);
 CREATE TEMP TABLE statements(id bigint PRIMARY KEY,owner_id bigint,account_id bigint,filename text,validated bool,period_start date,period_end date);
 CREATE TEMP TABLE transactions(id bigserial PRIMARY KEY,statement_id bigint,owner_id bigint,tx_date date,label text,display_label text,merchant text,label_kind text,amount numeric(12,2),domain_id bigint,classification_source text DEFAULT 'unknown');
 CREATE TEMP TABLE transaction_suggestions(transaction_id bigint PRIMARY KEY,owner_id bigint,domain_id bigint,tags text[],source text);
 CREATE TEMP TABLE tags(id bigserial PRIMARY KEY,owner_id bigint,name text,UNIQUE(owner_id,name));
 CREATE TEMP TABLE transaction_tags(transaction_id bigint,tag_id bigint,PRIMARY KEY(transaction_id,tag_id));
 INSERT INTO domains VALUES(1,1,null,false),(2,1,null,true),(3,1,2,false),(4,1,1,false),(10,2,null,true);
 INSERT INTO accounts VALUES(1,1,'TEST CURRENT'),(2,1,'TEST SAVINGS'),(10,2,'SECRET ACCOUNT');
 INSERT INTO statements VALUES(1,1,1,'TEST SOURCE',true,'2020-01-01','2020-01-31'),(2,1,2,'TEST SAVINGS',false,null,null),(10,2,10,'SECRET SOURCE',true,'2020-01-01','2020-01-31');
 INSERT INTO transactions(statement_id,owner_id,tx_date,label,display_label,merchant,label_kind,amount,domain_id) VALUES
 (1,1,'2020-01-01','RAW PAYPAL 111','PayPal','paypal','',-12.34,1),
 (1,1,'2020-01-01','RAW PAYPAL 222','PayPal','paypal','',-12.34,1),
 (1,1,'2020-01-02','TEST SALARY','Salary','employer','',3249.47,1),
 (1,1,'2020-01-02','TEST EXPENSE','Expense','shop','',-4172.46,4),
 (1,1,'2020-01-03','TEST TRANSFER OUT','Transfer','own','',-22,2),
 (2,1,'2020-01-03','TEST TRANSFER IN','Transfer','own','',22,3),
 (1,1,'2020-01-04','TEST CREDIT UNLINKED','Credit','shop','',12.34,1),
 (1,1,'2020-01-05','TEST SAVINGS','Savings','pel','savings',-100,1),
 (null,1,'2020-01-06','TEST UNKNOWN','Unknown','unknown','',-0.01,null),
 (1,1,'2020-01-07','TEST INVALID CATEGORY','Unknown','unknown','',-0.01,10),
 (1,1,'2020-02-01','TEST NEXT MONTH','Next','next','',-999,1),
 (10,2,'2020-01-01','SECRET RAW','SECRET DISPLAY','SECRET MERCHANT','',-999,10);
 INSERT INTO tags(owner_id,name) VALUES(1,'taxes'),(1,'verified'),(2,'secret tag');
 INSERT INTO transaction_tags VALUES(1,1),(1,2),(1,3);
 INSERT INTO transaction_suggestions VALUES(1,1,4,ARRAY['verified'],'ai'),(2,2,10,ARRAY['secret tag'],'ai');`)
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{DB: db}
	call := func(method, path, body string, owner int64) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req = req.WithContext(context.WithValue(req.Context(), ctxUserKey, &authUser{ID: owner}))
		if method == "PATCH" {
			req.SetPathValue("id", strings.TrimPrefix(path, "/api/transactions/"))
		}
		w := httptest.NewRecorder()
		if method == "PATCH" {
			s.updateTransaction(w, req)
		} else {
			s.listTransactions(w, req)
		}
		return w
	}
	get := func(query string) transactionReport {
		t.Helper()
		w := call("GET", "/api/transactions?view=report&"+query, "", 1)
		if w.Code != 200 {
			t.Fatalf("report %s: %d %s", query, w.Code, w.Body)
		}
		if strings.Contains(w.Body.String(), "SECRET") || strings.Contains(w.Body.String(), "secret tag") {
			t.Fatal("owner leak")
		}
		var report transactionReport
		if err = json.Unmarshal(w.Body.Bytes(), &report); err != nil {
			t.Fatal(err)
		}
		return report
	}
	r := get("month=2020-01&limit=2")
	if r.Total != 10 || len(r.Transactions) != 2 || r.Summary.Expenses != "4197.16" || r.Summary.Income != "3261.81" || r.Summary.Net != "-935.35" || r.Summary.TransferDebits != "122.00" || r.Summary.TransferCredits != "22.00" || r.Summary.UncategorizedCount != 2 {
		t.Fatalf("exact unpaginated summary: %+v", r)
	}
	page := get("month=2020-01&limit=2&offset=2")
	if page.Total != r.Total || page.Summary != r.Summary || page.Transactions[0].ID == r.Transactions[0].ID {
		t.Fatal("pagination changed financial scope")
	}
	for query, want := range map[string]int64{"domain=1": 6, "domain=none": 2, "domain=10": 0, "account=1": 8, "account=2": 1, "account=none": 1, "account=10": 0, "type=transfer": 3, "type=debit": 5, "type=credit": 2, "tag=taxes": 1, "tag=secret+tag": 0, "q=paypal": 2, "q=RAW+PAYPAL+111": 1, "q=verified": 1, "q=-12%2C34": 2, "q=12%2C34": 1, "from=2020-01-01&to=2020-01-01": 2} {
		got := get("month=2020-01&" + query)
		if got.Total != want {
			t.Errorf("%s total %d != %d", query, got.Total, want)
		}
	}
	row := get("ids=1&all=1").Transactions[0]
	if row.AmountCents != -1234 || row.Amount != "-12.34" || row.Label != "RAW PAYPAL 111" || row.StatementFilename != "TEST SOURCE" || row.AccountID == nil || *row.AccountID != 1 || row.StatementValidated == nil || !*row.StatementValidated || len(row.Tags) != 2 || row.ClassificationSource != "unknown" || row.Suggestion == nil || row.Suggestion.DomainID != 4 || row.DomainID == nil || *row.DomainID != 1 {
		t.Fatalf("detail provenance: %+v", row)
	}
	invalid := get("ids=10").Transactions[0]
	if invalid.DomainID != nil || invalid.FinancialType != "debit" {
		t.Fatal("foreign category influenced classification")
	}
	empty := get("month=2019-01")
	if empty.Total != 0 || len(empty.Transactions) != 0 || empty.Summary.Expenses != "0.00" || empty.Summary.Net != "0.00" {
		t.Fatal("empty report invalid")
	}
	legacy := call("GET", "/api/transactions?ids=1,2,12&month=2020-01", "", 1)
	if legacy.Code != 200 || !strings.HasPrefix(legacy.Body.String(), "[") || strings.Contains(legacy.Body.String(), "SECRET") {
		t.Fatalf("legacy contract: %s", legacy.Body)
	}
	for _, body := range []string{`{"domain_id":10}`, `{"domain_id":"no"}`, `{"tags":null}`, `{"tags":42}`, `{"domain_id":1,"tags":[""]}`, `{"amount":5}`, `{"domain_id":1} {}`, `null`, `{}`} {
		w := call("PATCH", "/api/transactions/1", body, 1)
		if w.Code != http.StatusBadRequest {
			t.Errorf("invalid patch %s: %d %s", body, w.Code, w.Body)
		}
	}
	for _, owner := range []int64{2, 3} {
		if w := call("PATCH", "/api/transactions/1", `{"domain_id":1}`, owner); w.Code != 404 {
			t.Fatalf("foreign edit %d: %d", owner, w.Code)
		}
	}
	w := call("PATCH", "/api/transactions/1", `{"discard_suggestion":true}`, 1)
	if w.Code != 204 {
		t.Fatalf("discard suggestion: %d %s", w.Code, w.Body)
	}
	row = get("ids=1").Transactions[0]
	if row.Suggestion != nil || row.DomainID == nil || *row.DomainID != 1 || row.ClassificationSource != "unknown" {
		t.Fatal("discard changed assigned classification")
	}
	w = call("PATCH", "/api/transactions/1", `{"domain_id":4,"tags":[" Verified ","taxes","TAXES"]}`, 1)
	if w.Code != 204 {
		t.Fatalf("patch: %d %s", w.Code, w.Body)
	}
	row = get("ids=1").Transactions[0]
	if row.DomainID == nil || *row.DomainID != 4 || row.ClassificationSource != "manual" || len(row.Tags) != 2 || row.Label != "RAW PAYPAL 111" || row.AmountCents != -1234 || row.Suggestion != nil {
		t.Fatalf("edit damaged source or provenance: %+v", row)
	}
	// Add/remove operates on current persisted tags and preserves unrelated tags.
	w = call("PATCH", "/api/transactions/1", `{"add_tags":["new","TAXES"]}`, 1)
	if w.Code != 204 {
		t.Fatalf("add tags %d %s", w.Code, w.Body)
	}
	row = get("ids=1").Transactions[0]
	if len(row.Tags) != 3 {
		t.Fatal("add overwrote existing tags")
	}
	w = call("PATCH", "/api/transactions/1", `{"remove_tags":["taxes"]}`, 1)
	if w.Code != 204 {
		t.Fatalf("remove tags %d %s", w.Code, w.Body)
	}
	row = get("ids=1").Transactions[0]
	if len(row.Tags) != 2 || row.Tags[0] != "new" || row.Tags[1] != "verified" {
		t.Fatal("remove damaged unrelated tags")
	}
	if w = call("PATCH", "/api/transactions/1", `{"tags":[],"add_tags":["new"]}`, 1); w.Code != 400 {
		t.Fatal("ambiguous tags mutation accepted")
	}
	// A tag write error must roll back the category too, leaving source rows intact.
	if _, err = db.Exec(`ALTER TABLE tags RENAME COLUMN name TO broken_name`); err != nil {
		t.Fatal(err)
	}
	w = call("PATCH", "/api/transactions/1", `{"domain_id":2,"tags":["new"]}`, 1)
	if w.Code != 500 {
		t.Fatalf("expected failed edit: %d %s", w.Code, w.Body)
	}
	var domain int64
	if err = db.QueryRow(`SELECT domain_id FROM transactions WHERE id=1`).Scan(&domain); err != nil || domain != 4 {
		t.Fatal("partial edit committed")
	}
	if _, err = db.Exec(`ALTER TABLE tags RENAME COLUMN broken_name TO name`); err != nil {
		t.Fatal(err)
	}
	w = call("PATCH", "/api/transactions/1", `{"domain_id":null,"tags":[]}`, 1)
	if w.Code != 204 {
		t.Fatalf("clear category: %d %s", w.Code, w.Body)
	}
	row = get("ids=1").Transactions[0]
	if row.DomainID != nil || len(row.Tags) != 0 || row.ClassificationSource != "manual" {
		t.Fatalf("clear fields: %+v", row)
	}
	var count int
	if err = db.QueryRow(`SELECT count(*) FROM transactions`).Scan(&count); err != nil || count != 12 {
		t.Fatal("source rows altered")
	}
	// A large filtered period is paginated server-side without truncating totals.
	_, err = db.Exec(`INSERT INTO transactions(owner_id,tx_date,label,display_label,merchant,label_kind,amount,domain_id) SELECT 1,'2020-04-01','TEST LARGE','TEST LARGE','TEST LARGE','',-0.01,1 FROM generate_series(1,1001)`)
	if err != nil {
		t.Fatal(err)
	}
	large := get("month=2020-04&limit=1000&offset=1000")
	if large.Total != 1001 || len(large.Transactions) != 1 || large.Summary.Expenses != "10.01" || large.Summary.Net != "-10.01" {
		t.Fatalf("large period truncation: %+v", large)
	}
	_, err = db.Exec(`INSERT INTO transactions(owner_id,tx_date,label,display_label,merchant,label_kind,amount,domain_id) VALUES(1,'2020-05-01','TEST ZERO','TEST ZERO','TEST ZERO','',0,1)`)
	if err != nil {
		t.Fatal(err)
	}
	neutral := get("month=2020-05&type=neutral")
	if neutral.Total != 1 || neutral.Transactions[0].FinancialType != "neutral" || neutral.Summary.Net != "0.00" {
		t.Fatal("zero transaction misclassified")
	}
	// Exact aggregate reconciliation across all pages, including the cent example.
	_, err = db.Exec(`INSERT INTO transactions(owner_id,tx_date,label,display_label,merchant,label_kind,amount,domain_id) VALUES(1,'2020-03-01','TEST','TEST','TEST','',3249.47,1),(1,'2020-03-02','TEST','TEST','TEST','',-4197.14,1)`)
	if err != nil {
		t.Fatal(err)
	}
	r = get("month=2020-03&limit=1")
	if r.Summary.Net != "-947.67" || r.Total != 2 {
		t.Fatalf("one-cent reconciliation: %+v", r)
	}
	for _, query := range []string{"month=2020-99", "limit=0", "type=cancelled", "from=2020-02-30&to=2020-03-01"} {
		w = call("GET", "/api/transactions?view=report&"+query, "", 1)
		if w.Code != 400 {
			t.Fatalf("invalid filter %s: %d", query, w.Code)
		}
	}
}
