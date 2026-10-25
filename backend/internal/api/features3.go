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
			if err != nil || id <= 0 {
				writeErr(w, 400, fmt.Errorf("catégorie invalide"))
				return
			}
			args = append(args, id)
			ph = append(ph, fmt.Sprintf("$%d", len(args)))
		}
		if len(ph) > 0 {
			catFilter = " AND (d.id IN (" + strings.Join(ph, ",") + ") OR p.id IN (" + strings.Join(ph, ",") + "))"
		}
	}

	// Honor the documented date range. An invalid boundary must never silently
	// broaden an otherwise scoped financial query.
	var first, last time.Time
	for _, bound := range []struct{ name, op string }{{"from", ">="}, {"to", "<="}} {
		if raw := q.Get(bound.name); raw != "" {
			date, err := time.Parse("2006-01-02", raw)
			if err != nil || date.Format("2006-01-02") != raw {
				writeErr(w, http.StatusBadRequest, fmt.Errorf("date %s invalide", bound.name))
				return
			}
			if bound.name == "from" {
				first = date
			} else {
				last = date
			}
			args = append(args, raw)
			catFilter += fmt.Sprintf(" AND t.tx_date %s $%d::date", bound.op, len(args))
		}
	}
	if !first.IsZero() && !last.IsZero() && first.After(last) {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("période inversée"))
		return
	}

	sql := fmt.Sprintf(`
		SELECT date_trunc('%s', t.tx_date) AS bucket,
		       COALESCE(p.name,d.name, 'Sans domaine') AS cat,
		       COALESCE(p.color,d.color, '#66736C') AS color,
		       (-SUM(t.amount) FILTER (WHERE t.amount < 0))::text AS spent
		FROM transactions t
		LEFT JOIN domains d ON d.id = t.domain_id AND d.owner_id=t.owner_id
		LEFT JOIN domains p ON p.id=d.parent_id AND p.owner_id=t.owner_id
        WHERE t.amount < 0 AND t.owner_id = $1 `+notTransferT+` %s
		GROUP BY 1,2,3
		ORDER BY bucket`, trunc, catFilter)

	rows, err := s.DB.Query(sql, args...)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	// pivot : une ligne par bucket, une colonne par catégorie
	bucketOrder := []string{}
	byBucket := map[string]map[string]json.Number{}
	colors := map[string]string{}
	for rows.Next() {
		var bucket time.Time
		var cat, color string
		var spentText string
		if err := rows.Scan(&bucket, &cat, &color, &spentText); err != nil {
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
			byBucket[key] = map[string]json.Number{}
			bucketOrder = append(bucketOrder, key)
		}
		spent, err := parseAnalysisAmount(spentText)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		byBucket[key][cat] = json.Number(spent.String())
		colors[cat] = color
	}

	if err = rows.Err(); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
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
	writeJSON(w, http.StatusOK, map[string]any{"series": series, "categories": cats, "query_from": q.Get("from"), "query_to": q.Get("to"), "coverage": "unknown", "missing_buckets": "not_observed"})
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
