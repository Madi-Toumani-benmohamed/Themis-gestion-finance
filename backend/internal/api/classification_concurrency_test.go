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

func classificationPatch(s *Server, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("PATCH", "/api/transactions/1", strings.NewReader(body))
	r.SetPathValue("id", "1")
	r = r.WithContext(context.WithValue(r.Context(), ctxUserKey, &authUser{ID: 1}))
	w := httptest.NewRecorder()
	s.updateTransaction(w, r)
	return w
}
func TestClassificationConcurrentSnapshotPreconditionPostgres(t *testing.T) {
	db := reimbursementTestDB(t)
	s := &Server{DB: db}
	rows, e := agentSnapshots(context.Background(), db, 1, []int64{1}, false)
	if e != nil {
		t.Fatal(e)
	}
	old := rows[0]
	version := classificationVersion(1, old.ID, old.Domain, old.Source, old.Tags)
	var wg sync.WaitGroup
	codes := make(chan int, 2)
	for _, tag := range []string{"alpha", "beta"} {
		wg.Add(1)
		go func(tag string) {
			defer wg.Done()
			body, _ := json.Marshal(map[string]any{"tags": []string{tag}, "expected_classification_version": version})
			codes <- classificationPatch(s, string(body)).Code
		}(tag)
	}
	wg.Wait()
	close(codes)
	success, conflict := 0, 0
	for code := range codes {
		if code == 204 {
			success++
		} else if code == 409 {
			conflict++
		} else {
			t.Fatal(code)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatal(success, conflict)
	}
	current, e := agentSnapshots(context.Background(), db, 1, []int64{1}, false)
	if e != nil {
		t.Fatal(e)
	}
	if current[0].Amount != old.Amount || current[0].Date != old.Date || current[0].Label != old.Label || len(current[0].Tags) != 1 {
		t.Fatal(current)
	}
	if classificationPatch(s, fmt.Sprintf(`{"domain_id":null,"expected_classification_version":%q}`, version)).Code != 409 {
		t.Fatal("stale domain update accepted")
	}
	fresh := classificationVersion(1, current[0].ID, current[0].Domain, current[0].Source, current[0].Tags)
	if classificationPatch(s, fmt.Sprintf(`{"domain_id":null,"expected_classification_version":%q}`, fresh)).Code != 204 {
		t.Fatal("explicit manual NULL rejected")
	}
	if _, e = db.Exec(`INSERT INTO rules(owner_id,pattern,domain_id,tags) VALUES(1,'TEST RESTAURANT',9001,'resto')`); e != nil {
		t.Fatal(e)
	}
	if _, e = s.applyAllRules(1); e != nil {
		t.Fatal(e)
	}
	var protected bool
	if e = db.QueryRow(`SELECT domain_id IS NULL AND classification_source='manual' FROM transactions WHERE id=1`).Scan(&protected); e != nil || !protected {
		t.Fatal("manual NULL not persisted", e)
	}
}

func TestManualTagRemovalPersistenceLimitIsExplicitPostgres(t *testing.T) {
	db := reimbursementTestDB(t)
	s := &Server{DB: db}
	if _, e := db.Exec(`INSERT INTO rules(owner_id,pattern,domain_id,tags) VALUES(1,'TEST RESTAURANT',9001,'resto')`); e != nil {
		t.Fatal(e)
	}
	if _, e := s.applyAllRules(1); e != nil {
		t.Fatal(e)
	}
	if w := classificationPatch(s, `{"remove_tags":["resto"]}`); w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	var source string
	if e := db.QueryRow(`SELECT classification_source FROM transactions WHERE id=1`).Scan(&source); e != nil || source == "manual" {
		t.Fatal("tag-only edit must not invent a manual domain lock", e)
	}
	// No tombstone exists: the removed tag can still be proposed/applied on an
	// explicit global reapplication. This is an unmet sprint-1B requirement.
	if _, e := s.applyAllRules(1); e != nil {
		t.Fatal(e)
	}
	var n int
	if e := db.QueryRow(`SELECT count(*) FROM transaction_tags tt JOIN tags tg ON tg.id=tt.tag_id WHERE tt.transaction_id=1 AND tg.name='resto'`).Scan(&n); e != nil || n != 1 {
		t.Fatal("persistence contract changed; review protection design", e)
	}
}

func TestClassificationApplicationRollsBackDomainOnTagFailurePostgres(t *testing.T) {
	db := reimbursementTestDB(t)
	s := &Server{DB: db}
	if _, e := db.Exec(`UPDATE transactions SET domain_id=NULL,classification_source='unknown' WHERE id=1;
ALTER TABLE tags ADD CONSTRAINT synthetic_tag_failure CHECK(name<>'reject-synthetic-tag');
INSERT INTO rules(owner_id,pattern,domain_id,tags) VALUES(1,'TEST RESTAURANT',9001,'reject-synthetic-tag');`); e != nil {
		t.Fatal(e)
	}
	if _, e := s.applyAllRules(1); e == nil {
		t.Fatal("tag insertion error ignored")
	}
	var unchanged bool
	if e := db.QueryRow(`SELECT domain_id IS NULL AND classification_source='unknown' AND NOT EXISTS(SELECT 1 FROM transaction_tags WHERE transaction_id=1) FROM transactions WHERE id=1`).Scan(&unchanged); e != nil || !unchanged {
		t.Fatal("partial classification committed", e)
	}
}
