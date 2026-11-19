package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

type unavailableObservationModel struct{ calls int }

func (m *unavailableObservationModel) Name() string { return "SYNTHETIC OFFLINE" }
func (m *unavailableObservationModel) Complete(context.Context, string, string, int, bool) (string, error) {
	m.calls++
	return "", fmt.Errorf("synthetic provider unavailable")
}
func TestAgentSpendingObservationDoesNotIgnoreQualifiers(t *testing.T) {
	if p, e := resolveAgentPeriod("Ma plus grosse dépense 2025", time.Date(2026, 11, 19, 0, 0, 0, 0, time.UTC), agentPeriod{}, ""); e != nil || p.From != "2025-01-01" {
		t.Fatalf("financial year lost: %+v %v", p, e)
	}
	for _, q := range []string{"ma top dépense de 100", "ma plus grosse dépense de plus de 100"} {
		if agentIntent(q, "") == "top_expense" {
			t.Fatalf("top request dropped amount qualifier: %s", q)
		}
	}

	for _, q := range []string{"Combien ai-je dépensé de 100 ?", "Combien ai-je dépensé plus de 100 ?", "Combien ai-je dépensé > 100 ?", "Combien ai-je dépensé chez TEST ?", "Quel est le total de mes dépenses en catégorie Restaurants ?", "Combien ai-je dépensé au-delà de 100 € ?", "Combien vais-je dépenser le mois prochain ?", "Compare le total des dépenses à mes revenus"} {
		if agentSimpleSpending(q) {
			t.Fatalf("qualifier ignored: %s", q)
		}
	}
	for _, q := range []string{"Quel est le montant de mes dépenses du 1er au 10 décembre 2025 ?", "Quel est le montant de mes dépenses du 1er au 10 décembre 2025 s’il te plaît ?", "Combien ai-je dépensé pendant les trente derniers jours ?", "Combien ai-je dépensé ce mois ?", "Quel est le montant de mes dépenses en décembre 2025 ?", "et le mois précédent ?", "Combien ai-je dépensé pour ces opérations ?"} {
		if !agentSimpleSpending(q) {
			t.Fatalf("simple observation not recognized: %s", q)
		}
	}
}
func TestAgentSpendingObservationsOfflineAndScopedPostgres(t *testing.T) {
	db := reimbursementTestDB(t)
	model := &unavailableObservationModel{}
	s := &Server{DB: db}
	s.ai.client = model
	var sumBefore, sumAfter string
	if e := db.QueryRow(`SELECT sum(amount)::text FROM transactions WHERE owner_id=1`).Scan(&sumBefore); e != nil {
		t.Fatal(e)
	}
	question := `{"messages":[{"role":"user","content":"Combien ai-je dépensé en décembre 2025 ?"}],"context":{"timezone":"Europe/Paris"}}`
	first := agentReq(t, s, s.chatHandler, 1, "POST", "/", question, 0, 200)
	result := agentBody[map[string]any](t, first)
	conversation := int64(result["conversation_id"].(float64))
	if !strings.Contains(result["answer"].(string), "205,00") || result["status"] != "ok" || model.calls != 0 {
		t.Fatalf("observation depended on model: %s calls=%d", first.Body, model.calls)
	}
	if len(result["steps"].([]any)) != 1 || len(result["evidence"].([]any)) != 1 {
		t.Fatal(first.Body.String())
	}
	proof := result["evidence"].([]any)[0].(map[string]any)
	if _, present := proof["data"].(map[string]any)["income"]; present {
		t.Fatal("expense-only observation claims an income total")
	}
	if _, present := proof["data"].(map[string]any)["net"]; present {
		t.Fatal("expense-only observation claims an overall net")
	}
	if proof["tool"] != "get_financial_summary" || proof["data"].(map[string]any)["expenses"].(float64) != 205 || result["data_quality"].(map[string]any)["status"] != "unknown" {
		t.Fatalf("unverified observation: %s", first.Body)
	}
	follow := fmt.Sprintf(`{"conversation_id":%d,"messages":[{"role":"user","content":"et le mois précédent ?"}]}`, conversation)
	second := agentReq(t, s, s.chatHandler, 1, "POST", "/", follow, 0, 200)
	if !strings.Contains(second.Body.String(), `"status":"insufficient_data"`) || !strings.Contains(second.Body.String(), `"from":"2025-11-01"`) || strings.Contains(second.Body.String(), "représentent 0") || model.calls != 0 {
		t.Fatal(second.Body.String())
	}
	selected := agentReq(t, s, s.chatHandler, 1, "POST", "/", `{"messages":[{"role":"user","content":"Combien ai-je dépensé pour ces opérations ?"}],"context":{"page_context":{"page":"transactions","transaction_ids":[1,3]}}}`, 0, 200)
	if !strings.Contains(selected.Body.String(), "180,00") || !strings.Contains(selected.Body.String(), `"from":"2025-12-10"`) || model.calls != 0 {
		t.Fatal(selected.Body.String())
	}
	foreign := agentReq(t, s, s.chatHandler, 2, "POST", "/", question, 0, 200)
	if !strings.Contains(foreign.Body.String(), "60,00") || strings.Contains(foreign.Body.String(), "205,00") {
		t.Fatal(foreign.Body.String())
	}
	agentReq(t, s, s.chatHandler, 2, "POST", "/", `{"messages":[{"role":"user","content":"Combien ai-je dépensé pour ces opérations ?"}],"context":{"page_context":{"page":"transactions","transaction_ids":[1]}}}`, 0, 400)
	for _, q := range []string{"Quel est le montant de mes dépenses du 1er au 10 décembre 2025 ?", "Total des dépenses 2025"} {
		b, _ := json.Marshal(map[string]any{"messages": []map[string]string{{"role": "user", "content": q}}})
		response := agentReq(t, s, s.chatHandler, 1, "POST", "/", string(b), 0, 200)
		expected := "60,00"
		if strings.Contains(q, "Total") {
			expected = "205,00"
		}
		if !strings.Contains(response.Body.String(), expected) || model.calls != 0 {
			t.Fatalf("calendar observation: %s", response.Body)
		}
	}
	// A merchant qualifier must keep the planner path, even while it is offline.
	qualified := agentReq(t, s, s.chatHandler, 1, "POST", "/", `{"messages":[{"role":"user","content":"Combien ai-je dépensé chez TEST en décembre 2025 ?"}]}`, 0, 200)
	if !strings.Contains(qualified.Body.String(), `"status":"error"`) || model.calls != 1 {
		t.Fatalf("qualified request broadened: %s calls=%d", qualified.Body, model.calls)
	}
	qualifiedData := agentBody[map[string]any](t, qualified)
	qualifiedConversation := int64(qualifiedData["conversation_id"].(float64))
	for i, q := range []string{"et le mois précédent ?", "Combien ai-je dépensé en novembre 2025 ?"} {
		b, _ := json.Marshal(map[string]any{"conversation_id": qualifiedConversation, "messages": []map[string]string{{"role": "user", "content": q}}})
		response := agentReq(t, s, s.chatHandler, 1, "POST", "/", string(b), 0, 200)
		if !strings.Contains(response.Body.String(), `"status":"error"`) || model.calls != i+2 {
			t.Fatalf("prior merchant scope forgotten: %s calls=%d", response.Body, model.calls)
		}
	}
	history := agentReq(t, s, s.agentConversations, 1, "GET", "/", "", conversation, 200)
	if !strings.Contains(history.Body.String(), "205,00") {
		t.Fatal("observation not persisted")
	}
	if e := db.QueryRow(`SELECT sum(amount)::text FROM transactions WHERE owner_id=1`).Scan(&sumAfter); e != nil {
		t.Fatal(e)
	}
	if sumBefore != sumAfter {
		t.Fatal("read changed bank amounts")
	}

}

