package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func analysisTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db := qualityTestDB(t)
	_, err := db.Exec(`CREATE TEMP TABLE domains(id bigint PRIMARY KEY,owner_id bigint,parent_id bigint,name text,color text,is_transfer bool DEFAULT false);
 CREATE TEMP TABLE accounts(id bigint PRIMARY KEY,owner_id bigint,name text,tracking_mode text NOT NULL DEFAULT 'statements');
 CREATE TEMP TABLE statements(id bigint PRIMARY KEY,owner_id bigint,account_id bigint,filename text,validated bool,period_start date,period_end date,uploaded_at timestamptz DEFAULT now());
 CREATE TEMP TABLE transactions(id bigserial PRIMARY KEY,statement_id bigint,owner_id bigint,tx_date date,label text DEFAULT 'TEST',display_label text DEFAULT '',merchant text DEFAULT '',label_kind text DEFAULT '',amount numeric(12,2),domain_id bigint);
 CREATE TEMP TABLE tags(id bigint PRIMARY KEY,owner_id bigint,name text);
 CREATE TEMP TABLE transaction_tags(transaction_id bigint,tag_id bigint,PRIMARY KEY(transaction_id,tag_id));
 CREATE TEMP TABLE tag_folder_prefs(owner_id bigint,tag_name text,state text,PRIMARY KEY(owner_id,tag_name));
 INSERT INTO accounts(id,owner_id,name) VALUES(1,1,'TEST'),(2,2,'SECRET');
 INSERT INTO domains VALUES(1,1,NULL,'Courses','#ffffff',false),(2,1,1,'Enfant','#ffffff',false),(3,1,NULL,'Interne','#ffffff',true),(4,1,3,'Interne enfant','#ffffff',false),(5,2,NULL,'SECRET','#ffffff',true);
 INSERT INTO statements(id,owner_id,account_id,filename,validated,period_start,period_end) VALUES(1,1,1,'TEST.csv',true,'2019-10-01','2020-02-29'),(2,2,2,'SECRET.csv',true,'2019-10-01','2020-02-29');
 INSERT INTO transactions(statement_id,owner_id,tx_date,amount,domain_id,label_kind,merchant) VALUES
 (1,1,'2020-01-09',-10.01,2,'','CAFE TEST'),(1,1,'2020-01-10',-20.02,1,'tax','CAFE TEST'),
 (1,1,'2020-01-11',5.01,1,'','REMBOURSEMENT TEST'),(1,1,'2020-01-12',-100,4,'','VIREMENT TEST'),(1,1,'2020-01-13',100,4,'','ANNULATION TEST'),
 (1,1,'2020-01-14',-50,1,'savings','PEL TEST'),(1,1,'2020-01-15',50,1,'savings','PEL ENTRANT TEST'),
 (1,1,'2020-01-16',-0.01,5,'','CATEGORIE ETRANGERE TEST'),(2,2,'2020-01-09',-9999,5,'','SECRET'),
 (1,1,'2020-02-10',-99,1,'','FUTURE WINDOW TEST'),(1,1,'2019-10-10',-0.01,1,'','REFERENCE TEST'),(1,1,'2019-11-10',-0.01,NULL,'','REFERENCE TEST');
 INSERT INTO tags VALUES(1,1,'projet test'),(2,1,'second test'),(3,2,'secret');
 INSERT INTO transaction_tags VALUES(1,1),(1,2),(1,3),(2,1),(3,1),(9,1),(10,1);`)
	if err != nil {
		t.Fatal(err)
	}
	seedDocumentedStatementBalances(t, db)
	return db
}
func TestAnalysisReportPostgres(t *testing.T) {
	db := analysisTestDB(t)
	srv := &Server{DB: db}
	now := time.Date(2020, 2, 21, 0, 0, 0, 0, time.UTC)
	report, err := srv.loadAnalysisReport(context.Background(), 1, "2020-01", now)
	if err != nil {
		t.Fatal(err)
	}
	if report.Expenses != "30.04" || report.Income != "5.01" || report.Net != "-25.03" || report.TotalOut != "180.04" || report.Savings != "50.00" || report.TransferIn != "150.00" || report.TransferOut != "100.00" {
		t.Fatalf("non-reconciled: %+v", report)
	}
	if report.Period.Completeness != "complete" || len(report.Categories) != 2 || report.Categories[0].Spent != "30.03" || report.UncategorizedSpent != "0.01" {
		t.Fatalf("categories/coverage: %+v", report)
	}
	empty, err := srv.loadAnalysisReport(context.Background(), 1, "2019-12", now)
	if err != nil || !empty.Available || empty.Expenses != "0.00" {
		t.Fatalf("documented zero: %+v %v", empty, err)
	}
	absent, err := srv.loadAnalysisReport(context.Background(), 1, "2019-09", now)
	if err != nil || absent.Available || absent.Period.Completeness == "complete" {
		t.Fatalf("absent != zero: %+v %v", absent, err)
	}
	current, err := srv.loadAnalysisReport(context.Background(), 1, "2020-02", now)
	if err != nil || current.Period.Completeness == "complete" || current.Income != "0.00" {
		t.Fatalf("current month: %+v %v", current, err)
	}
	_, err = db.Exec(`UPDATE transactions SET statement_id=NULL WHERE id=1`)
	if err != nil {
		t.Fatal(err)
	}
	orphan, err := srv.loadAnalysisReport(context.Background(), 1, "2020-01", now)
	if err != nil || orphan.Period.Completeness == "complete" || orphan.UnattributedCount != 1 {
		t.Fatalf("orphan certification: %+v %v", orphan, err)
	}
}
func TestAnalysisEndpointsPostgres(t *testing.T) {
	db := analysisTestDB(t)
	srv := &Server{DB: db}
	call := func(path string, handler func(http.ResponseWriter, *http.Request)) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", path, nil)
		r = r.WithContext(context.WithValue(r.Context(), ctxUserKey, &authUser{ID: 1}))
		w := httptest.NewRecorder()
		handler(w, r)
		return w
	}
	for _, tc := range []struct {
		path    string
		handler func(http.ResponseWriter, *http.Request)
		want    string
	}{
		{"/api/intel/nature?month=2020-01", srv.natureBreakdown, `"expenses":30.04`},
		{"/api/intel/merchants?month=2020-01&months=1", srv.merchantsView, `"total":30.04`},
		{"/api/intel/comparison?month=2020-01", srv.categoryComparison, `"denominator":3`},
		{"/api/analytics/tags-in-category?category=1&month=2020-01", srv.tagsInCategory, `"category_total":30.03`},
		{"/api/analytics/tag-series?tags=projet%20test&month=2020-01&months=1", srv.tagSeries, `30.03`},
		{"/api/tag-folders?month=2020-01", srv.tagFolders, `"total":30.03`},
	} {
		w := call(tc.path, tc.handler)
		if w.Code != 200 || !strings.Contains(w.Body.String(), tc.want) || strings.Contains(w.Body.String(), "SECRET") || strings.Contains(w.Body.String(), "FUTURE WINDOW") {
			t.Fatalf("%s => %d %s", tc.path, w.Code, w.Body)
		}
	}
	w := call("/api/analytics/tags-in-category?category=1&month=2020-01", srv.tagsInCategory)
	var tags map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &tags); err != nil {
		t.Fatal(err)
	}
	if tags["overlap_count"] != float64(1) || len(tags["slices"].([]any)) != 2 {
		t.Fatalf("overlapping tags: %s", w.Body)
	}
	_, err := db.Exec(`UPDATE statements SET period_start='2019-11-01' WHERE id=1`)
	if err != nil {
		t.Fatal(err)
	}
	w = call("/api/intel/comparison?month=2020-01", srv.categoryComparison)
	if !strings.Contains(w.Body.String(), `"comparable":false`) || !strings.Contains(w.Body.String(), `"average":null`) {
		t.Fatalf("unknown reference: %s", w.Body)
	}
	for _, path := range []string{"/api/intel/nature?month=2020-13", "/api/intel/nature?month=2999-01"} {
		if w = call(path, srv.natureBreakdown); w.Code != 400 {
			t.Fatalf("bad month: %d %s", w.Code, w.Body)
		}
	}
	// SQL failure must not become successful zero totals or an empty state.
	if _, err = db.Exec(`ALTER TABLE transactions RENAME COLUMN amount TO unavailable_amount`); err != nil {
		t.Fatal(err)
	}
	if w = call("/api/intel/nature?month=2020-01", srv.natureBreakdown); w.Code != 500 {
		t.Fatalf("error masked: %d %s", w.Code, w.Body)
	}
}

