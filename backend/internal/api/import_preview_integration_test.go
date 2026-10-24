package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/lib/pq"
	"themis/internal/models"
	"themis/internal/parser"
)

func TestImportPreviewPostgres(t *testing.T) {
	dsn := os.Getenv("THEMIS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("THEMIS_TEST_DATABASE_URL non défini")
	}
	admin, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("import_preview_test_%d", time.Now().UnixNano())
	if _, err = admin.Exec("CREATE SCHEMA " + pq.QuoteIdentifier(schema)); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec("DROP SCHEMA " + pq.QuoteIdentifier(schema) + " CASCADE")
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
	defer db.Close()
	_, err = db.Exec(`
 CREATE TABLE accounts(id bigint PRIMARY KEY,owner_id bigint,name text);
 CREATE TABLE statements(id bigserial PRIMARY KEY,filename text,period_start date,period_end date,opening_balance numeric,closing_balance numeric,parser text,validated bool NOT NULL DEFAULT false,uploaded_at timestamptz DEFAULT now(),account_id bigint,owner_id bigint);
 CREATE TABLE transactions(id bigserial PRIMARY KEY,statement_id bigint,tx_date date,label text,amount numeric,domain_id bigint,hash text,owner_id bigint,display_label text,merchant text,label_kind text,classification_source text DEFAULT 'unknown',UNIQUE(owner_id,hash));
 CREATE TABLE domains(id bigint PRIMARY KEY,name text,color text,is_transfer bool,parent_id bigint,owner_id bigint);
 CREATE TABLE rules(id bigserial,pattern text,domain_id bigint,owner_id bigint,tags text);
 CREATE TABLE merchant_aliases(owner_id bigint,match_key text,display text,is_pattern bool DEFAULT false);
 INSERT INTO accounts VALUES(11,1,'COMPTE TEST'),(21,2,'AUTRE COMPTE');
 INSERT INTO statements(id,filename,owner_id,account_id) VALUES(100,'seed-1',1,11),(200,'seed-2',2,21);`)
	if err != nil {
		t.Fatal(err)
	}
	a := models.Transaction{Date: "2026-10-01", Label: "TEST A", Amount: -10}
	b := models.Transaction{Date: "2026-10-02", Label: "TEST B", Amount: 20}
	if _, err = db.Exec(`INSERT INTO transactions(statement_id,tx_date,label,amount,hash,owner_id) VALUES(100,$1,$2,$3,$4,1),(200,$5,$6,$7,$8,2)`, a.Date, a.Label, a.Amount, parser.TxHash(a), b.Date, b.Label, b.Amount, parser.TxHash(b)); err != nil {
		t.Fatal(err)
	}
	srv := &Server{DB: db}
	ownerRequest := func(method, path string, owner int64) *http.Request {
		req := httptest.NewRequest(method, path, nil)
		return req.WithContext(context.WithValue(req.Context(), ctxUserKey, &authUser{ID: owner}))
	}
	count := func(table string) int {
		t.Helper()
		var n int
		if err := db.QueryRow("SELECT count(*) FROM " + table + " WHERE owner_id=1").Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	preview := func(csv, account string) importPreviewResponse {
		t.Helper()
		var body bytes.Buffer
		form := multipart.NewWriter(&body)
		f, _ := form.CreateFormFile("file", "test.csv")
		_, _ = f.Write([]byte(csv))
		_ = form.WriteField("account_id", account)
		_ = form.Close()
		req := ownerRequest("POST", "/api/statements/previews", 1)
		req.Body = http.NoBody
		req = httptest.NewRequest("POST", "/api/statements/previews", &body).WithContext(req.Context())
		req.Header.Set("Content-Type", form.FormDataContentType())
		w := httptest.NewRecorder()
		srv.uploadStatement(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("preview %d: %s", w.Code, w.Body)
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("financial preview must not be cached")
		}
		var out importPreviewResponse
		if err := json.NewDecoder(w.Body).Decode(&out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	confirm := func(token string, owner int64) *httptest.ResponseRecorder {
		req := ownerRequest("POST", "/api/statements/previews/"+token+"/confirm", owner)
		req.SetPathValue("token", token)
		w := httptest.NewRecorder()
		srv.confirmImportPreview(w, req)
		return w
	}
	ambiguous := preview("Date;Libelle;Montant\n02/10/2026;TEST B;20,00\n02/10/2026;TEST B;20,00\n", "11")
	if w := confirm(ambiguous.Token, 1); w.Code < 400 || count("transactions") != 1 || count("statements") != 1 {
		t.Fatal("ambiguous identical occurrences were silently merged")
	}
	csv := "Date;Libelle;Montant\n01/10/2026;TEST A;-10,00\n02/10/2026;TEST B;20,00\n"
	out := preview(csv, "11")
	if out.NewCount != 1 || out.Duplicates != 1 || out.AccountID != 11 || out.BalanceDifference != nil {
		t.Fatalf("unexpected preview: %+v", out)
	}
	if count("statements") != 1 || count("transactions") != 1 {
		t.Fatal("preview wrote financial data")
	}
	if w := confirm(out.Token, 2); w.Code != http.StatusGone {
		t.Fatal("another owner confirmed preview")
	}
	cancel := ownerRequest("DELETE", "/api/statements/previews/"+out.Token, 1)
	cancel.SetPathValue("token", out.Token)
	srv.cancelImportPreview(httptest.NewRecorder(), cancel)
	if w := confirm(out.Token, 1); w.Code != http.StatusGone {
		t.Fatal("cancelled preview confirmed")
	}
	if count("statements") != 1 || count("transactions") != 1 {
		t.Fatal("cancel changed financial data")
	}
	out = preview(csv, "11")
	// Un autre import survient après l'aperçu : la contrainte SQL reste décisive.
	if _, err = db.Exec(`INSERT INTO transactions(statement_id,tx_date,label,amount,hash,owner_id) VALUES(100,$1,$2,$3,$4,1)`, b.Date, b.Label, b.Amount, parser.TxHash(b)); err != nil {
		t.Fatal(err)
	}
	w := confirm(out.Token, 1)
	if w.Code != http.StatusOK || count("statements") != 1 || count("transactions") != 2 {
		t.Fatalf("concurrent duplicates: %d %s", w.Code, w.Body)
	}
	var result struct {
		Inserted   int `json:"inserted"`
		Duplicates int `json:"duplicates"`
	}
	_ = json.NewDecoder(w.Body).Decode(&result)
	if result.Inserted != 0 || result.Duplicates != 2 {
		t.Fatalf("duplicates not rechecked: %+v", result)
	}
	out = preview("Date;Libelle;Montant\n03/10/2026;TEST C;-5,00\n", "11")
	w = confirm(out.Token, 1)
	if w.Code != http.StatusCreated || count("transactions") != 3 || count("statements") != 2 {
		t.Fatalf("confirmation failed: %d %s", w.Code, w.Body)
	}
	if w = confirm(out.Token, 1); w.Code != http.StatusGone || count("transactions") != 3 {
		t.Fatal("confirmation replay was accepted")
	}
	// Vérifier qu'une collision sur un autre compte ne supprime aucune ligne.
	if _, err = db.Exec(`INSERT INTO accounts VALUES(12,1,'SECOND COMPTE TEST')`); err != nil {
		t.Fatal(err)
	}
	collision := preview("Date;Libelle;Montant\n01/10/2026;TEST A;-10,00\n", "12")
	if w := confirm(collision.Token, 1); w.Code < 400 || count("transactions") != 3 || count("statements") != 2 {
		t.Fatal("cross-account collision was silently discarded")
	}
	// La somme persistée doit être exactement celle des trois sources.
	var total string
	if err = db.QueryRow(`SELECT sum(amount)::text FROM transactions WHERE owner_id=1`).Scan(&total); err != nil || total != "5.00" {
		t.Fatalf("unreconciled database total %s: %v", total, err)
	}
	out = preview(csv, "11")
	if _, err = db.Exec(`UPDATE accounts SET owner_id=2 WHERE id=11`); err != nil {
		t.Fatal(err)
	}
	if w = confirm(out.Token, 1); w.Code != http.StatusBadRequest || count("transactions") != 3 {
		t.Fatal("confirmation did not recheck account ownership")
	}
	// API/dashboard source : un mois ancien doit rester interrogeable même si
	// des mois plus récents existent ; les catégories doivent avoir la même borne.
	if _, err = db.Exec(`INSERT INTO transactions(statement_id,owner_id,tx_date,label,amount,hash) VALUES(100,1,'2020-01-03','HISTORIQUE TEST',-12.34,'historical-test')`); err != nil {
		t.Fatal(err)
	}
	summaryResponse := httptest.NewRecorder()
	srv.summary(summaryResponse, ownerRequest("GET", "/api/summary?months=1&month=2020-01", 1))
	if summaryResponse.Code != http.StatusOK {
		t.Fatalf("summary: %d %s", summaryResponse.Code, summaryResponse.Body)
	}
	var summary struct {
		Monthly []struct {
			Month        string  `json:"month"`
			Income       float64 `json:"income"`
			Expenses     float64 `json:"expenses"`
			Completeness string  `json:"completeness"`
		} `json:"monthly"`
		Categories []struct {
			Month string  `json:"month"`
			Total float64 `json:"total"`
		} `json:"by_category"`
	}
	if err = json.Unmarshal(summaryResponse.Body.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	if len(summary.Monthly) != 1 || summary.Monthly[0].Month != "2020-01" || summary.Monthly[0].Expenses != 12.34 || summary.Monthly[0].Income != 0 || summary.Monthly[0].Completeness == "complete" {
		t.Fatalf("wrong source period or certainty: %s", summaryResponse.Body)
	}
	if len(summary.Categories) != 1 || summary.Categories[0].Month != "2020-01" || summary.Categories[0].Total != summary.Monthly[0].Expenses {
		t.Fatalf("category totals diverge: %s", summaryResponse.Body)
	}

}
