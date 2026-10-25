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

func TestStatementMigrationLegacyAndGuardedRollback(t *testing.T) {
	dsn := os.Getenv("THEMIS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("THEMIS_TEST_DATABASE_URL non défini")
	}
	root, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	schema := fmt.Sprintf("statement_migration_%d", time.Now().UnixNano())
	if _, err = root.Exec(`CREATE SCHEMA ` + pq.QuoteIdentifier(schema)); err != nil {
		t.Fatal(err)
	}
	defer root.Exec(`DROP SCHEMA ` + pq.QuoteIdentifier(schema) + ` CASCADE`)
	u, _ := url.Parse(dsn)
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	db, err := sql.Open("postgres", u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(`CREATE TABLE schema_migrations(name text PRIMARY KEY,applied_at timestamptz DEFAULT now())`); err != nil {
		t.Fatal(err)
	}
	entries, _ := migrations.ReadDir("migrations")
	for _, entry := range entries {
		if entry.Name() >= "025_" {
			continue
		}
		b, _ := migrations.ReadFile("migrations/" + entry.Name())
		if _, err = db.Exec(string(b)); err != nil {
			t.Fatal(entry.Name(), err)
		}
		db.Exec(`INSERT INTO schema_migrations(name) VALUES($1)`, entry.Name())
	}
	if _, err = db.Exec(`INSERT INTO users(id,username,password_hash,role) VALUES(1,'MIGRATION DOC TEST','TEST','admin');UPDATE accounts SET owner_id=1;INSERT INTO accounts(id,name,kind,owner_id) VALUES(12,'Ordinary arbitrary name','livret_a',1);INSERT INTO statements(id,filename,account_id,owner_id,opening_balance,closing_balance,validated) VALUES(101,'HISTORICAL TEST',1,1,33.33,44.44,true)`); err != nil {
		t.Fatal(err)
	}
	if err = Migrate(db); err != nil {
		t.Fatal(err)
	}
	if err = Migrate(db); err != nil {
		t.Fatal("repeat", err)
	}
	var current, manual, opening, closing string
	var validated bool
	if err = db.QueryRow(`SELECT (SELECT tracking_mode FROM accounts WHERE id=1),(SELECT tracking_mode FROM accounts WHERE id=12),opening_balance::text,closing_balance::text,validated FROM statements WHERE id=101`).Scan(&current, &manual, &opening, &closing, &validated); err != nil || current != "statements" || manual != "manual" || opening != "33.33" || closing != "44.44" || !validated {
		t.Fatal("legacy modified", current, manual, opening, closing, validated, err)
	}
	rollback, err := os.ReadFile("rollbacks/025_statement_controls.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(string(rollback)); err != nil {
		t.Fatal("unused rollback", err)
	}
	if err = Migrate(db); err != nil {
		t.Fatal("reapply", err)
	}
	if _, err = db.Exec(`UPDATE accounts SET tracking_mode='statements' WHERE id=12`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(string(rollback)); err == nil {
		t.Fatal("custom tracking lost")
	}
	if _, err = db.Exec(`UPDATE accounts SET tracking_mode='manual' WHERE id=12;INSERT INTO statement_validation_history(statement_id,resource_id,owner_id,actor_id,action,previous_validated,note,snapshot) VALUES(101,101,1,1,'reopen',true,'Original reviewed TEST','{}')`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(string(rollback)); err == nil {
		t.Fatal("manual proof lost")
	}
	var n int
	if err = db.QueryRow(`SELECT count(*) FROM statement_validation_history`).Scan(&n); err != nil || n != 1 {
		t.Fatal("guard not atomic")
	}
}
