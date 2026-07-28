package api

import "net/http"

// ── Vague 1 : rendre les données plus intelligentes, sans rien deviner ──
// Trois lectures purement factuelles des données déjà collectées :
//   1. par marchand    : chez qui, combien, à quelle fréquence, quelle tendance
//   2. par nature       : vraies dépenses vs impôts vs épargne (via label_kind)
//   3. comparaison      : ce mois vs la moyenne récente, par catégorie
// Tout est vérifiable au centime ; aucune interprétation.

// merchantsView agrège les dépenses par marchand normalisé sur une fenêtre
// (par défaut 3 mois). Répond à « chez qui je dépense, et est-ce que ça
// dérive ? » — impossible sans la couche de normalisation.
func (s *Server) merchantsView(w http.ResponseWriter, r *http.Request) {
	owner := s.ownerID(r)
	months := 3
	if m := atoiDefault(r.URL.Query().Get("months"), 3); m >= 1 && m <= 24 {
		months = m
	}
	// Fenêtre courante vs fenêtre précédente de même durée, pour la tendance.
	rows, err := s.DB.Query(`
		WITH win AS (
			SELECT COALESCE(NULLIF(merchant,''), display_label, label) AS m,
			       tx_date, -amount AS spent
			FROM transactions
			WHERE owner_id = $1 AND amount < 0
			  AND (domain_id IS NULL OR domain_id NOT IN (SELECT id FROM domains WHERE is_transfer))
			  AND tx_date >= (CURRENT_DATE - ($2 * INTERVAL '1 month'))
		),
		prev AS (
			SELECT COALESCE(NULLIF(merchant,''), display_label, label) AS m,
			       -amount AS spent
			FROM transactions
			WHERE owner_id = $1 AND amount < 0
			  AND (domain_id IS NULL OR domain_id NOT IN (SELECT id FROM domains WHERE is_transfer))
			  AND tx_date >= (CURRENT_DATE - ($2 * 2 * INTERVAL '1 month'))
			  AND tx_date <  (CURRENT_DATE - ($2 * INTERVAL '1 month'))
		)
		SELECT w.m,
		       COUNT(*)            AS visits,
		       SUM(w.spent)        AS total,
		       AVG(w.spent)        AS avg_basket,
		       MAX(w.tx_date)::text AS last_seen,
		       COALESCE((SELECT SUM(p.spent) FROM prev p WHERE p.m = w.m), 0) AS prev_total
		FROM win w
		GROUP BY w.m
		HAVING COUNT(*) >= 1
		ORDER BY total DESC
		LIMIT 40`, owner, months)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	type merchant struct {
		Name      string  `json:"name"`
		Visits    int     `json:"visits"`
		Total     float64 `json:"total"`
		AvgBasket float64 `json:"avg_basket"`
		LastSeen  string  `json:"last_seen"`
		PrevTotal float64 `json:"prev_total"`
		TrendPct  *int    `json:"trend_pct"` // nil si pas de base de comparaison
	}
	out := []merchant{}
	for rows.Next() {
		var m merchant
		if err := rows.Scan(&m.Name, &m.Visits, &m.Total, &m.AvgBasket, &m.LastSeen, &m.PrevTotal); err != nil {
			continue
		}
		if m.PrevTotal > 0 {
			p := int((m.Total - m.PrevTotal) / m.PrevTotal * 100)
			m.TrendPct = &p
		}
		out = append(out, m)
	}
	writeJSON(w, http.StatusOK, map[string]any{"months": months, "merchants": out})
}

