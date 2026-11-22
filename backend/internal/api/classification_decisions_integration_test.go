package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"themis/internal/ai"
	"themis/internal/classification"
)

func decisionState(t *testing.T, db *sql.DB) classification.State {
	t.Helper()
	s, e := classification.Read(context.Background(), db, 1, 1)
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func decisionRequest(s *Server, body, operation string, owner int64) *httptest.ResponseRecorder {
	r := httptest.NewRequest("PATCH", "/api/transactions/1", strings.NewReader(body))
	r.SetPathValue("id", "1")
	r.Header.Set("Idempotency-Key", operation)
	r = r.WithContext(context.WithValue(r.Context(), ctxUserKey, &authUser{ID: owner}))
	w := httptest.NewRecorder()
	s.updateTransaction(w, r)
	return w
}
func requireDecision(t *testing.T, s *Server, body string) {
	t.Helper()
	w := decisionRequest(s, body, "", 1)
	if w.Code != 204 {
		t.Fatalf("decision %d: %s", w.Code, w.Body)
	}
}
func TestClassificationRevisionABAReplayNoopAndIsolationPostgres(t *testing.T) {
	db := reimbursementTestDB(t)
	s := &Server{DB: db}
	requireDecision(t, s, `{"domain_id":9001}`)
	initial := decisionState(t, db)
	body := fmt.Sprintf(`{"domain_id":9002,"expected_revision":"%d"}`, initial.Revision)
	if w := decisionRequest(s, body, "synthetic-replay", 1); w.Code != 204 {
		t.Fatal(w.Code, w.Body)
	}
	first := decisionState(t, db)
	if w := decisionRequest(s, body, "synthetic-replay", 1); w.Code != 204 {
		t.Fatal("committed replay rejected", w.Code, w.Body)
	}
	if decisionState(t, db).Revision != first.Revision {
		t.Fatal("replay changed revision")
	}
	if w := decisionRequest(s, `{"domain_id":9001}`, "synthetic-replay", 1); w.Code != 409 {
		t.Fatal("operation reused with different request", w.Code)
	}
	requireDecision(t, s, `{"domain_id":9001}`)
	restored := decisionState(t, db)
	if restored.Revision <= first.Revision || *restored.Domain != *initial.Domain || restored.Source != initial.Source {
		t.Fatal("ABA setup", initial, restored)
	}
	if w := decisionRequest(s, fmt.Sprintf(`{"tags":["stale"],"expected_revision":"%d"}`, initial.Revision), "", 1); w.Code != 409 {
		t.Fatal("ABA accepted", w.Code)
	}
	requireDecision(t, s, `{"domain_id":9001,"domain_locked":false,"tags_locked":false,"tags":[]}`)
	if decisionState(t, db).Revision != restored.Revision {
		t.Fatal("no-op incremented revision")
	}
	if w := decisionRequest(s, `{"tags":["foreign"]}`, "", 2); w.Code != 404 {
		t.Fatal("owner isolation", w.Code)
	}
	if w := decisionRequest(s, `{"add_tags":["x"],"remove_tags":["x"]}`, "", 1); w.Code != 400 {
		t.Fatal("ambiguous tag request", w.Code)
	}
	var n int
	if e := db.QueryRow(`SELECT count(*) FROM classification_events WHERE owner_id=1 AND transaction_id=1`).Scan(&n); e != nil || int64(n) != restored.Revision {
		t.Fatal("journal/revision mismatch", n, restored.Revision, e)
	}
}

func TestClassificationIndependentLocksTombstonesAndRenamePostgres(t *testing.T) {
	db := reimbursementTestDB(t)
	s := &Server{DB: db}
	if _, e := db.Exec(`UPDATE transactions SET domain_id=NULL WHERE id=1; INSERT INTO rules(owner_id,pattern,domain_id,tags) VALUES(1,'TEST RESTAURANT',9001,'resto')`); e != nil {
		t.Fatal(e)
	}
	requireDecision(t, s, `{"domain_locked":true}`)
	if _, e := s.applyAllRules(1); e != nil {
		t.Fatal(e)
	}
	state := decisionState(t, db)
	if state.Domain != nil || len(state.Tags) != 1 {
		t.Fatal("domain lock should permit tags", state)
	}
	requireDecision(t, s, `{"domain_locked":false,"tags_locked":true}`)
	if _, e := db.Exec(`UPDATE rules SET tags='resto,nouveau' WHERE owner_id=1`); e != nil {
		t.Fatal(e)
	}
	if _, e := s.applyAllRules(1); e != nil {
		t.Fatal(e)
	}
	state = decisionState(t, db)
	if state.Domain == nil || *state.Domain != 9001 || len(state.Tags) != 1 {
		t.Fatal("tags lock should permit domain", state)
	}
	requireDecision(t, s, `{"tags_locked":false,"remove_tags":["resto"]}`)
	if _, e := db.Exec(`UPDATE tags SET name='renommé' WHERE owner_id=1 AND name='resto'; UPDATE rules SET tags='renommé' WHERE owner_id=1`); e != nil {
		t.Fatal(e)
	}
	if _, e := s.applyAllRules(1); e != nil {
		t.Fatal(e)
	}
	state = decisionState(t, db)
	if len(state.Tags) != 0 {
		t.Fatal("renaming bypassed tombstone", state)
	}
	requireDecision(t, s, `{"add_tags":["renommé"]}`)
	var decision string
	if e := db.QueryRow(`SELECT d.decision FROM classification_tag_decisions d JOIN tags g ON g.id=d.tag_id WHERE d.owner_id=1 AND d.transaction_id=1 AND g.name='renommé'`).Scan(&decision); e != nil || decision != "added" {
		t.Fatal("manual re-add did not clear tombstone", decision, e)
	}
	requireDecision(t, s, `{"domain_id":null,"domain_locked":true}`)
	requireDecision(t, s, `{"domain_locked":false}`)
	if _, e := s.applyAllRules(1); e != nil {
		t.Fatal(e)
	}
	state = decisionState(t, db)
	if state.Domain != nil || state.Source != "manual" {
		t.Fatal("unlock lost manual priority", state)
	}
}

func TestClassificationJournalAtomicPrivateAndImmutablePostgres(t *testing.T) {
	db := reimbursementTestDB(t)
	s := &Server{DB: db}
	before := decisionState(t, db)
	if _, e := db.Exec(`CREATE FUNCTION synthetic_journal_failure() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'synthetic journal failure'; END $$; CREATE TRIGGER synthetic_journal_failure BEFORE INSERT ON classification_events FOR EACH ROW EXECUTE FUNCTION synthetic_journal_failure()`); e != nil {
		t.Fatal(e)
	}
	if w := decisionRequest(s, `{"domain_id":9002}`, "", 1); w.Code != 500 {
		t.Fatal("journal failure ignored", w.Code)
	}
	after := decisionState(t, db)
	if after.Revision != before.Revision || *after.Domain != *before.Domain || after.Source != before.Source {
		t.Fatal("partial decision committed", before, after)
	}
	if _, e := db.Exec(`DROP TRIGGER synthetic_journal_failure ON classification_events`); e != nil {
		t.Fatal(e)
	}
	requireDecision(t, s, `{"tags":["synthetic"]}`)
	if _, e := db.Exec(`UPDATE classification_events SET reason='modified' WHERE owner_id=1 AND transaction_id=1`); e == nil {
		t.Fatal("mutable history")
	}
	r := httptest.NewRequest("GET", "/api/transactions/1/classification/history", nil)
	r.SetPathValue("id", "1")
	r = r.WithContext(context.WithValue(r.Context(), ctxUserKey, &authUser{ID: 1}))
	w := httptest.NewRecorder()
	s.classificationHistory(w, r)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	var result struct {
		Events []map[string]json.RawMessage `json:"events"`
	}
	if e := json.Unmarshal(w.Body.Bytes(), &result); e != nil || len(result.Events) == 0 {
		t.Fatal(e, w.Body)
	}
	for _, event := range result.Events {
		for _, name := range []string{"before", "after"} {
			v := string(event[name])
			if strings.Contains(v, "TEST RESTAURANT") || strings.Contains(v, "amount") || strings.Contains(v, "label") {
				t.Fatal("bank data in journal", v)
			}
		}
	}
	r = r.WithContext(context.WithValue(r.Context(), ctxUserKey, &authUser{ID: 2}))
	w = httptest.NewRecorder()
	s.classificationHistory(w, r)
	if w.Code != 404 {
		t.Fatal("foreign journal visible", w.Code)
	}
}

type decisionProposalClient struct {
	callback func()
	provider string
}

func (c decisionProposalClient) Name() string { return c.provider }
func (c decisionProposalClient) Complete(context.Context, string, string, int, bool) (string, error) {
	if c.callback != nil {
		c.callback()
	}
	return `{"assignments":[{"id":1,"domain":"Restaurants TEST","tags":["facture"]}]}`, nil
}

func TestClassificationDurableRejectionContextReevaluationAndLateAIPostgres(t *testing.T) {
	db := reimbursementTestDB(t)
	s := &Server{DB: db}
	ctx := context.Background()
	if _, e := db.Exec(`UPDATE transactions SET domain_id=NULL WHERE id=1`); e != nil {
		t.Fatal(e)
	}
	generate := func(client decisionProposalClient) ai.EnrichResult {
		t.Helper()
		r, e := ai.AutoEnrich(ctx, db, client, 1, 0, "")
		if e != nil {
			t.Fatal(e)
		}
		return r
	}
	if generate(decisionProposalClient{provider: "first"}).Suggested != 1 {
		t.Fatal("no initial proposal")
	}
	state, key, e := classification.Context(ctx, db, 1, 1)
	if e != nil {
		t.Fatal(e)
	}
	identity, identityErr := ai.SuggestionIdentity(1, 1, 9001, []string{"facture"})
	if identityErr != nil {
		t.Fatal(identityErr)
	}
	requireDecision(t, s, fmt.Sprintf(`{"discard_suggestion":true,"expected_suggestion_identity":%q,"expected_revision":"%d"}`, identity, state.Revision))
	_, rejectedKey, e := classification.Context(ctx, db, 1, 1)
	if e != nil || rejectedKey != key {
		t.Fatal("rejection invalidated itself", e)
	}
	if generate(decisionProposalClient{provider: "different-model"}).Suggested != 0 {
		t.Fatal("rejected suggestion returned")
	}
	if _, e = db.Exec(`INSERT INTO rules(owner_id,pattern,domain_id,tags) VALUES(1,'UNRELATED SYNTHETIC',9002,'other')`); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(`UPDATE rules SET tags='changed unrelated' WHERE pattern='UNRELATED SYNTHETIC'; INSERT INTO transactions(id,owner_id,statement_id,tx_date,label,amount,hash) VALUES(80001,1,101,'2025-12-20','UNRELATED IMPORT',-1.01,'synthetic-unrelated-import')`); e != nil {
		t.Fatal(e)
	}
	if generate(decisionProposalClient{provider: "third"}).Suggested != 0 {
		t.Fatal("unrelated rule reset rejection")
	}
	requireDecision(t, s, `{"reevaluate":true}`)
	if generate(decisionProposalClient{}).Suggested != 1 {
		t.Fatal("explicit reevaluation did not reset context")
	}
	requireDecision(t, s, `{"discard_suggestion":true}`)
	if _, e = db.Exec(`INSERT INTO rules(owner_id,pattern,domain_id,tags) VALUES(1,'TEST RESTAURANT',9001,'facture')`); e != nil {
		t.Fatal(e)
	}
	if generate(decisionProposalClient{}).Suggested != 1 {
		t.Fatal("relevant rule change did not open new context")
	}
	if _, e = db.Exec(`UPDATE rules SET tags='new relevant context' WHERE owner_id=1 AND pattern='TEST RESTAURANT'`); e != nil {
		t.Fatal(e)
	}
	reportRequest := httptest.NewRequest("GET", "/api/transactions?view=report&all=1&ids=1", nil)
	reportRequest = reportRequest.WithContext(context.WithValue(reportRequest.Context(), ctxUserKey, &authUser{ID: 1}))
	reportResponse := httptest.NewRecorder()
	s.listTransactions(reportResponse, reportRequest)
	var report transactionReport
	if e = json.Unmarshal(reportResponse.Body.Bytes(), &report); e != nil || reportResponse.Code != 200 || len(report.Transactions) != 1 {
		t.Fatal("report revalidation", e, reportResponse.Code, reportResponse.Body)
	}
	if report.Transactions[0].Suggestion != nil {
		t.Fatal("stale proposal presented after relevant rule change")
	}
	requireDecision(t, s, `{"reevaluate":true}`)
	if generate(decisionProposalClient{callback: func() { requireDecision(t, s, `{"domain_locked":true}`) }}).Suggested != 0 {
		t.Fatal("late model response survived changed revision")
	}
	var count int
	if e = db.QueryRow(`SELECT count(*) FROM classification_rejections WHERE owner_id=1 AND transaction_id=1`).Scan(&count); e != nil || count != 2 {
		t.Fatal("rejections not durable", count, e)
	}
}

func TestClassificationLegacyPreconditionTransitionPostgres(t *testing.T) {
	db := reimbursementTestDB(t)
	s := &Server{DB: db}
	if _, e := db.Exec(`INSERT INTO transactions(id,owner_id,statement_id,tx_date,label,amount,hash) VALUES(80001,1,101,'2025-12-20','SYNTHETIC LEGACY',-1.01,'synthetic-legacy-precondition')`); e != nil {
		t.Fatal(e)
	}
	initial, e := classification.Read(context.Background(), db, 1, 80001)
	if e != nil {
		t.Fatal(e)
	}
	if initial.Revision != 0 || initial.Configured {
		t.Fatal("untracked baseline", initial)
	}
	legacy := classification.LegacyVersion(1, 80001, initial.Domain, initial.Source, initial.Tags)
	call := func() *httptest.ResponseRecorder {
		body := fmt.Sprintf(`{"domain_id":9001,"expected_classification_version":%q}`, legacy)
		r := httptest.NewRequest("PATCH", "/api/transactions/80001", strings.NewReader(body))
		r.SetPathValue("id", "80001")
		r = r.WithContext(context.WithValue(r.Context(), ctxUserKey, &authUser{ID: 1}))
		w := httptest.NewRecorder()
		s.updateTransaction(w, r)
		return w
	}
	if w := call(); w.Code != 204 {
		t.Fatal("legacy baseline rejected", w.Code, w.Body)
	}
	if w := call(); w.Code != 409 {
		t.Fatal("old fingerprint bypassed durable revision", w.Code, w.Body)
	}
}

func TestClassificationAgentProtectionStalenessAndDismissalPostgres(t *testing.T) {
	db := reimbursementTestDB(t)
	s := &Server{DB: db}
	var conversation int64
	if e := db.QueryRow(`INSERT INTO chat_conversations(owner_id,title) VALUES(1,'SYNTHETIC CLASSIFICATION AGENT') RETURNING id`).Scan(&conversation); e != nil {
		t.Fatal(e)
	}
	prepare := func() agentAction {
		t.Helper()
		action, e := s.prepareAgentAction(context.Background(), 1, conversation, 0, agentActionInput{Kind: "tag", TransactionIDs: []int64{1}, Tag: "confirmed-agent"})
		if e != nil {
			t.Fatal(e)
		}
		return action
	}
	stale := prepare()
	requireDecision(t, s, `{"tags_locked":true}`)
	agentReq(t, s, s.agentActionDecision, 1, "POST", "/api/agent/actions/decision", `{"decision":"apply"}`, stale.ID, 409)
	fresh := prepare()
	agentReq(t, s, s.agentActionDecision, 1, "POST", "/api/agent/actions/decision", `{"decision":"apply"}`, fresh.ID, 200)
	state := decisionState(t, db)
	if !state.TagsLocked || len(state.Tags) != 1 {
		t.Fatal("explicit confirmed Agent edit lost protection", state)
	}
	var events int
	if e := db.QueryRow(`SELECT count(*) FROM classification_events WHERE owner_id=1 AND transaction_id=1 AND origin='agent' AND actor_id=1 AND operation_id=$1`, fmt.Sprintf("agent:%d:1", fresh.ID)).Scan(&events); e != nil || events == 0 {
		t.Fatal("Agent journal missing", events, e)
	}
	dismissed := prepare()
	agentReq(t, s, s.agentActionDecision, 1, "POST", "/api/agent/actions/decision", `{"decision":"dismiss"}`, dismissed.ID, 200)
	if decisionState(t, db).Revision != state.Revision {
		t.Fatal("dismissal changed classification")
	}
	if e := db.QueryRow(`SELECT count(*) FROM classification_rejections WHERE owner_id=1 AND transaction_id=1`).Scan(&events); e != nil || events != 0 {
		t.Fatal("Agent dismissal confused with classification rejection", events, e)
	}
}
