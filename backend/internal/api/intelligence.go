package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"themis/internal/money"
)

func (s *Server) natureBreakdown(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	month, err := s.analysisMonth(r.Context(), s.ownerID(r), r.URL.Query().Get("month"), now)
	if err != nil {
		analysisError(w, r, err)
		return
	}
	report, err := s.loadAnalysisReport(r.Context(), s.ownerID(r), month, now)
	if err != nil {
		writeErr(w, 500, fmt.Errorf("rapport analytique indisponible"))
		return
	}
	writeJSON(w, 200, report)
}
func analysisError(w http.ResponseWriter, r *http.Request, err error) {
	if _, _, _, e := reliabilityWindow(r.URL.Query().Get("month"), time.Now()); e != nil {
		writeErr(w, 400, e)
		return
	}
	writeErr(w, 500, fmt.Errorf("période analytique indisponible"))
}

type analysisComparisonCategory struct {
	ID          *int64       `json:"id"`
	Name        string       `json:"name"`
	Current     json.Number  `json:"current"`
	Average     *json.Number `json:"average"`
	Delta       *json.Number `json:"delta"`
	DeltaPct    *json.Number `json:"delta_pct"`
	SourceCount int          `json:"source_count"`
}

func (s *Server) categoryComparison(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	owner := s.ownerID(r)
	month, err := s.analysisMonth(r.Context(), owner, r.URL.Query().Get("month"), now)
	if err != nil {
		analysisError(w, r, err)
		return
	}
	tx, err := s.DB.BeginTx(r.Context(), &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		writeErr(w, 500, fmt.Errorf("comparaison indisponible"))
		return
	}
	defer tx.Rollback()
	current, err := analysisReportFrom(r.Context(), tx, owner, month, now)
	if err != nil {
		writeErr(w, 500, fmt.Errorf("comparaison indisponible"))
		return
	}
	first, _, _, _ := reliabilityWindow(month, now)
	refs := []string{}
	for i := 3; i > 0; i-- {
		refs = append(refs, first.AddDate(0, -i, 0).Format("2006-01"))
	}
	comparable := current.Period.Completeness == "complete"
	reason := "Comparaison sur trois mois civils documentés, mois sans opération inclus."
	if !comparable {
		reason = "Le mois sélectionné n'est pas intégralement documenté. Aucune comparaison avec des mois complets."
	}
	entries := map[string]*analysisComparisonCategory{}
	totals := map[string]money.Cents{}
	keyOf := func(id *int64) string {
		if id == nil {
			return "none"
		}
		return strconv.FormatInt(*id, 10)
	}
	for _, c := range current.Categories {
		key := keyOf(c.ID)
		entries[key] = &analysisComparisonCategory{ID: c.ID, Name: c.Name, Current: c.Spent, SourceCount: c.Count}
	}
	if comparable {
		for _, ref := range refs {
			past, e := analysisReportFrom(r.Context(), tx, owner, ref, now)
			if e != nil {
				writeErr(w, 500, fmt.Errorf("référence historique indisponible"))
				return
			}
			if past.Period.Completeness != "complete" {
				comparable = false
				reason = "Au moins un des trois mois de référence a une couverture inconnue ou partielle."
			}
			for _, c := range past.Categories {
				key := keyOf(c.ID)
				if entries[key] == nil {
					entries[key] = &analysisComparisonCategory{ID: c.ID, Name: c.Name, Current: "0.00"}
				}
				amount, _ := parseAnalysisAmount(string(c.Spent))
				totals[key] += amount
			}
		}
	}
	keys := make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	values := make([]money.Cents, len(keys))
	for i, key := range keys {
		values[i] = totals[key]
	}
	averages := analysisApportion(values, 3)
	cats := []analysisComparisonCategory{}
	for i, key := range keys {
		c := entries[key]
		if comparable {
			avg := simulationNumber(averages[i])
			cur, _ := parseAnalysisAmount(string(c.Current))
			delta := simulationNumber(cur - averages[i])
			c.Average = &avg
			c.Delta = &delta
			c.DeltaPct = analysisPercent(cur-averages[i], averages[i])
		}
		cats = append(cats, *c)
	}
	sort.SliceStable(cats, func(i, j int) bool {
		a, _ := parseAnalysisAmount(string(cats[i].Current))
		b, _ := parseAnalysisAmount(string(cats[j].Current))
		return a > b
	})
	if err = tx.Commit(); err != nil {
		writeErr(w, 500, fmt.Errorf("comparaison interrompue"))
		return
	}
	denominator := 0
	if comparable {
		denominator = 3
	}
	writeJSON(w, 200, map[string]any{"available": current.Available, "month": month, "query_from": current.From, "query_to": current.To, "period": current.Period, "comparable": comparable, "reason": reason, "reference_months": refs, "reference_from": first.AddDate(0, -3, 0).Format("2006-01-02"), "reference_to": first.AddDate(0, 0, -1).Format("2006-01-02"), "denominator": denominator, "categories": cats})
}