// natureBreakdown sépare, pour le mois de référence, ce qui sort du compte en
// trois natures : dépenses réelles, prélèvements obligatoires (impôts,
// cotisations) et épargne. « 2 000 € de sorties » dont 800 d'épargne, ce n'est
// pas 2 000 € de dépenses — cette lecture change la compréhension.
func (s *Server) natureBreakdown(w http.ResponseWriter, r *http.Request) {
	owner := s.ownerID(r)
	var ref string
	if err := s.DB.QueryRow(`SELECT to_char(max(tx_date),'YYYY-MM') FROM transactions WHERE owner_id = $1`, owner).Scan(&ref); err != nil || ref == "" {
		writeJSON(w, http.StatusOK, map[string]any{"available": false})
		return
	}
	// On classe par label_kind : tax → obligatoire, savings → épargne, le
	// reste des débits (hors transferts) → dépenses réelles.
	var realSpend, mandatory, savings, income float64
	_ = s.DB.QueryRow(`
		SELECT
			COALESCE(-SUM(amount) FILTER (WHERE amount < 0 AND label_kind NOT IN ('tax','savings')
			         AND (domain_id IS NULL OR domain_id NOT IN (SELECT id FROM domains WHERE is_transfer))), 0),
			COALESCE(-SUM(amount) FILTER (WHERE amount < 0 AND label_kind = 'tax'), 0),
			COALESCE(-SUM(amount) FILTER (WHERE amount < 0 AND label_kind = 'savings'), 0),
			COALESCE(SUM(amount) FILTER (WHERE amount > 0 AND label_kind NOT IN ('savings')), 0)
		FROM transactions
		WHERE owner_id = $1 AND to_char(tx_date,'YYYY-MM') = $2`, owner, ref).
		Scan(&realSpend, &mandatory, &savings, &income)

	writeJSON(w, http.StatusOK, map[string]any{
		"available":   true,
		"month":       ref,
		"income":      income,
		"real_spend":  realSpend,
		"mandatory":   mandatory, // impôts, cotisations
		"savings":     savings,   // versements épargne
		"total_out":   realSpend + mandatory + savings,
	})
}

// categoryComparison compare, pour chaque catégorie, la dépense du mois de
// référence à la moyenne des mois précédents. Fait ressortir où ça dérive,
// sans jugement : juste l'écart chiffré.
func (s *Server) categoryComparison(w http.ResponseWriter, r *http.Request) {
	owner := s.ownerID(r)
	var ref string
	if err := s.DB.QueryRow(`SELECT to_char(max(tx_date),'YYYY-MM') FROM transactions WHERE owner_id = $1`, owner).Scan(&ref); err != nil || ref == "" {
		writeJSON(w, http.StatusOK, map[string]any{"available": false})
		return
	}
	rows, err := s.DB.Query(`
		WITH cur AS (
			SELECT d.name, -SUM(t.amount) AS cur_spent
			FROM transactions t JOIN domains d ON d.id = t.domain_id AND NOT d.is_transfer
			WHERE t.owner_id = $1 AND t.amount < 0 AND to_char(t.tx_date,'YYYY-MM') = $2
			GROUP BY d.name
		),
		hist AS (
			SELECT d.name,
			       -SUM(t.amount) / GREATEST(COUNT(DISTINCT to_char(t.tx_date,'YYYY-MM')),1) AS avg_spent
			FROM transactions t JOIN domains d ON d.id = t.domain_id AND NOT d.is_transfer
			WHERE t.owner_id = $1 AND t.amount < 0 AND to_char(t.tx_date,'YYYY-MM') < $2
			GROUP BY d.name
		)
		SELECT COALESCE(cur.name, hist.name),
		       COALESCE(cur.cur_spent, 0),
		       COALESCE(hist.avg_spent, 0)
		FROM cur FULL OUTER JOIN hist ON cur.name = hist.name
		ORDER BY COALESCE(cur.cur_spent, 0) DESC`, owner, ref)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	type cat struct {
		Name     string  `json:"name"`
		Current  float64 `json:"current"`
		Average  float64 `json:"average"`
		DeltaPct *int    `json:"delta_pct"`
	}
	out := []cat{}
	for rows.Next() {
		var c cat
		if rows.Scan(&c.Name, &c.Current, &c.Average) != nil {
			continue
		}
		if c.Average > 0 {
			d := int((c.Current - c.Average) / c.Average * 100)
			c.DeltaPct = &d
		}
		out = append(out, c)
	}
	writeJSON(w, http.StatusOK, map[string]any{"available": true, "month": ref, "categories": out})
}

// atoiDefault : petit helper local de parsing avec valeur de repli.
func atoiDefault(s string, def int) int {
	n := 0
	if s == "" {
		return def
	}
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return def
		}
		n = n*10 + int(ch-'0')
	}
	return n
}
