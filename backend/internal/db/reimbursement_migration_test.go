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

func TestReimbursementMigrationPreservesTransactionsAndSafeRollbackPostgres(t *testing.T) {
	dsn := os.Getenv("THEMIS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("THEMIS_TEST_DATABASE_URL non défini")
	}
	admin, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("reimbursement_migration_%d", time.Now().UnixNano())
	if _, err = admin.Exec(`CREATE SCHEMA ` + pq.QuoteIdentifier(schema)); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(`DROP SCHEMA ` + pq.QuoteIdentifier(schema) + ` CASCADE`)
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	query := u.Query()
	query.Set("search_path", schema)
	u.RawQuery = query.Encode()
	db, err := sql.Open("postgres", u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = Migrate(db); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO users(id,username,password_hash,role) VALUES(1,'REFUND MIGRATION TEST','TEST','admin');
	 INSERT INTO accounts(id,owner_id,name,kind) VALUES(9001,1,'MIGRATION TEST','courant');
	 INSERT INTO statements(id,owner_id,account_id,filename,opening_balance,closing_balance,validated) VALUES(9001,1,9001,'SYNTHETIC TEST',100,90,true);
	 INSERT INTO transactions(id,statement_id,owner_id,tx_date,label,amount,hash) VALUES(9001,9001,1,'2026-01-10','SYNTHETIC ORIGINAL',-10.00,'synthetic-original');`); err != nil {
		t.Fatal(err)
	}
	var beforeAmount, afterAmount string
	var beforeDate, afterDate, beforeHash, afterHash string
	if err = db.QueryRow(`SELECT amount::text,tx_date::text,hash FROM transactions WHERE id=9001`).Scan(&beforeAmount, &beforeDate, &beforeHash); err != nil {
		t.Fatal(err)
	}
	if err = Migrate(db); err != nil {
		t.Fatal("repeat migration", err)
	}
	if err = db.QueryRow(`SELECT amount::text,tx_date::text,hash FROM transactions WHERE id=9001`).Scan(&afterAmount, &afterDate, &afterHash); err != nil {
		t.Fatal(err)
	}
	if beforeAmount != afterAmount || beforeDate != afterDate || beforeHash != afterHash || beforeAmount != "-10.00" {
		t.Fatalf("bank transaction changed: before=%s/%s/%s after=%s/%s/%s", beforeAmount, beforeDate, beforeHash, afterAmount, afterDate, afterHash)
	}
	rollback, err := os.ReadFile("rollbacks/027_reimbursements.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(string(rollback)); err != nil {
		t.Fatal("empty-schema rollback", err)
	}
	if err = Migrate(db); err != nil {
		t.Fatal("reapply after rollback", err)
	}
	if _, err = db.Exec(`INSERT INTO reimbursement_cases(owner_id,credit_transaction_id,credit_snapshot,kind) VALUES(1,9001,'{"transaction_id":9001,"amount_cents":1000}','personal_expense')`); err == nil {
		t.Fatal("debit accepted as reimbursement credit")
	}
	if _, err = db.Exec(`INSERT INTO transactions(id,statement_id,owner_id,tx_date,label,amount,hash) VALUES(9002,9001,1,'2026-01-11','SYNTHETIC CREDIT',10.00,'synthetic-credit')`); err != nil {
		t.Fatal("synthetic credit", err)
	}
	if _, err = db.Exec(`INSERT INTO reimbursement_cases(owner_id,credit_transaction_id,credit_snapshot,kind) VALUES(1,9002,'{"transaction_id":9002,"amount_cents":1000}','personal_expense')`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(string(rollback)); err == nil {
		t.Fatal("rollback should refuse after explicit reimbursement use")
	}
	if err = db.QueryRow(`SELECT amount::text FROM transactions WHERE id=9001`).Scan(&afterAmount); err != nil || afterAmount != "-10.00" {
		t.Fatal("guarded rollback changed financial source", afterAmount, err)
	}
}
