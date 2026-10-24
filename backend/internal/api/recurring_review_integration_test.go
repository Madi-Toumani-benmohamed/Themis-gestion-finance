package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/lib/pq"
)

func TestRecurringReviewPostgres(t *testing.T) {
	dsn := os.Getenv("THEMIS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("THEMIS_TEST_DATABASE_URL non défini")
	}
	admin, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("recurring_review_test_%d", time.Now().UnixNano())
	if _, err = admin.Exec("CREATE SCHEMA " + pq.QuoteIdentifier(schema)); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec("DROP SCHEMA " + pq.QuoteIdentifier(schema) + " CASCADE")
	connection, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	query := connection.Query()
	query.Set("search_path", schema)
	connection.RawQuery = query.Encode()
	db, err := sql.Open("postgres", connection.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE users(id bigint PRIMARY KEY);
 CREATE TABLE accounts(id bigint PRIMARY KEY,owner_id bigint,name text);
 CREATE TABLE domains(id bigint PRIMARY KEY,owner_id bigint,parent_id bigint,name text,is_transfer bool);
 CREATE TABLE statements(id bigint PRIMARY KEY,owner_id bigint,account_id bigint,filename text,period_start date,period_end date,validated bool,uploaded_at timestamptz DEFAULT now());
 CREATE TABLE transactions(id bigserial PRIMARY KEY,owner_id bigint,statement_id bigint,tx_date date,label text,amount numeric(12,2),domain_id bigint,merchant text DEFAULT '',display_label text DEFAULT '',label_kind text DEFAULT '');
 CREATE TABLE schema_migrations(name text PRIMARY KEY);
 INSERT INTO schema_migrations VALUES('020_recurring_reviews.sql');
 INSERT INTO users VALUES(1),(2);
 INSERT INTO accounts VALUES(11,1,'TEST'),(21,2,'OTHER TEST');
 INSERT INTO domains VALUES(1,1,null,'Abonnements',false),(2,1,null,'Épargne',true),(3,1,2,'Sous-catégorie',false),(4,1,null,'Retraits',false),(20,2,null,'OTHER PRIVATE DOMAIN',false);
 INSERT INTO statements(id,owner_id,account_id,filename,period_start,period_end,validated) VALUES(100,1,11,'TEST','2020-01-01','2020-03-31',true);
 INSERT INTO transactions(owner_id,statement_id,tx_date,label,amount,domain_id,merchant,display_label)
 SELECT 1,100,date_trunc('month',date '2020-01-01')+n*interval '1 month'+interval '9 days','PRLV NETFLIX',-12.99,1,'Netflix','Netflix' FROM generate_series(0,2) n;
 INSERT INTO transactions(owner_id,statement_id,tx_date,label,amount,domain_id,merchant,display_label)
 SELECT 1,100,date_trunc('month',date '2020-01-01')+n*interval '1 month'+interval '14 days','VIREMENT INTERNE',-100,3,'Test Savings','Test Savings' FROM generate_series(0,2) n;
 INSERT INTO transactions(owner_id,statement_id,tx_date,label,amount,domain_id,merchant,display_label)
 SELECT 1,100,date_trunc('month',date '2020-01-01')+n*interval '1 month'+interval '14 days','VIREMENT INTERNE',100,3,'Test Savings','Test Savings' FROM generate_series(0,2) n;
 INSERT INTO transactions(owner_id,statement_id,tx_date,label,amount,domain_id,merchant,display_label)
 SELECT 1,100,date_trunc('month',date '2020-01-01')+n*interval '1 month'+interval '19 days','RET DAB LES ULIS',-20,4,'Les Ulis','Les Ulis' FROM generate_series(0,2) n;
 INSERT INTO transactions(owner_id,statement_id,tx_date,label,amount,domain_id,merchant,display_label) VALUES
 (1,100,'2020-04-10','PRLV NETFLIX',-99.99,1,'Netflix','Netflix'),
 (2,null,'2020-03-10','OTHER USER',-999.99,20,'PRIVATE MERCHANT','PRIVATE MERCHANT');`)
	if err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile("../db/migrations/020_recurring_reviews.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(string(migration)); err != nil {
		t.Fatal(err)
	}
	srv := &Server{DB: db}
	request := func(method, path, body string, owner int64, key string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req = req.WithContext(context.WithValue(req.Context(), ctxUserKey, &authUser{ID: owner}))
		req.SetPathValue("key", key)
		w := httptest.NewRecorder()
		if method == "GET" {
			srv.listRecurringReview(w, req)
		} else {
			srv.updateRecurringReview(w, req)
		}
		return w
	}
	get := func() recurringReviewReport {
		t.Helper()
		w := request("GET", "/api/recurring/review?month=2020-03", "", 1, "")
		if w.Code != 200 {
			t.Fatalf("review GET: %d %s", w.Code, w.Body)
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("financial review must not be cached")
		}
		var report recurringReviewReport
		if err := json.Unmarshal(w.Body.Bytes(), &report); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(w.Body.String(), "PRIVATE") {
			t.Fatal("another owner's data leaked")
		}
		return report
	}
	report := get()
	if len(report.Items) != 4 {
		t.Fatalf("missing categories, directions or distinct sources: %+v", report.Items)
	}
	legacy, legacyErr := computeRecurringForMonth(db, 1, 2, "2020-03")
	if legacyErr != nil || len(legacy) != 1 || !legacy[0].Regular || legacy[0].Amount != 12.99 {
		t.Fatalf("legacy consumers diverge from reviewed sources: %+v %v", legacy, legacyErr)
	}
	var subscription, cash, transfer recurringReviewItem
	for _, item := range report.Items {
		if item.Nature == "subscription" {
			subscription = item
		}
		if item.Nature == "variable_recurring" {
			cash = item
		}
		if item.Nature == "transfer_savings" {
			transfer = item
		}
		if item.LastDate > "2020-03-31" || item.OccurrenceCount != 3 || len(item.TransactionIDs) != 3 {
			t.Fatal("future transaction included or sources lost")
		}
	}
	if subscription.LastAmount != "12.99" || subscription.AveragePerTransaction != "12.99" || subscription.Confidence != "probable" || subscription.EstimatedNextDate == nil || *subscription.EstimatedNextDate != "2020-04-10" {
		t.Fatalf("subscription evidence: %+v", subscription)
	}
	if transfer.Key == "" || cash.Key == "" || transfer.Key == subscription.Key {
		t.Fatal("transfer child or DAB not classified safely")
	}
	for _, key := range []string{transfer.Key, cash.Key} {
		w := request("PUT", "/api/recurring/review/"+key+"?month=2020-03", `{"status":"confirmed","nature":"subscription"}`, 1, key)
		if w.Code != 400 {
			t.Fatalf("protected transfer/cash became subscription: %d %s", w.Code, w.Body)
		}
	}
	w := request("PUT", "/api/recurring/review/"+subscription.Key+"?month=2020-03", `{"status":"confirmed","nature":"subscription"}`, 2, subscription.Key)
	if w.Code != 404 {
		t.Fatalf("another owner reviewed a detection: %d %s", w.Code, w.Body)
	}
	w = request("PUT", "/api/recurring/review/"+subscription.Key+"?month=2020-03", `{"status":"confirmed","nature":"subscription"}`, 1, subscription.Key)
	if w.Code != 200 {
		t.Fatalf("confirm: %d %s", w.Code, w.Body)
	}
	report = get()
	for _, item := range report.Items {
		if item.Key == subscription.Key && (item.Status != "confirmed" || item.ReviewedAt == nil) {
			t.Fatal("review decision not persisted")
		}
	}
	if _, err = db.Exec(`UPDATE statements SET validated=false`); err != nil {
		t.Fatal(err)
	}
	report = get()
	for _, item := range report.Items {
		if item.Confidence != "insufficient" || item.CoverageComplete {
			t.Fatal("confirmation fabricated documentation")
		}
	}
	w = request("PUT", "/api/recurring/review/"+subscription.Key+"?month=2020-03", `{"status":"ignored"}`, 1, subscription.Key)
	if w.Code != 200 {
		t.Fatalf("ignore: %d %s", w.Code, w.Body)
	}
	report = get()
	for _, item := range report.Items {
		if item.Key == subscription.Key && item.Status != "ignored" {
			t.Fatal("ignored detection not retained")
		}
	}
	legacy, legacyErr = computeRecurringForMonth(db, 1, 2, "2020-03")
	if legacyErr != nil || len(legacy) != 0 {
		t.Fatal("ignored detections still affect legacy consumers")
	}
	w = request("PUT", "/api/recurring/review/"+subscription.Key+"?month=2020-03", `{"status":"detected"}`, 1, subscription.Key)
	if w.Code != 200 {
		t.Fatal("review cannot be restored")
	}
	w = request("GET", "/api/recurring/review?month=not-a-month", "", 1, "")
	if w.Code != 400 {
		t.Fatal("invalid period silently replaced")
	}
	w = request("GET", "/api/recurring/review?month=2020-03", "", 3, "")
	if w.Code != 200 || strings.Contains(w.Body.String(), subscription.Key) {
		t.Fatal("empty owner sees another detection")
	}
	var before, after int
	if err = db.QueryRow(`SELECT count(*) FROM transactions`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	rollback, err := os.ReadFile("../db/rollbacks/020_recurring_reviews.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(string(rollback)); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`SELECT count(*) FROM transactions`).Scan(&after); err != nil || before != after {
		t.Fatal("rollback changed source transactions")
	}
	w = request("GET", "/api/recurring/review?month=2020-03", "", 1, "")
	if w.Code != 500 {
		t.Fatal("missing review schema hidden as an empty list")
	}
}
