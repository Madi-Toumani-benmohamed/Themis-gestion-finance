package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lib/pq"
	"themis/internal/money"
)

func engagementTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("THEMIS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("THEMIS_TEST_DATABASE_URL non défini")
	}
	admin, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("engagement_test_%d", time.Now().UnixNano())
	if _, err = admin.Exec(`CREATE SCHEMA ` + pq.QuoteIdentifier(schema)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Exec(`DROP SCHEMA ` + pq.QuoteIdentifier(schema) + ` CASCADE`); admin.Close() })
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	db, err := sql.Open("postgres", u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	db.SetMaxOpenConns(8)
	_, err = db.Exec(`CREATE TABLE users(id bigint PRIMARY KEY);INSERT INTO users VALUES(1),(2);
 CREATE TABLE credits(id bigserial PRIMARY KEY,owner_id bigint REFERENCES users(id),name text NOT NULL,monthly_amount numeric(12,2) NOT NULL CHECK(monthly_amount>0),months_total int NOT NULL CHECK(months_total>0),start_month char(7) NOT NULL,note text NOT NULL DEFAULT '');
 CREATE TABLE taxes(id bigserial PRIMARY KEY,owner_id bigint REFERENCES users(id),name text NOT NULL,amount numeric(12,2) NOT NULL CHECK(amount>0),due_date date NOT NULL,paid boolean NOT NULL DEFAULT false);
 INSERT INTO credits(owner_id,name,monthly_amount,months_total,start_month) VALUES(1,'HISTORIQUE TEST',33.33,12,'2020-01');
 INSERT INTO taxes(owner_id,name,amount,due_date,paid) VALUES(1,'ANCIENNE TEST',100.01,'2020-01-01',true);`)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{"022_credit_payments.sql", "023_tax_history.sql", "024_engagement_integrity.sql"} {
		data, err := os.ReadFile("../db/migrations/" + file)
		if err != nil {
			t.Fatal(err)
		}
		tx, err := db.Begin()
		if err != nil {
			t.Fatal(err)
		}
		if _, err = tx.Exec(string(data)); err != nil {
			tx.Rollback()
			t.Fatal(file, err)
		}
		if err = tx.Commit(); err != nil {
			t.Fatal(err)
		}
	}
	return db
}
func engagementRequest(t *testing.T, s *Server, handler http.HandlerFunc, method, path, body string, owner int64, want int) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r = r.WithContext(context.WithValue(r.Context(), ctxUserKey, &authUser{ID: owner}))
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) > 2 {
		r.SetPathValue("id", parts[2])
	}
	if len(parts) > 4 {
		r.SetPathValue("payment_id", parts[4])
	}
	w := httptest.NewRecorder()
	handler(w, r)
	if w.Code != want {
		t.Fatalf("%s %s status=%d want=%d body=%s", method, path, w.Code, want, w.Body)
	}
	return w
}
func TestCreditConfirmedPaymentsAndUnknownHistory(t *testing.T) {
	now := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	c := creditState{Monthly: 3333, Months: 12, StartMonth: "2020-01"}
	v := buildCredit(c, now)
	if v.Remaining != nil || v.Progress != nil || v.MonthsPaid != nil || v.Status != "unknown" || *v.ScheduledRemaining != 0 {
		t.Fatalf("calendar invented payments: %+v", v)
	}
	c.Opening = sql.NullInt64{Int64: 10000, Valid: true}
	c.Recorded = 3333
	v = buildCredit(c, now)
	if string(*v.Remaining) != "266.63" || string(*v.PaidTotal) != "133.33" || *v.MonthsPaid != 4 || *v.AmountsCents["remaining"] != 26663 {
		t.Fatalf("incorrect cents %+v", v)
	}
	c.Recorded = 29996
	v = buildCredit(c, now)
	if v.Status != "paid" || *v.Remaining != "0.00" {
		t.Fatal("fully paid", v)
	}
	c.Recorded++
	if buildCredit(c, now).Status != "invalid" {
		t.Fatal("overpayment accepted")
	}
	for _, c := range []creditState{{Monthly: 100, Months: 1201, StartMonth: "2020-01"}, {Monthly: 100, Months: 12, StartMonth: "0000-01"}, {Monthly: 100, Months: 12, StartMonth: "9999-12"}, {Monthly: engagementMaxCents, Months: 2, StartMonth: "2020-01"}} {
		if buildCredit(c, now).Status != "invalid" {
			t.Fatal("invalid schedule accepted")
		}
	}
	huge := creditState{Monthly: 999999999999, Months: 1200, StartMonth: "2020-01", Opening: sql.NullInt64{Int64: 0, Valid: true}}
	if *buildCredit(huge, now).AmountsCents["remaining"] != 1199999999998800 {
		t.Fatal("large sum rounded")
	}
}
func TestTaxStatusesAndExactInputs(t *testing.T) {
	now := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	base := taxState{Name: "TEST", Amount: 10001, DueDate: "2026-07-14", PaidAmount: sql.NullInt64{Valid: true}}
	for _, tc := range []struct {
		date      string
		paid      int64
		confirmed bool
		status    string
	}{{"2026-07-14", 0, false, "to_verify"}, {"2026-07-15", 0, false, "today"}, {"2026-07-16", 0, false, "upcoming"}, {"2026-07-14", 1, false, "partial"}, {"2026-07-14", 1, true, "late_confirmed"}, {"2026-07-14", 10001, false, "paid"}} {
		c := base
		c.DueDate = tc.date
		c.PaidAmount.Int64 = tc.paid
		c.Paid = tc.paid == 10001
		c.OverdueConfirmed = tc.confirmed
		v, err := buildTax(c, now)
		if err != nil || v.Status != tc.status || *v.AmountsCents["remaining"] != 10001-tc.paid {
			t.Fatal(tc, v, err)
		}
	}
	for _, raw := range []string{"0.001", "-1", "NaN", "90071992547409.92", "1e999"} {
		if _, err := engagementAmount(raw, true); err == nil {
			t.Fatal("invalid amount", raw)
		}
	}
	if v, e := engagementAmount("33.33", true); e != nil || v != money.Cents(3333) {
		t.Fatal(v, e)
	}
}
func TestEngagementCRUDIsolationPaymentsHistoryPostgres(t *testing.T) {
	db := engagementTestDB(t)
	s := &Server{DB: db}
	w := engagementRequest(t, s, s.obligations, "GET", "/api/obligations", "", 1, 200)
	var report engagementReport
	if json.Unmarshal(w.Body.Bytes(), &report) != nil || report.CreditsRemaining != nil || report.UnknownCredits != 1 {
		t.Fatal("legacy unknown hidden", w.Body)
	}
	w = engagementRequest(t, s, s.createCredit, "POST", "/api/credits", `{"name":"AUTO TEST","monthly_amount":33.33,"months_total":3,"start_month":"2020-01","opening_paid":0}`, 1, 201)
	var c creditView
	json.Unmarshal(w.Body.Bytes(), &c)
	path := fmt.Sprintf("/api/credits/%d", c.ID)
	for _, owner := range []int64{2} {
		engagementRequest(t, s, s.updateCredit, "PATCH", path, `{"monthly_amount":1}`, owner, 404)
		engagementRequest(t, s, s.deleteCredit, "DELETE", path, "", owner, 404)
		engagementRequest(t, s, s.creditHistory, "GET", path+"/history", "", owner, 404)
		engagementRequest(t, s, s.recordCreditPayment, "POST", path+"/payments", `{"amount":1,"payment_date":"2020-01-01","idempotency_key":"isolated-payment-0001"}`, owner, 404)
	}
	body := `{"amount":22.22,"payment_date":"2020-01-01","idempotency_key":"isolated-payment-0001"}`
	engagementRequest(t, s, s.recordCreditPayment, "POST", path+"/payments", body, 1, 201)
	engagementRequest(t, s, s.recordCreditPayment, "POST", path+"/payments", body, 1, 200)
	engagementRequest(t, s, s.recordCreditPayment, "POST", path+"/payments", strings.Replace(body, "0001", "0002", 1), 1, 201)
	var total string
	db.QueryRow(`SELECT SUM(amount_cents)::text FROM credit_payments WHERE credit_id=$1`, c.ID).Scan(&total)
	if total != "4444" {
		t.Fatal("replay duplicated, distinct dropped", total)
	}
	engagementRequest(t, s, s.updateCredit, "PATCH", path, `{"closed":true}`, 1, 400)
	engagementRequest(t, s, s.recordCreditPayment, "POST", path+"/payments", `{"amount":55.55,"payment_date":"2020-01-01","idempotency_key":"isolated-payment-0003"}`, 1, 201)
	engagementRequest(t, s, s.updateCredit, "PATCH", path, `{"closed":true}`, 1, 200)
	engagementRequest(t, s, s.updateCredit, "PATCH", path, `{"closed":false}`, 1, 200)
	var pid int
	db.QueryRow(`SELECT min(id) FROM credit_payments WHERE credit_id=$1`, c.ID).Scan(&pid)
	voidpath := fmt.Sprintf("%s/payments/%d", path, pid)
	engagementRequest(t, s, s.voidCreditPayment, "DELETE", voidpath, `{"note":"Erreur de saisie TEST"}`, 2, 404)
	engagementRequest(t, s, s.voidCreditPayment, "DELETE", voidpath, `{"note":"Erreur de saisie TEST"}`, 1, 200)
	engagementRequest(t, s, s.recordCreditPayment, "POST", path+"/payments", body, 1, 409)
	engagementRequest(t, s, s.deleteCredit, "DELETE", path, "", 1, 200)
	engagementRequest(t, s, s.creditHistory, "GET", path+"/history", "", 1, 200)
	var n int
	db.QueryRow(`SELECT count(*) FROM credit_payments WHERE credit_id=$1`, c.ID).Scan(&n)
	if n != 3 {
		t.Fatal("archive destroyed payments")
	}
	// Tax deltas, explicit confirmation, correct owner, replay and all-or-nothing history.
	w = engagementRequest(t, s, s.createTax, "POST", "/api/taxes", `{"name":"IMPOT TEST","amount":100.01,"due_date":"2020-01-01"}`, 1, 201)
	var tax taxView
	json.Unmarshal(w.Body.Bytes(), &tax)
	tp := fmt.Sprintf("/api/taxes/%d", tax.ID)
	engagementRequest(t, s, s.patchTax, "PATCH", tp, `{"paid":true}`, 2, 404)
	engagementRequest(t, s, s.deleteTax, "DELETE", tp, "", 2, 404)
	engagementRequest(t, s, s.taxHistory, "GET", tp+"/history", "", 2, 404)
	payment := `{"payment_amount":33.33,"payment_date":"2020-01-01","idempotency_key":"tax-test-payment-0001"}`
	engagementRequest(t, s, s.patchTax, "PATCH", tp, payment, 1, 200)
	engagementRequest(t, s, s.patchTax, "PATCH", tp, payment, 1, 200)
	var paid int64
	db.QueryRow(`SELECT paid_amount_cents FROM taxes WHERE id=$1`, tax.ID).Scan(&paid)
	if paid != 3333 {
		t.Fatal("tax replay changed amount", paid)
	}
	engagementRequest(t, s, s.patchTax, "PATCH", tp, `{"amount":110.01,"name":"MODIF TEST"}`, 1, 200)
	if _, err := db.Exec(`ALTER TABLE tax_history RENAME COLUMN after_state TO unavailable`); err != nil {
		t.Fatal(err)
	}
	engagementRequest(t, s, s.patchTax, "PATCH", tp, `{"paid":true}`, 1, 500)
	db.QueryRow(`SELECT paid_amount_cents FROM taxes WHERE id=$1`, tax.ID).Scan(&paid)
	if paid != 3333 {
		t.Fatal("mutation survived failed history")
	}
	db.Exec(`ALTER TABLE tax_history RENAME COLUMN unavailable TO after_state`)
	engagementRequest(t, s, s.patchTax, "PATCH", tp, `{"paid":true}`, 1, 200)
	engagementRequest(t, s, s.taxHistory, "GET", tp+"/history", "", 1, 200)
	engagementRequest(t, s, s.deleteTax, "DELETE", tp, "", 1, 200)
	if _, err := db.Exec(`DELETE FROM credits WHERE id=$1`, c.ID); err == nil {
		t.Fatal("hard delete erased payment evidence")
	}

	// Cross-owner relationships are forbidden in PostgreSQL, including direct SQL.
	if _, err := db.Exec(`INSERT INTO credit_payments(credit_id,owner_id,amount_cents,payment_date) VALUES($1,2,1,'2020-01-01')`, c.ID); err == nil {
		t.Fatal("foreign payment accepted")
	}
}
func TestCreditConcurrentPaymentsPostgres(t *testing.T) {
	db := engagementTestDB(t)
	s := &Server{DB: db}
	var id int64
	err := db.QueryRow(`INSERT INTO credits(owner_id,name,monthly_amount,months_total,start_month,opening_paid_cents) VALUES(1,'CONCURRENCY TEST',1,1,'2020-01',0) RETURNING id`).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	codes := make(chan int, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r := httptest.NewRequest("POST", "/", strings.NewReader(fmt.Sprintf(`{"amount":0.75,"payment_date":"2020-01-01","idempotency_key":"concurrency-test-%04d"}`, i)))
			r = r.WithContext(context.WithValue(r.Context(), ctxUserKey, &authUser{ID: 1}))
			r.SetPathValue("id", fmt.Sprint(id))
			w := httptest.NewRecorder()
			s.recordCreditPayment(w, r)
			codes <- w.Code
		}(i)
	}
	wg.Wait()
	close(codes)
	created, rejected := 0, 0
	for code := range codes {
		if code == 201 {
			created++
		}
		if code == 409 {
			rejected++
		}
	}
	if created != 1 || rejected != 1 {
		t.Fatal("concurrent overpayment", created, rejected)
	}
}
func TestEngagementRealHandlerValidationPostgres(t *testing.T) {
	db := engagementTestDB(t)
	s := &Server{DB: db}
	for _, body := range []string{`{"name":"","monthly_amount":1,"months_total":1,"start_month":"2026-01"}`, `{"name":"TEST","monthly_amount":1.001,"months_total":1,"start_month":"2026-01"}`, `{"name":"TEST","monthly_amount":1,"months_total":1201,"start_month":"2026-01"}`, `{"name":"TEST","monthly_amount":1,"months_total":1,"start_month":"2026-13"}`, `{"name":"TEST","monthly_amount":1,"months_total":1,"start_month":"0000-01"}`, `{"name":"TEST","monthly_amount":1,"months_total":1,"start_month":"2026-01","opening_paid":2}`} {
		engagementRequest(t, s, s.createCredit, "POST", "/api/credits", body, 1, 400)
	}
	for _, body := range []string{`{}`, `{"name":""}`, `{"monthly_amount":0}`, `{"months_total":0}`, `{"name":null}`, `{"owner_id":2}`, `{"start_month":"bad"}`} {
		engagementRequest(t, s, s.updateCredit, "PATCH", "/api/credits/1", body, 1, 400)
	}
	for _, body := range []string{`{"name":"TEST","amount":0,"due_date":"2026-01-01"}`, `{"name":"TEST","amount":1.001,"due_date":"2026-01-01"}`, `{"name":"TEST","amount":1,"due_date":"2026-02-30"}`} {
		engagementRequest(t, s, s.createTax, "POST", "/api/taxes", body, 1, 400)
	}
}

