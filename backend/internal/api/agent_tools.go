package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"themis/internal/money"
)

func agentToolDates(args map[string]any) (agentPeriod, error) {
	a, _ := args["start_date"].(string)
	b, _ := args["end_date"].(string)
	p := agentPeriod{From: a, To: b}
	if !agentPeriodValid(p) {
		return p, fmt.Errorf("période invalide : dates ISO ordonnées, dix ans maximum")
	}
	return p, nil
}
func agentSearchTransactions(ctx context.Context, db *sql.DB, owner int64, args map[string]any, ids []int64) toolResult {
	p, e := agentToolDates(args)
	if e != nil {
		return toolResult{Error: e.Error()}
	}
	o, e := agentReadArguments(args, "search")
	if e != nil {
		return toolResult{Error: e.Error()}
	}
	if e = agentReadDomain(ctx, db, owner, &o); e != nil {
		return toolResult{Error: e.Error()}
	}
	limit := 20
	if o.Limit != nil {
		limit = *o.Limit
	}
	order := "t.tx_date DESC,t.id DESC"
	if o.Sort == "amount" {
		order = "abs(t.amount) DESC,t.id DESC"
	}
	params := append(o.params(owner, p, ids), limit+1)
	rows, e := db.QueryContext(ctx, `SELECT t.id,t.tx_date::text,COALESCE(NULLIF(t.display_label,''),t.label),t.amount::text,COALESCE(d.name,'Sans catégorie') FROM transactions t LEFT JOIN domains d ON d.id=t.domain_id AND d.owner_id=t.owner_id`+agentReadWhereSQL+` ORDER BY `+order+` LIMIT $11`, params...)
	if e != nil {
		return toolResult{Error: "transactions indisponibles"}
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id int64
		var date, label, amount, cat string
		if rows.Scan(&id, &date, &label, &amount, &cat) != nil {
			return toolResult{Error: "lecture interrompue"}
		}
		if _, e = money.Parse(amount); e != nil {
			return toolResult{Error: "montant source invalide"}
		}
		out = append(out, map[string]any{"id": id, "date": date, "label": label, "amount": json.Number(amount), "category": cat})
	}
	if rows.Err() != nil {
		return toolResult{Error: "lecture interrompue"}
	}
	more := len(out) > limit
	if more {
		out = out[:limit]
	}
	return toolResult{Content: string(agentJSON(map[string]any{"transactions": out, "has_more": more, "limit": limit, "returned_count": len(out), "filters": o.metadata(), "query_from": p.From, "query_to": p.To, "coverage": "unknown", "source": "transactions PostgreSQL du compte connecté, virements internes et épargne exclus"}))}
}
func agentCategories(ctx context.Context, db *sql.DB, owner int64) toolResult {
	rows, e := db.QueryContext(ctx, `SELECT id,name FROM domains WHERE owner_id=$1 ORDER BY name,id LIMIT 200`, owner)
	if e != nil {
		return toolResult{Error: "catégories indisponibles"}
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id int64
		var name string
		if rows.Scan(&id, &name) != nil {
			return toolResult{Error: "catégories illisibles"}
		}
		out = append(out, map[string]any{"id": id, "name": name})
	}
	if rows.Err() != nil {
		return toolResult{Error: "catégories indisponibles"}
	}
	return toolResult{Content: string(agentJSON(out))}
}
func agentFilteredTotals(ctx context.Context, db analysisReader, owner int64, p agentPeriod, o agentReadOptions, ids []int64) (money.Cents, money.Cents, int, error) {
	var income, expense string
	var count int
	err := db.QueryRowContext(ctx, `SELECT COALESCE(SUM(t.amount) FILTER(WHERE t.amount>0),0)::text,COALESCE(-SUM(t.amount) FILTER(WHERE t.amount<0),0)::text,count(*) FROM transactions t`+agentReadWhereSQL, o.params(owner, p, ids)...).Scan(&income, &expense, &count)
	if err != nil {
		return 0, 0, 0, err
	}
	a, e := parseAnalysisAmount(income)
	if e != nil {
		return 0, 0, 0, e
	}
	b, e := parseAnalysisAmount(expense)
	return a, b, count, e
}
func agentFinancialSummary(ctx context.Context, db *sql.DB, owner int64, args map[string]any, scope ...[]int64) toolResult {
	p, e := agentToolDates(args)
	if e != nil {
		return toolResult{Error: e.Error()}
	}
	o, e := agentReadArguments(args, "summary")
	if e != nil {
		return toolResult{Error: e.Error()}
	}
	if e = agentReadDomain(ctx, db, owner, &o); e != nil {
		return toolResult{Error: e.Error()}
	}
	if o.Group != "" && o.Group != "none" {
		return agentGroupedSummary(ctx, db, owner, args, scope...)
	}
	a, b, count, e := agentFilteredTotals(ctx, db, owner, p, o, agentScopeIDs(scope))
	if e != nil {
		return toolResult{Error: "synthèse financière indisponible"}
	}
	if count == 0 {
		return toolResult{Content: string(agentJSON(map[string]any{"filters": o.metadata(), "available": false, "income": nil, "expenses": nil, "net": nil, "transaction_count": 0, "query_from": p.From, "query_to": p.To, "coverage": "unknown", "reason": "Aucune opération enregistrée : absence d’observation, pas zéro réel."}))}
	}
	if validateAnalysisAmounts(a-b) != nil {
		return toolResult{Error: "montants hors limites exactes"}
	}
	return toolResult{Content: string(agentJSON(map[string]any{"filters": o.metadata(), "available": true, "transaction_count": count, "income": simulationNumber(a), "expenses": simulationNumber(b), "net": simulationNumber(a - b), "query_from": p.From, "query_to": p.To, "coverage": "unknown", "source": "transactions PostgreSQL, virements et épargne exclus"}))}
}

