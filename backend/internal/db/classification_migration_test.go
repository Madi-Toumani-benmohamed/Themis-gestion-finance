package db

import (
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"net/url"
	"os"
	"testing"
	"time"
)

func TestClassificationMigrationLegacyAndGuardedRollbackPostgres(t *testing.T) {
	dsn := os.Getenv("THEMIS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("THEMIS_TEST_DATABASE_URL non défini")
	}
	admin, e := sql.Open("postgres", dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer admin.Close()
	schema := fmt.Sprintf("classification_migration_%d", time.Now().UnixNano())
	if _, e = admin.Exec(`CREATE SCHEMA ` + pq.QuoteIdentifier(schema)); e != nil {
		t.Fatal(e)
	}
	defer admin.Exec(`DROP SCHEMA ` + pq.QuoteIdentifier(schema) + ` CASCADE`)
	u, e := url.Parse(dsn)
	if e != nil {
		t.Fatal(e)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	db, e := sql.Open("postgres", u.String())
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	if e = Migrate(db); e != nil {
		t.Fatal(e)
	}
	rollback, e := os.ReadFile("rollbacks/035_classification_decisions.sql")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(string(rollback)); e != nil {
		t.Fatal("unused inverse", e)
	}
	if _, e = db.Exec(`INSERT INTO users(id,username,password_hash,role) VALUES(1,'CLASSIFICATION SYNTHETIC','TEST','admin'),(2,'SECOND SYNTHETIC','TEST','user');
 INSERT INTO accounts(id,owner_id,name,kind) VALUES(9001,1,'SYNTHETIC','courant');
 INSERT INTO statements(id,owner_id,account_id,filename) VALUES(9001,1,9001,'SYNTHETIC');
 INSERT INTO domains(id,owner_id,name,color) VALUES(9001,1,'SYNTHETIC','#123456');
 INSERT INTO transactions(id,statement_id,owner_id,tx_date,label,amount,domain_id,classification_source,hash) VALUES(9001,9001,1,'2026-01-01','SYNTHETIC RAW',-10.01,9001,'manual','synthetic-classification');
 INSERT INTO tags(id,owner_id,name) VALUES(9001,1,'synthetic');INSERT INTO transaction_tags VALUES(9001,9001)`); e != nil {
		t.Fatal(e)
	}
	proof := func() string {
		t.Helper()
		var p string
		if e := db.QueryRow(`SELECT jsonb_build_array(t.*, (SELECT jsonb_agg(tt.*) FROM transaction_tags tt WHERE transaction_id=t.id))::text FROM transactions t WHERE id=9001`).Scan(&p); e != nil {
			t.Fatal(e)
		}
		return p
	}
	before := proof()
	for range 2 {
		if e = Migrate(db); e != nil {
			t.Fatal(e)
		}
	}
	if proof() != before {
		t.Fatal("migration changed legacy transaction")
	}
	var n int
	if e = db.QueryRow(`SELECT (SELECT count(*) FROM classification_state)+(SELECT count(*) FROM classification_events)+(SELECT count(*) FROM classification_tag_decisions)`).Scan(&n); e != nil || n != 0 {
		t.Fatal("fabricated legacy provenance", n, e)
	}
	if _, e = db.Exec(`INSERT INTO classification_state(owner_id,transaction_id) VALUES(2,9001)`); e == nil {
		t.Fatal("cross-owner FK accepted")
	}
	// An untouched historical installation remains reversible after validation.
	if _, e = db.Exec(string(rollback)); e != nil {
		t.Fatal("legacy-only inverse", e)
	}
	if e = Migrate(db); e != nil {
		t.Fatal("reapply", e)
	}
	if _, e = db.Exec(`BEGIN;SELECT classification_begin(1::bigint,9001::bigint,'manual','synthetic-operation',repeat('a',64),0::bigint);SELECT classification_set_locks(1::bigint,9001::bigint,true,NULL);SELECT classification_finish(1::bigint,9001::bigint);COMMIT`); e != nil {
		t.Fatal(e)
	}
	tx, e := db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	if _, e = tx.Exec(string(rollback)); e == nil {
		tx.Rollback()
		t.Fatal("inverse discarded decisions")
	}
	tx.Rollback()
	if proof() != before {
		t.Fatal("metadata decision changed financial source")
	}
	if e = db.QueryRow(`SELECT count(*) FROM classification_events WHERE owner_id=1 AND transaction_id=9001`).Scan(&n); e != nil || n != 1 {
		t.Fatal("guard lost journal", n, e)
	}
	if _, e = db.Exec(`DELETE FROM transactions WHERE id=9001`); e != nil {
		t.Fatal("existing transaction deletion blocked", e)
	}
	if e = db.QueryRow(`SELECT count(*) FROM classification_events WHERE owner_id=1 AND transaction_id=9001`).Scan(&n); e != nil || n != 1 {
		t.Fatal("source deletion erased journal", n, e)
	}
	if _, e = db.Exec(`DELETE FROM users WHERE id=1`); e != nil {
		t.Fatal("owner erasure blocked", e)
	}
	if e = db.QueryRow(`SELECT count(*) FROM classification_events WHERE owner_id=1`).Scan(&n); e != nil || n != 0 {
		t.Fatal("owner metadata survived erasure", n, e)
	}
}
