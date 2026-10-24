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

// Test optionnel sur PostgreSQL : toutes les tables sont temporaires, sans
// lecture ni écriture des données réelles de la base fournie.
func TestReliabilityPostgres(t *testing.T) {
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
 CREATE TEMP TABLE accounts(id bigint,owner_id bigint,name text);
 CREATE TEMP TABLE statements(id bigint,account_id bigint,owner_id bigint,filename text,period_start date,period_end date,validated bool,uploaded_at timestamptz);
 CREATE TEMP TABLE transactions(statement_id bigint,owner_id bigint,tx_date date);
 INSERT INTO accounts VALUES(11,1,'Compte test'),(12,1,'Compte sans couverture validée'),(21,2,'AUTRE UTILISATEUR');
 INSERT INTO statements VALUES
 (101,11,1,'cycle-a.pdf','2026-08-22','2026-09-21',true,'2026-09-23'),
 (102,11,1,'cycle-b.pdf','2026-09-22','2026-10-21',true,'2026-10-23'),
 (103,12,1,'controle.csv','2026-09-01','2026-09-30',false,'2026-10-23'),
 (104,12,1,'periode-inconnue.csv',null,null,false,'2026-10-23'),
 (201,21,2,'AUTRE-UTILISATEUR.pdf','2026-09-01','2026-09-30',true,'2026-10-23');
 INSERT INTO transactions VALUES(102,1,'2026-09-29'),(103,1,'2026-09-30'),(201,2,'2026-09-30');`)
	if err != nil {
		t.Fatal(err)
	}
	srv := &Server{DB: db}
	now := reliabilityTestDate(t, "2026-10-23")
	report, err := srv.loadReliability(context.Background(), 1, "2026-09", now)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Accounts) != 2 || report.Current || report.Through != "2026-09-30" {
		t.Fatalf("unexpected report: %+v", report)
	}
	a, b := report.Accounts[0], report.Accounts[1]
	if a.Status != "documented" || len(a.Gaps) != 0 || a.LastTransaction != "2026-09-29" || a.LastImport != "2026-10-23" {
		t.Fatalf("documented account: %+v", a)
	}
	if b.Status != "to_review" || b.ToReview != 2 || b.UnknownPeriods != 1 || len(b.Gaps) != 1 {
		t.Fatalf("review account: %+v", b)
	}
	raw, _ := json.Marshal(report)
	if strings.Contains(string(raw), "AUTRE") {
		t.Fatal("other user's data leaked")
	}
	if srv.heroMonthCovered(1, "2026-09") {
		t.Fatal("hero should agree coverage is insufficient")
	}
	if !srv.heroMonthCovered(2, "2026-09") {
		t.Fatal("fully documented other account should be covered")
	}
	current, err := srv.loadReliability(context.Background(), 1, "2026-10", now)
	if err != nil || !current.Current || current.Through != "2026-10-23" {
		t.Fatalf("current report: %+v, %v", current, err)
	}
	if got := current.Accounts[0].Gaps; len(got) != 1 || got[0].From != "2026-10-22" || got[0].To != "2026-10-23" {
		t.Fatalf("future days treated as missing: %v", got)
	}
	empty, err := srv.loadReliability(context.Background(), 3, "2026-09", now)
	if err != nil || len(empty.Accounts) != 0 {
		t.Fatal("empty owner should have no accounts")
	}
	// Le propriétaire provient de la session, jamais d'un paramètre fourni.
	req := httptest.NewRequest("GET", "/api/reliability?month=2026-09&owner_id=2", nil)
	req = req.WithContext(context.WithValue(req.Context(), ctxUserKey, &authUser{ID: 1}))
	w := httptest.NewRecorder()
	srv.reliability(w, req)
	if w.Code != http.StatusOK || strings.Contains(w.Body.String(), "AUTRE") {
		t.Fatalf("ownership failed: %d %s", w.Code, w.Body)
	}
}
