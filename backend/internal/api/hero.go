package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"
)

// heroPoint est un élément de la synthèse : un fait saillant de l'analyse.
type heroPoint struct {
	Kind string `json:"kind"` // savings | deficit | budgets_ok | budget_over | anomalies | category_up | income_drop
	Tone string `json:"tone"` // positive | neutral | attention | alert
	Text string `json:"text"`
}

// hero produit la conclusion de l'analyse : un ton global, une phrase de
// situation, et les faits réellement importants du mois. Tout est calculé
// depuis les données (déterministe, instantané, vérifiable) — le reste de la
// page vient ensuite expliquer cette conclusion.
func (s *Server) hero(w http.ResponseWriter, r *http.Request) {
	owner := s.ownerID(r)

	// Mois de référence : paramètre 'month' (YYYY-MM) ou le plus récent contenant des transactions.
	month := r.URL.Query().Get("month")
	var ref sql.NullString
	if month != "" {
		ref = sql.NullString{String: month, Valid: true}
	} else {
		_ = s.DB.QueryRow(`SELECT to_char(max(tx_date),'YYYY-MM') FROM transactions WHERE owner_id = $1`, owner).Scan(&ref)
	}
	if !ref.Valid {
		writeJSON(w, http.StatusOK, map[string]any{"available": false})
		return
	}

	info, periodErr := s.getPeriodInfo(owner, ref.String)
	if periodErr != nil {
		writeErr(w, http.StatusBadRequest, periodErr)
		return
	}
	complete := info.Completeness == "complete"
	// Comparaisons uniquement si les trois mois de référence sont documentés.
	comparable := complete
	start, _ := time.Parse("2006-01", ref.String)
	for i := 1; i <= 3 && comparable; i++ {
		comparable = s.heroMonthCovered(owner, start.AddDate(0, -i, 0).Format("2006-01"))
	}

	// Revenus / dépenses du mois (transferts exclus) — période observée.
	var income, spent float64
	dayFilter := ""
	if info.DaysObserved > 0 && info.IsCurrent {
		dayFilter = fmt.Sprintf(" AND EXTRACT(DAY FROM tx_date) <= %d", info.DaysObserved)
	}
	q := `SELECT COALESCE(SUM(amount) FILTER (WHERE amount > 0), 0),
		       COALESCE(-SUM(amount) FILTER (WHERE amount < 0), 0)
		FROM transactions WHERE owner_id = $2 AND to_char(tx_date,'YYYY-MM') = $1` + notTransfer + dayFilter
	if err := s.DB.QueryRow(q, ref.String, owner).Scan(&income, &spent); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	net := income - spent

	// Le dashboard et la page Budgets partagent les mêmes périmètres et seuils.
	budgetReport, budgetErr := s.loadBudgetReport(r.Context(), owner, ref.String, time.Now())
	if budgetErr != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Errorf("lecture des budgets indisponible"))
		return
	}
	totalBudgets, overBudgets := 0, budgetReport.Summary.OverCount
	for _, budget := range budgetReport.Budgets {
		if budget.Scope == "expense" {
			totalBudgets++
		}
	}

	// Dépenses inhabituelles : le même détecteur que l'onglet anomalies.
	anomalyCount, newRecurring := 0, 0
	if rec, err := computeRecurring(s.DB, owner, 2); err == nil {
		// Le détecteur travaille sur les dernières opérations : ne pas attribuer
		// à un mois historique des signaux apparus après celui-ci.
		selected := []recurring{}
		for _, g := range rec {
			if len(g.LastDate) >= 7 && g.LastDate[:7] == ref.String {
				selected = append(selected, g)
			}
		}
		for _, a := range detectAnomalies(selected) {
			if a.Kind == "subscription_gone" {
				continue
			}
			if a.Kind == "new_recurring" {
				newRecurring++
			} else {
				anomalyCount++
			}
		}
	}

	// Catégorie en forte hausse vs sa moyenne historique (>30 % et >40 €) — seulement pour mois complet.
	var upName string
	var upCur, upAvg float64
	if comparable {
		q = `WITH cur AS (
				SELECT d.name, -SUM(t.amount) AS dep
				FROM transactions t JOIN domains d ON d.id = t.domain_id AND t.owner_id = $2
				WHERE t.amount < 0 AND NOT d.is_transfer AND to_char(t.tx_date,'YYYY-MM') = $1
				GROUP BY d.name
			), hist AS (
				SELECT d.name, -SUM(t.amount) / GREATEST(COUNT(DISTINCT to_char(t.tx_date,'YYYY-MM')),1) AS avg_dep
				FROM transactions t JOIN domains d ON d.id = t.domain_id AND t.owner_id = $2
				WHERE t.amount < 0 AND NOT d.is_transfer AND to_char(t.tx_date,'YYYY-MM') < $1 AND t.tx_date >= ($1 || '-01')::date - interval '3 months'
				GROUP BY d.name
			)
			SELECT cur.name, cur.dep, hist.avg_dep FROM cur JOIN hist ON hist.name = cur.name
			WHERE hist.avg_dep > 0 AND cur.dep > hist.avg_dep * 1.3 AND cur.dep - hist.avg_dep > 40
			ORDER BY cur.dep - hist.avg_dep DESC LIMIT 1`
		_ = s.DB.QueryRow(q, ref.String, owner).Scan(&upName, &upCur, &upAvg)
	}

	// Baisse de revenus vs la moyenne des 3 mois précédents (>25 %) — seulement pour mois complet.
	var prevIncome sql.NullFloat64
	var incomeDrop bool
	if comparable {
		q = `SELECT AVG(m.inc) FROM (
				SELECT to_char(tx_date,'YYYY-MM') AS mth, SUM(amount) FILTER (WHERE amount > 0) AS inc
				FROM transactions
				WHERE owner_id = $2 AND to_char(tx_date,'YYYY-MM') < $1 AND tx_date >= ($1 || '-01')::date - interval '3 months'` + notTransfer + `
				GROUP BY 1 ORDER BY 1 DESC LIMIT 3
			) m`
		_ = s.DB.QueryRow(q, ref.String, owner).Scan(&prevIncome)
		incomeDrop = prevIncome.Valid && prevIncome.Float64 > 100 && income < prevIncome.Float64*0.75
	}

	// ---- Composition de la synthèse ----
	points := []heroPoint{}
	if totalBudgets > 0 {
		if overBudgets == 0 {
			points = append(points, heroPoint{Kind: "budgets_ok", Tone: "positive",
				Text: "Aucun dépassement budgétaire dans les flux enregistrés."})
		} else {
			// Le dépassement est observé, indépendamment du calendrier de salaire.
			tone := "alert"
			if info.IsCurrent {
				tone = "attention"
			}
			points = append(points, heroPoint{Kind: "budget_over", Tone: tone,
				Text: fmt.Sprintf("%d budget(s) dépassé(s) sur la période.", overBudgets)})
		}
	}
	if anomalyCount > 0 {
		plural := ""
		if anomalyCount > 1 {
			plural = "s"
		}
		points = append(points, heroPoint{Kind: "anomalies", Tone: "attention",
			Text: fmt.Sprintf("%d dépense%s potentiellement inhabituelle%s à examiner.", anomalyCount, plural, plural)})
	}
	if newRecurring > 0 {
		if newRecurring == 1 {
			points = append(points, heroPoint{Kind: "new_recurring", Tone: "attention",
				Text: "Un nouveau paiement récurrent semble apparaître."})
		} else {
			points = append(points, heroPoint{Kind: "new_recurring", Tone: "attention",
				Text: fmt.Sprintf("%d nouveaux paiements récurrents semblent apparaître.", newRecurring)})
		}
	}
	if upName != "" {
		points = append(points, heroPoint{Kind: "category_up", Tone: "attention",
			Text: fmt.Sprintf("Les dépenses %s sont en forte hausse (+%.0f %% vs habitude).", upName, (upCur-upAvg)/upAvg*100)})
	}
	if incomeDrop {
		points = append(points, heroPoint{Kind: "income_drop", Tone: "alert",
			Text: fmt.Sprintf("Revenus en baisse de %.0f %% par rapport aux mois précédents.", (1-income/prevIncome.Float64)*100)})
	}

	tone, headline := heroDiagnosis(info.IsCurrent, complete, net, points)

	// Période couverte.
	var lastTx, lastImport sql.NullString
	_ = s.DB.QueryRow(`SELECT max(tx_date)::text FROM transactions WHERE owner_id = $1 AND to_char(tx_date,'YYYY-MM') = $2`, owner, ref.String).Scan(&lastTx)
	_ = s.DB.QueryRow(`SELECT max(uploaded_at)::date::text FROM statements WHERE owner_id = $1`, owner).Scan(&lastImport)
	stale := false
	if info.IsCurrent && lastTx.Valid {
		if d, err := time.Parse("2006-01-02", lastTx.String); err == nil {
			stale = time.Since(d) > 35*24*time.Hour
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"available":       true,
		"month":           ref.String,
		"income":          income,
		"expenses":        spent,
		"net":             net,
		"period_complete": complete,
		"period":          info,
		"tone":            tone,
		"headline":        headline,
		"points":          points,
		"coverage": map[string]any{
			"last_tx":     lastTx.String,
			"last_import": lastImport.String,
			"stale":       stale,
		},
	})
}
