package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func projectRequest(s *Server, method, path, body string, owner int64) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r = r.WithContext(context.WithValue(r.Context(), ctxUserKey, &authUser{ID: owner}))
	parts := strings.Split(strings.Split(path, "?")[0], "/")
	if len(parts) > 3 {
		r.SetPathValue("id", parts[3])
	}
	if len(parts) > 5 {
		r.SetPathValue("rule_id", parts[5])
	}
	w := httptest.NewRecorder()
	switch {
	case len(parts) == 3 && method == "GET":
		s.projects(w, r)
	case len(parts) == 3 || method == "PUT":
		s.saveProject(w, r)
	case len(parts) == 4:
		s.projectReportHTTP(w, r)
	case parts[4] == "rules":
		s.projectRules(w, r)
	case parts[4] == "decisions":
		s.decideProjectTransactions(w, r)
	case parts[4] == "suggestions":
		s.suggestProject(w, r)
	case parts[4] == "transactions":
		s.projectTransactions(w, r)
	case parts[4] == "history":
		s.projectHistoryHTTP(w, r)
	default:
		panic(path)
	}
	return w
}
func newTestProject(t *testing.T, s *Server, owner int64, budget string) int64 {
	t.Helper()
	w := projectRequest(s, "POST", "/api/projects", `{"name":"Voyage TEST","budget_cents":`+budget+`,"start_date":"2025-12-01","end_date":"2026-02-28"}`, owner)
	var out struct {
		ID int64 `json:"id"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &out) != nil {
		t.Fatalf("create: %d %s", w.Code, w.Body)
	}
	return out.ID
}
func testProjectReport(t *testing.T, s *Server, id int64) projectReport {
	t.Helper()
	w := projectRequest(s, "GET", fmt.Sprintf("/api/projects/%d?month=2026-02", id), "", 1)
	var out projectReport
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &out) != nil {
		_, cause := s.loadProjectReport(context.Background(), 1, id, "2026-02")
		t.Fatalf("report: %d %s (cause %v)", w.Code, w.Body, cause)
	}
	return out
}
func TestProjectsExactCostsDecisionsIsolationPostgres(t *testing.T) {
	db := reimbursementTestDB(t)
	s := &Server{DB: db}
	id := newTestProject(t, s, 1, "10000")
	other := newTestProject(t, s, 2, "null")
	path := fmt.Sprintf("/api/projects/%d", id)
	for _, endpoint := range []string{"", "/transactions", "/history", "/rules"} {
		w := projectRequest(s, "GET", path+endpoint, "", 2)
		if w.Code != 404 {
			t.Fatalf("foreign read %s: %d", endpoint, w.Code)
		}
	}
	for _, tc := range []struct{ method, suffix, body string }{{"PUT", "", `{"name":"HACK"}`}, {"POST", "/decisions", `{"transaction_ids":[12],"state":"confirmed"}`}, {"POST", "/rules", `{"kind":"merchant","match_text":"shop","from_date":"2025-12-01","to_date":"2026-02-28"}`}, {"POST", "/suggestions", `{}`}} {
		w := projectRequest(s, tc.method, path+tc.suffix, tc.body, 2)
		if w.Code != 404 {
			t.Fatalf("foreign mutation %s: %d %s", tc.suffix, w.Code, w.Body)
		}
	}
	if other == id {
		t.Fatal("unstable identity")
	}
	empty := testProjectReport(t, s, id)
	if empty.Project.Cost != 0 || empty.Project.Count != 0 || empty.Project.Percent == nil {
		t.Fatalf("empty: %+v", empty)
	}
	for _, tid := range []int{2, 10, 15, 17, 12} {
		w := projectRequest(s, "POST", path+"/decisions", fmt.Sprintf(`{"transaction_ids":[1,%d],"state":"confirmed"}`, tid), 1)
		if w.Code != 400 && w.Code != 404 {
			t.Fatalf("ineligible %d: %d %s", tid, w.Code, w.Body)
		}
		if testProjectReport(t, s, id).Project.Count != 0 {
			t.Fatal("partial batch persisted")
		}
	}
	body := `{"transaction_ids":[1,3,6,8,1],"state":"confirmed"}`
	for range 2 {
		w := projectRequest(s, "POST", path+"/decisions", body, 1)
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body)
		}
	}
	r := testProjectReport(t, s, id)
	if r.Project.Cost != 19001 || r.Project.Count != 4 || *r.Project.Remaining != -9001 || *r.Project.Overrun != 9001 || *r.Project.Percent != "190.01" {
		t.Fatalf("exact cents %+v", r.Project)
	}
	for _, partition := range [][]projectSlice{r.Categories, r.Merchants, r.Months} {
		var total int64
		for _, row := range partition {
			total += row.Cents
		}
		if total != 19001 {
			t.Fatal("unreconciled partition", partition)
		}
	}
	if len(r.Observations) != 2 || r.Observations[0].Values["cost_cents"] != float64(19001) {
		t.Fatalf("observations %+v", r.Observations)
	}
	var n int
	if e := db.QueryRow(`SELECT count(*) FROM project_history WHERE project_id=$1 AND event='decision'`, id).Scan(&n); e != nil || n != 4 {
		t.Fatal("idempotent history", n, e)
	}
	second := newTestProject(t, s, 1, "null")
	w := projectRequest(s, "POST", fmt.Sprintf("/api/projects/%d/decisions", second), body, 1)
	if w.Code != 200 {
		t.Fatal(w.Body)
	}
	if testProjectReport(t, s, second).Project.Cost != 19001 {
		t.Fatal("multi-project cost")
	}
	w = projectRequest(s, "GET", "/api/projects?month=2026-02", "", 1)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"additive":false`) {
		t.Fatal("list", w.Code, w.Body)
	}
	w = projectRequest(s, "POST", path+"/decisions", `{"transaction_ids":[1],"state":"rejected"}`, 1)
	if w.Code != 200 || testProjectReport(t, s, id).Project.Cost != 13001 {
		t.Fatal("withdraw failed", w.Body)
	}
	w = projectRequest(s, "POST", path+"/decisions", `{"transaction_ids":[1],"state":"confirmed"}`, 1)
	if w.Code != 200 || testProjectReport(t, s, id).Project.Cost != 19001 {
		t.Fatal("reversible decision", w.Body)
	}
	// A refund credit exists in the fixture; no net cost is inferred.
	w = projectRequest(s, "PUT", path, `{"name":"Voyage TEST","budget_cents":0,"start_date":"2025-12-01","end_date":"2026-02-28","status":"completed"}`, 1)
	if w.Code != 200 {
		t.Fatal(w.Body)
	}
	r = testProjectReport(t, s, id)
	if r.Project.Percent != nil || *r.Project.Overrun != 19001 {
		t.Fatal("zero budget", r.Project)
	}
	w = projectRequest(s, "PUT", path, `{"name":"Voyage TEST","budget_cents":null,"start_date":"2025-12-01","end_date":"2026-02-28"}`, 1)
	if w.Code != 200 {
		t.Fatal(w.Body)
	}
	r = testProjectReport(t, s, id)
	if r.Project.Remaining != nil || r.Project.Percent != nil {
		t.Fatal("absent budget")
	}
	// Existing confirmations survive reclassification but no longer inflate ordinary cost.
	if _, e := db.Exec(`UPDATE transactions SET label_kind='savings' WHERE id=6`); e != nil {
		t.Fatal(e)
	}
	r = testProjectReport(t, s, id)
	if r.Project.Cost != 18001 || r.Excluded != 1 {
		t.Fatal("reclassification", r)
	}
	// Unknown documentary coverage stays provisional even when manually completed.
	if r.Coverage != "provisional" {
		t.Fatal("unsupported coverage", r.Coverage)
	}
	if _, e := db.Exec(`ALTER TABLE project_history ADD CONSTRAINT fail_test CHECK(event<>'decision') NOT VALID`); e != nil {
		t.Fatal(e)
	}
	w = projectRequest(s, "POST", path+"/decisions", `{"transaction_ids":[1],"state":"rejected"}`, 1)
	if w.Code != 500 || testProjectReport(t, s, id).Project.Cost != 18001 {
		t.Fatal("history failure not atomic", w.Code, w.Body)
	}
	if _, e := db.Exec(`DROP TABLE project_transactions`); e != nil {
		t.Fatal(e)
	}
	w = projectRequest(s, "GET", path, "", 1)
	if w.Code != 500 {
		t.Fatal("SQL failure masquerades as zero", w.Code, w.Body)
	}
}

