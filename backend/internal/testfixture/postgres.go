package testfixture

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/lib/pq"
	database "themis/internal/db"
)

// Postgres returns an isolated, migrated schema. No public or live table is used.
func Postgres(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("THEMIS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("THEMIS_TEST_DATABASE_URL non défini")
	}
	admin, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("classification_fixture_%d", time.Now().UnixNano())
	if _, err = admin.Exec(`CREATE SCHEMA ` + pq.QuoteIdentifier(schema)); err != nil {
		admin.Close()
		t.Fatal(err)
	}
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
	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		db.Close()
		admin.Exec(`DROP SCHEMA ` + pq.QuoteIdentifier(schema) + ` CASCADE`)
		admin.Close()
	})
	if err = database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO users(id,username,password_hash,role) VALUES(1,'SYNTHETIC ONE','TEST','admin'),(2,'SYNTHETIC TWO','TEST','user')`); err != nil {
		t.Fatal(err)
	}
	return db
}

// ClassificationShadows attaches production guards to legacy temporary fixtures.
// LIKE retains checks/indices but deliberately has no foreign keys to shadowed
// base tables; full-schema tests exercise those owner and referential constraints.
func ClassificationShadows(t *testing.T, db *sql.DB) {
	t.Helper()
	statements := []string{}
	for _, table := range []string{"classification_state", "classification_tag_decisions", "classification_rejections", "classification_events", "classification_operations", "classification_rule_versions"} {
		statements = append(statements, `CREATE TEMP TABLE `+table+` (LIKE `+table+` INCLUDING ALL)`)
	}
	statements = append(statements, `DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_class WHERE relname='transaction_suggestions' AND relnamespace=pg_my_temp_schema()) THEN
 ALTER TABLE transaction_suggestions ADD COLUMN IF NOT EXISTS identity text NOT NULL DEFAULT '';
 ALTER TABLE transaction_suggestions ADD COLUMN IF NOT EXISTS context_key text NOT NULL DEFAULT '';
 ALTER TABLE transaction_suggestions ADD COLUMN IF NOT EXISTS base_revision bigint NOT NULL DEFAULT 0;
 END IF;
 END $$`)
	for _, def := range []struct{ table, sql string }{
		{"transactions", `CREATE TRIGGER classification_transaction_insert AFTER INSERT ON transactions FOR EACH ROW EXECUTE FUNCTION classification_transaction_guard(); CREATE TRIGGER classification_transaction_update AFTER UPDATE OF domain_id,classification_source ON transactions FOR EACH ROW EXECUTE FUNCTION classification_transaction_guard()`},
		{"transaction_tags", `CREATE TRIGGER classification_tag_before BEFORE INSERT OR DELETE ON transaction_tags FOR EACH ROW EXECUTE FUNCTION classification_tag_guard(); CREATE TRIGGER classification_tag_after AFTER INSERT OR DELETE ON transaction_tags FOR EACH ROW EXECUTE FUNCTION classification_tag_guard()`},
		{"rules", `CREATE TRIGGER classification_rules AFTER INSERT OR UPDATE OF pattern,domain_id,tags ON rules FOR EACH ROW EXECUTE FUNCTION classification_rule_version()`},
	} {
		var exists bool
		if err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM pg_class WHERE relname=$1 AND relnamespace=pg_my_temp_schema())`, def.table).Scan(&exists); err != nil {
			t.Fatal(err)
		}
		if exists {
			statements = append(statements, def.sql)
		}
	}
	statements = append(statements, `DISCARD PLANS`)
	for _, s := range statements {
		if _, err := db.Exec(s); err != nil {
			t.Fatal(err)
		}
	}
}