func TestAnalysisSignalsPostgres(t *testing.T) {
	db := analysisTestDB(t)
	srv := &Server{DB: db}
	now := time.Date(2020, 2, 21, 0, 0, 0, 0, time.UTC)
	_, err := db.Exec(`INSERT INTO tags VALUES(4,1,'trend test');
 INSERT INTO transactions(id,statement_id,owner_id,tx_date,amount,domain_id,merchant) VALUES
 (100,1,1,'2019-10-10',-10,1,'TREND TEST'),(101,1,1,'2019-11-10',-20,1,'TREND TEST'),(102,1,1,'2019-12-10',-30,1,'TREND TEST'),(103,1,1,'2020-01-10',-100,1,'TREND TEST'),
 (104,1,1,'2018-02-10',-80,1,'ANNUAL TEST'),(105,1,1,'2019-02-10',-80,1,'ANNUAL TEST'),
 (106,1,1,'2020-02-10',-999,1,'FUTURE TEST');
 INSERT INTO transaction_tags VALUES(100,4),(101,4),(102,4),(103,4);UPDATE statements st SET closing_balance=opening_balance+COALESCE((SELECT SUM(t.amount) FROM transactions t WHERE t.statement_id=st.id AND t.owner_id=st.owner_id),0);`)
	if err != nil {
		t.Fatal(err)
	}
	signals, limits, err := srv.loadAnalysisSignals(context.Background(), 1, "2020-01", now)
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]bool{}
	for _, signal := range signals {
		kinds[signal["kind"].(string)] = true
	}
	for _, kind := range []string{"merchant_drift", "tag_surge", "seasonal"} {
		if !kinds[kind] {
			t.Fatalf("missing %s: %+v limits=%v", kind, signals, limits)
		}
	}
	raw, _ := json.Marshal(signals)
	if strings.Contains(string(raw), "SECRET") || strings.Contains(string(raw), "FUTURE TEST") {
		t.Fatal("signals outside owner/window")
	}
	_, err = db.Exec(`UPDATE statements SET period_start='2019-11-01' WHERE id=1`)
	if err != nil {
		t.Fatal(err)
	}
	signals, limits, err = srv.loadAnalysisSignals(context.Background(), 1, "2020-01", now)
	if err != nil {
		t.Fatal(err)
	}
	for _, signal := range signals {
		if signal["kind"] != "seasonal" {
			t.Fatalf("incomparable monthly totals: %+v", signals)
		}
	}
	if len(limits) < 2 {
		t.Fatal("missing explicit limitation")
	}
}
func TestAnalysisFolderConcentrationAndSourcesPostgres(t *testing.T) {
	db := analysisTestDB(t)
	srv := &Server{DB: db}
	_, err := db.Exec(`INSERT INTO transactions(id,statement_id,owner_id,tx_date,amount,domain_id) VALUES(100,1,1,'2020-01-09',-80,1),(101,1,1,'2020-01-09',-80,1);INSERT INTO transaction_tags VALUES(100,1),(101,1);`)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("GET", "/api/tag-folders/projet%20test?month=2020-01", nil)
	r.SetPathValue("name", "projet test")
	r = r.WithContext(context.WithValue(r.Context(), ctxUserKey, &authUser{ID: 1}))
	w := httptest.NewRecorder()
	srv.tagFolderDetail(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"total":190.03`) || !strings.Contains(w.Body.String(), `"count":4`) {
		t.Fatalf("detail: %d %s", w.Code, w.Body)
	}
	source, err := parseTransactionFilter(url.Values{"tag": {"projet test"}, "type": {"debit"}, "from": {"2020-01-09"}, "to": {"2020-01-31"}}, 1, true, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	var total string
	var count int
	err = db.QueryRow(`SELECT (-SUM(t.amount))::text,count(*) FROM transactions t WHERE `+source.where, source.args...).Scan(&total, &count)
	if err != nil || total != "190.03" || count != 4 {
		t.Fatalf("detail/source discrepancy %s %d %v", total, count, err)
	}
}

type analysisTestLLM struct {
	response string
	err      error
	prompt   string
	json     bool
}

func (c *analysisTestLLM) Name() string { return "TEST MODEL" }
func (c *analysisTestLLM) Complete(_ context.Context, system, user string, _ int, wantJSON bool) (string, error) {
	c.prompt = system + user
	c.json = wantJSON
	return c.response, c.err
}
func TestAnalysisNarrativeContractPostgres(t *testing.T) {
	db := analysisTestDB(t)
	srv := &Server{DB: db}
	client := &analysisTestLLM{response: `{"observation_ids":["flows"]}`}
	srv.ai.client = client
	call := func() *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "/api/intel/narrative?month=2020-01", nil)
		r = r.WithContext(context.WithValue(r.Context(), ctxUserKey, &authUser{ID: 1}))
		w := httptest.NewRecorder()
		srv.intelNarrative(w, r)
		return w
	}
	w := call()
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"selection_mode":"verified_observations_only"`) || !strings.Contains(w.Body.String(), `30.04`) || !client.json || strings.Contains(client.prompt, "SECRET") {
		t.Fatalf("narrative contract: %d %s", w.Code, w.Body)
	}
	client.response = `{"observation_ids":["flows"],"narrative":"FICTION TEST"}`
	w = call()
	if !strings.Contains(w.Body.String(), `"available":false`) || strings.Contains(w.Body.String(), "FICTION TEST") {
		t.Fatalf("invented response displayed: %s", w.Body)
	}
	client.err = fmt.Errorf("SENSITIVE PROVIDER BODY")
	w = call()
	if !strings.Contains(w.Body.String(), `"available":false`) || strings.Contains(w.Body.String(), "SENSITIVE") {
		t.Fatalf("provider failure: %s", w.Body)
	}
	srv.ai.client = nil
	w = call()
	if !strings.Contains(w.Body.String(), "déterministes") {
		t.Fatalf("optional AI unavailable: %s", w.Body)
	}
}
func TestAnalysisFolderConcentrationPostgres(t *testing.T) {
	db := analysisTestDB(t)
	srv := &Server{DB: db}
	_, err := db.Exec(`INSERT INTO tags VALUES(10,1,'separated test');INSERT INTO transactions(id,statement_id,owner_id,tx_date,amount,domain_id) VALUES
 (200,1,1,'2018-01-20',-100,1),(201,1,1,'2018-01-20',-100,1),(202,1,1,'2019-01-20',-100,1),(203,1,1,'2019-01-20',-100,1);
 INSERT INTO transaction_tags VALUES(200,10),(201,10),(202,10),(203,10);`)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("GET", "/api/tag-folders?month=2020-01", nil)
	r = r.WithContext(context.WithValue(r.Context(), ctxUserKey, &authUser{ID: 1}))
	w := httptest.NewRecorder()
	srv.tagFolders(w, r)
	var result struct {
		Others []tagFolder `json:"others"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &result) != nil {
		t.Fatalf("folders: %d %s", w.Code, w.Body)
	}
	found := false
	for _, f := range result.Others {
		if f.Name == "separated test" {
			found = true
			if f.IsProject || f.Total != "400.00" || f.Count != 4 {
				t.Fatalf("multiplied anchor dates: %+v", f)
			}
		}
	}
	if !found {
		t.Fatal("false project, repeated same-day anchors")
	}
}
func TestAnalysisCategoryAndMerchantSourcesPostgres(t *testing.T) {
	db := analysisTestDB(t)
	srv := &Server{DB: db}
	r := httptest.NewRequest("GET", "/api/category-series?categories=2&from=2020-01-01&to=2020-01-31", nil)
	r = r.WithContext(context.WithValue(r.Context(), ctxUserKey, &authUser{ID: 1}))
	w := httptest.NewRecorder()
	srv.categorySeries(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `10.01`) {
		t.Fatalf("leaf category filter: %d %s", w.Code, w.Body)
	}
	r = httptest.NewRequest("GET", "/api/category-series?categories=oops", nil).WithContext(r.Context())
	w = httptest.NewRecorder()
	srv.categorySeries(w, r)
	if w.Code != 400 {
		t.Fatal("invalid filter broadened series")
	}
	filter, err := parseTransactionFilter(url.Values{"merchant": {"CAFE TEST"}, "month": {"2020-01"}, "type": {"debit"}}, 1, true, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	var amount string
	var count int
	err = db.QueryRow(`SELECT (-SUM(t.amount))::text,count(*) FROM transactions t WHERE `+filter.where, filter.args...).Scan(&amount, &count)
	if err != nil || amount != "30.03" || count != 2 {
		t.Fatalf("merchant exact sources: %s %d %v", amount, count, err)
	}
}

func TestAnalysisMerchantsComparableWindowsPostgres(t *testing.T) {
	db := analysisTestDB(t)
	srv := &Server{DB: db}
	_, err := db.Exec(`INSERT INTO transactions(statement_id,owner_id,tx_date,amount,domain_id,merchant) VALUES(1,1,'2019-12-10',-10,1,'CAFE TEST');UPDATE statements SET closing_balance=closing_balance-10 WHERE id=1`)
	if err != nil {
		t.Fatal(err)
	}
	call := func() map[string]any {
		r := httptest.NewRequest("GET", "/api/intel/merchants?month=2020-01&months=1", nil)
		r = r.WithContext(context.WithValue(r.Context(), ctxUserKey, &authUser{ID: 1}))
		w := httptest.NewRecorder()
		srv.merchantsView(w, r)
		var result map[string]any
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &result) != nil {
			t.Fatalf("merchants %d %s", w.Code, w.Body)
		}
		return result
	}
	report := call()
	if report["comparable"] != true || report["reference_from"] != "2019-12-01" || report["reference_to"] != "2019-12-31" {
		t.Fatalf("windows: %+v", report)
	}
	for _, value := range report["merchants"].([]any) {
		m := value.(map[string]any)
		if m["name"] == "CAFE TEST" && (m["prev_total"] != float64(10) || m["trend_pct"] != 200.3) {
			t.Fatalf("exact trend %+v", m)
		}
	}
	_, err = db.Exec(`UPDATE statements SET period_start='2020-01-01' WHERE id=1`)
	if err != nil {
		t.Fatal(err)
	}
	report = call()
	if report["comparable"] != false {
		t.Fatal("incomparable windows compared")
	}
	for _, value := range report["merchants"].([]any) {
		if value.(map[string]any)["trend_pct"] != nil {
			t.Fatal("unknown trend not null")
		}
	}
	_, err = db.Exec(`UPDATE statements SET period_start='2019-10-01' WHERE id=1;UPDATE transactions SET statement_id=NULL WHERE id=1`)
	if err != nil {
		t.Fatal(err)
	}
	if call()["comparable"] != false {
		t.Fatal("unattributed sources certified")
	}
}
