package api

import (
	"fmt"
	"math"
	"net/http"
	"time"
)

// pulse produit une lecture synthétique et immédiate du mois demandé :
// un "ton" (calme / attention / alerte) et une phrase en français, calculés
// sans IA (déterministe, instantané, fiable). L'objectif est de répondre d'un
// coup d'œil à « est-ce que tout va bien ? ».
func (s *Server) pulse(w http.ResponseWriter, r *http.Request) {
	owner := s.ownerID(r)

	// Mois de référence : paramètre 'month' (YYYY-MM) ou mois le plus récent contenant des transactions.
	month := r.URL.Query().Get("month")
	var ref string
	if month != "" {
		ref = month
	} else if err := s.DB.QueryRow(`SELECT to_char(max(tx_date), 'YYYY-MM') FROM transactions WHERE owner_id = $1`, owner).Scan(&ref); err != nil || ref == "" {
		writeJSON(w, http.StatusOK, map[string]any{
			"tone":       "neutral",
			"headline":   "Importe un relevé pour voir l'état de tes comptes.",
			"details":    []string{},
			"period":     map[string]any{"month": "", "status": "none"},
		})
		return
	}

	info, _ := s.getPeriodInfo(owner, ref)

	// Dépenses et revenus du mois de référence (période observée).
	var spent, income float64
	dayFilter := ""
	if info.DaysObserved > 0 && info.IsCurrent {
		dayFilter = fmt.Sprintf(" AND EXTRACT(DAY FROM tx_date) <= %d", info.DaysObserved)
	}
	q := `SELECT COALESCE(-SUM(amount) FILTER (WHERE amount < 0), 0),
		       COALESCE(SUM(amount) FILTER (WHERE amount > 0), 0)
		FROM transactions WHERE owner_id = $2 AND to_char(tx_date,'YYYY-MM') = $1` + notTransfer + dayFilter
	s.DB.QueryRow(q, ref, owner).Scan(&spent, &income)

	// Comparaison à des périodes comparables (mêmes jours du mois).
	current, average, hasHistory := s.comparePeriod(owner, ref, info.DaysObserved, "expenses", true)

	details := []string{}
	tone := "calm"

	// Solde net du mois observé.
	net := income - spent
	if income > 0 {
		if net >= 0 {
			details = append(details, fmt.Sprintf("Solde à date : +%s épargnés.", Euro(net)))
		} else {
			details = append(details, fmt.Sprintf("Solde à date : %s dans le rouge.", Euro(net)))
			tone = "warn"
		}
	}

	var headline string
	if info.IsCurrent {
		// Mois en cours : formulation adaptée.
		headline = fmt.Sprintf("Au %d %s, %s de dépenses enregistrées.", info.DaysObserved, monthNameFr(info.Month), Euro(spent))
		if income > 0 {
			headline += fmt.Sprintf(" Revenus : %s. Solde : %s.", Euro(income), Euro(net))
		}
		headline += " Le mois est en cours : ce n'est pas le bilan définitif."
	} else if info.Status == "incomplete" {
		// Mois passé mais incomplet.
		headline = fmt.Sprintf("%s de dépenses sur %d/%d jours (%s). Données possiblement incomplètes.", Euro(spent), info.DaysObserved, info.DaysInMonth, info.Month)
	} else {
		// Mois complet historique.
		headline = fmt.Sprintf("Bilan %s : %s de dépenses, %s de revenus.", monthNameFr(info.Month), Euro(spent), Euro(income))
	}

	// Comparaison à la moyenne — SEULEMENT avec au moins 2 mois d'historique comparable.
	if hasHistory && average > 0 && !info.IsCurrent {
		delta := current - average
		pct := delta / average * 100
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
		details = append(details, fmt.Sprintf("Dépenses observées : %s (moyenne historique comparable : %s).", Euro(current), Euro(average)))
	} else if info.IsCurrent && hasHistory && average > 0 {
		// Mois courant : comparer aux mêmes jours des mois précédents.
		delta := current - average
		pct := delta / average * 100
		details = append(details, fmt.Sprintf("Comparé aux %d premiers jours des mois précédents : %s (moyenne : %s, %+.0f%%).", info.DaysObserved, Euro(current), Euro(average), pct))
	} else if !hasHistory {
		// Pas assez de recul.
		details = append(details, "Encore trop peu d'historique comparable pour te situer par rapport à tes habitudes.")
	}

	// Le poste (domaine) qui dérive le plus — seulement avec assez d'historique.
	if hasHistory && !info.IsCurrent {
		var domName string
		var domNow, domAvg float64
		q = `WITH cur AS (
				SELECT d.name, -SUM(t.amount) AS dep
				FROM transactions t JOIN domains d ON d.id = t.domain_id AND t.owner_id = $2
				WHERE t.amount < 0 AND NOT d.is_transfer AND to_char(t.tx_date,'YYYY-MM') = $1` + dayFilter + `
				GROUP BY d.name
			), hist AS (
				SELECT d.name, -SUM(t.amount) / GREATEST(COUNT(DISTINCT to_char(t.tx_date,'YYYY-MM')),1) AS avg_dep
				FROM transactions t JOIN domains d ON d.id = t.domain_id AND t.owner_id = $2
				WHERE t.amount < 0 AND NOT d.is_transfer AND to_char(t.tx_date,'YYYY-MM') < $1` + dayFilter + `
				GROUP BY d.name
			)
			SELECT cur.name, cur.dep, COALESCE(hist.avg_dep, 0)
			FROM cur LEFT JOIN hist ON hist.name = cur.name
			WHERE COALESCE(hist.avg_dep,0) > 0 AND cur.dep > hist.avg_dep * 1.3 AND cur.dep - hist.avg_dep > 40
			ORDER BY (cur.dep - COALESCE(hist.avg_dep,0)) DESC LIMIT 1`
		err := s.DB.QueryRow(q, ref, owner).Scan(&domName, &domNow, &domAvg)
		if err == nil && domName != "" {
			details = append(details, fmt.Sprintf("Poste en hausse : %s (%s vs %s d'ordinaire).", domName, Euro(domNow), Euro(domAvg)))
			if tone == "calm" {
				tone = "warn"
			}
		}
	}

	// Transactions non rangées : petite invitation à trier.
	var untagged int
	q = `SELECT COUNT(*) FROM transactions WHERE owner_id = $2 AND domain_id IS NULL AND to_char(tx_date,'YYYY-MM') = $1` + dayFilter
	s.DB.QueryRow(q, ref, owner).Scan(&untagged)
	if untagged > 0 {
		details = append(details, fmt.Sprintf("%d transaction%s à ranger.", untagged, Plural(untagged)))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"tone":     tone,
		"headline": headline,
		"details":  details,
		"period":   info,
		"untagged": untagged,
	})
}

func monthNameFr(ym string) string {
	mois := []string{"janvier", "février", "mars", "avril", "mai", "juin", "juillet", "août", "septembre", "octobre", "novembre", "décembre"}
	if len(ym) >= 7 {
		var y, m int
		fmt.Sscanf(ym, "%d-%d", &y, &m)
		if m >= 1 && m <= 12 {
			return mois[m-1]
		}
	}
	return ym
}

var _ = time.Now