package api

import "time"

// Le signe des flux ne diagnostique une tension que sur une période documentée.
// Aucun salaire attendu ni solde bancaire n'est inféré.
func heroDiagnosis(current, complete bool, net float64, points []heroPoint) (string, string) {
	tone := "neutral"
	for _, p := range points {
		if p.Tone == "alert" {
			tone = "alert"
			break
		}
		if p.Tone == "attention" {
			tone = "attention"
		}
	}
	if current {
		return tone, "Résultat provisoire : les flux du mois ne sont pas encore complets."
	}
	if !complete {
		return tone, "Données incomplètes : le résultat porte uniquement sur les flux enregistrés."
	}
	if net < 0 {
		return "alert", "Les dépenses ont dépassé les revenus sur le mois."
	}
	if tone == "neutral" {
		return "positive", "Les revenus couvrent les dépenses du mois."
	}
	return tone, "Des points d’attention nécessitent un examen."
}

type heroDateSpan struct{ start, end time.Time }

// Les intervalles sont triés par début. Des relevés successifs peuvent couvrir
// un mois civil même si chacun se clôture le 21. Un trou reste incomplet.
func heroSpansCover(spans []heroDateSpan, first, last time.Time) bool {
	next := first
	for _, span := range spans {
		if span.start.After(next) {
			return false
		}
		if !span.end.Before(next) {
			next = span.end.AddDate(0, 0, 1)
		}
		if next.After(last) {
			return true
		}
	}
	return false
}

// La date d'une transaction ou d'un import ne prouve pas la couverture.
// Exiger des relevés validés continus pour chaque compte connu (et les
// relevés sans compte). En cas d'erreur ou de métadonnées absentes : prudent.
func (s *Server) heroMonthCovered(owner int64, month string) bool {
	first, err := time.Parse("2006-01", month)
	if err != nil {
		return false
	}
	last := first.AddDate(0, 1, -1)
	rows, err := s.DB.Query(`
 WITH scope AS (
  SELECT id AS account_id FROM accounts WHERE owner_id = $1
  UNION SELECT account_id FROM statements WHERE owner_id = $1
 )
 SELECT COALESCE(c.account_id,0), COALESCE(s.period_start::text,''), COALESCE(s.period_end::text,'')
 FROM scope c LEFT JOIN statements s ON s.account_id IS NOT DISTINCT FROM c.account_id
  AND s.owner_id = $1 AND s.validated = true
  AND s.period_start <= $3::date AND s.period_end >= $2::date
 ORDER BY COALESCE(c.account_id,0), s.period_start`, owner, first.Format("2006-01-02"), last.Format("2006-01-02"))
	if err != nil {
		return false
	}
	defer rows.Close()
	groups := map[int64][]heroDateSpan{}
	for rows.Next() {
		var account int64
		var from, to string
		if rows.Scan(&account, &from, &to) != nil {
			return false
		}
		if _, ok := groups[account]; !ok {
			groups[account] = nil
		}
		a, e1 := time.Parse("2006-01-02", from)
		b, e2 := time.Parse("2006-01-02", to)
		if e1 == nil && e2 == nil && !b.Before(a) {
			groups[account] = append(groups[account], heroDateSpan{a, b})
		}
	}
	if rows.Err() != nil || len(groups) == 0 {
		return false
	}
	for _, spans := range groups {
		if !heroSpansCover(spans, first, last) {
			return false
		}
	}
	return true
}
