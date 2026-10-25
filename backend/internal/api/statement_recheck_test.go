package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type statementReviewLLM struct {
	complete func(context.Context, string, string, int, bool) (string, error)
}

func (c statementReviewLLM) Complete(ctx context.Context, system, user string, tokens int, json bool) (string, error) {
	return c.complete(ctx, system, user, tokens, json)
}
func (c statementReviewLLM) Name() string { return "TEST PROVIDER / MODEL" }

func TestStatementRechecksAreReadOnlyOwnedAndOptionalPostgres(t *testing.T) {
	db := statementTestDB(t)
	s := &Server{DB: db}
	calls := 0
	s.ai.client = statementReviewLLM{complete: func(ctx context.Context, system, user string, _ int, json bool) (string, error) {
		calls++
		if !json {
			t.Error("structured evidence required")
		}
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 30*time.Second {
			t.Error("unbounded review")
		}
		for _, secret := range []string{"SECRET LABEL", "TEST DEBIT", "B TEST.pdf", "opening_balance", "100000", "PRIVATE CONTEXT", "statement_id"} {
			if strings.Contains(user, secret) {
				t.Errorf("unnecessary bank data/context in prompt: %s", secret)
			}
		}
		return `{"observation_ids":["control_0"]}`, nil
	}}
	s.ai.config.Context = "PRIVATE CONTEXT"
	call := func(mode, id string, owner int64) statementCheckResult {
		t.Helper()
		w := statementCall(s, s.recheckStatement, "POST", "/api/statements/"+id+"/checks", `{"mode":"`+mode+`"}`, owner)
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body)
		}
		var result statementCheckResult
		if json.Unmarshal(w.Body.Bytes(), &result) != nil {
			t.Fatal("invalid result")
		}
		return result
	}
	result := call("deterministic", "102", 1)
	if calls != 0 || result.AI.Status != "not_requested" || result.CheckedAt == "" || result.Statement.ID != 102 {
		t.Fatal("deterministic check called AI or lacks trace")
	}
	result = call("combined", "102", 1)
	if calls != 1 || result.AI.Status != "available" || len(result.AI.Evidence) < 1 || result.AI.Evidence[0].ID != "coverage" {
		t.Fatal("missing verified evidence", result)
	}
	for _, id := range []string{"201", "999999"} {
		w := statementCall(s, s.recheckStatement, "POST", "/api/statements/"+id+"/checks", `{"mode":"combined"}`, 1)
		if w.Code != 404 {
			t.Fatal("foreign/absent source checked", w.Code)
		}
	}
	if calls != 1 {
		t.Fatal("AI called before authorization")
	}
	for _, body := range []string{`{"mode":"ai"}`, `{"mode":"combined","owner_id":2}`, `{"mode":"deterministic"} {}`, `invalid`} {
		w := statementCall(s, s.recheckStatement, "POST", "/api/statements/102/checks", body, 1)
		if w.Code != 400 {
			t.Fatal("invalid check accepted", w.Code)
		}
	}
	unauth := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/statements/102/checks", strings.NewReader(`{"mode":"combined"}`))
	r.SetPathValue("id", "102")
	s.recheckStatement(unauth, r)
	if unauth.Code != 401 {
		t.Fatal("bootstrap guard missing")
	}
	s.ai.client = nil
	result = call("combined", "102", 1)
	if result.AI.Status != "not_configured" || result.Statement.Status != "validated" {
		t.Fatal("missing provider lost deterministic checks")
	}
	s.ai.client = statementReviewLLM{complete: func(context.Context, string, string, int, bool) (string, error) {
		return "", errors.New("PRIVATE KEY BANK PROMPT")
	}}
	result = call("combined", "102", 1)
	raw, _ := json.Marshal(result)
	if result.AI.Status != "unavailable" || strings.Contains(string(raw), "PRIVATE") {
		t.Fatal("provider failure leaked or hid deterministic result")
	}
	s.ai.client = statementReviewLLM{complete: func(context.Context, string, string, int, bool) (string, error) {
		return `{"observation_ids":["invented"],"amount":9999}`, nil
	}}
	result = call("combined", "102", 1)
	if result.AI.Status != "invalid_response" || len(result.AI.Evidence) != 0 {
		t.Fatal("unverified AI content accepted")
	}
	s.ai.client = statementReviewLLM{complete: func(context.Context, string, string, int, bool) (string, error) { return "", context.DeadlineExceeded }}
	result = call("combined", "102", 1)
	if result.AI.Status != "unavailable" || !strings.Contains(result.AI.Detail, "Délai dépassé") {
		t.Fatal("timeout not explicit")
	}
	var history, transactions int
	var validated bool
	if err := db.QueryRow(`SELECT (SELECT count(*) FROM statement_validation_history),(SELECT count(*) FROM transactions WHERE owner_id=1),validated FROM statements WHERE id=102`).Scan(&history, &transactions, &validated); err != nil || history != 0 || transactions != 2 || !validated {
		t.Fatal("recheck wrote financial or validation data", history, transactions, validated, err)
	}
}