// Grouping uses the same exact amounts, owner and exclusions as the totals.
func agentGroupedSummary(ctx context.Context, db *sql.DB, owner int64, args map[string]any, scope ...[]int64) toolResult {
	p, e := agentToolDates(args)
	if e != nil {
		return toolResult{Error: e.Error()}
	}
	o, e := agentReadArguments(args, "summary")
	if e != nil {
		return toolResult{Error: e.Error()}
	}
	if e = agentReadDomain(ctx, db, owner, &o); e != nil {
		return toolResult{Error: e.Error()}
	}
	column := "COALESCE(d.name,'Sans catégorie')"
	if args["group_by"] == "merchant" {
		column = "COALESCE(NULLIF(t.merchant,''),NULLIF(t.display_label,''),t.label)"
	}
	rows, e := db.QueryContext(ctx, `SELECT `+column+`,COALESCE(SUM(t.amount) FILTER(WHERE t.amount>0),0)::text,COALESCE(-SUM(t.amount) FILTER(WHERE t.amount<0),0)::text,count(*) FROM transactions t LEFT JOIN domains d ON d.id=t.domain_id AND d.owner_id=t.owner_id`+agentReadWhereSQL+` GROUP BY 1 ORDER BY CASE WHEN $10='income' THEN COALESCE(SUM(t.amount) FILTER(WHERE t.amount>0),0) ELSE COALESCE(-SUM(t.amount) FILTER(WHERE t.amount<0),0) END DESC,1 LIMIT 201`, o.params(owner, p, agentScopeIDs(scope))...)
	if e != nil {
		return toolResult{Error: "regroupement indisponible"}
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var key, income, expense string
		var count int
		if rows.Scan(&key, &income, &expense, &count) != nil {
			return toolResult{Error: "lecture du regroupement interrompue"}
		}
		a, e := parseAnalysisAmount(income)
		if e != nil {
			return toolResult{Error: "montant source invalide"}
		}
		b, e := parseAnalysisAmount(expense)
		if e != nil {
			return toolResult{Error: "montant source invalide"}
		}
		out = append(out, map[string]any{"key": key, "income": simulationNumber(a), "expenses": simulationNumber(b), "transaction_count": count})
	}
	if rows.Err() != nil {
		return toolResult{Error: "lecture du regroupement interrompue"}
	}
	truncated := len(out) > 200
	if truncated {
		out = out[:200]
	}
	return toolResult{Content: string(agentJSON(map[string]any{"filters": o.metadata(), "groups": out, "truncated": truncated, "query_from": p.From, "query_to": p.To, "coverage": "unknown", "source": "transactions PostgreSQL, virements et épargne exclus"}))}
}
func agentCategoryBreakdown(ctx context.Context, db *sql.DB, owner int64, args map[string]any, scope ...[]int64) toolResult {
	p, e := agentToolDates(args)
	if e != nil {
		return toolResult{Error: e.Error()}
	}
	o, e := agentReadArguments(args, "categories")
	if e != nil {
		return toolResult{Error: e.Error()}
	}
	if e = agentReadDomain(ctx, db, owner, &o); e != nil {
		return toolResult{Error: e.Error()}
	}
	rows, e := db.QueryContext(ctx, `SELECT COALESCE(d.name,'Sans catégorie'),COALESCE(SUM(abs(t.amount)),0)::text,count(*) FROM transactions t LEFT JOIN domains d ON d.id=t.domain_id AND d.owner_id=t.owner_id`+agentReadWhereSQL+` GROUP BY 1 ORDER BY SUM(abs(t.amount)) DESC,1 LIMIT 201`, o.params(owner, p, agentScopeIDs(scope))...)
	if e != nil {
		return toolResult{Error: "ventilation indisponible"}
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var name, value string
		var n int
		if rows.Scan(&name, &value, &n) != nil {
			return toolResult{Error: "ventilation illisible"}
		}
		if _, e = parseAnalysisAmount(value); e != nil {
			return toolResult{Error: "montants hors limites exactes"}
		}
		out = append(out, map[string]any{"category": name, "amount": json.Number(value), "transaction_count": n})
	}
	if rows.Err() != nil {
		return toolResult{Error: "ventilation indisponible"}
	}
	truncated := len(out) > 200
	if truncated {
		out = out[:200]
	}
	return toolResult{Content: string(agentJSON(map[string]any{"filters": o.metadata(), "categories": out, "truncated": truncated, "query_from": p.From, "query_to": p.To, "coverage": "unknown", "source": "transactions PostgreSQL, virements et épargne exclus"}))}
}
func (s *Server) agentCompare(ctx context.Context, owner int64, args map[string]any, p agentPeriod, scope ...[]int64) toolResult {
	// Retain the earlier nested argument contract for stored model conversations.
	for _, key := range []string{"a", "b"} {
		if value, ok := args["period_"+key].(map[string]any); ok {
			if args["start_date_"+key] == nil {
				args["start_date_"+key] = value["start_date"]
			}
			if args["end_date_"+key] == nil {
				args["end_date_"+key] = value["end_date"]
			}
		}
	}

	filters := map[string]any{}
	for key, value := range args {
		switch key {
		case "period_a", "period_b", "start_date_a", "end_date_a", "start_date_b", "end_date_b", "metric":
			continue
		}
		filters[key] = value
	}
	options, err := agentReadArguments(filters, "comparison")
	if err != nil {
		return toolResult{Error: err.Error()}
	}
	if err = agentReadDomain(ctx, s.DB, owner, &options); err != nil {
		return toolResult{Error: err.Error()}
	}
	get := func(a, b string, fallback agentPeriod) (agentPeriod, error) {
		from, _ := args[a].(string)
		to, _ := args[b].(string)
		if from == "" && to == "" {
			return fallback, nil
		}
		x := agentPeriod{From: from, To: to}
		if !agentPeriodValid(x) {
			return x, fmt.Errorf("période de comparaison invalide")
		}
		return x, nil
	}
	a, e := get("start_date_a", "end_date_a", p)
	if e != nil {
		return toolResult{Error: e.Error()}
	}
	start, _ := time.Parse("2006-01-02", a.From)
	previous := agentMonthRange(start.AddDate(0, -1, 0), time.Now(), "comparison")
	b, e := get("start_date_b", "end_date_b", previous)
	if e != nil {
		return toolResult{Error: e.Error()}
	}
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return toolResult{Error: "comparaison indisponible"}
	}
	defer tx.Rollback()
	ai, ae, na, e := agentFilteredTotals(ctx, tx, owner, a, options, agentScopeIDs(scope))
	if e != nil {
		return toolResult{Error: "première période indisponible"}
	}
	bi, be, nb, e := agentFilteredTotals(ctx, tx, owner, b, options, agentScopeIDs(scope))
	if e != nil {
		return toolResult{Error: "seconde période indisponible"}
	}
	nullable := func(n int, v money.Cents) any {
		if n == 0 {
			return nil
		}
		return simulationNumber(v)
	}
	out := map[string]any{"filters": options.metadata(), "period_a": a, "period_b": b, "income_a": nullable(na, ai), "expenses_a": nullable(na, ae), "income_b": nullable(nb, bi), "expenses_b": nullable(nb, be), "count_a": na, "count_b": nb, "comparable": false, "reason": "Couverture comparable non établie. Présenter les observations séparément, sans conclure à une hausse/baisse ni calculer un pourcentage."}
	fullMonth := func(p agentPeriod) bool {
		first, e := time.Parse("2006-01-02", p.From)
		return e == nil && first.Day() == 1 && first.AddDate(0, 1, -1).Format("2006-01-02") == p.To && p.To < time.Now().Format("2006-01-02")
	}
	if na > 0 && nb > 0 && fullMonth(a) && fullMonth(b) {
		ra, ea := analysisReportFrom(ctx, tx, owner, a.From[:7], time.Now())
		rb, eb := analysisReportFrom(ctx, tx, owner, b.From[:7], time.Now())
		if ea != nil || eb != nil {
			return toolResult{Error: "couverture de comparaison indisponible"}
		}
		if ra.Period.Completeness == "complete" && rb.Period.Completeness == "complete" {
			if validateAnalysisAmounts(ae-be, ai-bi) != nil {
				return toolResult{Error: "écart hors limites exactes"}
			}
			out["comparable"] = true
			out["reason"] = "Deux mois civils terminés et documentés."
			out["expense_change"] = simulationNumber(ae - be)
			out["expense_change_percent"] = analysisPercent(ae-be, be)
			out["income_change"] = simulationNumber(ai - bi)
		}
	}
	if tx.Commit() != nil {
		return toolResult{Error: "comparaison interrompue"}
	}
	return toolResult{Content: string(agentJSON(out))}
}
func agentDataQuality(ctx context.Context, db *sql.DB, owner int64) toolResult {
	var count int
	var first, last sql.NullString
	e := db.QueryRowContext(ctx, `SELECT count(*),min(tx_date)::text,max(tx_date)::text FROM transactions WHERE owner_id=$1`, owner).Scan(&count, &first, &last)
	if e != nil {
		return toolResult{Error: "qualité des données indisponible"}
	}
	return toolResult{Content: string(agentJSON(map[string]any{"transaction_count": count, "first_transaction": agentNullableString(first), "last_transaction": agentNullableString(last), "coverage": "unknown", "reason": "Des opérations importées ne prouvent pas une période complète."}))}
}
