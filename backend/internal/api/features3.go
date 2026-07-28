package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

/* ==========================================================================
   Graphe par catégories : séries temporelles multi-catégories, période au choix
   GET /api/category-series?categories=1,2,3&period=month|quarter|year&from=&to=
   ========================================================================== */

func (s *Server) categorySeries(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	// bucket temporel
	var trunc, keyFmt string
	switch q.Get("period") {
	case "year":
		trunc, keyFmt = "year", "2006"
	case "quarter":
		trunc, keyFmt = "quarter", "" // libellé calculé à la main
	default:
		trunc, keyFmt = "month", "2006-01"
	}

	// filtre de catégories (ids), vide = toutes
	var catFilter string
	args := []any{s.ownerID(r)}
	if cs := strings.TrimSpace(q.Get("categories")); cs != "" {
		ph := make([]string, 0)
		for _, raw := range strings.Split(cs, ",") {
			id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
			if err != nil {
				continue // on ignore silencieusement les IDs non numériques
			}
			args = append(args, id)
			ph = append(ph, fmt.Sprintf("$%d", len(args)))
		}
		if len(ph) > 0 {
			catFilter = " AND t.domain_id IN (" + strings.Join(ph, ",") + ")"
		}
	}

	sql := fmt.Sprintf(`
		SELECT date_trunc('%s', t.tx_date) AS bucket,
		       COALESCE(d.name, 'Sans domaine') AS cat,
		       COALESCE(d.color, '#66736C') AS color,
		       -SUM(t.amount) FILTER (WHERE t.amount < 0) AS spent
		FROM transactions t
		LEFT JOIN domains d ON d.id = t.domain_id
		WHERE t.amount < 0 AND t.owner_id = $1 AND (t.domain_id IS NULL OR t.domain_id NOT IN (SELECT id FROM domains WHERE is_transfer)) %s
		GROUP BY bucket, cat, color
		ORDER BY bucket`, trunc, catFilter)

	rows, err := s.DB.Query(sql, args...)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	// pivot : une ligne par bucket, une colonne par catégorie
	bucketOrder := []string{}
	byBucket := map[string]map[string]float64{}
	colors := map[string]string{}
	for rows.Next() {
		var bucket time.Time
		var cat, color string
		var spent float64
		if err := rows.Scan(&bucket, &cat, &color, &spent); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		var key string
		if trunc == "quarter" {
			key = fmt.Sprintf("%d-T%d", bucket.Year(), (int(bucket.Month())-1)/3+1)
		} else {
			key = bucket.Format(keyFmt)
		}
		if _, ok := byBucket[key]; !ok {
			byBucket[key] = map[string]float64{}
			bucketOrder = append(bucketOrder, key)
		}
		byBucket[key][cat] = round2(spent)
		colors[cat] = color
	}

	series := make([]map[string]any, 0, len(bucketOrder))
	for _, k := range bucketOrder {
		row := map[string]any{"period": k}
		for cat, v := range byBucket[k] {
			row[cat] = v
		}
		series = append(series, row)
	}
	cats := make([]map[string]string, 0, len(colors))
	for name, color := range colors {
		cats = append(cats, map[string]string{"name": name, "color": color})
	}
	writeJSON(w, http.StatusOK, map[string]any{"series": series, "categories": cats})
}

func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}

/* ==========================================================================
   Simulation budgétaire
   POST /api/simulate
   {
     "base_income": 2500,           // revenu mensuel (défaut : moyenne récente)
     "cuts": [{"category":"Loisirs","reduce_pct":30}],
     "new_expenses": [{"label":"Salle de sport","amount":40}],
     "loan": {"amount":15000, "rate":4.5, "months":48}   // optionnel
   }
   ========================================================================== */