func TestProjectSuggestionsRulesAndImportReplayPostgres(t *testing.T) {
	db := reimbursementTestDB(t)
	s := &Server{DB: db}
	id := newTestProject(t, s, 1, "null")
	path := fmt.Sprintf("/api/projects/%d", id)
	if _, e := db.Exec(`INSERT INTO tags(id,owner_id,name) VALUES(4001,1,'Voyage TEST'),(4002,1,'Second TEST'),(4003,2,'SECRET'); INSERT INTO transaction_tags(transaction_id,tag_id) VALUES(6,4001),(6,4002),(8,4001),(12,4003);`); e != nil {
		t.Fatal(e)
	}
	for _, rule := range []string{`{"kind":"tag","tag_id":4001,"from_date":"2025-12-01","to_date":"2026-02-28"}`, `{"kind":"tag","tag_id":4002,"from_date":"2025-12-01","to_date":"2026-02-28"}`, `{"kind":"merchant","match_text":"shop","from_date":"2025-12-01","to_date":"2026-02-28"}`} {
		for range 2 {
			w := projectRequest(s, "POST", path+"/rules", rule, 1)
			if w.Code != 200 {
				t.Fatal(w.Code, w.Body)
			}
		}
	}
	for _, rule := range []string{`{"kind":"tag","tag_id":4003,"from_date":"2025-12-01","to_date":"2026-02-28"}`, `{"kind":"merchant","match_text":"shop"}`, `{"kind":"merchant","match_text":"sh","from_date":"2025-12-01","to_date":"2026-02-28"}`, `{"kind":"category","domain_id":9201,"from_date":"2025-12-01","to_date":"2026-02-28"}`} {
		w := projectRequest(s, "POST", path+"/rules", rule, 1)
		if w.Code != 400 {
			t.Fatal("bad rule", w.Code, w.Body)
		}
	}
	for range 2 {
		if _, e := s.refreshProjectSuggestions(context.Background(), 1, 0); e != nil {
			t.Fatal(e)
		}
	}
	r := testProjectReport(t, s, id)
	if r.Project.Cost != 0 || r.Project.Suggestions != 3 {
		t.Fatalf("suggestions counted as cost or duplicated: %+v", r.Project)
	}
	w := projectRequest(s, "GET", path+"/transactions?state=suggested&month=2026-02", "", 1)
	if w.Code != 200 || strings.Contains(w.Body.String(), "SECRET") || !strings.Contains(w.Body.String(), `"kind":"tag"`) {
		t.Fatal("candidate evidence", w.Code, w.Body)
	}
	w = projectRequest(s, "POST", path+"/decisions", `{"transaction_ids":[6,8],"state":"confirmed"}`, 1)
	if w.Code != 200 {
		t.Fatal(w.Body)
	}
	w = projectRequest(s, "POST", path+"/decisions", `{"transaction_ids":[14],"state":"rejected"}`, 1)
	if w.Code != 200 {
		t.Fatal(w.Body)
	}
	for range 2 {
		if _, e := s.refreshProjectSuggestions(context.Background(), 1, 0); e != nil {
			t.Fatal(e)
		}
	}
	r = testProjectReport(t, s, id)
	if r.Project.Cost != 1001 || r.Project.Suggestions != 0 || len(r.Tags) != 2 || r.Tags[0].Cents != 1001 {
		t.Fatalf("decisions/tag overlap %+v", r)
	}
	// Simulate a committed import: only its persisted row is discovered, once.
	if _, e := db.Exec(`INSERT INTO transactions(id,owner_id,statement_id,tx_date,label,merchant,amount,hash) VALUES(1001,1,101,'2025-12-21','IMPORTED TEST','shop',-12.34,'project-import-test')`); e != nil {
		t.Fatal(e)
	}
	n, e := s.refreshProjectSuggestions(context.Background(), 1, 0)
	if e != nil || n != 1 {
		t.Fatal("new persisted import", n, e)
	}
	n, e = s.refreshProjectSuggestions(context.Background(), 1, 0)
	if e != nil || n != 0 {
		t.Fatal("import replay", n, e)
	}
	var count int
	if e = db.QueryRow(`SELECT count(*) FROM project_transactions WHERE project_id=$1 AND transaction_id=1001`, id).Scan(&count); e != nil || count != 1 {
		t.Fatal(count, e)
	}
	// Composite FKs defend direct SQL too.
	if _, e = db.Exec(`INSERT INTO project_transactions(owner_id,project_id,transaction_id,state,reason) VALUES(1,$1,12,'confirmed','{}')`, id); e == nil {
		t.Fatal("foreign transaction FK absent")
	}
	if _, e = db.Exec(`INSERT INTO project_transactions(owner_id,project_id,transaction_id,state,reason) VALUES(2,$1,12,'confirmed','{}')`, id); e == nil {
		t.Fatal("foreign project FK absent")
	}
	if _, e = db.Exec(`UPDATE financial_projects SET status='archived' WHERE id=$1`, id); e != nil {
		t.Fatal(e)
	}
	w = projectRequest(s, "POST", path+"/decisions", `{"transaction_ids":[6],"state":"rejected"}`, 1)
	if w.Code != 409 {
		t.Fatal("archive mutable")
	}
}

