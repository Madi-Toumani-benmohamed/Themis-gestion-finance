package api

import (
	"fmt"
	"strings"
	"testing"
)

func TestAgentSimpleIncomePreservesQualifiers(t *testing.T) {
	for _, q := range []string{"Mes revenus ce mois ?", "Combien ai-je reçu en janvier 2026 ?", "Et mes revenus ?", "Quel est le montant de mes revenus du 1er au 10 janvier 2026 ?"} {
		if !agentSimpleIncome(q) || agentIntent(q, "") != "income" {
			t.Errorf("simple income missed: %s", q)
		}
	}
	for _, q := range []string{"Mes revenus et mes dépenses", "Mes revenus et ma dépense", "Ma top entrée du mois", "Mes plus gros revenus", "Mes revenus de plus de 100 €", "Mes revenus chez TEST", "Mes revenus fiscaux 2025", "Mon salaire net", "Combien vais-je recevoir le mois prochain ?", "Mes revenus moins mes achats"} {
		if agentSimpleIncome(q) {
			t.Errorf("qualifier discarded: %s", q)
		}
	}
}
func TestAgentIncomeOfflinePostgres(t *testing.T) {
	db := reimbursementTestDB(t)
	model := &unavailableObservationModel{}
	s := &Server{DB: db}
	s.ai.client = model
	var sumBefore, sumAfter string
	if err := db.QueryRow(`SELECT sum(amount)::text FROM transactions`).Scan(&sumBefore); err != nil {
		t.Fatal(err)
	}
	ask := func(owner int64, q string, conv int64) map[string]any {
		t.Helper()
		id := ""
		if conv > 0 {
			id = fmt.Sprintf(`"conversation_id":%d,`, conv)
		}
		w := agentReq(t, s, s.chatHandler, owner, "POST", "/", fmt.Sprintf(`{%s"messages":[{"role":"user","content":%q}]}`, id, q), 0, 200)
		return agentBody[map[string]any](t, w)
	}
	first := ask(1, "Combien ai-je reçu en janvier 2026 ?", 0)
	if first["status"] != "ok" || !strings.Contains(first["answer"].(string), "205,01") || !strings.Contains(first["answer"].(string), "remboursements") || model.calls != 0 {
		t.Fatalf("income not grounded: %+v calls=%d", first, model.calls)
	}
	proof := first["evidence"].([]any)[0].(map[string]any)["data"].(map[string]any)
	if proof["income"] != 205.01 || proof["transaction_count"] != float64(5) || proof["filters"].(map[string]any)["direction"] != "income" {
		t.Fatal(proof)
	}
	for _, key := range []string{"expenses", "net"} {
		if _, ok := proof[key]; ok {
			t.Fatalf("directional proof claims %s: %+v", key, proof)
		}
	}
	conv := int64(first["conversation_id"].(float64))
	follow := ask(1, "et le mois précédent ?", conv)
	if follow["status"] != "insufficient_data" || !strings.Contains(follow["answer"].(string), "ne signifie pas") || follow["context"].(map[string]any)["period"].(map[string]any)["from"] != "2025-12-01" {
		t.Fatal(follow)
	}
	other := ask(2, "Mes revenus en décembre 2025 ?", 0)
	if other["status"] != "ok" || !strings.Contains(other["answer"].(string), "30,00") {
		t.Fatal(other)
	}
	w := agentReq(t, s, s.chatHandler, 1, "POST", "/", `{"messages":[{"role":"user","content":"Mes revenus pour ces opérations ?"}],"context":{"page_context":{"transaction_ids":[2,4,11,16]}}}`, 0, 200)
	if !strings.Contains(w.Body.String(), "70,00") || !strings.Contains(w.Body.String(), `"transaction_count":2`) {
		t.Fatal(w.Body.String())
	}
	expense := ask(1, "Et mes dépenses en décembre 2025 ?", conv)
	if expense["status"] != "ok" || !strings.Contains(expense["answer"].(string), "205,00") {
		t.Fatal(expense)
	}
	if model.calls != 0 {
		t.Fatalf("simple observations used provider %d times", model.calls)
	}
	qualified := ask(1, "Mes revenus chez TEST en janvier 2026 ?", 0)
	if qualified["status"] != "error" || model.calls != 1 {
		t.Fatal(qualified)
	}
	qualifiedConv := int64(qualified["conversation_id"].(float64))
	after := ask(1, "et le mois précédent ?", qualifiedConv)
	if after["status"] != "error" || model.calls != 2 {
		t.Fatalf("lost merchant filter: %+v", after)
	}
	if err := db.QueryRow(`SELECT sum(amount)::text FROM transactions`).Scan(&sumAfter); err != nil || sumAfter != sumBefore {
		t.Fatalf("observation wrote financial data: %s -> %s (%v)", sumBefore, sumAfter, err)
	}
}