func TestStatementConfirmedFailuresStayFailedDuringAIReviewPostgres(t *testing.T) {
	db := statementTestDB(t)
	s := &Server{DB: db}
	if _, err := db.Exec(`UPDATE statements SET closing_balance=950.01 WHERE id=102;
 INSERT INTO statements(id,owner_id,account_id,filename,period_start,period_end,opening_balance,closing_balance,validated) VALUES(301,1,11,'OVERLAP TEST','2026-09-20','2026-10-22',0,0,false)`); err != nil {
		t.Fatal(err)
	}
	s.ai.client = statementReviewLLM{complete: func(context.Context, string, string, int, bool) (string, error) {
		return `{"observation_ids":["control_0"]}`, nil
	}}
	w := statementCall(s, s.recheckStatement, "POST", "/api/statements/102/checks", `{"mode":"combined"}`, 1)
	var result statementCheckResult
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &result) != nil {
		t.Fatal(w.Code, w.Body)
	}
	if result.Statement.Status != "failed" || result.Statement.CanValidate || result.Statement.EffectiveValidated || result.Statement.DifferenceCents == nil || *result.Statement.DifferenceCents != 1 {
		t.Fatal("AI changed confirmed failure", result.Statement)
	}
	failedEvidence := false
	for _, f := range result.AI.Evidence {
		if f.Source["status"] == "failed" {
			failedEvidence = true
		}
	}
	if !failedEvidence {
		t.Fatal("model hid failed check")
	}
	if w = statementCall(s, s.validateStatement, "PATCH", "/api/statements/102/validation", `{"action":"validate","note":"AI review is not proof"}`, 1); w.Code != 409 {
		t.Fatal("AI review bypassed financial invariant")
	}
}

func TestStatementReviewLimitsPayloadAndPrioritizesFailedControls(t *testing.T) {
	st := checkedStatement{}
	for i := 0; i < 30; i++ {
		st.Controls = append(st.Controls, statementControl{Code: "overlap", Status: "suspected", Severity: "warning", Detail: "Overlap TEST"})
	}
	st.Controls = append(st.Controls, statementControl{Code: "reconciliation", Status: "failed", Severity: "error", Detail: "Failure TEST"})
	facts := statementReviewFacts(st)
	if len(facts) != 21 || facts[1].Source["status"] != "failed" || !strings.Contains(facts[0].Text, "20 contrôles prioritaires sur 31") {
		t.Fatal("review budget or priority incorrect", len(facts))
	}
}

func TestStatementReviewLabelsInformationalAndNotApplicableControls(t *testing.T) {
	if controlReviewLabel("informational") != "Information" || controlReviewLabel("not_applicable") != "Non applicable" {
		t.Fatal("informational control was misrepresented as an unevaluated failure")
	}
}

func TestAutomaticStatementCanBeManuallyAttestedOncePostgres(t *testing.T) {
	db := statementTestDB(t)
	s := &Server{DB: db}
	for i, want := range []int{204, 409} {
		w := statementCall(s, s.validateStatement, "PATCH", "/api/statements/102/validation", `{"action":"validate","note":"Original document reviewed TEST"}`, 1)
		if w.Code != want {
			t.Fatal("manual attestation", i, w.Code, w.Body)
		}
	}
	var history int
	var validated bool
	if err := db.QueryRow(`SELECT (SELECT count(*) FROM statement_validation_history WHERE resource_id=102),validated FROM statements WHERE id=102`).Scan(&history, &validated); err != nil || history != 1 || !validated {
		t.Fatal("attestation missing/duplicated", history, err)
	}
}
