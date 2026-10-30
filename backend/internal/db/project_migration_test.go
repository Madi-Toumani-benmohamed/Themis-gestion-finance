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

func TestProjectMigrationKeepsLegacyAndGuardsRollbackPostgres(t *testing.T) {
	dsn := os.Getenv("THEMIS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("THEMIS_TEST_DATABASE_URL non défini")
	}
	admin, e := sql.Open("postgres", dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer admin.Close()
	schema := fmt.Sprintf("project_migration_%d", time.Now().UnixNano())
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
	if _, e = db.Exec(`CREATE TABLE schema_migrations(name text PRIMARY KEY,applied_at timestamptz DEFAULT now())`); e != nil {
		t.Fatal(e)
	}
	entries, _ := migrations.ReadDir("migrations")
	for _, entry := range entries {
		if entry.Name() >= "031_" {
			continue
		}
		data, e := migrations.ReadFile("migrations/" + entry.Name())
		if e != nil {
			t.Fatal(e)
		}
		if _, e = db.Exec(string(data)); e != nil {
			t.Fatal(entry.Name(), e)
		}
		if _, e = db.Exec(`INSERT INTO schema_migrations(name) VALUES($1)`, entry.Name()); e != nil {
			t.Fatal(e)
		}
	}
	if _, e = db.Exec(`INSERT INTO users(id,username,password_hash,role) VALUES(1,'PROJECT MIGRATION TEST','TEST','admin');
 INSERT INTO tags(id,owner_id,name) VALUES(9001,1,'Voyage historique TEST');
 INSERT INTO tag_folder_prefs(owner_id,tag_name,state) VALUES(1,'Voyage historique TEST','pinned');
 INSERT INTO accounts(id,owner_id,name,kind) VALUES(9001,1,'PROJECT TEST','courant');
 INSERT INTO statements(id,owner_id,account_id,filename) VALUES(9001,1,9001,'PROJECT SYNTHETIC TEST');
 INSERT INTO transactions(id,statement_id,owner_id,tx_date,label,amount,hash) VALUES(9001,9001,1,'2025-12-10','SOURCE TEST',-10.01,'project-migration-test');
 INSERT INTO transaction_tags(transaction_id,tag_id) VALUES(9001,9001)`); e != nil {
		t.Fatal(e)
	}
	for range 2 {
		if e = Migrate(db); e != nil {
			t.Fatal(e)
		}
	}
	var amount, state string
	var count int
	if e = db.QueryRow(`SELECT t.amount::text,p.state FROM transactions t JOIN transaction_tags tt ON tt.transaction_id=t.id JOIN tags tg ON tg.id=tt.tag_id JOIN tag_folder_prefs p ON p.owner_id=tg.owner_id AND p.tag_name=tg.name WHERE t.id=9001`).Scan(&amount, &state); e != nil || amount != "-10.01" || state != "pinned" {
		t.Fatal("legacy changed", amount, state, e)
	}
	if e = db.QueryRow(`SELECT count(*) FROM financial_projects`).Scan(&count); e != nil || count != 0 {
		t.Fatal("hypotheses silently confirmed", count, e)
	}
	rollback, e := os.ReadFile("rollbacks/031_financial_projects.sql")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(string(rollback)); e != nil {
		t.Fatal("empty rollback", e)
	}
	if e = Migrate(db); e != nil {
		t.Fatal("reapply", e)
	}
	var id int64
	if e = db.QueryRow(`INSERT INTO financial_projects(owner_id,name) VALUES(1,'PROJET TEST') RETURNING id`).Scan(&id); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(string(rollback)); e == nil {
		t.Fatal("rollback lost projects")
	}
	if _, e = db.Exec(`INSERT INTO project_transactions(owner_id,project_id,transaction_id,state,reason) VALUES(1,$1,9001,'confirmed','{"kind":"manual"}')`, id); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(`DELETE FROM transactions WHERE id=9001`); e == nil {
		t.Fatal("confirmed source deleted")
	}
	if _, e = db.Exec(`UPDATE project_transactions SET state='rejected' WHERE project_id=$1`, id); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(`DELETE FROM transactions WHERE id=9001`); e != nil {
		t.Fatal("withdrawn source undeletable", e)
	}
}
