package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
)

// Optional bounded series window. Legacy callers without dates retain history.
func analysisSeriesWindow(r *http.Request) (from, to string, err error) {
	q := r.URL.Query()
	if q.Get("month") != "" && (q.Get("from") != "" || q.Get("to") != "") {
		return "", "", fmt.Errorf("choisir un mois de référence ou des bornes explicites")
	}
	if q.Get("months") != "" && q.Get("month") == "" {
		return "", "", fmt.Errorf("un mois de référence est requis avec une durée")
	}
	if month := q.Get("month"); month != "" {
		first, last, _, e := reliabilityWindow(month, time.Now())
		if e != nil {
			return "", "", e
		}
		months := 6
		if raw := q.Get("months"); raw != "" {
			months, e = strconv.Atoi(raw)
			if e != nil || months < 1 || months > 600 {
				return "", "", fmt.Errorf("fenêtre de 1 à 600 mois attendue")
			}
		}
		start := first.AddDate(0, 1-months, 0)
		if start.Year() < 1 {
			return "", "", fmt.Errorf("début de période invalide")
		}
		return start.Format("2006-01-02"), last.Format("2006-01-02"), nil
	}
	from, to = q.Get("from"), q.Get("to")
	for _, v := range []string{from, to} {
		if v != "" {
			d, e := time.Parse("2006-01-02", v)
			if e != nil || d.Year() < 1 || d.Format("2006-01-02") != v || v > time.Now().Format("2006-01-02") {
				return "", "", fmt.Errorf("date invalide")
			}
		}
	}
	if from != "" && to != "" && from > to {
		return "", "", fmt.Errorf("période inversée")
	}
	if from != "" && to != "" {
		first, _ := time.Parse("2006-01-02", from)
		last, _ := time.Parse("2006-01-02", to)
		if (last.Year()-first.Year())*12+int(last.Month()-first.Month())+1 > 600 {
			return "", "", fmt.Errorf("fenêtre maximale de 600 mois par requête")
		}
	}
	return
}

