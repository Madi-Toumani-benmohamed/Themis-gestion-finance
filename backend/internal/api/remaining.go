package api

import (
	"fmt"
	"net/http"
	"themis/internal/money"
	"time"
)

// remainingItem : un prélèvement récurrent encore attendu d'ici la fin du mois.
type remainingItem struct {
	Label  string  `json:"label"`
	Amount float64 `json:"amount"`  // positif (montant qui va sortir)
	DueDay int     `json:"due_day"` // jour habituel dans le mois
}

// remaining calcule le « reste à vivre » projeté : ce qu'il reste sur le compte
// une fois retirés les prélèvements récurrents encore attendus d'ici la fin du
// mois. Version « engagements seulement » : on ne soustrait que des
// prélèvements réels identifiés, jamais une estimation de dépenses variables.
// Chaque euro retiré correspond donc à un prélèvement que l'on peut nommer.
// Pour les mois historiques, renvoie le solde réel (revenus - dépenses) au lieu
// d'une projection, car le mois est terminé.
func (s *Server) remaining(w http.ResponseWriter, r *http.Request) {
	owner := s.ownerID(r)

	// Mois de référence : paramètre 'month' (YYYY-MM) ou mois courant.
	month := r.URL.Query().Get("month")
	if month == "" {
		month = time.Now().Format("2006-01")
	}

	info, periodErr := s.getPeriodInfo(owner, month)
	if periodErr != nil {
		writeErr(w, http.StatusBadRequest, periodErr)
		return
	}

	// Pour les mois historiques (pas le mois en cours), on renvoie le bilan réel.
	if !info.IsCurrent {
		var income, spent float64
		dayFilter := ""
		if info.Status == "incomplete" && info.DaysObserved > 0 {
			dayFilter = fmt.Sprintf(" AND EXTRACT(DAY FROM tx_date) <= %d", info.DaysObserved)
		}
		q := `SELECT COALESCE(SUM(amount) FILTER (WHERE amount > 0), 0),
			       COALESCE(-SUM(amount) FILTER (WHERE amount < 0), 0)
			FROM transactions WHERE owner_id = $2 AND to_char(tx_date,'YYYY-MM') = $1` + notTransfer + dayFilter
		s.DB.QueryRow(q, month, owner).Scan(&income, &spent)
		net := income - spent

		writeJSON(w, http.StatusOK, map[string]any{
			"available":      true,
			"period":         info,
			"balance":        nil,
			"balance_date":   nil,
			"committed":      0,
			"remaining":      net,
			"upcoming":       []remainingItem{},
			"upcoming_count": 0,
			"is_historical":  true,
			"income":         income,
			"expenses":       spent,
		})
		return
	}

	// Mois en cours : calcul de la projection (logique existante).
	// 1. Solde de départ : clôture du dernier relevé qui en fournit un.
	var balance float64
	var balanceDate string
	err := s.DB.QueryRow(`
		SELECT closing_balance, period_end::text
		FROM statements
		WHERE owner_id = $1 AND closing_balance IS NOT NULL AND period_end IS NOT NULL
		ORDER BY period_end DESC, id DESC LIMIT 1`, owner).Scan(&balance, &balanceDate)
	if err != nil {
		// Pas de solde connu : on ne peut pas projeter honnêtement.
		writeJSON(w, http.StatusOK, map[string]any{"available": false, "period": info})
		return
	}

	// 2. Affiner : ajouter les transactions saisies APRÈS la date du solde
	//    (elles ne sont pas encore reflétées dans closing_balance).
	var delta float64
	_ = s.DB.QueryRow(`
		SELECT COALESCE(SUM(amount),0) FROM transactions
		WHERE owner_id = $1 AND tx_date > $2::date`, owner, balanceDate).Scan(&delta)
	estimated := balance + delta

	// 3. Prélèvements récurrents encore attendus d'ici la fin du mois. On repart
	//    du moteur de récurrences (régulières = montant stable = engagement).
	//    Seuil à 2 mois : un prélèvement vu en mai et juin est un engagement
	//    crédible pour juillet, même si l'occurrence de juillet n'a pas encore eu
	//    lieu (elle n'existe donc pas encore comme transaction).
	rec, err := computeRecurring(s.DB, owner, 2)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	now := time.Now()
	upcoming := []remainingItem{}
	var committedCents money.Cents
	seen := map[string]bool{}
	for _, g := range rec {
		key := g.Key
		if key == "" {
			key = g.Label
		}
		if !g.Regular || g.EstimatedNextDate == nil || seen[key] {
			continue
		}
		next, e := time.Parse("2006-01-02", *g.EstimatedNextDate)
		if e != nil || !next.After(now) || next.Format("2006-01") != month {
			continue
		}
		amount, e := money.FromLegacy(g.AvgAmount)
		if e != nil || amount <= 0 {
			continue
		}
		upcoming = append(upcoming, remainingItem{Label: cleanMerchant(g.Label), Amount: float64(amount) / 100, DueDay: next.Day()})
		committedCents += amount
		seen[key] = true
	}
	committed := float64(committedCents) / 100

	writeJSON(w, http.StatusOK, map[string]any{
		"available":      true,
		"period":         info,
		"balance":        estimated,
		"balance_date":   balanceDate,
		"committed":      committed,
		"remaining":      estimated - committed,
		"upcoming":       upcoming,
		"upcoming_count": len(upcoming),
		"is_historical":  false,
	})
}