func (s *Server) merchantsView(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	owner := s.ownerID(r)
	month, err := s.analysisMonth(r.Context(), owner, r.URL.Query().Get("month"), now)
	if err != nil {
		analysisError(w, r, err)
		return
	}
	months := 3
	if raw := r.URL.Query().Get("months"); raw != "" {
		months, err = strconv.Atoi(raw)
		if err != nil || months < 1 || months > 24 {
			writeErr(w, 400, fmt.Errorf("months doit être compris entre 1 et 24"))
			return
		}
	}
	first, last, current, _ := reliabilityWindow(month, now)
	first = first.AddDate(0, 1-months, 0)
	previousFirst := first.AddDate(0, -months, 0)
	previousLast := first.AddDate(0, 0, -1)
	tx, err := s.DB.BeginTx(r.Context(), &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		writeErr(w, 500, fmt.Errorf("marchands indisponibles"))
		return
	}
	defer tx.Rollback()
	coverage, err := loadReliabilityRangeFrom(r.Context(), tx, owner, previousFirst, last, current)
	if err != nil {
		writeErr(w, 500, fmt.Errorf("couverture des marchands indisponible"))
		return
	}
	comparable := !current
	required := 0
	for _, account := range coverage.Accounts {
		if account.TrackingMode == "manual" {
			continue
		}
		required++
		if account.Status != "documented" {
			comparable = false
		}
	}
	comparable = comparable && required > 0
	if comparable {
		var orphan bool
		err = tx.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM transactions t WHERE t.owner_id=$1 AND t.tx_date BETWEEN $2::date AND $3::date
 AND NOT EXISTS(SELECT 1 FROM statements st JOIN accounts a ON a.id=st.account_id AND a.owner_id=st.owner_id WHERE st.id=t.statement_id AND st.owner_id=t.owner_id))`, owner, previousFirst.Format("2006-01-02"), last.Format("2006-01-02")).Scan(&orphan)
		if err != nil {
			writeErr(w, 500, fmt.Errorf("sources des marchands indisponibles"))
			return
		}
		comparable = !orphan
	}
	rows, err := tx.QueryContext(r.Context(), `WITH source AS (
 SELECT COALESCE(NULLIF(t.merchant,''),NULLIF(t.display_label,''),t.label) AS name,t.tx_date,-t.amount AS spent
 FROM transactions t WHERE t.owner_id=$1 AND t.amount<0 AND t.tx_date BETWEEN $2::date AND $4::date`+notTransferT+`),
 cur AS (SELECT name,count(*) AS visits,SUM(spent) AS spent,max(tx_date)::text AS last_seen FROM source WHERE tx_date>=$3::date GROUP BY name),
 prev AS (SELECT name,SUM(spent) AS spent FROM source WHERE tx_date<$3::date GROUP BY name)
 SELECT cur.name,cur.visits,cur.spent::text,cur.last_seen,COALESCE(prev.spent,0)::text,SUM(cur.spent) OVER()::text,count(*) OVER()
 FROM cur LEFT JOIN prev ON prev.name=cur.name ORDER BY cur.spent DESC,cur.name LIMIT 40`, owner, previousFirst.Format("2006-01-02"), first.Format("2006-01-02"), last.Format("2006-01-02"))
	if err != nil {
		writeErr(w, 500, fmt.Errorf("marchands indisponibles"))
		return
	}
	merchants := []map[string]any{}
	var fullTotal money.Cents
	var merchantCount int
	for rows.Next() {
		var name, raw, lastSeen, previous, total string
		var count int
		if err = rows.Scan(&name, &count, &raw, &lastSeen, &previous, &total, &merchantCount); err != nil {
			rows.Close()
			writeErr(w, 500, fmt.Errorf("marchands invalides"))
			return
		}
		spent, e := parseAnalysisAmount(raw)
		if e != nil {
			rows.Close()
			writeErr(w, 500, e)
			return
		}
		fullTotal, e = parseAnalysisAmount(total)
		if e != nil {
			rows.Close()
			writeErr(w, 500, e)
			return
		}
		var previousAmount any
		var trend *json.Number
		if comparable {
			amount, e := parseAnalysisAmount(previous)
			if e != nil {
				rows.Close()
				writeErr(w, 500, e)
				return
			}
			previousAmount = simulationNumber(amount)
			trend = analysisPercent(spent-amount, amount)
		}
		merchants = append(merchants, map[string]any{"name": name, "visits": count, "total": simulationNumber(spent), "avg_basket": analysisMean(spent, count), "last_seen": lastSeen, "prev_total": previousAmount, "trend_pct": trend, "share_pct": analysisPercent(spent, fullTotal)})
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		writeErr(w, 500, fmt.Errorf("marchands interrompus"))
		return
	}
	if err = tx.Commit(); err != nil {
		writeErr(w, 500, fmt.Errorf("rapport des marchands interrompu"))
		return
	}
	reason := "Montants observés ; panier moyen par transaction. Tendances non établies sans couverture documentaire commune."
	if comparable {
		reason = "Deux fenêtres de même durée, intégralement documentées ; panier moyen par transaction, variation des totaux observés."
	}
	writeJSON(w, 200, map[string]any{"month": month, "months": months, "query_from": first.Format("2006-01-02"), "query_to": last.Format("2006-01-02"), "reference_from": previousFirst.Format("2006-01-02"), "reference_to": previousLast.Format("2006-01-02"), "merchants": merchants, "total": simulationNumber(fullTotal), "merchant_count": merchantCount, "truncated": merchantCount > 40, "comparable": comparable, "reason": reason})
}

func atoiDefault(s string, def int) int {
	n, e := strconv.Atoi(s)
	if e != nil || n < 0 {
		return def
	}
	return n
}
