package api

import (
	"context"
	"time"
)

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
	return len(uncoveredStatementDates(spans, first, last)) == 0
}

// La date d'une transaction ou d'un import ne prouve pas la couverture.
// Exiger des relevés validés continus pour chaque compte soumis au contrôle documentaire (et les
// relevés sans compte). En cas d'erreur ou de métadonnées absentes : prudent.
func (s *Server) heroMonthCovered(owner int64, month string) bool {
	report, err := s.loadReliability(context.Background(), owner, month, time.Now())
	if err != nil {
		return false
	}
	required := 0
	for _, a := range report.Accounts {
		if a.TrackingMode == "manual" {
			continue
		}
		required++
		if a.Status != "documented" {
			return false
		}
	}
	return required > 0
}
