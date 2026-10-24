package api

import (
	"context"
	"database/sql"
	"os"
	"testing"
)

func TestTransactionRuleProvenancePostgres(t *testing.T) {
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
CREATE TEMP TABLE domains(id bigint PRIMARY KEY,owner_id bigint);
CREATE TEMP TABLE rules(id bigserial PRIMARY KEY,owner_id bigint,domain_id bigint,pattern text,tags text);
CREATE TEMP TABLE transactions(id bigint PRIMARY KEY,owner_id bigint,statement_id bigint,label text,domain_id bigint,classification_source text DEFAULT 'unknown');
CREATE TEMP TABLE tags(id bigserial PRIMARY KEY,owner_id bigint,name text,UNIQUE(owner_id,name));
CREATE TEMP TABLE transaction_tags(transaction_id bigint,tag_id bigint,UNIQUE(transaction_id,tag_id));
INSERT INTO domains VALUES(1,1),(2,2);
INSERT INTO transactions(id,owner_id,statement_id,label,domain_id) VALUES(1,1,1,'PRLV EDF',null),(2,1,2,'prlv edf',null),(3,1,2,'A_B%',null),(4,1,2,'AXXB',null),(5,2,3,'PRLV EDF',null),(6,1,2,'PRLV EDF',1);
INSERT INTO rules(owner_id,domain_id,pattern,tags) VALUES(1,1,'re:^PRLV EDF$','factures'),(1,2,'AXXB','secret');`)
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{DB: db}
	// Import applies rules only to newly imported sources. It does not tag or
	// recategorize historical or foreign transactions as a side effect.
	n, err := s.applyAllRulesForStatement(1, 1)
	if err != nil || n != 1 {
		t.Fatalf("statement rule %d %v", n, err)
	}
	var count int
	if err = db.QueryRow(`SELECT count(*) FROM transaction_tags`).Scan(&count); err != nil || count != 1 {
		t.Fatal("import changed historical tags", err)
	}
	var source string
	if err = db.QueryRow(`SELECT classification_source FROM transactions WHERE id=1`).Scan(&source); err != nil || source != "rule" {
		t.Fatal("missing rule provenance", err)
	}
	if err = s.validateRule(context.Background(), 1, 2, "EDF"); err == nil {
		t.Fatal("foreign category accepted")
	}
	if _, err = s.applyRuleToTransactions(1, "EDF", 2, ""); err == nil {
		t.Fatal("foreign rule applied")
	}
	if _, err = s.applyRuleToTransactions(1, "re:[", 1, ""); err == nil {
		t.Fatal("invalid regex accepted")
	}
	matched, err := s.applyRuleToTransactions(1, "A_B%", 1, "")
	if err != nil || matched != 1 {
		t.Fatalf("literal matching %d %v", matched, err)
	}
	n, err = s.applyAllRules(1)
	if err != nil || n != 1 {
		t.Fatalf("historical explicit application %d %v", n, err)
	}
	var untouched int
	if err = db.QueryRow(`SELECT count(*) FROM transactions WHERE id IN(4,5) AND domain_id IS NULL`).Scan(&untouched); err != nil || untouched != 2 {
		t.Fatal("literal/owner isolation failed", err)
	}
	if err = db.QueryRow(`SELECT classification_source FROM transactions WHERE id=6`).Scan(&source); err != nil || source != "unknown" {
		t.Fatal("existing provenance rewritten", err)
	}
	// Tag errors roll back a newly assigned category rather than silently
	// leaving a partially applied rule. Temp tables keep public data isolated.
	_, err = db.Exec(`UPDATE transactions SET domain_id=null,classification_source='unknown' WHERE id=3; ALTER TABLE tags RENAME COLUMN name TO broken_name`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.applyRuleToTransactions(1, "A_B%", 1, "test"); err == nil {
		t.Fatal("tag failure ignored")
	}
	var empty bool
	if err = db.QueryRow(`SELECT domain_id IS NULL AND classification_source='unknown' FROM transactions WHERE id=3`).Scan(&empty); err != nil || !empty {
		t.Fatal("partial rule committed", err)
	}
}
