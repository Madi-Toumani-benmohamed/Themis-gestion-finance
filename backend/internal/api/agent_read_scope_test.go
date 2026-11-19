package api

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestAgentReadRejectsUnsupportedFilters(t *testing.T) {
	if _, e := agentReadArguments(map[string]any{"start_date": "2025-01-01"}, "comparison"); e == nil {
		t.Fatal("ambiguous comparison dates accepted")
	}
	for _, args := range []map[string]any{
		{"category": "Restaurants"}, {"limit": 0}, {"limit": 51}, {"limit": 1.5}, {"direction": "out"}, {"domain_id": 0}, {"min_amount": "0.001"}, {"min_amount": "-1.00"}, {"min_amount": "10.01", "max_amount": "10.00"}, {"tag": ""}, {"max_amount": "99999999999"},
	} {
		if _, e := agentReadArguments(args, "search"); e == nil {
			t.Fatalf("silently ignored filter: %+v", args)
		}
	}
	if _, e := agentReadArguments(map[string]any{"limit": 1}, "summary"); e == nil {
		t.Fatal("truncated total accepted")
	}
}
func TestAgentReadScopesAndExactTotalsPostgres(t *testing.T) {
	db := reimbursementTestDB(t)
	ctx := context.Background()
	base := map[string]any{"start_date": "2025-12-01", "end_date": "2025-12-31", "direction": "expense", "min_amount": "60.00", "max_amount": "120.00"}
	args := func(extra map[string]any) map[string]any {
		out := map[string]any{}
		for k, v := range base {
			out[k] = v
		}
		for k, v := range extra {
			out[k] = v
		}
		return out
	}
	read := agentSearchTransactions(ctx, db, 1, args(map[string]any{"sort": "amount", "limit": json.Number("1")}), nil)
	if read.Error != "" || !strings.Contains(read.Content, `"has_more":true`) {
		t.Fatalf("partial source list not declared: %+v", read)
	}
	var search struct {
		Transactions []struct {
			ID int64 `json:"id"`
		} `json:"transactions"`
	}
	json.Unmarshal([]byte(read.Content), &search)
	if len(search.Transactions) != 1 || search.Transactions[0].ID != 3 {
		t.Fatal(read.Content)
	}
	total := agentFinancialSummary(ctx, db, 1, base)
	if total.Error != "" || !strings.Contains(total.Content, `"expenses":180.00`) {
		t.Fatalf("complete total: %+v", total)
	}
	selected := agentFinancialSummary(ctx, db, 1, base, []int64{1})
	if selected.Error != "" || !strings.Contains(selected.Content, `"expenses":60.00`) {
		t.Fatal(selected)
	}
	for _, mode := range []string{"category", "merchant"} {
		grouped := agentFinancialSummary(ctx, db, 1, args(map[string]any{"group_by": mode}))
		if grouped.Error != "" || !strings.Contains(grouped.Content, `"expenses":180.00`) {
			t.Fatal(grouped)
		}
	}
	if _, e := db.Exec(`INSERT INTO tags(name,owner_id) VALUES('scope test',1); INSERT INTO transaction_tags(transaction_id,tag_id) SELECT 1,id FROM tags WHERE owner_id=1 AND name='scope test'`); e != nil {
		t.Fatal(e)
	}
	tagged := agentFinancialSummary(ctx, db, 1, args(map[string]any{"tag": "SCOPE TEST", "domain_id": 9001}))
	if tagged.Error != "" || !strings.Contains(tagged.Content, `"expenses":60.00`) || !strings.Contains(tagged.Content, `"category":"Restaurants TEST"`) {
		t.Fatal(tagged)
	}
	categories := agentCategoryBreakdown(ctx, db, 1, args(map[string]any{"tag": "scope test"}))
	if categories.Error != "" || !strings.Contains(categories.Content, `"amount":60.00`) {
		t.Fatal(categories)
	}
	if foreign := agentFinancialSummary(ctx, db, 1, args(map[string]any{"domain_id": 9201})); foreign.Error == "" {
		t.Fatalf("foreign category accepted: %+v", foreign)
	}
	absent := agentFinancialSummary(ctx, db, 2, args(map[string]any{"merchant": "TEST GROUP EXPENSE 120"}))
	if absent.Error != "" || !strings.Contains(absent.Content, `"available":false`) || !strings.Contains(absent.Content, `"expenses":null`) {
		t.Fatalf("owner scope/absence: %+v", absent)
	}
	for _, merchant := range []string{"TEST_%", `TEST\_%`} {
		if _, e := db.Exec(`UPDATE transactions SET merchant=$1 WHERE id=1`, merchant); e != nil {
			t.Fatal(e)
		}
		exact := agentSearchTransactions(ctx, db, 1, args(map[string]any{"merchant": merchant}), nil)
		if exact.Error != "" || !strings.Contains(exact.Content, `"returned_count":1`) {
			t.Fatalf("merchant wildcard expanded %s: %+v", merchant, exact)
		}
	}
	compared := (&Server{DB: db}).agentCompare(ctx, 1, map[string]any{"start_date_a": "2025-12-01", "end_date_a": "2025-12-31", "start_date_b": "2026-01-01", "end_date_b": "2026-01-31", "domain_id": 9001}, agentPeriod{}, []int64{1, 2})
	if compared.Error != "" || !strings.Contains(compared.Content, `"expenses_a":60.00`) || !strings.Contains(compared.Content, `"income_b":30.00`) {
		t.Fatalf("comparison lost selection/category: %+v", compared)
	}
	income := agentCategoryBreakdown(ctx, db, 1, map[string]any{"start_date": "2026-01-01", "end_date": "2026-01-31", "direction": "income"}, []int64{2, 4})
	if income.Error != "" || !strings.Contains(income.Content, `"amount":70.00`) {
		t.Fatal(income)
	}
	if _, e := db.Exec(`UPDATE transactions SET domain_id=NULL WHERE id=4`); e != nil {
		t.Fatal(e)
	}
	ranked := agentFinancialSummary(ctx, db, 1, map[string]any{"start_date": "2026-01-01", "end_date": "2026-01-31", "group_by": "category", "direction": "income"}, []int64{2, 4})
	var groups struct {
		Groups []struct {
			Key    string      `json:"key"`
			Income json.Number `json:"income"`
		} `json:"groups"`
	}
	if ranked.Error != "" || json.Unmarshal([]byte(ranked.Content), &groups) != nil || len(groups.Groups) != 2 || groups.Groups[0].Income.String() != "40.00" {
		t.Fatalf("income grouping not ranked by income: %+v", ranked)
	}

}

