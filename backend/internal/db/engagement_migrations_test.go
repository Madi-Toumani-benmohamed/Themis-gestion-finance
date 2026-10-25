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

func TestEngagementMigrationsFreshAndLegacyPostgres(t *testing.T) {
	dsn := os.Getenv("THEMIS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("THEMIS_TEST_DATABASE_URL non défini")
	}
	admin, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	for _, legacy := range []bool{false, true} {
		t.Run(fmt.Sprint("legacy=", legacy), func(t *testing.T) {
			schema := fmt.Sprintf("engagement_migrations_%d", time.Now().UnixNano())
			if _, err = admin.Exec(`CREATE SCHEMA ` + pq.QuoteIdentifier(schema)); err != nil {
				t.Fatal(err)
			}
			defer admin.Exec(`DROP SCHEMA ` + pq.QuoteIdentifier(schema) + ` CASCADE`)
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
			if legacy {
				if _, err = db.Exec(`CREATE TABLE schema_migrations(name text PRIMARY KEY,applied_at timestamptz DEFAULT now())`); err != nil {
					t.Fatal(err)
				}
				entries, _ := migrations.ReadDir("migrations")
				for _, e := range entries {
					if e.Name() >= "024_" {
						continue
					}
					data, _ := migrations.ReadFile("migrations/" + e.Name())
					if _, err = db.Exec(string(data)); err != nil {
						t.Fatal(e.Name(), err)
					}
					db.Exec(`INSERT INTO schema_migrations(name) VALUES($1)`, e.Name())
				}
				_, err = db.Exec(`INSERT INTO users(id,username,password_hash,role) VALUES(1,'MIGRATION TEST','TEST','admin');INSERT INTO credits(id,owner_id,name,monthly_amount,months_total,start_month) VALUES(1,1,'CREDIT TEST',33.33,12,'2020-01');INSERT INTO taxes(id,owner_id,name,amount,due_date) VALUES(1,1,'TAX TEST',100.01,'2020-01-01');INSERT INTO tax_history(tax_id,paid_before,paid_after,changed_by) VALUES(1,false,true,1)`)
				if err != nil {
					t.Fatal(err)
				}
			}
			if err = Migrate(db); err != nil {
				t.Fatal(err)
			}
			if err = Migrate(db); err != nil {
				t.Fatal("repeat runner", err)
			}
			if legacy {
				var opening sql.NullInt64
				var amount string
				if err = db.QueryRow(`SELECT opening_paid_cents,monthly_amount::text FROM credits WHERE id=1`).Scan(&opening, &amount); err != nil || opening.Valid || amount != "33.33" {
					t.Fatal("financial backfill", opening, amount, err)
				}
				var owner, resource int64
				var snapshot sql.NullString
				if err = db.QueryRow(`SELECT owner_id,resource_id,after_state::text FROM tax_history WHERE tax_id=1`).Scan(&owner, &resource, &snapshot); err != nil || owner != 1 || resource != 1 || snapshot.Valid {
					t.Fatal("legacy history", owner, resource, snapshot, err)
				}
			}
			rollback, err := os.ReadFile("rollbacks/024_engagement_integrity.sql")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = db.Exec(string(rollback)); err != nil {
				t.Fatal("safe rollback", err)
			}
			if err = Migrate(db); err != nil {
				t.Fatal("reapply after rollback", err)
			}
			if legacy {
				if _, err = db.Exec(`UPDATE credits SET opening_paid_cents=0 WHERE id=1`); err != nil {
					t.Fatal(err)
				}
				if _, err = db.Exec(string(rollback)); err == nil {
					t.Fatal("rollback lost financial declarations")
				}
				var preserved int64
				if err = db.QueryRow(`SELECT opening_paid_cents FROM credits WHERE id=1`).Scan(&preserved); err != nil || preserved != 0 {
					t.Fatal("guard failed", preserved, err)
				}
				// Removing an old actor must not erase another owner's financial history.
				if _, err = db.Exec(`INSERT INTO users(id,username,password_hash,role) VALUES(2,'ACTOR TEST','TEST','user');UPDATE tax_history SET changed_by=2 WHERE tax_id=1;DELETE FROM users WHERE id=2`); err != nil {
					t.Fatal("actor removal", err)
				}
				var actor sql.NullInt64
				if err = db.QueryRow(`SELECT changed_by FROM tax_history WHERE tax_id=1`).Scan(&actor); err != nil || actor.Valid {
					t.Fatal("history erased with actor", actor, err)
				}
				// Explicit user deletion retains its existing cascade semantics.
				if _, err = db.Exec(`INSERT INTO credit_payments(credit_id,owner_id,amount_cents,payment_date) VALUES(1,1,1,'2020-01-01');DELETE FROM users WHERE id=1`); err != nil {
					t.Fatal("owner cascade regression", err)
				}

			}

		})
	}
}