func TestProjectInputValidation(t *testing.T) {
	for _, body := range []string{`{"name":""}`, `{"name":"TEST","budget_cents":1.2}`, `{"name":"TEST","budget_cents":-1}`, `{"name":"TEST","start_date":"2026-02-30"}`, `{"name":"TEST","start_date":"2026-02-28","end_date":"2026-02-01"}`, `{"name":"TEST","status":"unknown"}`, `{"name":"TEST","extra":1}`, `{"name":"TEST"} {}`} {
		w := projectRequest(&Server{}, "POST", "/api/projects", body, 1)
		if w.Code != 400 {
			t.Fatal(body, w.Code)
		}
	}
}

func TestProjectsSourcesDeletionAssistantAndExportPostgres(t *testing.T) {
	db := reimbursementTestDB(t)
	s := &Server{DB: db}
	id := newTestProject(t, s, 1, "10000")
	path := fmt.Sprintf("/api/projects/%d", id)
	w := projectRequest(s, "POST", path+"/decisions", `{"transaction_ids":[1],"state":"confirmed"}`, 1)
	if w.Code != 200 {
		t.Fatal(w.Body)
	}
	sourceDelete := func() *httptest.ResponseRecorder {
		r := httptest.NewRequest("DELETE", "/api/statements/101", nil)
		r.SetPathValue("id", "101")
		r = r.WithContext(context.WithValue(r.Context(), ctxUserKey, &authUser{ID: 1}))
		w := httptest.NewRecorder()
		s.deleteStatement(w, r)
		return w
	}
	if w = sourceDelete(); w.Code != 409 || !strings.Contains(w.Body.String(), "projets") {
		t.Fatal("confirmed statement deletion", w.Code, w.Body)
	}
	result := toolProjectAnalysis(context.Background(), db, 1, map[string]any{"project_id": fmt.Sprint(id), "month": "2026-02"})
	if result.Error != "" || !strings.Contains(result.Content, `"cost_cents":6000`) {
		t.Fatal("assistant exact source", result)
	}
	if result = toolProjectAnalysis(context.Background(), db, 2, map[string]any{"project_id": id}); result.Error == "" || result.Content != "" {
		t.Fatal("foreign assistant", result)
	}
	r := httptest.NewRequest("GET", "/api/export", nil)
	r = r.WithContext(context.WithValue(r.Context(), ctxUserKey, &authUser{ID: 1}))
	w = httptest.NewRecorder()
	s.exportAll(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"financial_projects"`) || strings.Contains(w.Body.String(), "SECRET") {
		t.Fatal("owned export", w.Code, w.Body)
	}
	w = projectRequest(s, "POST", path+"/decisions", `{"transaction_ids":[1],"state":"rejected"}`, 1)
	if w.Code != 200 {
		t.Fatal(w.Body)
	}
	if w = sourceDelete(); w.Code != 204 {
		t.Fatal("withdrawn statement deletion", w.Code, w.Body)
	}
	if testProjectReport(t, s, id).Project.Cost != 0 {
		t.Fatal("deleted source retained in cost")
	}
	w = projectRequest(s, "GET", path+"/history", "", 1)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"amount":"-60.00"`) {
		t.Fatal("financial source history lost", w.Code, w.Body)
	}
}