func TestEngagementHistoryAtomicityAndReconciliationPostgres(t *testing.T) {
	db := engagementTestDB(t)
	s := &Server{DB: db}
	w := engagementRequest(t, s, s.createCredit, "POST", "/api/credits", `{"name":"EXACT TEST","monthly_amount":"33.33","months_total":3,"start_month":"2020-01","opening_paid":"0.00"}`, 2, 201)
	var c creditView
	json.Unmarshal(w.Body.Bytes(), &c)
	path := fmt.Sprintf("/api/credits/%d", c.ID)
	_, err := db.Exec(`ALTER TABLE credit_history RENAME COLUMN after_state TO unavailable`)
	if err != nil {
		t.Fatal(err)
	}
	body := `{"amount":"33.33","payment_date":"2020-01-01","idempotency_key":"atomic-payment-0001"}`
	engagementRequest(t, s, s.recordCreditPayment, "POST", path+"/payments", body, 2, 500)
	var n int
	db.QueryRow(`SELECT count(*) FROM credit_payments WHERE credit_id=$1`, c.ID).Scan(&n)
	if n != 0 {
		t.Fatal("payment survived failed history")
	}
	engagementRequest(t, s, s.updateCredit, "PATCH", path, `{"monthly_amount":22.22}`, 2, 500)
	var amount string
	db.QueryRow(`SELECT monthly_amount::text FROM credits WHERE id=$1`, c.ID).Scan(&amount)
	if amount != "33.33" {
		t.Fatal("credit mutation survived failed history")
	}
	db.Exec(`ALTER TABLE credit_history RENAME COLUMN unavailable TO after_state`)
	engagementRequest(t, s, s.recordCreditPayment, "POST", path+"/payments", body, 2, 201)
	engagementRequest(t, s, s.createTax, "POST", "/api/taxes", `{"name":"TAX TEST","amount":"10.01","due_date":"2020-01-01"}`, 2, 201)
	w = engagementRequest(t, s, s.obligations, "GET", "/api/obligations", "", 2, 200)
	var report engagementReport
	if err = json.Unmarshal(w.Body.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Credits) != 1 || len(report.Taxes) != 1 || report.CreditsRemaining == nil || *report.CreditsRemaining != "66.66" || *report.TotalDebt != "76.67" || *report.AmountsCents["total_debt"] != 7667 || report.Credits[0].PaymentsRecorded != "33.33" {
		t.Fatal("totals not reconciled", w.Body)
	}
	// No-op does not create repeated history. Different same-amount payments remain valid.
	before := 0
	db.QueryRow(`SELECT count(*) FROM credit_history WHERE credit_id=$1`, c.ID).Scan(&before)
	engagementRequest(t, s, s.updateCredit, "PATCH", path, `{"name":"EXACT TEST"}`, 2, 200)
	db.QueryRow(`SELECT count(*) FROM credit_history WHERE credit_id=$1`, c.ID).Scan(&n)
	if n != before {
		t.Fatal("no-op history duplicated")
	}
	engagementRequest(t, s, s.recordCreditPayment, "POST", path+"/payments", strings.Replace(body, "33.33", "33.34", 1), 2, 409)
	// A ledger edit failure also rolls back its void.
	var pid int64
	db.QueryRow(`SELECT id FROM credit_payments WHERE credit_id=$1`, c.ID).Scan(&pid)
	db.Exec(`ALTER TABLE credit_history RENAME COLUMN after_state TO unavailable`)
	engagementRequest(t, s, s.voidCreditPayment, "DELETE", fmt.Sprintf("%s/payments/%d", path, pid), `{"note":"Correction TEST"}`, 2, 500)
	var voided bool
	db.QueryRow(`SELECT voided_at IS NOT NULL FROM credit_payments WHERE id=$1`, pid).Scan(&voided)
	if voided {
		t.Fatal("void survived failed history")
	}
}

