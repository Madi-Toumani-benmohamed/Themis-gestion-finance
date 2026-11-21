package api

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestAgentComparisonRecognitionAndCalendar(t *testing.T) {
	for _, q := range []string{"Compare mes dépenses ce mois avec le mois dernier ?", "Peux-tu comparer mes revenus de janvier 2026 au mois précédent ?", "Et par rapport au mois précédent ?", "Compare avec le mois précédent"} {
		if _, ok := parseAgentComparison(q, "expenses"); !ok {
			t.Errorf("simple comparison missed: %s", q)
		}
	}
	for _, q := range []string{"Compare mes dépenses chez TEST avec le mois précédent", "Compare mes dépenses sans catégorie avec le mois précédent", "Compare ma top dépense avec le mois précédent", "Compare cette opération avec le mois précédent", "Compare cette dépense avec le mois précédent", "Compare la sélection avec le mois précédent", "Compare mes dépenses par catégorie avec le mois précédent", "Compare mes revenus et dépenses avec le mois précédent", "Compare mes dépenses de plus de 100 € avec le mois précédent", "Compare mes dépenses de janvier et février 2026 avec le mois précédent", "Compare les dépenses de novembre 2025 et novembre 2026 avec le mois précédent"} {
		if _, ok := parseAgentComparison(q, "expenses"); ok {
			t.Errorf("constraint ignored: %s", q)
		}
	}
	if _, ok := parseAgentComparison("Et par rapport au mois précédent ?", "top_expense"); ok {
		t.Fatal("ranking became a total")
	}
	for _, tt := range []struct {
		now, from, to, wantFrom, wantTo string
		calendar                        bool
	}{
		{"2026-11-21", "2026-11-01", "2026-11-21", "2026-10-01", "2026-10-21", false},
		{"2024-03-31", "2024-03-01", "2024-03-31", "2024-02-01", "2024-02-29", false},
		{"2026-02-28", "2026-02-01", "2026-02-28", "2026-01-01", "2026-01-28", false},
		{"2026-11-21", "2025-02-01", "2025-02-28", "2025-01-01", "2025-01-31", false},
		{"2026-11-21", "2026-01-01", "2026-01-10", "2025-12-01", "2025-12-10", false},
		{"2026-11-21", "2026-01-01", "2026-01-31", "2026-10-01", "2026-10-31", true},
	} {
		now, _ := time.Parse("2006-01-02", tt.now)
		p, ok := agentPreviousComparisonMonth(agentPeriod{From: tt.from, To: tt.to}, now, tt.calendar)
		if !ok || p.From != tt.wantFrom || p.To != tt.wantTo {
			t.Errorf("calendar %+v => %+v, %v", tt, p, ok)
		}
	}
	now, _ := time.Parse("2006-01-02", "2026-11-21")
	for _, p := range []agentPeriod{{From: "2026-10-10", To: "2026-10-20"}, {From: "2026-10-01", To: "2026-11-01"}, {From: "2027-01-01", To: "2027-01-31"}} {
		if _, ok := agentPreviousComparisonMonth(p, now, false); ok {
			t.Fatalf("non-month range accepted: %+v", p)
		}
	}
}
func TestAgentComparisonOfflinePostgres(t *testing.T) {
	db := reimbursementTestDB(t)
	model := &unavailableObservationModel{}
	s := &Server{DB: db}
	s.ai.client = model
	if _, e := db.Exec(`INSERT INTO transactions(id,statement_id,owner_id,tx_date,label,amount,domain_id,hash) VALUES(1001,101,1,'2025-11-10','SYNTHETIC PREVIOUS',-100,9001,'comparison-1001'),(1002,101,1,'2025-11-20','SYNTHETIC LATER',-50,9001,'comparison-1002')`); e != nil {
		t.Fatal(e)
	}
	var before, after string
	if e := db.QueryRow(`SELECT sum(amount)::text FROM transactions WHERE owner_id=1`).Scan(&before); e != nil {
		t.Fatal(e)
	}
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
	first := ask(1, "Combien ai-je dépensé en décembre 2025 ?", 0, nil)
	conv := int64(first["conversation_id"].(float64))
	result := ask(1, "Compare avec le mois précédent ?", conv, nil)
	if result["status"] != "ok" || !strings.Contains(result["answer"].(string), "205,00") || !strings.Contains(result["answer"].(string), "150,00") || model.calls != 0 {
		t.Fatalf("not grounded: %+v calls=%d", result, model.calls)
	}
	source := result["evidence"].([]any)[0].(map[string]any)["data"].(map[string]any)
	if source["expenses_a"] != float64(205) || source["expenses_b"] != float64(150) || source["comparable"] != false || source["period_b"].(map[string]any)["from"] != "2025-11-01" {
		t.Fatal(source)
	}
	for _, key := range []string{"income_a", "income_b", "expense_change", "expense_change_percent"} {
		if _, present := source[key]; present {
			t.Fatalf("unobserved metric %s: %+v", key, source)
		}
	}
	if !strings.Contains(result["answer"].(string), "ne suffisent pas") {
		t.Fatal(result)
	}
	// An explicit first request resolves its baseline before removing the target.
	direct := ask(1, "Compare mes dépenses du 1er au 10 décembre 2025 avec le mois précédent ?", 0, nil)
	if direct["status"] != "ok" || !strings.Contains(direct["answer"].(string), "60,00") || !strings.Contains(direct["answer"].(string), "100,00") {
		t.Fatal(direct)
	}
	income := ask(1, "Compare mes revenus en janvier 2026 avec le mois précédent ?", 0, nil)
	if income["status"] != "insufficient_data" || !strings.Contains(income["answer"].(string), "205,01") || !strings.Contains(income["answer"].(string), "montant bancaire nul") {
		t.Fatal(income)
	}
	foreign := ask(2, "Compare mes dépenses en décembre 2025 avec le mois précédent ?", 0, nil)
	if !strings.Contains(foreign["answer"].(string), "60,00") || strings.Contains(foreign["answer"].(string), "150,00") {
		t.Fatal(foreign)
	}
	for _, q := range []string{"Compare mes dépenses chez TEST en décembre 2025 avec le mois précédent ?", "Compare mes dépenses sans catégorie avec le mois précédent ?", "Compare ma top dépense avec le mois précédent ?"} {
		r := ask(1, q, 0, nil)
		if r["status"] != "error" {
			t.Fatalf("qualifier discarded: %s %+v", q, r)
		}
	}
	selected := ask(1, "Compare mes dépenses de décembre 2025 avec le mois précédent ?", 0, []int64{1, 3})
	if selected["status"] != "error" {
		t.Fatal(selected)
	}
	legacy := ask(1, "Compare mes dépenses en décembre 2025 avec le mois précédent ?", conv, nil)
	if legacy["status"] != "error" {
		t.Fatalf("comparison context broadened: %+v", legacy)
	}
	if _, e := db.Exec(`INSERT INTO users(id,username,password_hash,role) VALUES(3,'SYNTHETIC DOCUMENTED COMPARISON','TEST','user');
 INSERT INTO accounts(id,owner_id,name,kind,tracking_mode) VALUES(31,3,'SYNTHETIC COMPARISON','courant','statements');
 INSERT INTO statements(id,owner_id,account_id,filename,period_start,period_end,opening_balance,closing_balance,validated) VALUES
 (301,3,31,'SYNTHETIC NOVEMBER','2025-11-01','2025-11-30',1000,900,true),(302,3,31,'SYNTHETIC DECEMBER','2025-12-01','2025-12-31',900,850.25,true);
 INSERT INTO transactions(id,statement_id,owner_id,tx_date,label,amount,hash) VALUES(3001,301,3,'2025-11-10','SYNTHETIC EXPENSE',-100,'documented-3001'),(3002,302,3,'2025-12-10','SYNTHETIC EXPENSE',-49.75,'documented-3002')`); e != nil {
		t.Fatal(e)
	}
	documented := ask(3, "Compare mes dépenses en décembre 2025 avec le mois précédent ?", 0, nil)
	if documented["status"] != "ok" || !strings.Contains(documented["answer"].(string), "-50,25") || !strings.Contains(documented["answer"].(string), "documentés") {
		t.Fatal(documented)
	}
	documentedSource := documented["evidence"].([]any)[0].(map[string]any)["data"].(map[string]any)
	if documentedSource["comparable"] != true || documentedSource["expense_change"] != float64(-50.25) {
		t.Fatal(documentedSource)
	}
	ny, e := time.LoadLocation("America/New_York")
	if e != nil {
		t.Fatal(e)
	}
	localNow := time.Date(2025, 12, 31, 23, 30, 0, 0, ny)
	localResult, e := s.agentObservedComparison(context.Background(), 3, agentPeriod{From: "2025-12-01", To: "2025-12-31"}, agentPeriod{From: "2025-11-01", To: "2025-11-30"}, "expense", localNow)
	if e != nil {
		t.Fatal(e)
	}
	localSource := localResult.Evidence[0].(map[string]any)["data"].(map[string]any)
	if localSource["comparable"] != false || !strings.Contains(localResult.Answer, "ne suffisent pas") {
		t.Fatalf("current local month became final: %+v", localSource)
	}
	if _, present := localSource["expense_change"]; present {
		t.Fatal("current local month exposes a final delta")
	}
	if model.calls != 5 {
		t.Fatalf("planner routing changed: %d", model.calls)
	}
	if e := db.QueryRow(`SELECT sum(amount)::text FROM transactions WHERE owner_id=1`).Scan(&after); e != nil || after != before {
		t.Fatalf("read changed bank amounts: %s -> %s (%v)", before, after, e)
	}
}
