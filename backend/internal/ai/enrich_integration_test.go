package ai

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

type proposalClient struct {
	db         *sql.DB
	concurrent bool
}

func (c proposalClient) Name() string { return "isolated-test" }
func (c proposalClient) Complete(context.Context, string, string, int, bool) (string, error) {
	if c.concurrent {
		if _, err := c.db.Exec(`UPDATE transactions SET domain_id=1,classification_source='manual' WHERE id=1`); err != nil {
			return "", err
		}
	}
	return `{"assignments":[{"id":1,"domain":"Courses","tags":[" FACTURE ","facture"]},{"id":1,"domain":"Courses","tags":["duplicate"]},{"id":2,"domain":"SECRET CATEGORY","tags":["secret"]},{"id":3,"domain":"Courses","tags":["secret"]},{"id":999,"domain":"Courses","tags":["invented"]}]}`, nil
}

func TestAIEnrichmentProposesWithoutAssigningPostgres(t *testing.T) {
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
CREATE TEMP TABLE domains(id bigint PRIMARY KEY,owner_id bigint,parent_id bigint,name text);
CREATE TEMP TABLE transactions(id bigint PRIMARY KEY,owner_id bigint,statement_id bigint,tx_date date,label text,amount numeric(12,2),domain_id bigint,classification_source text DEFAULT 'unknown');
CREATE TEMP TABLE transaction_suggestions(transaction_id bigint PRIMARY KEY,owner_id bigint,domain_id bigint,tags text[],source text DEFAULT 'ai',created_at timestamptz DEFAULT now());
CREATE TEMP TABLE tags(id bigint PRIMARY KEY,owner_id bigint,name text);
CREATE TEMP TABLE transaction_tags(transaction_id bigint,tag_id bigint);
INSERT INTO domains VALUES(1,1,null,'Courses'),(2,2,null,'SECRET CATEGORY');
INSERT INTO transactions(id,owner_id,statement_id,tx_date,label,amount) VALUES(1,1,1,'2020-01-01','TEST RAW',-12.34),(2,1,1,'2020-01-02','TEST RAW 2',-22),(3,2,2,'2020-01-02','SECRET RAW',-22);
INSERT INTO tags VALUES(1,1,'existing');INSERT INTO transaction_tags VALUES(1,1);`)
	if err != nil {
		t.Fatal(err)
	}
	result, err := AutoEnrich(context.Background(), db, proposalClient{db: db}, 1, 0, "2020-01")
	if err != nil || result.Suggested != 1 || result.DomainsSet != 0 || result.TagsSet != 0 {
		t.Fatalf("proposals %+v %v", result, err)
	}
	var n int
	if err = db.QueryRow(`SELECT count(*) FROM transactions WHERE domain_id IS NOT NULL OR classification_source<>'unknown'`).Scan(&n); err != nil || n != 0 {
		t.Fatal("AI silently assigned category", err)
	}
	if err = db.QueryRow(`SELECT count(*) FROM transaction_tags`).Scan(&n); err != nil || n != 1 {
		t.Fatal("AI silently changed tags", err)
	}
	if err = db.QueryRow(`SELECT count(*) FROM transaction_suggestions WHERE transaction_id=1 AND owner_id=1 AND domain_id=1 AND tags=ARRAY['facture']`).Scan(&n); err != nil || n != 1 {
		t.Fatal("proposal validation failed", err)
	}
	var raw string
	if err = db.QueryRow(`SELECT label FROM transactions WHERE id=1`).Scan(&raw); err != nil || raw != "TEST RAW" {
		t.Fatal("source label changed", err)
	}
	_, err = db.Exec(`DELETE FROM transaction_suggestions`)
	if err != nil {
		t.Fatal(err)
	}
	result, err = AutoEnrich(context.Background(), db, proposalClient{db: db, concurrent: true}, 1, 0, "2020-01")
	if err != nil || result.Suggested != 0 {
		t.Fatalf("manual edit during LLM call lost: %+v %v", result, err)
	}
}