type agentMalformedToolModel struct {
	calls     int
	arguments string
}

func (m *agentMalformedToolModel) Name() string { return "SYNTHETIC MALFORMED TOOL" }
func (m *agentMalformedToolModel) Complete(_ context.Context, _, _ string, _ int, _ bool) (string, error) {
	m.calls++
	if m.calls == 1 {
		return `{"tool_calls":[{"name":"search_transactions","arguments":` + m.arguments + `}]}`, nil
	}
	return `{"answer":"Il faut préciser la recherche.","status":"ok"}`, nil
}
func TestAgentMalformedToolNeverBroadensSearch(t *testing.T) {
	for _, arguments := range []string{`null`, `[]`, `"invalid object"`, `42`} {
		s := &Server{}
		s.ai.client = &agentMalformedToolModel{arguments: arguments}
		// No DB: malformed arguments must not execute a query or panic on a nil map.
		result, e := s.orchestrate(context.Background(), nil, 1, nil, "Cherche une opération")
		if e != nil || len(result.Steps) != 1 || result.Steps[0].Status != "error" || len(result.Evidence) != 0 || result.Status != "partial" {
			t.Fatalf("arguments=%s: %+v %v", arguments, result, e)
		}
	}
}

func TestAgentFrenchDatesGroundedPostgres(t *testing.T) {
	db := reimbursementTestDB(t)
	s := &Server{DB: db}
	s.ai.client = scriptedLLM{reply: `{"answer":"Unused model for the simple top query.","status":"ok"}`}
	for _, q := range []string{"Donne-moi ma top dépense du 1 au 10 décembre 2025", "Donne-moi ma top dépense du 01/12/2025 au 10/12/2025"} {
		b, _ := json.Marshal(map[string]any{"messages": []map[string]string{{"role": "user", "content": q}}})
		w := agentReq(t, s, s.chatHandler, 1, "POST", "/", string(b), 0, 200)
		if !strings.Contains(w.Body.String(), "60,00") || !strings.Contains(w.Body.String(), `"from":"2025-12-01"`) || !strings.Contains(w.Body.String(), `"to":"2025-12-10"`) {
			t.Fatalf("natural dates lost: %s", w.Body.String())
		}
	}
	w := agentReq(t, s, s.chatHandler, 1, "POST", "/", `{"messages":[{"role":"user","content":"ma top dépense le 31 novembre 2025"}]}`, 0, 200)
	if !strings.Contains(w.Body.String(), `"status":"needs_clarification"`) {
		t.Fatal(w.Body.String())
	}
}
