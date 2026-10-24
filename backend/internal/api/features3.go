package api

import (
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
