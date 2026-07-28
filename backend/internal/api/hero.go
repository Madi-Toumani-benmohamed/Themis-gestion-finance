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

	// Mois de référence : le plus récent contenant des transactions.
	var ref sql.NullString
	_ = s.DB.QueryRow(`SELECT to_char(max(tx_date),'YYYY-MM') FROM transactions WHERE owner_id = $1`, owner).Scan(&ref)
	if !ref.Valid {
		writeJSON(w, http.StatusOK, map[string]any{"available": false})
		return
	}

	// Revenus / dépenses du mois (transferts exclus).
	var income, spent float64
	_ = s.DB.QueryRow(`SELECT COALESCE(SUM(amount) FILTER (WHERE amount > 0), 0),
	                          COALESCE(-SUM(amount) FILTER (WHERE amount < 0), 0)
	                   FROM transactions WHERE owner_id = $2 AND to_char(tx_date,'YYYY-MM') = $1`+notTransfer,
		ref.String, owner).Scan(&income, &spent)
	net := income - spent

	// Budgets du mois : combien sont définis, combien sont dépassés.
	var totalBudgets, overBudgets int
	_ = s.DB.QueryRow(`SELECT count(*) FROM budgets WHERE owner_id = $1`, owner).Scan(&totalBudgets)
	_ = s.DB.QueryRow(`
		SELECT count(*) FROM budgets b
		WHERE b.owner_id = $1 AND b.monthly_limit < (
			SELECT COALESCE(-SUM(t.amount),0) FROM transactions t
			WHERE t.owner_id = $1 AND t.domain_id = b.domain_id
			  AND t.amount < 0 AND to_char(t.tx_date,'YYYY-MM') = $2)`,
		owner, ref.String).Scan(&overBudgets)

	// Dépenses inhabituelles : le même détecteur que l'onglet anomalies.
	// Les nouveaux paiements récurrents sont comptés à part : c'est un signal
	// différent (un engagement qui s'installe, pas une dépense ponctuelle).
	anomalyCount, newRecurring := 0, 0
	if rec, err := computeRecurring(s.DB, owner, 2); err == nil {
		for _, a := range detectAnomalies(rec) {
			if a.Kind == "new_recurring" {
				newRecurring++
			} else {
				anomalyCount++
			}
		}
	}

	// Catégorie en forte hausse vs sa moyenne historique (>30 % et >40 €).
	var upName string
	var upCur, upAvg float64
	_ = s.DB.QueryRow(`
		WITH cur AS (
			SELECT d.name, -SUM(t.amount) AS dep
			FROM transactions t JOIN domains d ON d.id = t.domain_id AND t.owner_id = $2
			WHERE t.amount < 0 AND NOT d.is_transfer AND to_char(t.tx_date,'YYYY-MM') = $1
			GROUP BY d.name
		), hist AS (
			SELECT d.name, -SUM(t.amount) / GREATEST(COUNT(DISTINCT to_char(t.tx_date,'YYYY-MM')),1) AS avg_dep
			FROM transactions t JOIN domains d ON d.id = t.domain_id AND t.owner_id = $2
			WHERE t.amount < 0 AND NOT d.is_transfer AND to_char(t.tx_date,'YYYY-MM') < $1
			GROUP BY d.name
		)
		SELECT cur.name, cur.dep, hist.avg_dep FROM cur JOIN hist ON hist.name = cur.name
		WHERE hist.avg_dep > 0 AND cur.dep > hist.avg_dep * 1.3 AND cur.dep - hist.avg_dep > 40
		ORDER BY cur.dep - hist.avg_dep DESC LIMIT 1`,
		ref.String, owner).Scan(&upName, &upCur, &upAvg)

	// Baisse de revenus vs la moyenne des 3 mois précédents (>25 %).
	var prevIncome sql.NullFloat64
	_ = s.DB.QueryRow(`
		SELECT AVG(m.inc) FROM (
			SELECT to_char(tx_date,'YYYY-MM') AS mth, SUM(amount) FILTER (WHERE amount > 0) AS inc
			FROM transactions
			WHERE owner_id = $2 AND to_char(tx_date,'YYYY-MM') < $1`+notTransfer+`
			GROUP BY 1 ORDER BY 1 DESC LIMIT 3
		) m`, ref.String, owner).Scan(&prevIncome)
	incomeDrop := prevIncome.Valid && prevIncome.Float64 > 100 && income < prevIncome.Float64*0.75

	// ---- Composition de la synthèse ----
	points := []heroPoint{}
	if net > 0 {
		points = append(points, heroPoint{Kind: "savings", Tone: "positive",
			Text: fmt.Sprintf("Tu as épargné %s ce mois-ci.", euro(net))})
	} else if net < 0 {
		points = append(points, heroPoint{Kind: "deficit", Tone: "attention",
			Text: fmt.Sprintf("Le mois est déficitaire de %s.", euro(-net))})
	}
	if totalBudgets > 0 {
		if overBudgets == 0 {
			points = append(points, heroPoint{Kind: "budgets_ok", Tone: "positive",
				Text: "Tous tes budgets sont respectés."})
		} else {
			points = append(points, heroPoint{Kind: "budget_over", Tone: "alert",
				Text: fmt.Sprintf("%d budget(s) dépassé(s) ce mois-ci.", overBudgets)})
		}
	}
	if anomalyCount > 0 {
		plural := ""
		if anomalyCount > 1 {
			plural = "s"
		}
		points = append(points, heroPoint{Kind: "anomalies", Tone: "attention",
			Text: fmt.Sprintf("%d dépense%s inhabituelle%s détectée%s.", anomalyCount, plural, plural, plural)})
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

	// Ton global et phrase de situation : le pire état l'emporte.
	tone, headline := "sain", "Situation financière saine."
	hasAlert := overBudgets > 0 || incomeDrop
	hasAttention := anomalyCount > 0 || upName != "" || net < 0
	switch {
	case hasAlert:
		tone, headline = "alerte", "Situation sous tension ce mois-ci."
	case hasAttention:
		tone, headline = "attention", "Situation correcte, quelques points à surveiller."
	}

	// Période couverte (l'ex-ligne de couverture vit dans le hero).
	var lastTx, lastImport sql.NullString
	_ = s.DB.QueryRow(`SELECT max(tx_date)::text FROM transactions WHERE owner_id = $1`, owner).Scan(&lastTx)
	_ = s.DB.QueryRow(`SELECT max(uploaded_at)::date::text FROM statements WHERE owner_id = $1`, owner).Scan(&lastImport)
	stale := false
	if lastTx.Valid {
		if d, err := time.Parse("2006-01-02", lastTx.String); err == nil {
			stale = time.Since(d) > 35*24*time.Hour
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"available": true,
		"month":     ref.String,
		"tone":      tone,
		"headline":  headline,
		"points":    points,
		"coverage": map[string]any{
			"last_tx":     lastTx.String,
			"last_import": lastImport.String,
			"stale":       stale,
		},
	})
}
