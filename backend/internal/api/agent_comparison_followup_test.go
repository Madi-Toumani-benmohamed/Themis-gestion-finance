package api

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestAgentComparisonFollowupScope(t *testing.T) {
	now, _ := time.Parse("2006-01-02", "2026-12-01")
	a := agentPeriod{From: "2026-11-01", To: "2026-11-21"}
	b := agentPeriod{From: "2026-10-01", To: "2026-10-21"}
	ctx := agentContext{Period: a, Intent: "expense_comparison", Comparison: &agentComparisonContext{A: a, B: b, Direction: "expense", Scope: agentComparisonUnfiltered}}
	for _, tt := range []struct{ q, d string }{{"Et mes revenus ?", "income"}, {"Et pour les dépenses ?", "expense"}, {"Et mes entrées d’argent ?", "income"}, {"Les revenus sur les mêmes périodes ?", "income"}, {"Et concernant mes dépenses dans ces deux périodes ?", "expense"}} {
		if d, ok := agentComparisonFollowup(tt.q, ctx, now); !ok || d != tt.d {
			t.Errorf("missed follow-up %q: %s %v", tt.q, d, ok)
		}
	}
	for _, q := range []string{"Et mes revenus en décembre ?", "Et mes dépenses chez TEST ?", "Et mes revenus fiscaux ?", "Et mes dépenses sans catégorie ?", "Et mes dépenses de plus de 100 € ?", "Et mes revenus et dépenses ?", "Et par catégorie ?", "Et mes revenus pour cette sélection ?", "Et au total ?"} {
		if _, ok := agentComparisonFollowup(q, ctx, now); ok {
			t.Errorf("qualifier lost: %s", q)
		}
	}
	bad := ctx
	bad.Comparison = nil
	if _, ok := agentComparisonFollowup("Et mes revenus ?", bad, now); ok {
		t.Fatal("legacy context accepted")
	}
	bad = ctx
	bad.Intent = "expenses"
	if _, ok := agentComparisonFollowup("Et mes revenus ?", bad, now); ok {
		t.Fatal("non-comparison context accepted")
	}
	for _, change := range []func(*agentComparisonContext){func(c *agentComparisonContext) { c.Scope = "filtered" }, func(c *agentComparisonContext) { c.Direction = "all" }, func(c *agentComparisonContext) { c.A.To = "2027-01-01" },
		func(c *agentComparisonContext) { c.B = agentPeriod{From: "2027-01-01", To: "2027-01-31"} }, func(c *agentComparisonContext) { c.A.From = "2026-11-02" }, func(c *agentComparisonContext) { c.B = c.A }} {
		copyContext := *ctx.Comparison
		change(&copyContext)
		bad = ctx
		bad.Comparison = &copyContext
		if _, ok := agentComparisonFollowup("Et mes revenus ?", bad, now); ok {
			t.Fatalf("unsafe context accepted: %+v", copyContext)
		}
	}
	// The calendar has changed, but the stored windows remain November/October.
	if _, ok := agentComparisonFollowup("Et mes revenus ?", ctx, now); !ok {
		t.Fatal("month rollover lost the stored comparison")
	}
}
func TestAgentComparisonDirectionFollowupsPostgres(t *testing.T) {
	db := reimbursementTestDB(t)
	model := &unavailableObservationModel{}
	s := &Server{DB: db}
	s.ai.client = model
	if _, e := db.Exec(`INSERT INTO transactions(id,statement_id,owner_id,tx_date,label,amount,domain_id,hash) VALUES
 (1001,101,1,'2025-11-10','SYNTHETIC PRIOR EXPENSE',-100,9001,'follow-1001'),(1002,101,1,'2025-11-20','SYNTHETIC PRIOR INCOME',30,9001,'follow-1002'),(1003,101,1,'2025-12-20','SYNTHETIC CURRENT INCOME',50,9001,'follow-1003')`); e != nil {
		t.Fatal(e)
	}
	var before, after string
	if e := db.QueryRow(`SELECT sum(amount)::text FROM transactions`).Scan(&before); e != nil {
		t.Fatal(e)
	}
	ask := func(q string, conv int64) map[string]any {
		t.Helper()
		body := map[string]any{"messages": []map[string]string{{"role": "user", "content": q}}}
		if conv > 0 {
			body["conversation_id"] = conv
		}
		raw, _ := json.Marshal(body)
		return agentBody[map[string]any](t, agentReq(t, s, s.chatHandler, 1, "POST", "/", string(raw), 0, 200))
	}
	first := ask("Compare mes dépenses de décembre 2025 avec le mois précédent ?", 0)
	conv := int64(first["conversation_id"].(float64))
	if first["status"] != "ok" {
		t.Fatal(first)
	}
	original := first["evidence"].([]any)[0].(map[string]any)["data"].(map[string]any)
	reference := "2025-12-01|2025-12-31|2025-11-01|2025-11-30"
	income := agentBody[map[string]any](t, agentReq(t, s, s.chatHandler, 1, "POST", "/", fmt.Sprintf(`{"conversation_id":%d,"comparison_ref":%q,"messages":[{"role":"user","content":"Et mes revenus ?"}]}`, conv, reference), 0, 200))
	if income["status"] != "ok" || !strings.Contains(income["answer"].(string), "50,00") || !strings.Contains(income["answer"].(string), "30,00") || !strings.Contains(income["answer"].(string), "deux périodes") || model.calls != 0 {
		t.Fatalf("follow-up used model/lost scope: %+v calls=%d", income, model.calls)
	}
	proof := income["evidence"].([]any)[0].(map[string]any)["data"].(map[string]any)
	for _, key := range []string{"period_a", "period_b"} {
		x, _ := json.Marshal(proof[key])
		y, _ := json.Marshal(original[key])
		if string(x) != string(y) {
			t.Fatalf("window changed: %s -> %s", y, x)
		}
	}
	if proof["income_a"] != float64(50) || proof["income_b"] != float64(30) || proof["filters"].(map[string]any)["direction"] != "income" {
		t.Fatal(proof)
	}
	if _, present := proof["expenses_a"]; present {
		t.Fatal("income source claims expense observation")
	}
	// Re-read the persisted context as a new HTTP turn after a data change.
	if _, e := db.Exec(`UPDATE transactions SET amount=55.01 WHERE id=1003`); e != nil {
		t.Fatal(e)
	}
	refreshedBody := fmt.Sprintf(`{"conversation_id":%d,"messages":[{"role":"user","content":"Et les revenus sur les mêmes périodes ?"}],"context":{"timezone":"Europe/Paris","page_context":{"page":"transactions","month":"2026-09"}}}`, conv)
	refreshed := agentBody[map[string]any](t, agentReq(t, s, s.chatHandler, 1, "POST", "/", refreshedBody, 0, 200))
	if refreshed["status"] != "ok" || !strings.Contains(refreshed["answer"].(string), "55,01") {
		t.Fatalf("cached an old amount: %+v", refreshed)
	}
	if refreshed["context"].(map[string]any)["period"].(map[string]any)["from"] != "2025-12-01" {
		t.Fatal("displayed page month replaced the saved windows")
	}
	if _, e := db.Exec(`UPDATE transactions SET amount=50 WHERE id=1003`); e != nil {
		t.Fatal(e)
	}
	expenses := ask("Et mes dépenses ?", conv)
	if expenses["status"] != "ok" || !strings.Contains(expenses["answer"].(string), "205,00") || !strings.Contains(expenses["answer"].(string), "100,00") || model.calls != 0 {
		t.Fatal(expenses)
	}
	agentReq(t, s, s.chatHandler, 2, "POST", "/", fmt.Sprintf(`{"conversation_id":%d,"messages":[{"role":"user","content":"Et mes revenus ?"}]}`, conv), 0, 404)
	agentReq(t, s, s.chatHandler, 1, "POST", "/", `{"messages":[{"role":"user","content":"Et mes revenus ?"}],"context":{"comparison":{"scope":"unfiltered_months"}}}`, 0, 400)
	agentReq(t, s, s.chatHandler, 1, "POST", "/", `{"comparison_ref":"invalid","messages":[{"role":"user","content":"Et mes revenus ?"}]}`, 0, 400)

	// A qualification clears the eligibility; a later simple follow-up must not
	// revive an older unfiltered comparison after the provider error.
	var messagesBefore, messagesAfter int
	if e := db.QueryRow(`SELECT count(*) FROM chat_messages WHERE conversation_id=$1`, conv).Scan(&messagesBefore); e != nil {
		t.Fatal(e)
	}
	agentReq(t, s, s.chatHandler, 1, "POST", "/", fmt.Sprintf(`{"conversation_id":%d,"comparison_ref":"2025-12-01|2025-12-31|2025-11-01|2025-11-29","messages":[{"role":"user","content":"Et mes revenus ?"}]}`, conv), 0, 409)
	if e := db.QueryRow(`SELECT count(*) FROM chat_messages WHERE conversation_id=$1`, conv).Scan(&messagesAfter); e != nil || messagesBefore != messagesAfter {
		t.Fatal("stale request changed history")
	}
	qualified := ask("Et mes revenus chez TEST ?", conv)
	if qualified["status"] != "error" || model.calls != 1 {
		t.Fatal(qualified)
	}
	agentReq(t, s, s.chatHandler, 1, "POST", "/", fmt.Sprintf(`{"conversation_id":%d,"comparison_ref":%q,"messages":[{"role":"user","content":"Et mes revenus ?"}]}`, conv, reference), 0, 409)
	if model.calls != 1 {
		t.Fatal("stale reference called the provider")
	}
	again := ask("Et mes revenus ?", conv)
	if again["status"] != "error" || model.calls != 2 {
		t.Fatal(again)
	}
	var stored map[string]any
	var raw []byte
	if e := db.QueryRow(`SELECT agent_context FROM chat_conversations WHERE id=$1`, conv).Scan(&raw); e != nil {
		t.Fatal(e)
	}
	json.Unmarshal(raw, &stored)
	if _, ok := stored["comparison"]; ok {
		t.Fatal("stale eligibility retained")
	}
	// Empty observations still have verified windows and can change direction.
	empty := ask("Compare mes dépenses de janvier 2026 avec le mois précédent ?", 0)
	if empty["status"] != "insufficient_data" {
		t.Fatal(empty)
	}
	follow := ask("Et mes revenus ?", int64(empty["conversation_id"].(float64)))
	if follow["status"] != "ok" || !strings.Contains(follow["answer"].(string), "205,01") || !strings.Contains(follow["answer"].(string), "50,00") || model.calls != 2 {
		t.Fatal(follow)
	}
	interruptedConv := int64(empty["conversation_id"].(float64))
	if _, e := db.Exec(`INSERT INTO chat_messages(conversation_id,role,content) VALUES($1,'user','Et mes revenus chez TEST ?')`, interruptedConv); e != nil {
		t.Fatal(e)
	}
	agentReq(t, s, s.chatHandler, 1, "POST", "/", fmt.Sprintf(`{"conversation_id":%d,"comparison_ref":"2026-01-01|2026-01-31|2025-12-01|2025-12-31","messages":[{"role":"user","content":"Et mes revenus ?"}]}`, interruptedConv), 0, 409)
	interrupted := ask("Et mes revenus ?", interruptedConv)
	if interrupted["status"] != "error" || model.calls != 3 {
		t.Fatalf("interrupted qualification ignored: %+v", interrupted)
	}
	missionComparison := ask("Compare mes dépenses de décembre 2025 avec le mois précédent ?", 0)
	missionConv := int64(missionComparison["conversation_id"].(float64))
	mission := agentBody[map[string]any](t, agentReq(t, s, s.agentMissions, 1, "POST", "/", fmt.Sprintf(`{"objective":"SYNTHETIC MISSION","conversation_id":%d}`, missionConv), 0, 201))
	missionTurn := agentBody[map[string]any](t, agentReq(t, s, s.chatHandler, 1, "POST", "/", fmt.Sprintf(`{"conversation_id":%d,"mission_id":%d,"messages":[{"role":"user","content":"Et mes revenus ?"}]}`, missionConv, int64(mission["id"].(float64))), 0, 200))
	if missionTurn["status"] != "error" || model.calls != 4 {
		t.Fatalf("mission goal bypassed: %+v", missionTurn)
	}
	selectionComparison := ask("Compare mes dépenses de décembre 2025 avec le mois précédent ?", 0)
	selectionConv := int64(selectionComparison["conversation_id"].(float64))
	selectionTurn := agentBody[map[string]any](t, agentReq(t, s, s.chatHandler, 1, "POST", "/", fmt.Sprintf(`{"conversation_id":%d,"messages":[{"role":"user","content":"Et mes revenus ?"}],"context":{"page_context":{"transaction_ids":[1,3]}}}`, selectionConv), 0, 200))
	if selectionTurn["status"] != "error" || model.calls != 5 {
		t.Fatalf("selection bypassed: %+v", selectionTurn)
	}
	if e := db.QueryRow(`SELECT sum(amount)::text FROM transactions`).Scan(&after); e != nil || before != after {
		t.Fatalf("bank data changed: %s -> %s (%v)", before, after, e)
	}
}
