package db

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/lib/pq"
)

func TestTransactionClassificationMigrationPostgres(t *testing.T) {
	dsn := os.Getenv("THEMIS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("THEMIS_TEST_DATABASE_URL non défini")
	}
	admin, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("classification_test_%d", time.Now().UnixNano())
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
	_, err = db.Exec(`CREATE TABLE users(id bigint PRIMARY KEY);CREATE TABLE domains(id bigint PRIMARY KEY);
CREATE TABLE schema_migrations(name text PRIMARY KEY);
CREATE TABLE transactions(id bigint PRIMARY KEY,owner_id bigint,tx_date date,label text,amount numeric(12,2),domain_id bigint);
INSERT INTO users VALUES(1);INSERT INTO domains VALUES(1);
INSERT INTO transactions VALUES(1,1,'2020-01-01','ORIGINAL TEST',-12.34,1);`)
	if err != nil {
		t.Fatal(err)
	}
	migration, err := migrations.ReadFile("migrations/021_transaction_classification.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(string(migration)); err != nil {
		t.Fatal(err)
	}
	var source, label, amount string
	var domain int
	if err = db.QueryRow(`SELECT classification_source,label,amount::text,domain_id FROM transactions WHERE id=1`).Scan(&source, &label, &amount, &domain); err != nil || source != "unknown" || label != "ORIGINAL TEST" || amount != "-12.34" || domain != 1 {
		t.Fatalf("migration changed history %s %s %s %d %v", source, label, amount, domain, err)
	}
	if _, err = db.Exec(`UPDATE transactions SET classification_source='confirmed' WHERE id=1`); err == nil {
		t.Fatal("unrecognised provenance accepted")
	}
	if _, err = db.Exec(`INSERT INTO transaction_suggestions(transaction_id,owner_id,domain_id,tags) VALUES(1,1,1,ARRAY['test']);INSERT INTO schema_migrations VALUES('021_transaction_classification.sql')`); err != nil {
		t.Fatal(err)
	}
	rollback, err := os.ReadFile("rollbacks/021_transaction_classification.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(string(rollback)); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`SELECT label,amount::text,domain_id FROM transactions WHERE id=1`).Scan(&label, &amount, &domain); err != nil || label != "ORIGINAL TEST" || amount != "-12.34" || domain != 1 {
		t.Fatal("rollback damaged originals", err)
	}
	if _, err = db.Exec(string(migration)); err != nil {
		t.Fatal("migration not reapplicable after rollback", err)
	}
}
