package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"time"

	"themis/internal/money"
)

type projectSlice struct {
	Name  string `json:"name"`
	Cents int64  `json:"amount_cents"`
	Count int    `json:"count"`
}
type projectObservation struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Text     string         `json:"text"`
	Values   map[string]any `json:"values"`
	Source   map[string]any `json:"source"`
	Validity string         `json:"validity"`
}
type projectReport struct {
	Project        financialProject     `json:"project"`
	From           string               `json:"query_from"`
	To             string               `json:"query_to"`
	Last           string               `json:"last_observed"`
	Coverage       string               `json:"coverage"`
	CoverageReason string               `json:"coverage_reason"`
	Categories     []projectSlice       `json:"by_category"`
	Merchants      []projectSlice       `json:"by_merchant"`
	Months         []projectSlice       `json:"by_month"`
	Tags           []projectSlice       `json:"by_tag"`
	Observations   []projectObservation `json:"observations"`
	Excluded       int                  `json:"excluded_confirmed_count"`
}

// Aggregates and coverage use a single repeatable-read snapshot; no LLM and no stored totals.
func (s *Server) loadProjectReport(ctx context.Context, owner, id int64, month string) (projectReport, error) {
	out := projectReport{Categories: []projectSlice{}, Merchants: []projectSlice{}, Months: []projectSlice{}, Tags: []projectSlice{}, Observations: []projectObservation{}, Coverage: "provisional", CoverageReason: "Périmètre non borné ou couverture documentaire non établie ; coût observé provisoire."}
	_, last, _, err := reliabilityWindow(month, time.Now())
	if err != nil {
		return out, projectBad(err.Error())
	}
	out.To = last.Format("2006-01-02")
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	if err = scanProject(tx.QueryRowContext(ctx, `SELECT `+projectColumns+` FROM financial_projects p WHERE p.owner_id=$1 AND p.id=$2`, owner, id), &out.Project); err != nil {
		return out, err
	}
	out.From = out.Project.Start
	if out.Project.End != "" && out.Project.End < out.To {
		out.To = out.Project.End
	}
	if out.From != "" && out.From > out.To {
		out.CoverageReason = "Le projet n'a pas encore commencé à la date consultée."
	}
	const source = ` FROM financial_projects p JOIN project_transactions pt ON pt.project_id=p.id AND pt.owner_id=p.owner_id JOIN transactions t ON t.id=pt.transaction_id AND t.owner_id=p.owner_id LEFT JOIN domains d ON d.id=t.domain_id AND d.owner_id=t.owner_id WHERE p.owner_id=$1 AND p.id=$2 AND pt.state='confirmed' AND ` + projectEligible
	// GROUPING SETS produces additive partitions and the unique total in SQL.
	rows, err := tx.QueryContext(ctx, `WITH source AS (SELECT t.id,t.tx_date,-t.amount amt,COALESCE(d.name,'Sans catégorie') category,COALESCE(NULLIF(t.merchant,''),NULLIF(t.display_label,''),t.label) merchant,to_char(t.tx_date,'YYYY-MM') period_key
 `+source+`)
 SELECT CASE WHEN GROUPING(category)=0 THEN 'category' WHEN GROUPING(merchant)=0 THEN 'merchant' WHEN GROUPING(period_key)=0 THEN 'month' ELSE 'total' END,
 COALESCE(category,merchant,period_key,''),COALESCE(SUM(amt),0)::text,count(*),COALESCE(max(tx_date)::text,'') FROM source GROUP BY GROUPING SETS((category),(merchant),(period_key),())`, owner, id, out.To)
	// Category join is kept in the source, before WHERE, to preserve one row per transaction.
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var kind, name, raw, date string
		var n int
		if err = rows.Scan(&kind, &name, &raw, &n, &date); err != nil {
			rows.Close()
			return out, err
		}
		amount, e := parseAnalysisAmount(raw)
		if e != nil {
			rows.Close()
			return out, e
		}
		item := projectSlice{name, int64(amount), n}
		switch kind {
		case "category":
			out.Categories = append(out.Categories, item)
		case "merchant":
			out.Merchants = append(out.Merchants, item)
		case "month":
			out.Months = append(out.Months, item)
		case "total":
			out.Project.Cost = int64(amount)
			out.Project.Count = n
			out.Last = date
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	projectMetrics(&out.Project)
	if err = tx.QueryRowContext(ctx, `SELECT count(*) FILTER(WHERE pt.state='confirmed' AND NOT (`+projectEligible+`)),count(*) FILTER(WHERE pt.state='suggested' AND `+projectEligible+`) FROM financial_projects p JOIN project_transactions pt ON pt.project_id=p.id AND pt.owner_id=p.owner_id JOIN transactions t ON t.id=pt.transaction_id AND t.owner_id=p.owner_id WHERE p.owner_id=$1 AND p.id=$2`, owner, id, out.To).Scan(&out.Excluded, &out.Project.Suggestions); err != nil {
		return out, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT tg.name,(-SUM(t.amount))::text,count(*) FROM tags tg JOIN transaction_tags tt ON tt.tag_id=tg.id JOIN transactions t ON t.id=tt.transaction_id AND t.owner_id=tg.owner_id JOIN project_transactions pt ON pt.transaction_id=t.id AND pt.owner_id=t.owner_id JOIN financial_projects p ON p.id=pt.project_id AND p.owner_id=pt.owner_id WHERE p.owner_id=$1 AND p.id=$2 AND pt.state='confirmed' AND `+projectEligible+` GROUP BY tg.id,tg.name`, owner, id, out.To)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var item projectSlice
		var raw string
		if err = rows.Scan(&item.Name, &raw, &item.Count); err != nil {
			rows.Close()
			return out, err
		}
		c, e := parseAnalysisAmount(raw)
		if e != nil {
			rows.Close()
			return out, e
		}
		item.Cents = int64(c)
		out.Tags = append(out.Tags, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	for _, items := range [][]projectSlice{out.Categories, out.Merchants, out.Tags} {
		sort.Slice(items, func(i, j int) bool {
			if items[i].Cents == items[j].Cents {
				return items[i].Name < items[j].Name
			}
			return items[i].Cents > items[j].Cents
		})
	}
	sort.Slice(out.Months, func(i, j int) bool { return out.Months[i].Name < out.Months[j].Name })
	if out.From != "" && out.From <= out.To {
		first, _ := time.Parse("2006-01-02", out.From)
		last, _ := time.Parse("2006-01-02", out.To)
		coverage, e := loadReliabilityRangeFrom(ctx, tx, owner, first, last, out.To == time.Now().Format("2006-01-02"))
		if e != nil {
			return out, e
		}
		complete := len(coverage.Accounts) > 0 && !coverage.Current
		for _, a := range coverage.Accounts {
			if a.Status != "documented" {
				complete = false
			}
		}
		var orphan int
		if e = tx.QueryRowContext(ctx, `SELECT count(*) FROM transactions t WHERE t.owner_id=$1 AND t.tx_date BETWEEN $2::date AND $3::date AND NOT EXISTS(SELECT 1 FROM statements st WHERE st.id=t.statement_id AND st.owner_id=t.owner_id)`, owner, out.From, out.To).Scan(&orphan); e != nil {
			return out, e
		}
		if complete && orphan == 0 {
			out.Coverage = "documented"
			out.CoverageReason = "Relevés documentés sur la période observée ; cela ne prouve pas l'exhaustivité des rattachements du projet."
		}
	}
	out.Observations = projectObservations(out)
	if err = tx.Commit(); err != nil {
		return out, err
	}
	return out, nil
}
func projectObservations(r projectReport) []projectObservation {
	out := []projectObservation{}
	p := r.Project
	add := func(kind, text string, values map[string]any, source map[string]any) {
		source["project_id"] = p.ID
		source["state"] = "confirmed"
		source["from"] = r.From
		source["to"] = r.To
		out = append(out, projectObservation{fmt.Sprintf("project:%d:%s:%s", p.ID, kind, r.To), kind, text, values, source, "Opérations confirmées éligibles uniquement. " + r.CoverageReason})
	}
	if p.Overrun != nil && *p.Overrun > 0 {
		add("budget_overrun", "Le coût confirmé observé dépasse le budget.", map[string]any{"budget_cents": p.Budget, "cost_cents": p.Cost, "overrun_cents": p.Overrun}, map[string]any{})
	}
	if p.Count >= 3 && len(r.Categories) > 0 && p.Cost > 0 && r.Categories[0].Cents*100 >= p.Cost*40 {
		c := r.Categories[0]
		add("category_concentration", "Ce poste concentre au moins 40 % du coût enregistré.", map[string]any{"category": c.Name, "amount_cents": c.Cents, "total_cents": p.Cost, "share_pct": analysisPercent(money.Cents(c.Cents), money.Cents(p.Cost))}, map[string]any{"category": c.Name})
	}
	return out
}
func (s *Server) projectReportHTTP(w http.ResponseWriter, r *http.Request) {
	id, e := projectID(r)
	if e != nil {
		projectHTTPError(w, e)
		return
	}
	out, e := s.loadProjectReport(r.Context(), s.ownerID(r), id, r.URL.Query().Get("month"))
	if e != nil {
		projectHTTPError(w, e)
		return
	}
	writeJSON(w, 200, out)
}