func TestProjectsCategoryLabelRulesBoundsAndReadOnlyPostgres(t *testing.T) {
	db := reimbursementTestDB(t)
	s := &Server{DB: db}
	id := newTestProject(t, s, 1, "null")
	path := fmt.Sprintf("/api/projects/%d", id)
	for _, rule := range []string{`{"kind":"category","domain_id":9001,"from_date":"2025-12-10","to_date":"2025-12-11"}`, `{"kind":"label","match_text":"PENNY EXPENSE","from_date":"2026-02-01","to_date":"2026-02-28"}`} {
		w := projectRequest(s, "POST", path+"/rules", rule, 1)
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body)
		}
	}
	// The existing durable worker path handles project suggestions before intelligence.
	if e := s.runIntelligenceJob(context.Background(), 1); e != nil {
		t.Fatal("shared worker", e)
	}
	report := testProjectReport(t, s, id)
	if report.Project.Suggestions != 3 || report.Project.Cost != 0 {
		t.Fatal("bounded deterministic rules", report.Project)
	}
	var before, after int
	if e := db.QueryRow(`SELECT count(*) FROM project_history`).Scan(&before); e != nil {
		t.Fatal(e)
	}
	for _, suffix := range []string{"?month=2026-02", "/rules", "/transactions?state=suggested&month=2026-02", "/history"} {
		w := projectRequest(s, "GET", path+suffix, "", 1)
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body)
		}
	}
	if e := db.QueryRow(`SELECT count(*) FROM project_history`).Scan(&after); e != nil || before != after {
		t.Fatal("GET mutated state", before, after, e)
	}
	for _, suffix := range []string{"?month=invalid", "?month=9999-12", "/transactions?state=unknown", "/transactions?offset=-1"} {
		w := projectRequest(s, "GET", path+suffix, "", 1)
		if w.Code != 400 {
			t.Fatal("bad filter broadened", suffix, w.Code)
		}
	}
	// Read-time period excludes later confirmations without changing membership.
	w := projectRequest(s, "POST", path+"/decisions", `{"transaction_ids":[1,8],"state":"confirmed"}`, 1)
	if w.Code != 200 {
		t.Fatal(w.Body)
	}
	w = projectRequest(s, "GET", path+"?month=2025-12", "", 1)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"cost_cents":6000`) {
		t.Fatal("period scope", w.Code, w.Body)
	}
	if _, e := db.Exec(`UPDATE financial_projects SET start_date='2026-02-01',end_date='2026-02-28' WHERE id=$1`, id); e != nil {
		t.Fatal(e)
	}
	report = testProjectReport(t, s, id)
	if report.Project.Cost != 1 || report.Excluded != 1 {
		t.Fatal("project date scope", report)
	}
}

func TestProjectCoverageUsesDocumentsNotManualStatusPostgres(t *testing.T) {
	db := reimbursementTestDB(t)
	s := &Server{DB: db}
	id := newTestProject(t, s, 1, "null")
	if _, e := db.Exec(`UPDATE statements SET opening_balance=773,closing_balance=1005.01 WHERE id=102;UPDATE statements SET opening_balance=1005.01,closing_balance=1000 WHERE id=103`); e != nil {
		t.Fatal(e)
	}
	report := testProjectReport(t, s, id)
	if report.Coverage != "documented" {
		t.Fatal("documented period not recognized", report.CoverageReason)
	}
	if _, e := db.Exec(`WITH changed AS (UPDATE financial_projects SET status='completed' WHERE id=$1) UPDATE statements SET validated=false WHERE id=103`, id); e != nil {
		t.Fatal(e)
	}
	report = testProjectReport(t, s, id)
	if report.Coverage != "provisional" || report.Project.Status != "completed" {
		t.Fatal("manual completion certified coverage", report)
	}
}

func TestProjectConcurrentConfirmationAndIdempotentMetadataPostgres(t *testing.T) {
	db := reimbursementTestDB(t)
	s := &Server{DB: db}
	id := newTestProject(t, s, 1, "10000")
	path := fmt.Sprintf("/api/projects/%d", id)
	body := `{"name":"Voyage TEST","budget_cents":10000,"start_date":"2025-12-01","end_date":"2026-02-28","status":"active"}`
	for range 2 {
		w := projectRequest(s, "PUT", path, body, 1)
		if w.Code != 200 {
			t.Fatal(w.Body)
		}
	}
	rule := `{"kind":"merchant","match_text":"shop","from_date":"2025-12-01","to_date":"2026-02-28"}`
	var ruleID int64
	for range 2 {
		w := projectRequest(s, "POST", path+"/rules", rule, 1)
		if w.Code != 200 {
			t.Fatal(w.Body)
		}
		var out projectRule
		if e := json.Unmarshal(w.Body.Bytes(), &out); e != nil {
			t.Fatal(e)
		}
		ruleID = out.ID
	}
	var wait sync.WaitGroup
	codes := make(chan int, 2)
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			w := projectRequest(s, "POST", path+"/decisions", `{"transaction_ids":[1],"state":"confirmed"}`, 1)
			codes <- w.Code
		}()
	}
	wait.Wait()
	close(codes)
	for code := range codes {
		if code != 200 {
			t.Fatal("concurrent decision", code)
		}
	}
	var count int
	if e := db.QueryRow(`SELECT count(*) FROM project_history WHERE project_id=$1`, id).Scan(&count); e != nil || count != 3 {
		t.Fatal("no-op history duplicated", count, e)
	}
	for range 2 {
		w := projectRequest(s, "DELETE", fmt.Sprintf("%s/rules/%d", path, ruleID), "", 1)
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body)
		}
	}
	if e := db.QueryRow(`SELECT count(*) FROM project_history WHERE project_id=$1`, id).Scan(&count); e != nil || count != 4 {
		t.Fatal("repeated rule removal", count, e)
	}
	if testProjectReport(t, s, id).Project.Cost != 6000 {
		t.Fatal("rule removal changed explicit confirmation")
	}
}

func TestProjectUncategorizedObservationSourcesPostgres(t *testing.T) {
	db := reimbursementTestDB(t)
	s := &Server{DB: db}
	id := newTestProject(t, s, 1, "null")
	path := fmt.Sprintf("/api/projects/%d", id)
	if _, e := db.Exec(`UPDATE transactions SET domain_id=NULL WHERE id IN(1,3,6)`); e != nil {
		t.Fatal(e)
	}
	w := projectRequest(s, "POST", path+"/decisions", `{"transaction_ids":[1,3,6],"state":"confirmed"}`, 1)
	if w.Code != 200 {
		t.Fatal(w.Body)
	}
	report := testProjectReport(t, s, id)
	if report.Categories[0].Name != "Sans catégorie" || len(report.Observations) != 1 {
		t.Fatal("uncategorized proof missing", report)
	}
	w = projectRequest(s, "GET", path+"/transactions?month=2026-02&category=Sans+cat%C3%A9gorie", "", 1)
	var sources struct {
		Transactions []map[string]any `json:"transactions"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &sources) != nil || len(sources.Transactions) != 3 {
		t.Fatal("uncategorized sources lost", w.Code, w.Body)
	}
	if result := toolProjectAnalysis(context.Background(), db, 1, map[string]any{"project_id": id, "month": 123}); result.Error == "" {
		t.Fatal("invalid period broadened assistant report")
	}
}

func TestProjectOwnerDeletionStillCascadesPostgres(t *testing.T) {
	db := reimbursementTestDB(t)
	s := &Server{DB: db}
	id := newTestProject(t, s, 2, "null")
	w := projectRequest(s, "POST", fmt.Sprintf("/api/projects/%d/decisions", id), `{"transaction_ids":[12],"state":"confirmed"}`, 2)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	if _, e := db.Exec(`DELETE FROM users WHERE id=2`); e != nil {
		t.Fatal("existing owner deletion blocked", e)
	}
	var count int
	if e := db.QueryRow(`SELECT count(*) FROM financial_projects WHERE owner_id=2`).Scan(&count); e != nil || count != 0 {
		t.Fatal("orphaned project", count, e)
	}
	if e := db.QueryRow(`SELECT count(*) FROM users WHERE id=1`).Scan(&count); e != nil || count != 1 {
		t.Fatal("other owner affected", count, e)
	}
}
