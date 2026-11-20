package api

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAgentCategoryQuestionScope(t *testing.T) {
	for _, q := range []string{"Et par catégorie ?", "Répartis mes dépenses par catégorie ce mois", "Détaille mes revenus par catégorie en janvier 2026", "Où va mon argent ?"} {
		if agentCategoryIntent(q, "") == "" {
			t.Errorf("simple category question missed: %s", q)
		}
	}
	for _, q := range []string{"Mes revenus et mes dépenses par catégorie", "Répartis mes dépenses par catégorie chez TEST", "Mes dépenses de plus de 100 € par catégorie", "Mes dépenses par catégorie Alimentation", "Mes revenus fiscaux par catégorie", "Ma catégorie Revenus", "Mes opérations par catégorie Revenus", "Compare mes dépenses par catégorie", "Ma top catégorie", "Les 5 catégories de dépenses", "Pourquoi mes dépenses augmentent par catégorie"} {
		if agentCategoryIntent(q, "") != "" {
			t.Errorf("constraint dropped: %s", q)
		}
	}
	if agentCategoryIntent("et par catégorie ?", "income") != "income_categories" || agentCategoryIntent("et par catégorie ?", "income_categories") != "income_categories" {
		t.Fatal("income context lost")
	}
}
func TestAgentCategoryObservationsOfflinePostgres(t *testing.T) {
	db := reimbursementTestDB(t)
	if _, e := db.Exec(`UPDATE transactions SET domain_id=NULL WHERE id=6`); e != nil {
		t.Fatal(e)
	}
	model := &unavailableObservationModel{}
	s := &Server{DB: db}
	s.ai.client = model
	var before, after string
	db.QueryRow(`SELECT sum(amount)::text FROM transactions`).Scan(&before)
	ask := func(owner int64, text string, conv int64, ids []int64) map[string]any {
		t.Helper()
		body := map[string]any{"messages": []map[string]string{{"role": "user", "content": text}}}
		if conv > 0 {
			body["conversation_id"] = conv
		}
		if ids != nil {
			body["context"] = map[string]any{"page_context": map[string]any{"transaction_ids": ids}}
		}
		raw, _ := json.Marshal(body)
		return agentBody[map[string]any](t, agentReq(t, s, s.chatHandler, owner, "POST", "/", string(raw), 0, 200))
	}
	total := ask(1, "Combien ai-je dépensé en décembre 2025 ?", 0, nil)
	conv := int64(total["conversation_id"].(float64))
	result := ask(1, "Et par catégorie ?", conv, nil)
	if result["status"] != "ok" || !strings.Contains(result["answer"].(string), "195,00") || !strings.Contains(result["answer"].(string), "10,00") || !strings.Contains(result["answer"].(string), "Sans catégorie") || model.calls != 0 {
		t.Fatalf("breakdown not grounded: %+v calls=%d", result, model.calls)
	}
	proof := result["evidence"].([]any)[0].(map[string]any)["data"].(map[string]any)
	if len(proof["categories"].([]any)) != 2 || proof["filters"].(map[string]any)["direction"] != "expense" || proof["query_from"] != "2025-12-01" {
		t.Fatal(proof)
	}
	overall := ask(1, "Et au total ?", conv, nil)
	if overall["status"] != "ok" || !strings.Contains(overall["answer"].(string), "205,00") || overall["context"].(map[string]any)["intent"] != "expenses" {
		t.Fatal(overall)
	}
	income := ask(1, "Mes revenus en janvier 2026 ?", 0, nil)
	incomeConv := int64(income["conversation_id"].(float64))
	byIncome := ask(1, "Et par catégorie ?", incomeConv, nil)
	if byIncome["status"] != "ok" || !strings.Contains(byIncome["answer"].(string), "205,01") || !strings.Contains(byIncome["answer"].(string), "remboursements") {
		t.Fatal(byIncome)
	}
	incomeTotal := ask(1, "Et au total ?", incomeConv, nil)
	if incomeTotal["status"] != "ok" || !strings.Contains(incomeTotal["answer"].(string), "205,01") || incomeTotal["context"].(map[string]any)["intent"] != "income" {
		t.Fatal(incomeTotal)
	}
	ask(1, "Et par catégorie ?", incomeConv, nil)
	previous := ask(1, "et le mois précédent ?", incomeConv, nil)
	if previous["status"] != "insufficient_data" || previous["context"].(map[string]any)["period"].(map[string]any)["from"] != "2025-12-01" {
		t.Fatal(previous)
	}
	expense := ask(1, "Répartis mes dépenses par catégorie", incomeConv, nil)
	if expense["status"] != "ok" || !strings.Contains(expense["answer"].(string), "195,00") {
		t.Fatal(expense)
	}
	selected := ask(1, "Répartis mes dépenses par catégorie pour ces opérations", 0, []int64{1, 3})
	if selected["status"] != "ok" || !strings.Contains(selected["answer"].(string), "180,00") {
		t.Fatal(selected)
	}
	foreign := ask(2, "Mes dépenses par catégorie en décembre 2025 ?", 0, nil)
	if foreign["status"] != "ok" || !strings.Contains(foreign["answer"].(string), "60,00") || strings.Contains(foreign["answer"].(string), "195,00") {
		t.Fatal(foreign)
	}
	if model.calls != 0 {
		t.Fatalf("simple observations called provider: %d", model.calls)
	}
	qualified := ask(1, "Mes dépenses par catégorie chez TEST en décembre 2025 ?", 0, nil)
	if qualified["status"] != "error" || model.calls != 1 {
		t.Fatal(qualified)
	}
	follow := ask(1, "Et par catégorie ?", int64(qualified["conversation_id"].(float64)), nil)
	if follow["status"] != "error" || model.calls != 2 {
		t.Fatalf("merchant constraint lost: %+v", follow)
	}
	// A selected exchange must never silently become a complete-month query.
	selectedFollow := ask(1, "et le mois précédent ?", int64(selected["conversation_id"].(float64)), nil)
	if selectedFollow["status"] != "error" || model.calls != 3 {
		t.Fatal(selectedFollow)
	}
	db.QueryRow(`SELECT sum(amount)::text FROM transactions`).Scan(&after)
	if before != after {
		t.Fatalf("read changed bank amounts: %s -> %s", before, after)
	}
	if _, e := db.Exec(`INSERT INTO domains(id,owner_id,name,color) SELECT 10000+n,1,'SYNTHETIC CATEGORY '||n,'#123456' FROM generate_series(1,201) n;
 INSERT INTO transactions(id,statement_id,owner_id,tx_date,label,amount,domain_id,hash) SELECT 10000+n,101,1,'2025-12-22','SYNTHETIC CATEGORY',-1,10000+n,'category-'||n FROM generate_series(1,201) n`); e != nil {
		t.Fatal(e)
	}
	partial := ask(1, "Mes dépenses par catégorie en décembre 2025 ?", 0, nil)
	source := partial["evidence"].([]any)[0].(map[string]any)["data"].(map[string]any)
	if source["truncated"] != true || len(source["categories"].([]any)) != 200 || !strings.Contains(partial["answer"].(string), "200 premières") || !strings.Contains(partial["answer"].(string), "six premières") {
		t.Fatal(partial)
	}
}
