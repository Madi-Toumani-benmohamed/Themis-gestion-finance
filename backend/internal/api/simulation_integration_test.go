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

func TestSimulationPostgres(t *testing.T) {
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
	_, err = db.Exec(`CREATE TEMP TABLE accounts(id bigint,owner_id bigint,name text);
 CREATE TEMP TABLE domains(id bigint,name text,is_transfer bool,owner_id bigint DEFAULT 1,parent_id bigint);
 CREATE TEMP TABLE statements(id bigint,account_id bigint,owner_id bigint,filename text,period_start date,period_end date,validated bool,uploaded_at timestamptz DEFAULT now());
 CREATE TEMP TABLE transactions(statement_id bigint,owner_id bigint,tx_date date,amount numeric(12,2),domain_id bigint,hash text UNIQUE,label_kind text DEFAULT '');
 INSERT INTO accounts VALUES(1,1,'TEST');
 INSERT INTO domains(id,name,is_transfer) VALUES(1,'A',false),(2,'B',false),(3,'Virement interne',true);
 INSERT INTO statements(id,account_id,owner_id,filename,period_start,period_end,validated)
 SELECT n,1,1,'TEST',date_trunc('month',CURRENT_DATE)-n*interval '1 month',date_trunc('month',CURRENT_DATE)-(n-1)*interval '1 month'-interval '1 day',true FROM generate_series(1,3) n;
 INSERT INTO transactions(statement_id,owner_id,tx_date,amount,domain_id,hash)
 SELECT n,1,date_trunc('month',CURRENT_DATE)-n*interval '1 month',-100,1,'A'||n FROM generate_series(1,3) n;
 INSERT INTO transactions(statement_id,owner_id,tx_date,amount,domain_id,hash) VALUES
 (1,1,date_trunc('month',CURRENT_DATE)-interval '1 month',1000,null,'income1'),
 (3,1,date_trunc('month',CURRENT_DATE)-interval '3 months',1000,null,'income3'),
 (1,1,date_trunc('month',CURRENT_DATE)-interval '1 month',-30,2,'B'),
 (1,1,date_trunc('month',CURRENT_DATE)-interval '1 month',-10,1,'similar1'),
 (1,1,date_trunc('month',CURRENT_DATE)-interval '1 month',-10,1,'similar2'),
 (1,1,date_trunc('month',CURRENT_DATE)-interval '1 month',9900,3,'transferIn'),
 (1,1,date_trunc('month',CURRENT_DATE)-interval '1 month',-9900,3,'transferOut'),
 (1,1,CURRENT_DATE,50000,null,'currentIncome'),
 (1,1,CURRENT_DATE,-90000,1,'currentExpense'),
 (1,2,date_trunc('month',CURRENT_DATE)-interval '1 month',99999,null,'otherOwner');`)
	if err != nil {
		t.Fatal(err)
	}
	srv := &Server{DB: db}
	run := func(owner int64) (*httptest.ResponseRecorder, map[string]json.RawMessage) {
		t.Helper()
		req := httptest.NewRequest("POST", "/api/simulate", nil)
		req = httptest.NewRequest("POST", "/api/simulate", strings.NewReader(`{}`)).WithContext(context.WithValue(req.Context(), ctxUserKey, &authUser{ID: owner}))
		w := httptest.NewRecorder()
		srv.simulate(w, req)
		var out map[string]json.RawMessage
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		return w, out
	}
	w, out := run(1)
	if w.Code != 200 {
		t.Fatalf("simulate %d: %s", w.Code, w.Body)
	}
	for key, want := range map[string]string{"income": "666.67", "base_spend": "116.67", "current_net": "550.00", "reference_complete": "true"} {
		if string(out[key]) != want {
			t.Errorf("%s: %s != %s", key, out[key], want)
		}
	}
	var periods []periodInfo
	_ = json.Unmarshal(out["reference_periods"], &periods)
	if len(periods) != 3 {
		t.Fatal("window must contain three months")
	}
	for _, p := range periods {
		if p.IsCurrent || p.Completeness != "complete" {
			t.Fatalf("incomplete period silently used: %+v", p)
		}
	}
	var categories []simulationCategory
	_ = json.Unmarshal(out["categories"], &categories)
	if len(categories) != 2 || categories[0].Before != "106.67" || categories[1].Before != "10.00" {
		t.Fatalf("category divisor or distinct transactions lost: %s", out["categories"])
	}
	// Une couverture insuffisante doit changer le statut, pas disparaître.
	if _, err = db.Exec(`UPDATE statements SET validated=false WHERE id=2`); err != nil {
		t.Fatal(err)
	}
	_, out = run(1)
	if string(out["reference_complete"]) != "false" {
		t.Fatal("missing coverage hidden")
	}
	_, out = run(3)
	if string(out["available"]) != "false" {
		t.Fatal("missing data presented as zero")
	}
	// Un mois sans opération peut être un zéro établi uniquement si documenté.
	if _, err = db.Exec(`DELETE FROM transactions WHERE owner_id=1;UPDATE statements SET validated=true`); err != nil {
		t.Fatal(err)
	}
	w, out = run(1)
	if w.Code != 200 || string(out["available"]) != "true" || string(out["income"]) != "0.00" {
		t.Fatalf("documented zero: %s", w.Body)
	}
	if _, err = db.Exec(`DROP TABLE domains`); err != nil {
		t.Fatal(err)
	}
	w, _ = run(1)
	if w.Code != 500 {
		t.Fatal("SQL error converted to financial zero")
	}
}