func TestAgentClarificationDoesNotRestoreUnqualifiedScopePostgres(t *testing.T) {
	db := reimbursementTestDB(t)
	model := &unavailableObservationModel{}
	s := &Server{DB: db}
	s.ai.client = model
	first := agentReq(t, s, s.chatHandler, 1, "POST", "/", `{"messages":[{"role":"user","content":"Ma top dépense en décembre 2025"}]}`, 0, 200)
	conversation := int64(agentBody[map[string]any](t, first)["conversation_id"].(float64))
	bad := agentReq(t, s, s.chatHandler, 1, "POST", "/", fmt.Sprintf(`{"conversation_id":%d,"messages":[{"role":"user","content":"Ma top dépense de plus de 100 € le 31 novembre 2025"}]}`, conversation), 0, 200)
	if !strings.Contains(bad.Body.String(), `"status":"needs_clarification"`) {
		t.Fatal(bad.Body.String())
	}
	follow := agentReq(t, s, s.chatHandler, 1, "POST", "/", fmt.Sprintf(`{"conversation_id":%d,"messages":[{"role":"user","content":"et le mois précédent ?"}]}`, conversation), 0, 200)
	if model.calls != 1 || !strings.Contains(follow.Body.String(), `"status":"error"`) {
		t.Fatalf("clarification revived an unqualified answer: %s", follow.Body)
	}
}

