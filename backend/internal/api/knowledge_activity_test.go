package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	database "themis/internal/db"
	"themis/internal/memory"
)

func knowledgeFixture(group string) memory.Result {
	return memory.Result{ID: "edge-" + group, GroupID: group, Source: "a-" + group, Target: "b-" + group, Type: "RECURRENT", Fact: "PRIVATE SYNTHETIC OWNER " + group, EpisodeIDs: []string{"ep-" + group}, Sources: []memory.Source{{ID: "ep-" + group, GroupID: group, Name: "Import TEST", Source: "statement:101"}}}
}
func knowledgeGraphFixture(group string) memory.Graph {
	n, e, p := int64(2), int64(1), int64(1)
	return memory.Graph{GroupID: group, NodeCount: &n, EdgeCount: &e, EpisodeCount: &p, Nodes: []memory.Node{{ID: "a-" + group, GroupID: group, Name: "Merchant OWNER " + group}, {ID: "b-" + group, GroupID: group, Name: "Habit TEST"}}, Edges: []memory.Result{knowledgeFixture(group)}, Sources: []memory.Source{{ID: "ep-" + group, GroupID: group, Name: "Import TEST", Source: "statement:101"}}}
}
func knowledgeRequest(s *Server, owner int64, method, path, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r = r.WithContext(context.WithValue(r.Context(), ctxUserKey, &authUser{ID: owner, Role: "user"}))
	w := httptest.NewRecorder()
	if method == "GET" {
		s.knowledgeOverview(w, r)
	} else {
		s.knowledgeSearch(w, r)
	}
	return w
}
func TestKnowledgeActivityIsolationFailuresAndContextPostgres(t *testing.T) {
	db := reimbursementTestDB(t)
	var failed atomic.Bool
	var empty atomic.Bool
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if failed.Load() {
			w.WriteHeader(500)
			w.Write([]byte(`{"detail":{"code":"model_missing","secret":"SECRET PROVIDER"}}`))
			return
		}
		if strings.HasPrefix(r.URL.Path, "/kg/inspect/") {
			group := strings.TrimPrefix(r.URL.Path, "/kg/inspect/")
			json.NewEncoder(w).Encode(knowledgeGraphFixture(group))
			return
		}
		var input struct {
			Group  string `json:"group_id"`
			Source string `json:"source_ref"`
		}
		json.NewDecoder(r.Body).Decode(&input)
		if r.URL.Path == "/kg/learn" {
			if input.Source != "statement:101" && input.Source != "manual" {
				t.Error(input.Source)
			}
			w.Write([]byte(`{"entities_added":2,"edges_added":1}`))
			return
		}
		facts := []memory.Result{}
		if !empty.Load() {
			facts = append(facts, knowledgeFixture(input.Group))
		}
		json.NewEncoder(w).Encode(map[string]any{"results": facts, "count": len(facts)})
	}))
	defer upstream.Close()
	s := &Server{DB: db, Memory: memory.New(memory.Config{Enabled: true, URL: upstream.URL, RequestTimeout: time.Second})}

	out, e := s.learnKnowledge(context.Background(), 1, 101, "statement", []string{"SYNTHETIC INPUT"})
	if e != nil || out.Status != "success" || *out.Entities != 2 {
		t.Fatal(out, e)
	}
	w := knowledgeRequest(s, 1, "POST", "/api/knowledge/search", `{"query":"OWNER TEST"}`)
	var trace knowledgeTrace
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &trace) != nil || !trace.AuditAvailable || len(trace.Facts) != 1 {
		t.Fatal(w.Code, w.Body)
	}
	firstID := trace.ActivityID
	// Body parameters can never select another owner.
	if w = knowledgeRequest(s, 1, "POST", "/api/knowledge/search", `{"query":"TEST","owner_id":2}`); w.Code != 400 {
		t.Fatal("owner override", w.Code)
	}
	w = knowledgeRequest(s, 2, "GET", "/api/knowledge/overview", "")
	if w.Code != 200 || strings.Contains(w.Body.String(), "PRIVATE SYNTHETIC OWNER 1") || strings.Contains(w.Body.String(), "Synthetic expenses TEST") {
		t.Fatal("foreign graph or journal leaked", w.Code, w.Body)
	}
	var overview struct {
		Activity []knowledgeActivity `json:"activity"`
		Last     *knowledgeActivity  `json:"last_learning"`
	}
	json.Unmarshal(w.Body.Bytes(), &overview)
	if len(overview.Activity) != 0 || overview.Last != nil {
		t.Fatal("foreign history leaked")
	}
	failed.Store(true)
	w = knowledgeRequest(s, 1, "POST", "/api/knowledge/search", `{"query":"TEST"}`)
	json.Unmarshal(w.Body.Bytes(), &trace)
	if trace.Status != "error" || trace.Code != "model_missing" || len(trace.Facts) != 0 || strings.Contains(w.Body.String(), "SECRET PROVIDER") {
		t.Fatal("error hidden", w.Body)
	}
	var count *int
	if e = db.QueryRow(`SELECT result_count FROM knowledge_activity WHERE id=$1`, trace.ActivityID).Scan(&count); e != nil || count != nil {
		t.Fatal("failed lookup replaced with zero", count, e)
	}
	w = knowledgeRequest(s, 1, "GET", "/api/knowledge/overview", "")
	var unavailable struct {
		Graph    *memory.Graph       `json:"graph"`
		Activity []knowledgeActivity `json:"activity"`
	}
	json.Unmarshal(w.Body.Bytes(), &unavailable)
	if w.Code != 200 || unavailable.Graph != nil || len(unavailable.Activity) == 0 {
		t.Fatal("outage erased journal", w.Code, w.Body)
	}
	failed.Store(false)
	empty.Store(true)
	w = knowledgeRequest(s, 1, "POST", "/api/knowledge/search", `{"query":"TEST"}`)
	json.Unmarshal(w.Body.Bytes(), &trace)
	if trace.Status != "empty" {
		t.Fatal(w.Body)
	}
	if e = db.QueryRow(`SELECT result_count FROM knowledge_activity WHERE id=$1`, trace.ActivityID).Scan(&count); e != nil || count == nil || *count != 0 {
		t.Fatal("genuine zero not preserved", count, e)
	}
	empty.Store(false)
	model := &knowledgePromptModel{}
	s.ai.client = model
	result, e := s.orchestrate(context.Background(), db, 1, nil, "SYNTHETIC QUESTION")
	if e != nil || result.Memory == nil || !result.Memory.Included || !result.Memory.AuditAvailable || !strings.Contains(model.system, "PRIVATE SYNTHETIC OWNER 1") || strings.Contains(model.system, "OWNER 2") {
		t.Fatal("context trace mismatch", result, e, model.system)
	}
	var included bool
	if e = db.QueryRow(`SELECT context_included FROM knowledge_activity WHERE id=$1 AND owner_id=1`, result.Memory.ActivityID).Scan(&included); e != nil || !included {
		t.Fatal("context not journalled", e)
	}
	w = knowledgeRequest(s, 1, "GET", fmt.Sprintf("/api/knowledge/overview?before_id=%d", firstID), "")
	json.Unmarshal(w.Body.Bytes(), &overview)
	for _, event := range overview.Activity {
		if event.ID >= firstID {
			t.Fatal("cursor ignored")
		}
	}
	if overview.Last == nil || overview.Last.Kind != "learn" {
		t.Fatal("last learning depends on journal page", w.Body)
	}
	if _, e = s.beginKnowledge(1, "learn", "statement", 201, 0); e == nil {
		t.Fatal("foreign statement association")
	}
	if _, e = db.Exec(`INSERT INTO knowledge_activity(owner_id,kind,source,statement_id,status) VALUES(1,'learn','statement',201,'running')`); e == nil {
		t.Fatal("database accepted foreign source")
	}

}

