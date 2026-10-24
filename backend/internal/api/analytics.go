package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/lib/pq"
)

// tagsInCategory : répartition des dépenses par tag à l'intérieur d'une
// catégorie (domain_id). Répond à « dans Alimentation, combien en courses vs
// restaurants ? ». Les transactions sans tag sont regroupées sous « (sans tag) ».
func (s *Server) tagsInCategory(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	catID, err := strconv.ParseInt(strings.TrimSpace(q.Get("category")), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("paramètre 'category' (id) requis"))
		return
	}

	args := []any{catID, s.ownerID(r)}
	monthClause := ""
	if m := strings.TrimSpace(q.Get("month")); m != "" {
		if !reMonth.MatchString(m) {
			writeErr(w, http.StatusBadRequest, fmt.Errorf("month doit être au format YYYY-MM"))
			return
		}
		args = append(args, m)
		monthClause = " AND to_char(t.tx_date,'YYYY-MM') = $3"
	}

	// Dépenses par tag (montants négatifs → dépense). Une transaction avec
	// plusieurs tags compte dans chacun ; c'est voulu (un resto « amoureux » +
	// « anniversaire » apparaît dans les deux). Le total « sans tag » ne
	// concerne que les transactions réellement sans aucun tag.
	rows, err := s.DB.Query(`
		SELECT COALESCE(tg.name, '(sans tag)') AS tag,
		       -SUM(t.amount) AS spent,
		       COUNT(DISTINCT t.id) AS n
		FROM transactions t
		LEFT JOIN transaction_tags tt ON tt.transaction_id = t.id
		LEFT JOIN tags tg ON tg.id = tt.tag_id
		WHERE t.domain_id = $1 AND t.owner_id = $2 AND t.amount < 0`+monthClause+`
		GROUP BY COALESCE(tg.name, '(sans tag)')
		HAVING -SUM(t.amount) > 0
		ORDER BY spent DESC`, args...)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	type slice struct {
		Tag   string  `json:"tag"`
		Spent float64 `json:"spent"`
		Count int     `json:"count"`
	}
	out := []slice{}
	for rows.Next() {
		var sl slice
		if err := rows.Scan(&sl.Tag, &sl.Spent, &sl.Count); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		out = append(out, sl)
	}
	writeJSON(w, http.StatusOK, map[string]any{"slices": out})
}

// tagSeries : évolution temporelle des dépenses pour un ou plusieurs tags,
// toutes catégories confondues. Répond à « combien je dépense en resto au fil
// des mois ? » — et, si on tague perso/amoureux/famille, récupère la lecture
// « domaines de vie » de façon optionnelle.
func (s *Server) tagSeries(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	var trunc, keyFmt string
	switch q.Get("period") {
	case "year":
		trunc, keyFmt = "year", "YYYY"
	default:
		trunc, keyFmt = "month", "YYYY-MM"
	}

	// Liste de tags demandés (noms). Vide → les 6 tags les plus dépensiers.
	tagsParam := strings.TrimSpace(q.Get("tags"))
	var tagNames []string
	if tagsParam != "" {
		for _, t := range strings.Split(tagsParam, ",") {
			if t = strings.TrimSpace(t); t != "" {
				tagNames = append(tagNames, t)
			}
		}
	}
	if len(tagNames) == 0 {
		trows, err := s.DB.Query(`SELECT tg.name FROM tags tg
 JOIN transaction_tags tt ON tt.tag_id=tg.id JOIN transactions t ON t.id=tt.transaction_id
 WHERE t.amount<0 AND t.owner_id=$1 AND tg.owner_id=t.owner_id`+notTransferT+`
 GROUP BY tg.name ORDER BY -SUM(t.amount) DESC LIMIT 6`, s.ownerID(r))
		if err != nil {
			writeErr(w, 500, fmt.Errorf("séries de tags indisponibles"))
			return
		}
		for trows.Next() {
			var name string
			if err = trows.Scan(&name); err != nil {
				trows.Close()
				writeErr(w, 500, fmt.Errorf("séries de tags indisponibles"))
				return
			}
			tagNames = append(tagNames, name)
		}
		err = trows.Err()
		trows.Close()
		if err != nil {
			writeErr(w, 500, fmt.Errorf("séries de tags interrompues"))
			return
		}
	}
	if len(tagNames) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{"buckets": []string{}, "series": []any{}})
		return
	}

	// Date granularity, format and tag array are bound values, never SQL text.
	// Audited: compile-time SQL constants only; all external values use PostgreSQL bind parameters.
	// nosemgrep: go.lang.security.audit.database.string-formatted-query.string-formatted-query
	const query = `SELECT tg.name,to_char(date_trunc($2::text,t.tx_date),$3) AS bucket,-SUM(t.amount) AS spent
 FROM transactions t JOIN transaction_tags tt ON tt.transaction_id=t.id
 JOIN tags tg ON tg.id=tt.tag_id AND tg.owner_id=t.owner_id
 WHERE t.amount<0 AND t.owner_id=$1 AND tg.name=ANY($4::text[])` + notTransferT + `
 GROUP BY tg.name,bucket ORDER BY bucket`
	rows, err := s.DB.Query(query, s.ownerID(r), trunc, keyFmt, pq.Array(tagNames))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	// Collecte : série par tag, indexée par bucket.
	bucketSet := map[string]bool{}
	byTag := map[string]map[string]float64{}
	for rows.Next() {
		var name, bucket string
		var spent float64
		if err := rows.Scan(&name, &bucket, &spent); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		bucketSet[bucket] = true
		if byTag[name] == nil {
			byTag[name] = map[string]float64{}
		}
		byTag[name][bucket] = spent
	}

	if rows.Err() != nil {
		writeErr(w, 500, fmt.Errorf("séries de tags interrompues"))
		return
	}
	buckets := make([]string, 0, len(bucketSet))
	for b := range bucketSet {
		buckets = append(buckets, b)
	}
	sortStrings(buckets)

	type serie struct {
		Tag    string    `json:"tag"`
		Values []float64 `json:"values"`
	}
	series := []serie{}
	for _, name := range tagNames {
		if byTag[name] == nil {
			continue
		}
		vals := make([]float64, len(buckets))
		for i, b := range buckets {
			vals[i] = byTag[name][b]
		}
		series = append(series, serie{Tag: name, Values: vals})
	}

	writeJSON(w, http.StatusOK, map[string]any{"buckets": buckets, "series": series})
}