func TestAgentObservationSelectionReferencesPostgres(t *testing.T) {
	db := reimbursementTestDB(t)
	model := &unavailableObservationModel{}
	s := &Server{DB: db}
	s.ai.client = model
	for _, q := range []string{"Combien ai-je dépensé pour cette opération ?", "Combien ai-je dépensé pour cette transaction ?"} {
		b, _ := json.Marshal(map[string]any{"messages": []map[string]string{{"role": "user", "content": q}}, "context": map[string]any{"page_context": map[string]any{"page": "transactions", "transaction_ids": []int64{1}}}})
		result := agentReq(t, s, s.chatHandler, 1, "POST", "/", string(b), 0, 200)
		if !strings.Contains(result.Body.String(), "60,00") || !strings.Contains(result.Body.String(), `"from":"2025-12-10"`) || model.calls != 0 {
			t.Fatalf("single selection lost: %s", result.Body)
		}
	}
	for _, ids := range [][]int64{{}, {1, 3}} {
		b, _ := json.Marshal(map[string]any{"messages": []map[string]string{{"role": "user", "content": "Combien ai-je dépensé pour cette opération ?"}}, "context": map[string]any{"page_context": map[string]any{"transaction_ids": ids}}})
		result := agentReq(t, s, s.chatHandler, 1, "POST", "/", string(b), 0, 200)
		if !strings.Contains(result.Body.String(), `"status":"needs_clarification"`) || model.calls != 0 {
			t.Fatalf("ambiguous selection guessed: %s", result.Body)
		}
	}
	first := agentReq(t, s, s.chatHandler, 1, "POST", "/", `{"messages":[{"role":"user","content":"Combien ai-je dépensé en décembre 2025 ?"}]}`, 0, 200)
	conversation := int64(agentBody[map[string]any](t, first)["conversation_id"].(float64))
	follow := agentReq(t, s, s.chatHandler, 1, "POST", "/", fmt.Sprintf(`{"conversation_id":%d,"messages":[{"role":"user","content":"et en novembre ?"}]}`, conversation), 0, 200)
	if !strings.Contains(follow.Body.String(), `"from":"2025-11-01"`) || !strings.Contains(follow.Body.String(), `"status":"insufficient_data"`) || model.calls != 0 {
		t.Fatalf("month follow-up requires a provider: %s", follow.Body)
	}
}

func TestAgentDirectObservationsRespectMissionAndLegacyContextPostgres(t *testing.T) {
	db := reimbursementTestDB(t)
	model := &unavailableObservationModel{}
	s := &Server{DB: db}
	s.ai.client = model
	var oldConversation int64
	if e := db.QueryRow(`INSERT INTO chat_conversations(owner_id,title) VALUES(1,'SYNTHETIC LEGACY') RETURNING id`).Scan(&oldConversation); e != nil {
		t.Fatal(e)
	}
	if _, e := db.Exec(`INSERT INTO chat_messages(conversation_id,role,content) VALUES($1,'user','Limiter la discussion aux restaurants TEST')`, oldConversation); e != nil {
		t.Fatal(e)
	}
	old := agentReq(t, s, s.chatHandler, 1, "POST", "/", fmt.Sprintf(`{"conversation_id":%d,"messages":[{"role":"user","content":"Combien ai-je dépensé en décembre 2025 ?"}]}`, oldConversation), 0, 200)
	if !strings.Contains(old.Body.String(), `"status":"error"`) || model.calls != 1 {
		t.Fatalf("legacy context discarded: %s", old.Body)
	}
	mission := agentBody[map[string]int64](t, agentReq(t, s, s.agentMissions, 1, "POST", "/", `{"objective":"Examiner seulement les restaurants TEST"}`, 0, 201))
	for i, q := range []string{"Combien ai-je dépensé en décembre 2025 ?", "Ma top dépense en décembre 2025"} {
		b, _ := json.Marshal(map[string]any{"mission_id": mission["id"], "messages": []map[string]string{{"role": "user", "content": q}}})
		result := agentReq(t, s, s.chatHandler, 1, "POST", "/", string(b), 0, 200)
		if !strings.Contains(result.Body.String(), `"status":"error"`) || model.calls != i+2 {
			t.Fatalf("mission objective discarded: %s", result.Body)
		}
	}
	selected := agentReq(t, s, s.chatHandler, 1, "POST", "/", `{"messages":[{"role":"user","content":"Combien ai-je dépensé pour ces opérations ?"}],"context":{"page_context":{"transaction_ids":[1,3]}}}`, 0, 200)
	conversation := int64(agentBody[map[string]any](t, selected)["conversation_id"].(float64))
	follow := agentReq(t, s, s.chatHandler, 1, "POST", "/", fmt.Sprintf(`{"conversation_id":%d,"messages":[{"role":"user","content":"et le mois précédent ?"}]}`, conversation), 0, 200)
	if !strings.Contains(follow.Body.String(), `"status":"error"`) || model.calls != 4 {
		t.Fatalf("selection was silently broadened: %s", follow.Body)
	}
}
