package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func qualityTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("THEMIS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("THEMIS_TEST_DATABASE_URL non défini")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	return db
}

func TestNormalizerOwnerIsolationAndAtomicVersionPostgres(t *testing.T) {
	db := qualityTestDB(t)
	_, err := db.Exec(`CREATE TEMP TABLE transactions(id bigint PRIMARY KEY,owner_id bigint,label text,amount numeric(12,2),display_label text,merchant text,label_kind text);
 CREATE TEMP TABLE merchant_aliases(owner_id bigint,match_key text,display text,is_pattern bool DEFAULT false);
 CREATE TEMP TABLE settings(key text,value jsonb,owner_id bigint,updated_at timestamptz DEFAULT now());
 CREATE UNIQUE INDEX quality_settings_key ON settings(COALESCE(owner_id,0),key);
 INSERT INTO transactions VALUES(1,1,'CB PAYPAL ORIGINAL 1',-10.01,'wrong','',''),(2,2,'CB PAYPAL ORIGINAL 2',-20.02,'wrong','',''),(3,3,'CB PAYPAL ORIGINAL 3',-30.03,'wrong','','');
 INSERT INTO merchant_aliases(owner_id,match_key,display) VALUES(1,'paypal','SURNOM UTILISATEUR 1'),(2,'PayPal','SURNOM UTILISATEUR 2');
 INSERT INTO merchant_aliases VALUES(1,'REF%_','MOTIF PERSONNEL',true),(2,'REF%_','AUTRE MOTIF PRIVE',true);
 INSERT INTO transactions VALUES(4,1,'CB TEST REF%_ VARIABLE',-1,'old','',''),(5,2,'CB TEST REF%_ VARIABLE',-1,'old','','');
 INSERT INTO settings(key,value,owner_id) VALUES('normalizer_version','"999"',1),('normalizer_version','"3"',NULL);`)
	if err != nil {
		t.Fatal(err)
	}
	srv := &Server{DB: db}
	if err = srv.backfillDisplayLabels(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []struct {
		id                     int
		display, label, amount string
	}{{1, "SURNOM UTILISATEUR 1", "CB PAYPAL ORIGINAL 1", "-10.01"}, {2, "SURNOM UTILISATEUR 2", "CB PAYPAL ORIGINAL 2", "-20.02"}, {3, "PayPal", "CB PAYPAL ORIGINAL 3", "-30.03"}} {
		var display, label, amount string
		if err = db.QueryRow(`SELECT display_label,label,amount::text FROM transactions WHERE id=$1`, want.id).Scan(&display, &label, &amount); err != nil {
			t.Fatal(err)
		}
		if display != want.display || label != want.label || amount != want.amount {
			t.Fatalf("owner/source altered: %d %q %q %q", want.id, display, label, amount)
		}
	}
	for _, expected := range []struct {
		id, owner int
		display   string
	}{{4, 1, "MOTIF PERSONNEL"}, {5, 2, "AUTRE MOTIF PRIVE"}} {
		var display string
		if err = db.QueryRow(`SELECT display_label FROM transactions WHERE id=$1`, expected.id).Scan(&display); err != nil || display != expected.display {
			t.Fatalf("pattern backfill %d %q %v", expected.id, display, err)
		}
		if display, ok := srv.aliasFor(int64(expected.owner), "TEST", "CB TEST REF%_ VARIABLE"); !ok || display != expected.display {
			t.Fatal("import alias differs from restart")
		}
		if _, ok := srv.aliasFor(int64(expected.owner), "TEST", "CB TEST REFXA VARIABLE"); ok {
			t.Fatal("alias wildcard used for literal pattern")
		}
	}
	var version string
	if err = db.QueryRow(`SELECT value #>> '{}' FROM settings WHERE owner_id IS NULL`).Scan(&version); err != nil || version != normalizerVersion {
		t.Fatalf("version: %q %v", version, err)
	}
	// An unchanged version must not repeatedly rewrite already normalized rows.
	if _, err = db.Exec(`UPDATE transactions SET display_label='PRESERVED' WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if err = srv.backfillDisplayLabels(); err != nil {
		t.Fatal(err)
	}
	var display string
	db.QueryRow(`SELECT display_label FROM transactions WHERE id=1`).Scan(&display)
	if display != "PRESERVED" {
		t.Fatal("repeat startup rewrote display")
	}
	// Fail before commit: all transaction displays and global version remain intact.
	if _, err = db.Exec(`UPDATE settings SET value='"0"' WHERE owner_id IS NULL;ALTER TABLE merchant_aliases RENAME COLUMN display TO broken_display`); err != nil {
		t.Fatal(err)
	}
	if err = srv.backfillDisplayLabels(); err == nil {
		t.Fatal("expected alias read error")
	}
	db.QueryRow(`SELECT display_label FROM transactions WHERE id=1`).Scan(&display)
	db.QueryRow(`SELECT value #>> '{}' FROM settings WHERE owner_id IS NULL`).Scan(&version)
	if display != "PRESERVED" || version != "0" {
		t.Fatal("failed backfill partially committed")
	}
}

func TestCategorySeriesFinancialScopePostgres(t *testing.T) {
	db := qualityTestDB(t)
	_, err := db.Exec(`CREATE TEMP TABLE domains(id bigint PRIMARY KEY,owner_id bigint,parent_id bigint,name text,color text,is_transfer bool);
 CREATE TEMP TABLE transactions(id bigint PRIMARY KEY,owner_id bigint,tx_date date,amount numeric(12,2),domain_id bigint,label_kind text);
 INSERT INTO domains VALUES(1,1,NULL,'Courses','#ffffff',false),(2,1,NULL,'Transfert','#ffffff',true),(3,1,2,'Enfant','#ffffff',false),(4,2,NULL,'SECRET','#ffffff',false);
 INSERT INTO transactions VALUES(1,1,'2020-01-10',-10.01,1,''),(2,1,'2020-01-10',-100,2,''),(3,1,'2020-01-10',-200,3,''),(4,1,'2020-01-10',-300,1,'savings'),(5,1,'2020-02-10',-99,1,''),(6,2,'2020-01-10',-999,4,''),(7,1,'2020-01-11',5,1,'');`)
	if err != nil {
		t.Fatal(err)
	}
	srv := &Server{DB: db}
	req := httptest.NewRequest("GET", "/api/category-series?from=2020-01-01&to=2020-01-31", nil)
	req = req.WithContext(context.WithValue(req.Context(), ctxUserKey, &authUser{ID: 1}))
	w := httptest.NewRecorder()
	srv.categorySeries(w, req)
	var report struct {
		Series []map[string]any `json:"series"`
	}
	if err = json.Unmarshal(w.Body.Bytes(), &report); err != nil || w.Code != 200 || len(report.Series) != 1 || report.Series[0]["Courses"] != 10.01 || strings.Contains(w.Body.String(), "SECRET") {
		t.Fatalf("scope: %d %s %v", w.Code, w.Body, err)
	}
	for _, query := range []string{"from=2020-02-31", "from=2020-02-01&to=2020-01-01"} {
		req = httptest.NewRequest("GET", "/api/category-series?"+query, nil).WithContext(req.Context())
		w = httptest.NewRecorder()
		srv.categorySeries(w, req)
		if w.Code != 400 {
			t.Fatalf("invalid dates: %d %s", w.Code, w.Body)
		}
	}
}

func TestAssistantMerchantSearchScopePostgres(t *testing.T) {
	db := qualityTestDB(t)
	_, err := db.Exec(`CREATE TEMP TABLE domains(id bigint PRIMARY KEY,owner_id bigint,parent_id bigint,name text,is_transfer bool);
 CREATE TEMP TABLE transactions(id bigint PRIMARY KEY,owner_id bigint,tx_date date,label text,display_label text,merchant text,amount numeric(12,2),domain_id bigint,label_kind text);
 INSERT INTO domains VALUES(1,1,NULL,'Courses',false),(2,1,NULL,'Transfert',true);
 INSERT INTO transactions VALUES(1,1,'2020-01-10','RAW','Affichage','MARCHAND TEST',-10.01,1,''),(2,2,'2020-01-10','RAW SECRET','SECRET','MARCHAND TEST',-999,1,''),(3,1,'2020-01-10','RAW','Affichage','MARCHAND TEST',-100,2,''),(4,1,'2020-02-10','RAW','Affichage','MARCHAND TEST',-200,1,'');`)
	if err != nil {
		t.Fatal(err)
	}
	result := toolSearchTransactions(db, 1, map[string]any{"start_date": "2020-01-01", "end_date": "2020-01-31", "merchant": "MARCHAND TEST"})
	if result.Error != "" {
		t.Fatal(result.Error)
	}
	var sources []map[string]any
	if err = json.Unmarshal([]byte(result.Content), &sources); err != nil || len(sources) != 1 || sources[0]["amount"] != -10.01 || sources[0]["id"] != float64(1) {
		t.Fatalf("merchant scope: %+v %v", result, err)
	}
	_, err = db.Exec(`INSERT INTO transactions VALUES(10,1,'2020-03-01','TEST SALARY','TEST','TEST',3249.47,1,''),(11,1,'2020-03-02','TEST EXPENSE','TEST','TEST',-4197.14,1,'')`)
	if err != nil {
		t.Fatal(err)
	}
	result = toolFinancialSummary(db, 1, map[string]any{"start_date": "2020-03-01", "end_date": "2020-03-31"})
	decoder := json.NewDecoder(strings.NewReader(result.Content))
	decoder.UseNumber()
	var exact map[string]any
	if err = decoder.Decode(&exact); err != nil || result.Error != "" || exact["net"] != json.Number("-947.67") || exact["completeness"] != "unknown" || exact["query_from"] != "2020-03-01" {
		t.Fatalf("assistant exact totals: %+v %v", result, err)
	}
	result = toolFinancialSummary(db, 1, map[string]any{"start_date": "2020-03-01", "end_date": "2020-03-31", "group_by": "category"})
	if result.Error != "" || !strings.Contains(result.Content, `"expenses":4197.14`) {
		t.Fatalf("group exact totals: %+v", result)
	}
}