func TestEngagementExportOwnerIsolationPostgres(t *testing.T) {
	db := engagementTestDB(t)
	s := &Server{DB: db}
	_, err := db.Exec(`CREATE TABLE accounts(id bigint,name text,kind text,interest_rate numeric,deposit_limit numeric,owner_id bigint);
 CREATE TABLE domains(id bigint,name text,parent_id bigint,color text,owner_id bigint);
 CREATE TABLE rules(id bigint,pattern text,domain_id bigint,owner_id bigint);
 CREATE TABLE budgets(id bigint,domain_id bigint,monthly_limit numeric,owner_id bigint);
 CREATE TABLE statements(id bigint,account_id bigint,filename text,period_start date,period_end date,opening_balance numeric,closing_balance numeric,parser text,validated bool,owner_id bigint);
 CREATE TABLE transactions(id bigint PRIMARY KEY,statement_id bigint,tx_date date,label text,amount numeric,domain_id bigint,owner_id bigint);
 CREATE TABLE tags(id bigint,name text,owner_id bigint);CREATE TABLE transaction_tags(transaction_id bigint,tag_id bigint);
 INSERT INTO tags VALUES(1,'VISIBLE_TAG_TEST',1),(2,'PRIVATE_TAG_OWNER2_TEST',2);
 INSERT INTO transactions VALUES(1,NULL,'2020-01-01','ORIGINAL LABEL TEST',-1.01,NULL,1);
 INSERT INTO transaction_tags VALUES(1,1),(1,2);`)
	if err != nil {
		t.Fatal(err)
	}
	for _, owner := range []int64{1, 2} {
		w := engagementRequest(t, s, s.createCredit, "POST", "/api/credits", fmt.Sprintf(`{"name":"CONFIDENTIAL_OWNER_%d","monthly_amount":"33.33","months_total":3,"start_month":"2020-01","opening_paid":"0.00"}`, owner), owner, 201)
		var c creditView
		json.Unmarshal(w.Body.Bytes(), &c)
		p := fmt.Sprintf("/api/credits/%d", c.ID)
		engagementRequest(t, s, s.recordCreditPayment, "POST", p+"/payments", `{"amount":"1.01","payment_date":"2020-01-01","idempotency_key":"export-payment-0001"}`, owner, 201)
		engagementRequest(t, s, s.createTax, "POST", "/api/taxes", fmt.Sprintf(`{"name":"CONFIDENTIAL_OWNER_%d","amount":"10.01","due_date":"2020-01-01"}`, owner), owner, 201)
		engagementRequest(t, s, s.deleteCredit, "DELETE", p, "", owner, 200)
	}
	w := engagementRequest(t, s, s.exportAll, "GET", "/api/export", "", 1, 200)
	if strings.Contains(w.Body.String(), "PRIVATE_TAG_OWNER2_TEST") || !strings.Contains(w.Body.String(), "VISIBLE_TAG_TEST") {
		t.Fatal("transaction tag export scope")
	}
	if strings.Contains(w.Body.String(), "CONFIDENTIAL_OWNER_2") {
		t.Fatal("export leaked foreign owner")
	}
	var dump map[string][]json.RawMessage
	var decoded map[string]json.RawMessage
	if err = json.Unmarshal(w.Body.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	dump = map[string][]json.RawMessage{}
	for _, key := range []string{"credits", "taxes", "credit_payments", "credit_history", "tax_history"} {
		var rows []json.RawMessage
		if err = json.Unmarshal(decoded[key], &rows); err != nil || len(rows) == 0 {
			t.Fatal("missing export evidence", key, err)
		}
		dump[key] = rows
	}
	if len(dump["credit_payments"]) != 1 || len(dump["credits"]) != 2 {
		t.Fatal("wrong export scope")
	}
	w = engagementRequest(t, s, s.listCredits, "GET", "/api/credits", "", 1, 200)
	if strings.Contains(w.Body.String(), "CONFIDENTIAL_OWNER_") {
		t.Fatal("archived credits in active list")
	}
	w = engagementRequest(t, s, s.listTaxes, "GET", "/api/taxes", "", 1, 200)
	if strings.Contains(w.Body.String(), "CONFIDENTIAL_OWNER_2") {
		t.Fatal("tax list leaked owner")
	}
}

func TestEngagementAuthenticationRequiredDuringBootstrap(t *testing.T) {
	s := &Server{}
	for _, owner := range []int64{0, -1, 1} {
		called := false
		wrapped := s.engagementAuthenticated(func(w http.ResponseWriter, r *http.Request) { called = true; w.WriteHeader(204) })
		r := httptest.NewRequest("GET", "/api/credits", nil)
		if owner != 0 {
			r = r.WithContext(context.WithValue(r.Context(), ctxUserKey, &authUser{ID: owner}))
		}
		w := httptest.NewRecorder()
		wrapped(w, r)
		if owner > 0 {
			if w.Code != 204 || !called {
				t.Fatal("valid owner blocked")
			}
		} else if w.Code != 401 || called {
			t.Fatal("missing owner accepted")
		}
	}
}

func TestTaxLegacyConfirmationPreservedWhenEditingName(t *testing.T) {
	now := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	tax := taxState{Name: "LEGACY TEST", Amount: 10001, DueDate: "2026-01-01", Paid: true}
	patch := map[string]json.RawMessage{"name": json.RawMessage(`"EDIT TEST"`), "amount": json.RawMessage(`"100.01"`)}
	if _, err := applyTaxPatch(&tax, patch, now); err != nil {
		t.Fatal(err)
	}
	out, err := buildTax(tax, now)
	if err != nil || tax.PaidAmount.Valid || out.PaymentSource != "legacy_confirmation" {
		t.Fatal("name edit changed historical provenance", out, err)
	}
	if _, err := applyTaxPatch(&tax, map[string]json.RawMessage{"amount": json.RawMessage(`"110.01"`)}, now); err != nil {
		t.Fatal(err)
	}
	out, err = buildTax(tax, now)
	if err != nil || out.PaidAmount != "100.01" || out.Remaining != "10.00" || out.Status != "partial" {
		t.Fatal("amount edit invented payments", out, err)
	}
}

func TestEngagementRouterBootstrapOwnerGuardPostgres(t *testing.T) {
	db := qualityTestDB(t)
	if _, err := db.Exec(`CREATE TEMP TABLE users(id bigint)`); err != nil {
		t.Fatal(err)
	}
	router := (&Server{DB: db}).Router()
	for _, route := range []string{"GET /api/credits", "POST /api/credits", "PATCH /api/credits/1", "DELETE /api/credits/1", "GET /api/taxes", "POST /api/taxes", "PATCH /api/taxes/1", "DELETE /api/taxes/1", "GET /api/obligations", "POST /api/credits/1/payments", "DELETE /api/credits/1/payments/1", "GET /api/credits/1/history", "GET /api/taxes/1/history"} {
		parts := strings.SplitN(route, " ", 2)
		r := httptest.NewRequest(parts[0], parts[1], nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != 401 {
			t.Fatalf("bootstrap %s returned %d", route, w.Code)
		}
	}
}