type knowledgePromptModel struct{ system string }

func (m *knowledgePromptModel) Name() string { return "SYNTHETIC TEST" }
func (m *knowledgePromptModel) Complete(_ context.Context, system, _ string, _ int, _ bool) (string, error) {
	m.system = system
	return `{"answer":"TEST ANSWER","status":"ok"}`, nil
}
func TestKnowledgeMigrationAuthenticationAndRollbackPostgres(t *testing.T) {
	db := reimbursementTestDB(t)
	s := &Server{DB: db}
	for _, route := range []struct{ method, path string }{{"GET", "/api/knowledge/overview"}, {"POST", "/api/knowledge/search"}, {"POST", "/api/knowledge/learn"}} {
		w := httptest.NewRecorder()
		s.Router().ServeHTTP(w, httptest.NewRequest(route.method, route.path, nil))
		if w.Code != 401 {
			t.Fatal(route.path, w.Code)
		}
	}
	rollback, e := os.ReadFile("../db/rollbacks/033_knowledge_activity.sql")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(string(rollback)); e != nil {
		t.Fatal(e)
	}
	for range 2 {
		if e = database.Migrate(db); e != nil {
			t.Fatal(e)
		}
	}
	s.skipKnowledgeImport(1, 101)
	if _, e = db.Exec(string(rollback)); e == nil {
		t.Fatal("rollback lost operational evidence")
	}
	var status string
	if e = db.QueryRow(`SELECT status FROM knowledge_activity WHERE owner_id=1`).Scan(&status); e != nil || status != "skipped" {
		t.Fatal(status, e)
	}
	var amount string
	if e = db.QueryRow(`SELECT amount::text FROM transactions WHERE id=1`).Scan(&amount); e != nil || amount != "-60.00" && amount != "-60" {
		t.Fatal("bank data changed", amount, e)
	}
}