func (s *Server) simulate(w http.ResponseWriter, r *http.Request) {
	var in struct {
		BaseIncome  *float64 `json:"base_income"`
		Cuts        []struct {
			Category  string  `json:"category"`
			ReducePct float64 `json:"reduce_pct"`
		} `json:"cuts"`
		NewExpenses []struct {
			Label  string  `json:"label"`
			Amount float64 `json:"amount"`
		} `json:"new_expenses"`
		Loan *struct {
			Amount float64 `json:"amount"`
			Rate   float64 `json:"rate"`
			Months int     `json:"months"`
		} `json:"loan"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("corps JSON invalide"))
		return
	}

	// Base : moyennes des 3 derniers mois observés
	var income float64
	if in.BaseIncome != nil {
		income = *in.BaseIncome
	} else {
		_ = s.DB.QueryRow(`
			SELECT COALESCE(AVG(m.inc), 0) FROM (
			  SELECT SUM(amount) FILTER (WHERE amount > 0) AS inc
			  FROM transactions
			  WHERE owner_id = $1 AND tx_date >= (SELECT COALESCE(max(tx_date), now()) FROM transactions WHERE owner_id = $1) - INTERVAL '3 months'
			  GROUP BY to_char(tx_date, 'YYYY-MM')) m`, s.ownerID(r)).Scan(&income)
	}

	// Dépenses moyennes par catégorie sur 3 mois
	type catAvg struct {
		name string
		avg  float64
	}
	baseline := []catAvg{}
	var baseSpend float64
	rows, err := s.DB.Query(`
		SELECT COALESCE(d.name, 'Sans domaine') AS cat, SUM(-t.amount) / GREATEST(COUNT(DISTINCT to_char(t.tx_date,'YYYY-MM')),1) AS avg_month
		FROM transactions t LEFT JOIN domains d ON d.id = t.domain_id
		WHERE t.amount < 0 AND t.owner_id = $1 AND (t.domain_id IS NULL OR t.domain_id NOT IN (SELECT id FROM domains WHERE is_transfer)) AND t.tx_date >= (SELECT COALESCE(max(tx_date), now()) FROM transactions WHERE owner_id = $1) - INTERVAL '3 months'
		GROUP BY cat ORDER BY avg_month DESC`, s.ownerID(r))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	for rows.Next() {
		var c catAvg
		if err := rows.Scan(&c.name, &c.avg); err == nil {
			baseline = append(baseline, c)
			baseSpend += c.avg
		}
	}
	rows.Close()

	cutByCat := map[string]float64{}
	for _, c := range in.Cuts {
		cutByCat[c.Category] = c.ReducePct
	}

	// Application des coupes
	adjusted := []map[string]any{}
	var newSpend, saved float64
	for _, b := range baseline {
		pct := cutByCat[b.name]
		if pct < 0 {
			pct = 0
		}
		if pct > 100 {
			pct = 100
		}
		after := b.avg * (1 - pct/100)
		newSpend += after
		saved += b.avg - after
		adjusted = append(adjusted, map[string]any{
			"category": b.name, "before": round2(b.avg), "after": round2(after), "reduce_pct": pct,
		})
	}

	// Nouvelles dépenses
	var added float64
	for _, e := range in.NewExpenses {
		if e.Amount > 0 {
			added += e.Amount
		}
	}

	// Emprunt : mensualité (amortissement constant)
	var loanPayment float64
	var loanTotal float64
	if in.Loan != nil && in.Loan.Amount > 0 && in.Loan.Months > 0 {
		mRate := in.Loan.Rate / 100 / 12
		if mRate > 0 {
			p := in.Loan.Amount * mRate / (1 - pow(1+mRate, -in.Loan.Months))
			loanPayment = round2(p)
		} else {
			loanPayment = round2(in.Loan.Amount / float64(in.Loan.Months))
		}
		loanTotal = round2(loanPayment * float64(in.Loan.Months))
	}

	currentNet := round2(income - baseSpend)
	simulatedNet := round2(income - newSpend - added - loanPayment)

	writeJSON(w, http.StatusOK, map[string]any{
		"income":          round2(income),
		"base_spend":      round2(baseSpend),
		"current_net":     currentNet,
		"simulated_spend": round2(newSpend + added + loanPayment),
		"simulated_net":   simulatedNet,
		"monthly_saved":   round2(saved),
		"new_expenses":    round2(added),
		"loan_payment":    loanPayment,
		"loan_total_cost": loanTotal,
		"loan_interest":   round2(loanTotal - loanAmount(in.Loan)),
		"categories":      adjusted,
		"delta":           round2(simulatedNet - currentNet),
	})
}

func loanAmount(l *struct {
	Amount float64 `json:"amount"`
	Rate   float64 `json:"rate"`
	Months int     `json:"months"`
}) float64 {
	if l == nil {
		return 0
	}
	return l.Amount
}

func pow(base float64, exp int) float64 {
	r := 1.0
	if exp < 0 {
		for i := 0; i < -exp; i++ {
			r /= base
		}
		return r
	}
	for i := 0; i < exp; i++ {
		r *= base
	}
	return r
}
