package api

import (
	"fmt"
	"math"
	"net/http"
	"time"
)

// pulse produit une lecture synthétique et immédiate du mois en cours :
// un "ton" (calme / attention / alerte) et une phrase en français, calculés
// sans IA (déterministe, instantané, fiable). L'objectif est de répondre d'un
// coup d'œil à « est-ce que tout va bien ? ».
func (s *Server) pulse(w http.ResponseWriter, r *http.Request) {
	// Mois de référence = mois le plus récent contenant des transactions.
	var ref string
	owner := s.ownerID(r)
	if err := s.DB.QueryRow(`SELECT to_char(max(tx_date), 'YYYY-MM') FROM transactions WHERE owner_id = $1`, owner).Scan(&ref); err != nil || ref == "" {
		writeJSON(w, http.StatusOK, map[string]any{
			"tone": "neutral",
			"headline": "Importe un relevé pour voir l'état de tes comptes.",
			"details": []string{},
		})
		return
	}

	// Dépenses et revenus du mois de référence.
	var spent, income float64
	s.DB.QueryRow(`SELECT COALESCE(-SUM(amount) FILTER (WHERE amount < 0), 0),
	                      COALESCE(SUM(amount) FILTER (WHERE amount > 0), 0)
	               FROM transactions WHERE owner_id = $2 AND to_char(tx_date,'YYYY-MM') = $1`+notTransfer, ref, owner).Scan(&spent, &income)

	// Moyenne des dépenses sur les mois complets précédents, et nombre de mois
	// d'historique disponibles (pour ne comparer que si c'est fiable).
	var avgSpent float64
	var histMonths int
	s.DB.QueryRow(`
		SELECT COALESCE(AVG(m.dep), 0), COUNT(*) FROM (
			SELECT to_char(tx_date,'YYYY-MM') AS mth, -SUM(amount) FILTER (WHERE amount < 0) AS dep
			FROM transactions
			WHERE owner_id = $2 AND to_char(tx_date,'YYYY-MM') < $1`+notTransfer+`
			GROUP BY 1 ORDER BY 1 DESC LIMIT 6
		) m`, ref, owner).Scan(&avgSpent, &histMonths)

	details := []string{}
	tone := "calm"

	// Solde net du mois.
	net := income - spent
	if income > 0 {
		if net >= 0 {
			details = append(details, fmt.Sprintf("Solde du mois : +%s épargnés.", euro(net)))
		} else {
			details = append(details, fmt.Sprintf("Solde du mois : %s dans le rouge.", euro(net)))
			tone = "warn"
		}
	}

	// Comparaison à la moyenne — SEULEMENT avec au moins 2 mois d'historique.
	// En dessous, une comparaison serait trompeuse : on s'abstient plutôt que
	// d'afficher un pourcentage absurde (une synthèse fausse est pire qu'aucune).
	var headline string
	if histMonths >= 2 && avgSpent > 0 {
		delta := spent - avgSpent
		pct := delta / avgSpent * 100
		switch {
		case pct >= 25:
			tone = "alert"
			headline = fmt.Sprintf("Tes dépenses sont nettement plus élevées que d'habitude (+%.0f %%).", pct)
		case pct >= 10:
			if tone == "calm" {
				tone = "warn"
			}
			headline = fmt.Sprintf("Tu dépenses un peu plus que ta moyenne ce mois-ci (+%.0f %%).", pct)
		case pct <= -10:
			headline = fmt.Sprintf("Mois plus sobre que d'habitude (%.0f %% de dépenses en moins). Beau travail.", math.Abs(pct))
		default:
			headline = "Ce mois-ci ressemble à tes mois habituels, rien d'inhabituel."
		}
		details = append(details, fmt.Sprintf("Dépenses : %s (moyenne récente : %s).", euro(spent), euro(avgSpent)))
	} else {
		// Pas assez de recul : on décrit sans juger.
		headline = fmt.Sprintf("%s de dépenses ce mois-ci.", euro(spent))
		details = append(details, "Encore trop peu d'historique pour te situer par rapport à tes habitudes — ça viendra avec les prochains relevés.")
	}

	// Le poste (domaine) qui dérive le plus — seulement avec assez d'historique.
	if histMonths >= 2 {
		var domName string
		var domNow, domAvg float64
		err := s.DB.QueryRow(`
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
			SELECT cur.name, cur.dep, COALESCE(hist.avg_dep, 0)
			FROM cur LEFT JOIN hist ON hist.name = cur.name
			WHERE COALESCE(hist.avg_dep,0) > 0 AND cur.dep > hist.avg_dep * 1.3 AND cur.dep - hist.avg_dep > 40
			ORDER BY (cur.dep - COALESCE(hist.avg_dep,0)) DESC LIMIT 1`, ref, owner).Scan(&domName, &domNow, &domAvg)
		if err == nil && domName != "" {
			details = append(details, fmt.Sprintf("Poste en hausse : %s (%s vs %s d'ordinaire).", domName, euro(domNow), euro(domAvg)))
			if tone == "calm" {
				tone = "warn"
			}
		}
	}

	// Transactions non rangées : petite invitation à trier.
	var untagged int
	s.DB.QueryRow(`SELECT COUNT(*) FROM transactions WHERE owner_id = $2 AND domain_id IS NULL AND to_char(tx_date,'YYYY-MM') = $1`, ref, owner).Scan(&untagged)
	if untagged > 0 {
		details = append(details, fmt.Sprintf("%d transaction%s à ranger.", untagged, plural(untagged)))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"tone":     tone,
		"headline": headline,
		"details":  details,
		"month":    ref,
		"untagged": untagged,
	})
}

// euro : format court « 1 234 € » (sans centimes) pour une phrase lisible.
func euro(v float64) string {
	v = math.Round(v)
	sign := ""
	if v < 0 {
		sign = "−"
		v = -v
	}
	s := fmt.Sprintf("%.0f", v)
	// séparateur de milliers (espace insécable fine)
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

var _ = time.Now
