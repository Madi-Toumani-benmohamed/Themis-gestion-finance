package api

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"
)

// periodInfo représente l'état d'une période mensuelle.
type periodInfo struct {
	Completeness  string  `json:"completeness"`   // complete | partial | unknown
	Month         string  `json:"month"`          // YYYY-MM
	IsCurrent     bool    `json:"is_current"`     // vrai si mois en cours
	LastTxDate    string  `json:"last_tx_date"`   // dernière transaction observée (YYYY-MM-DD)
	DaysObserved  int     `json:"days_observed"`  // jours écoulés dans le mois
	DaysInMonth   int     `json:"days_in_month"`  // jours total dans le mois
	CoverageRatio float64 `json:"coverage_ratio"` // DaysObserved / DaysInMonth
	Status        string  `json:"status"`         // "current" | "complete" | "incomplete"
}

// getPeriodInfo retourne l'info de période pour un mois donné (ou le mois de référence).
func (s *Server) getPeriodInfo(owner int64, month string) (periodInfo, error) {
	now := time.Now()
	var refMonth string
	if month == "" {
		// Mois de référence = mois le plus récent contenant des transactions.
		if err := s.DB.QueryRow(`SELECT to_char(max(tx_date), 'YYYY-MM') FROM transactions WHERE owner_id = $1`, owner).Scan(&refMonth); err != nil || refMonth == "" {
			refMonth = now.Format("2006-01")
		}
	} else {
		refMonth = month
	}

	first, _, isCurrent, err := reliabilityWindow(refMonth, now)
	if err != nil {
		return periodInfo{}, err
	}
	var lastTx sql.NullString
	if err := s.DB.QueryRow(`SELECT max(tx_date)::text FROM transactions WHERE owner_id=$1 AND to_char(tx_date,'YYYY-MM')=$2`, owner, refMonth).Scan(&lastTx); err != nil {
		return periodInfo{}, err
	}
	report, err := s.loadReliability(context.Background(), owner, refMonth, now)
	if err != nil {
		return periodInfo{}, err
	}
	return periodFromReliability(report, lastTx.String, first.AddDate(0, 1, -1).Day(), isCurrent, now.Day()), nil
}

// La présence d'une opération ne prouve aucun jour de couverture.
func periodFromReliability(report reliabilityReport, lastTx string, days int, current bool, today int) periodInfo {
	info := periodInfo{Month: report.Month, IsCurrent: current, LastTxDate: lastTx, DaysInMonth: days, Completeness: "unknown", Status: "incomplete"}
	documented := false
	required := 0
	covered := days
	for _, account := range report.Accounts {
		if account.TrackingMode == "manual" {
			continue
		}
		required++
		if required == 1 {
			documented = true
		}
		if account.Status != "documented" {
			documented = false
		}
		if len(account.Statements) > 0 && info.Completeness == "unknown" {
			info.Completeness = "partial"
		}
	}
	if current {
		covered = today
		info.DaysObserved = today // jours écoulés, pas une preuve de couverture
		info.Status = "current"
	}
	if documented {
		info.DaysObserved = covered
		info.CoverageRatio = float64(covered) / float64(days)
		info.Completeness = "partial"
		if !current {
			info.Completeness = "complete"
			info.Status = "complete"
		}
	}
	return info
}

// getComparableMonths retourne les mois historiques comparables à un mois de référence.
// Pour un mois courant (incomplet), on compare aux mêmes jours des mois précédents.
// Pour un mois complet, on compare aux mois complets.
func (s *Server) getComparableMonths(owner int64, refMonth string, daysObserved int, limit int) []string {
	if !s.heroMonthCovered(owner, refMonth) {
		return nil
	}
	// Candidats historiques ; la couverture est vérifiée après fermeture du curseur.
	var months []string
	// Audited: compile-time SQL constants only; all external values use PostgreSQL bind parameters.
	// nosemgrep: go.lang.security.audit.database.string-formatted-query.string-formatted-query
	const query = `
		SELECT to_char(tx_date,'YYYY-MM') AS mth
		FROM transactions
		WHERE owner_id = $1 AND to_char(tx_date,'YYYY-MM') < $2` + notTransfer + `
		GROUP BY 1
		ORDER BY 1 DESC
		LIMIT $3
	`
	rows, err := s.DB.Query(query, owner, refMonth, 60)
	if err != nil {
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var m string
		if err := rows.Scan(&m); err == nil {
			months = append(months, m)
		}
	}
	if rows.Err() != nil {
		return nil
	}
	rows.Close()
	comparable := []string{}
	for _, month := range months {
		if s.heroMonthCovered(owner, month) {
			comparable = append(comparable, month)
		}
		if len(comparable) >= limit {
			break
		}
	}
	return comparable
}

