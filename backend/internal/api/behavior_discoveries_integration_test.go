package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/lib/pq"
)

func TestBehaviorDiscoveriesPostgres(t *testing.T) {
	dsn := os.Getenv("THEMIS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("THEMIS_TEST_DATABASE_URL non défini")
	}
	admin, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("behavior_discoveries_%d", time.Now().UnixNano())
	if _, err = admin.Exec("CREATE SCHEMA " + pq.QuoteIdentifier(schema)); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec("DROP SCHEMA " + pq.QuoteIdentifier(schema) + " CASCADE")
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
	_, err = db.Exec(`CREATE TABLE accounts(id bigint PRIMARY KEY,owner_id bigint,name text,tracking_mode text);
 CREATE TABLE domains(id bigint PRIMARY KEY,owner_id bigint,parent_id bigint,name text,color text,is_transfer bool);
 CREATE TABLE statements(id bigserial PRIMARY KEY,owner_id bigint,account_id bigint,filename text,period_start date,period_end date,validated bool,opening_balance numeric(12,2),closing_balance numeric(12,2),uploaded_at timestamptz DEFAULT now());
 CREATE TABLE transactions(id bigserial PRIMARY KEY,owner_id bigint,statement_id bigint,tx_date date,label text,amount numeric(12,2),domain_id bigint,merchant text DEFAULT '',display_label text DEFAULT '',label_kind text DEFAULT '');
 INSERT INTO accounts VALUES(1,7,'Synthetic checking','statements');
	INSERT INTO domains VALUES(10,7,NULL,'Food','#0a0',false),(20,7,NULL,'Internal transfer','#aaa',true);`)
	if err != nil {
		t.Fatal(err)
	}
	months := []string{"2026-01", "2026-02", "2026-03", "2026-04", "2026-05", "2026-06"}
	balance := int64(100000)
	for i, month := range months {
		first, _ := time.Parse("2006-01", month)
		last := first.AddDate(0, 1, -1)
		amount := int64(10000 + i*2000)
		var statementID int64
		err = db.QueryRow(`INSERT INTO statements(owner_id,account_id,filename,period_start,period_end,validated,opening_balance,closing_balance)
		VALUES(7,1,'SYNTHETIC',$1,$2,true,$3::numeric/100,($3-$4-5000)::numeric/100) RETURNING id`, first.Format("2006-01-02"), last.Format("2006-01-02"), balance, amount).Scan(&statementID)
		if err != nil {
			t.Fatal(err)
		}
		balance -= amount + 5000
		if _, err = db.Exec(`INSERT INTO transactions(owner_id,statement_id,tx_date,label,amount,domain_id,merchant,display_label)
 VALUES(7,$1,$2,'Synthetic grocery',-$3::numeric/100,10,'Synthetic Market','Synthetic Market')`, statementID, first.AddDate(0, 0, 9).Format("2006-01-02"), amount); err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(`INSERT INTO transactions(owner_id,statement_id,tx_date,label,amount,domain_id,merchant,display_label)
		 VALUES(7,$1,$2,'Internal transfer',-50,20,'Transfer Test','Transfer Test')`, statementID, first.AddDate(0, 0, 12).Format("2006-01-02")); err != nil {
			t.Fatal(err)
		}
	}
	discoveries, complete, err := (&Server{DB: db}).loadBehaviorDiscoveries(context.Background(), 7, "2026-06", time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if !complete {
		t.Fatal("six reconciled monthly statements should establish complete coverage")
	}
	found := map[string]bool{}
	for _, d := range discoveries {
		found[d.Kind] = true
		if d.Title == "Hausse progressive : Transfer Test" {
			t.Fatal("internal transfer was classified as an ordinary behavioral expense")
		}
		if d.Coverage.Status != "complete" || d.Coverage.Periods != 6 || d.OwnerID != 7 {
			t.Fatalf("invalid coverage or owner: %+v", d)
		}
	}
	if !found["progressive_trend"] || !found["spend_variation_contributors"] {
		t.Fatalf("expected evidenced trend and variation, got %+v", discoveries)
	}
	if _, err = db.Exec(`UPDATE statements SET validated=false WHERE period_start='2026-06-01'`); err != nil {
		t.Fatal(err)
	}
	partial, complete, err := (&Server{DB: db}).loadBehaviorDiscoveries(context.Background(), 7, "2026-06", time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if complete || len(partial) != 0 {
		t.Fatalf("incomplete documentary coverage generated comparative signals: complete=%v discoveries=%+v", complete, partial)
	}
}
