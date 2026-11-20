package api

import (
	"context"
	"testing"
)

func TestAgentPeriodBriefingScopedPostgres(t *testing.T) {
	db := reimbursementTestDB(t)
	period := agentPeriod{From: "2025-12-01", To: "2025-12-31", Origin: "page"}
	read := func(owner int64, p agentPeriod) agentPeriodBriefing {
		t.Helper()
		v, e := agentReadPeriodBriefing(context.Background(), db, owner, p, "Europe/Paris")
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	v := read(1, period)
	if !v.Available || v.TransactionCount != 5 || v.ExpenseCount != 4 || v.UnclassifiedCount != 0 || len(v.UnclassifiedIDs) != 0 || v.Coverage != "unknown" {
		t.Fatalf("incorrect observations: %+v", v)
	}
	if _, e := db.Exec(`UPDATE transactions SET domain_id=NULL WHERE id IN(1,12)`); e != nil {
		t.Fatal(e)
	}
	v = read(1, period)
	if v.UnclassifiedCount != 1 || len(v.UnclassifiedIDs) != 1 || v.UnclassifiedIDs[0] != 1 {
		t.Fatalf("wrong source or owner: %+v", v)
	}
	foreign := read(2, period)
	if foreign.TransactionCount != 2 || foreign.ExpenseCount != 1 || len(foreign.UnclassifiedIDs) != 1 || foreign.UnclassifiedIDs[0] != 12 {
		t.Fatalf("owner isolation: %+v", foreign)
	}
	income := read(1, agentPeriod{From: "2026-01-01", To: "2026-01-31"})
	if !income.Available || income.ExpenseCount != 0 || income.TransactionCount != 7 {
		t.Fatalf("income-only mistaken for missing data: %+v", income)
	}
	empty := read(1, agentPeriod{From: "2025-11-01", To: "2025-11-30"})
	if empty.Available || empty.TransactionCount != 0 || empty.UnclassifiedIDs == nil {
		t.Fatalf("missing data: %+v", empty)
	}
	// 100 is a navigation bound, never the total. Stable ordering and a sentinel
	// distinguish a complete list of exactly 100 from a truncated list.
	if _, e := db.Exec(`INSERT INTO transactions(id,statement_id,owner_id,tx_date,label,amount,hash)
 SELECT 1000+n,101,1,'2025-12-21','SYNTHETIC BRIEFING',-1,'briefing-'||n FROM generate_series(1,99) n`); e != nil {
		t.Fatal(e)
	}
	v = read(1, period)
	if v.UnclassifiedCount != 100 || len(v.UnclassifiedIDs) != 100 || v.HasMore || v.UnclassifiedIDs[0] != 1099 {
		t.Fatalf("exact bound: %+v", v)
	}
	if _, e := db.Exec(`INSERT INTO transactions(id,statement_id,owner_id,tx_date,label,amount,hash) VALUES(1100,101,1,'2025-12-22','SYNTHETIC BRIEFING',-1,'briefing-100')`); e != nil {
		t.Fatal(e)
	}
	v = read(1, period)
	if v.UnclassifiedCount != 101 || len(v.UnclassifiedIDs) != 100 || !v.HasMore || v.UnclassifiedIDs[0] != 1100 {
		t.Fatalf("partial list: %+v", v)
	}
	// Savings/transfer exclusions remain identical to financial reports.
	if _, e := db.Exec(`UPDATE transactions SET domain_id=NULL WHERE id=15`); e != nil {
		t.Fatal(e)
	}
	feb := read(1, agentPeriod{From: "2026-02-01", To: "2026-02-28"})
	if feb.ExpenseCount != 1 || feb.UnclassifiedCount != 0 {
		t.Fatalf("savings or zero classified as expenses: %+v", feb)
	}
	s := &Server{DB: db}
	w := agentReq(t, s, s.agentBriefing, 1, "GET", "/api/agent/briefing?month=2025-12&timezone=Europe%2FParis", "", 0, 200)
	body := agentBody[struct {
		Summary agentPeriodBriefing `json:"period_summary"`
	}](t, w)
	if body.Summary.TransactionCount != 105 || body.Summary.Period.From != "2025-12-01" || body.Summary.Timezone != "Europe/Paris" || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal(w.Body.String())
	}
	for _, q := range []string{"month=oops", "month=2025-13", "month=2099-12", "timezone=NotAZone"} {
		agentReq(t, s, s.agentBriefing, 1, "GET", "/api/agent/briefing?"+q, "", 0, 400)
	}
}