// comparePeriod compare une métrique du mois de référence à sa moyenne historique
// sur des périodes comparables (mêmes jours du mois).
func (s *Server) comparePeriod(owner int64, refMonth string, daysObserved int, metric string, excludeTransfers bool) (current, average float64, hasHistory bool) {
	if metric != "income" && metric != "expenses" && metric != "net" {
		return 0, 0, false
	}
	months := s.getComparableMonths(owner, refMonth, daysObserved, 6)
	if len(months) == 0 {
		return 0, 0, false
	}

	// Requête pour le mois de référence (limité aux jours observés).
	var currentVal float64
	dayFilter := ""
	if daysObserved > 0 {
		dayFilter = fmt.Sprintf(" AND EXTRACT(DAY FROM tx_date) <= %d", daysObserved)
	}
	transferFilter := ""
	if excludeTransfers {
		transferFilter = notTransfer
	}

	metricSQL := ""
	switch metric {
	case "income":
		metricSQL = "SUM(amount) FILTER (WHERE amount > 0)"
	case "expenses":
		metricSQL = "-SUM(amount) FILTER (WHERE amount < 0)"
	case "net":
		metricSQL = "SUM(amount)"
	}

	// Valeur courante.
	err := s.DB.QueryRow(fmt.Sprintf(`
		SELECT COALESCE(%s, 0)
		FROM transactions
		WHERE owner_id = $1 AND to_char(tx_date,'YYYY-MM') = $2`+transferFilter+dayFilter, metricSQL),
		owner, refMonth).Scan(&currentVal)
	if err != nil {
		return 0, 0, false
	}

	// Moyenne historique sur périodes comparables.
	placeholders := make([]string, len(months))
	args := []any{owner}
	for i, m := range months {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, m)
	}
	avgQuery := fmt.Sprintf(`
		SELECT AVG(val) FROM (
			SELECT COALESCE(%s,0) AS val
			FROM transactions
			WHERE owner_id = $1 AND to_char(tx_date,'YYYY-MM') IN (%s)`+transferFilter+dayFilter+`
			GROUP BY to_char(tx_date,'YYYY-MM')
		) m`, metricSQL, strings.Join(placeholders, ","))

	var avgVal float64
	if err := s.DB.QueryRow(avgQuery, args...).Scan(&avgVal); err != nil {
		return 0, 0, false
	}

	return currentVal, avgVal, true
}

// periodHandler expose les infos de période pour le frontend.
func (s *Server) periodHandler(w http.ResponseWriter, r *http.Request) {
	owner := s.ownerID(r)
	month := r.URL.Query().Get("month")
	info, err := s.getPeriodInfo(owner, month)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, info)
}

// periodCompareHandler compare le mois de référence à des périodes historiques comparables.
func (s *Server) periodCompareHandler(w http.ResponseWriter, r *http.Request) {
	owner := s.ownerID(r)
	month := r.URL.Query().Get("month")
	metric := r.URL.Query().Get("metric") // income | expenses | net
	if metric == "" {
		metric = "expenses"
	}
	if metric != "income" && metric != "expenses" && metric != "net" {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("métrique invalide"))
		return
	}

	info, err := s.getPeriodInfo(owner, month)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	current, average, hasHistory := s.comparePeriod(owner, info.Month, info.DaysObserved, metric, true)

	var pctChange *float64
	if hasHistory && average != 0 {
		v := (current - average) / math.Abs(average) * 100
		pctChange = &v
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"period":            info,
		"metric":            metric,
		"current":           current,
		"average":           average,
		"pct_change":        pctChange,
		"has_history":       hasHistory,
		"comparable_months": len(s.getComparableMonths(owner, info.Month, info.DaysObserved, 6)),
	})
}

// Euro formate un montant en euros sans centimes (ex: "1 234 €").
func Euro(v float64) string {
	v = math.Round(v)
	sign := ""
	if v < 0 {
		sign = "−"
		v = -v
	}
	s := fmt.Sprintf("%.0f", v)
	out := ""
	for i, c := range reverse(s) {
		if i > 0 && i%3 == 0 {
			out = " " + out
		}
		out = string(c) + out
	}
	return sign + out + " €"
}

func reverse(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

// Plural retourne "s" si n > 1, sinon chaîne vide.
func Plural(n int) string {
	if n > 1 {
		return "s"
	}
	return ""
}

var _ = math.Round