// Bounded histories include every calendar interval. A missing observation must
// remain a null gap, even when none of the selected tags occurred that month.
func analysisSeriesBuckets(from, to, period string) []string {
	first, _ := time.Parse("2006-01-02", from)
	last, _ := time.Parse("2006-01-02", to)
	first = time.Date(first.Year(), first.Month(), 1, 0, 0, 0, 0, time.UTC)
	step, format := 1, "2006-01"
	if period == "year" {
		first = time.Date(first.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		step, format = 12, "2006"
	}
	buckets := []string{}
	for !first.After(last) {
		buckets = append(buckets, first.Format(format))
		first = first.AddDate(0, step, 0)
	}
	return buckets
}
func (s *Server) tagsInCategory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.URL.Query().Get("category"), 10, 64)
	if err != nil || id <= 0 {
		writeErr(w, 400, fmt.Errorf("catégorie requise"))
		return
	}
	from, to := "", ""
	if m := r.URL.Query().Get("month"); m != "" {
		first, last, _, e := reliabilityWindow(m, time.Now())
		if e != nil {
			writeErr(w, 400, e)
			return
		}
		from, to = first.Format("2006-01-02"), last.Format("2006-01-02")
	}
	// Per-transaction owned-tag array prevents external tags from multiplying the
	// untagged bucket. Window totals operate on sources BEFORE unnesting tags.
	rows, err := s.DB.QueryContext(r.Context(), `WITH source AS (
 SELECT t.id,-t.amount AS spent,ARRAY(SELECT DISTINCT tg.name FROM transaction_tags tt
 JOIN tags tg ON tg.id=tt.tag_id AND tg.owner_id=t.owner_id WHERE tt.transaction_id=t.id ORDER BY tg.name) AS names
 FROM transactions t JOIN domains d ON d.id=t.domain_id AND d.owner_id=t.owner_id
 WHERE t.owner_id=$1 AND (d.id=$2 OR d.parent_id=$2) AND t.amount<0`+notTransferT+`
 AND ($3='' OR t.tx_date>=NULLIF($3,'')::date) AND ($4='' OR t.tx_date<=NULLIF($4,'')::date)),
 totals AS (SELECT COALESCE(SUM(spent),0)::text AS total,count(*) AS n,count(*) FILTER(WHERE cardinality(names)>1) AS overlap FROM source),
 slices AS (SELECT name,SUM(spent)::text AS spent,count(*) AS n FROM source CROSS JOIN LATERAL
 unnest(CASE WHEN cardinality(names)=0 THEN ARRAY['(sans tag)'] ELSE names END) name GROUP BY name)
 SELECT slices.name,slices.spent,slices.n,totals.total,totals.n,totals.overlap FROM totals LEFT JOIN slices ON true ORDER BY slices.spent::numeric DESC NULLS LAST`, s.ownerID(r), id, from, to)
	if err != nil {
		writeErr(w, 500, fmt.Errorf("analyse des tags indisponible"))
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	var totalText string
	var count, overlap int
	for rows.Next() {
		var name, raw *string
		var n *int
		if err = rows.Scan(&name, &raw, &n, &totalText, &count, &overlap); err != nil {
			writeErr(w, 500, fmt.Errorf("tags invalides"))
			return
		}
		if name != nil {
			spent, e := parseAnalysisAmount(*raw)
			if e != nil {
				writeErr(w, 500, e)
				return
			}
			total, e := parseAnalysisAmount(totalText)
			if e != nil {
				writeErr(w, 500, e)
				return
			}
			out = append(out, map[string]any{"tag": *name, "spent": simulationNumber(spent), "count": *n, "share_pct": analysisPercent(spent, total)})
		}
	}
	if rows.Err() != nil {
		writeErr(w, 500, fmt.Errorf("tags interrompus"))
		return
	}
	total, err := parseAnalysisAmount(totalText)
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	writeJSON(w, 200, map[string]any{"slices": out, "category_total": simulationNumber(total), "category_count": count, "overlap_count": overlap, "month": r.URL.Query().Get("month"), "query_from": from, "query_to": to})
}
func (s *Server) tagSeries(w http.ResponseWriter, r *http.Request) {
	from, to, err := analysisSeriesWindow(r)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	trunc, format := "month", "YYYY-MM"
	if r.URL.Query().Get("period") == "year" {
		trunc, format = "year", "YYYY"
	}
	names := []string{}
	for _, v := range strings.Split(r.URL.Query().Get("tags"), ",") {
		if name := strings.TrimSpace(v); name != "" {
			names = append(names, name)
		}
	}
	if len(names) > 100 {
		writeErr(w, 400, fmt.Errorf("100 tags maximum"))
		return
	}
	if len(names) == 0 {
		rows, e := s.DB.QueryContext(r.Context(), `SELECT tg.name FROM tags tg JOIN transaction_tags tt ON tt.tag_id=tg.id
 JOIN transactions t ON t.id=tt.transaction_id AND t.owner_id=tg.owner_id
 WHERE t.owner_id=$1 AND t.amount<0`+notTransferT+` AND ($2='' OR t.tx_date>=NULLIF($2,'')::date) AND ($3='' OR t.tx_date<=NULLIF($3,'')::date)
 GROUP BY tg.name ORDER BY -SUM(t.amount) DESC LIMIT 6`, s.ownerID(r), from, to)
		if e != nil {
			writeErr(w, 500, fmt.Errorf("tags indisponibles"))
			return
		}
		for rows.Next() {
			var name string
			if e = rows.Scan(&name); e != nil {
				rows.Close()
				writeErr(w, 500, e)
				return
			}
			names = append(names, name)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			writeErr(w, 500, e)
			return
		}
	}
	rows, err := s.DB.QueryContext(r.Context(), `SELECT tg.name,to_char(date_trunc($2::text,t.tx_date),$3) AS bucket,(-SUM(t.amount))::text
 FROM transactions t JOIN transaction_tags tt ON tt.transaction_id=t.id JOIN tags tg ON tg.id=tt.tag_id AND tg.owner_id=t.owner_id
 WHERE t.owner_id=$1 AND t.amount<0 AND tg.name=ANY($4::text[])`+notTransferT+`
 AND ($5='' OR t.tx_date>=NULLIF($5,'')::date) AND ($6='' OR t.tx_date<=NULLIF($6,'')::date)
 GROUP BY tg.name,bucket ORDER BY bucket`, s.ownerID(r), trunc, format, pq.Array(names), from, to)
	if err != nil {
		writeErr(w, 500, fmt.Errorf("séries indisponibles"))
		return
	}
	defer rows.Close()
	buckets := []string{}
	seen := map[string]bool{}
	values := map[string]map[string]json.Number{}
	for rows.Next() {
		var name, bucket, raw string
		if err = rows.Scan(&name, &bucket, &raw); err != nil {
			writeErr(w, 500, err)
			return
		}
		value, e := parseAnalysisAmount(raw)
		if e != nil {
			writeErr(w, 500, e)
			return
		}
		if !seen[bucket] {
			buckets = append(buckets, bucket)
			seen[bucket] = true
		}
		if values[name] == nil {
			values[name] = map[string]json.Number{}
		}
		values[name][bucket] = simulationNumber(value)
	}
	if rows.Err() != nil {
		writeErr(w, 500, fmt.Errorf("séries interrompues"))
		return
	}
	if from != "" && to != "" && len(buckets) > 0 {
		buckets = analysisSeriesBuckets(from, to, trunc)
	}
	series := []map[string]any{}
	for _, name := range names {
		if values[name] == nil {
			continue
		}
		v := []any{}
		for _, bucket := range buckets {
			if amount, ok := values[name][bucket]; ok {
				v = append(v, amount)
			} else {
				v = append(v, nil)
			}
		}
		series = append(series, map[string]any{"tag": name, "values": v})
	}
	writeJSON(w, 200, map[string]any{"buckets": buckets, "series": series, "query_from": from, "query_to": to, "coverage": "unknown", "missing_buckets": "not_observed", "period": trunc})
}
