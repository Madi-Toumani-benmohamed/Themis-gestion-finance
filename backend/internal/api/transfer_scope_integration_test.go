package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"os"
	"testing"
)

func transferScopeTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("THEMIS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("THEMIS_TEST_DATABASE_URL non défini")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	return db
}

func TestTransferScopeSignalsAndBankFlowsPostgres(t *testing.T) {
	db := transferScopeTestDB(t)
	_, err := db.Exec(`CREATE TEMP TABLE domains(id bigint PRIMARY KEY,owner_id bigint,parent_id bigint,name text,is_transfer bool DEFAULT false);
CREATE TEMP TABLE transactions(id bigserial PRIMARY KEY,statement_id bigint,owner_id bigint,tx_date date,label text DEFAULT '',display_label text DEFAULT '',merchant text DEFAULT '',amount numeric(12,2),domain_id bigint,label_kind text DEFAULT '');
CREATE TEMP TABLE tags(id bigint,owner_id bigint,name text);
CREATE TEMP TABLE transaction_tags(transaction_id bigint,tag_id bigint);
INSERT INTO domains VALUES
 (1,1,NULL,'Ordinaire',false),(2,1,NULL,'Transfert interne',true),(3,1,2,'Compte courant / Livret A',false),
 (4,1,1,'Ordinaire enfant',false),(5,2,NULL,'Transfert privé',true),(6,2,5,'Enfant privé',false);
-- Three prior complete months establish the ordinary-category comparison.
INSERT INTO transactions(owner_id,tx_date,amount,domain_id) VALUES
 (1,'2020-01-10',-10,1),(1,'2020-02-10',-10,1),(1,'2020-03-10',-10,1),
 (1,'2020-01-10',-10,4),(1,'2020-02-10',-10,4),(1,'2020-03-10',-10,4),
 (1,'2020-04-10',-11,1),(1,'2020-04-10',-60,4),
 (1,'2020-04-10',-1000,3),(1,'2020-04-10',-500,2),
	(1,'2020-04-10',-39,1),(1,'2020-04-10',100,3),
 (2,'2020-04-10',-9000,6);
INSERT INTO transactions(owner_id,tx_date,amount,domain_id,label_kind) VALUES(1,'2020-04-10',-75,1,'savings');
INSERT INTO tags VALUES(1,1,'compte-courant/livret A');
INSERT INTO transaction_tags VALUES(11,1);`)
	if err != nil {
		t.Fatal(err)
	}

	// Both category signals must use shared ExpenseT: direct and inherited
	// transfer domains are absent, ordinary parent and child remain eligible.
	var heroName string
	if err := db.QueryRow(heroCategoryIncreaseSQL, "2020-04", int64(1)).Scan(&heroName, new(float64), new(float64)); err != nil {
		t.Fatalf("hero category query: %v", err)
	}
	if heroName != "Ordinaire enfant" {
		t.Fatalf("hero signal selected %q; want ordinary child domain", heroName)
	}
	rows, err := db.Query(insightCategorySpikeSQL, int64(1))
	if err != nil {
		t.Fatalf("insight category query: %v", err)
	}
	var insightNames []string
	for rows.Next() {
		var name string
		var current, average float64
		if err = rows.Scan(&name, &current, &average); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		insightNames = append(insightNames, name)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		t.Fatal(err)
	}
	rows.Close()
	if len(insightNames) != 2 || !hasString(insightNames, "Ordinaire") || !hasString(insightNames, "Ordinaire enfant") {
		t.Fatalf("insight categories: %v", insightNames)
	}

	// The TopWidget consumes the backend debit scope; it sees ordinary roots
	// and children while preserving no tag-based transfer semantics.
	srv := &Server{DB: db}
	req := httptest.NewRequest("GET", "/api/transactions?month=2020-04&type=debit", nil)
	req = req.WithContext(context.WithValue(req.Context(), ctxUserKey, &authUser{ID: 1}))
	w := httptest.NewRecorder()
	srv.listTransactions(w, req)
	if w.Code != 200 {
		t.Fatalf("filtered transactions: %d %s", w.Code, w.Body)
	}
	var txs []struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &txs); err != nil {
		t.Fatal(err)
	}
	if len(txs) != 3 {
		t.Fatalf("backend debit scope returned %d rows, want 3 ordinary debits", len(txs))
	}
	for _, tx := range txs {
		if tx.ID == 9 || tx.ID == 10 || tx.ID == 13 || tx.ID == 14 {
			t.Fatalf("transfer, savings or foreign-owner row leaked: id %d", tx.ID)
		}
	}

	// Economic filtering does not rewrite or remove any bank movement. This
	// exact raw count and net are the inputs used by balances/reconciliation.
	var rawCount int
	var rawNet string
	if err := db.QueryRow(`SELECT count(*),sum(amount)::text FROM transactions WHERE owner_id=1 AND tx_date='2020-04-10'`).Scan(&rawCount, &rawNet); err != nil {
		t.Fatal(err)
	}
	if rawCount != 7 || rawNet != "-1585.00" {
		t.Fatalf("raw bank movements changed: count=%d net=%s", rawCount, rawNet)
	}
	var ordinaryDebit string
	if err := db.QueryRow(`SELECT COALESCE(-sum(t.amount),0)::text FROM transactions t WHERE t.owner_id=1 AND t.tx_date='2020-04-10' AND t.amount<0 AND ` + financialExpenseSQLT).Scan(&ordinaryDebit); err != nil {
		t.Fatal(err)
	}
	if ordinaryDebit != "110.00" {
		t.Fatalf("economic debit total %s; want 110.00", ordinaryDebit)
	}
}

func hasString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
