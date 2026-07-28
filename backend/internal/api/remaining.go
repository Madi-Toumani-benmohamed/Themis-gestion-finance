package api

import (
	"net/http"
	"time"
)

// remainingItem : un prélèvement récurrent encore attendu d'ici la fin du mois.
type remainingItem struct {
	Label   string  `json:"label"`
	Amount  float64 `json:"amount"`   // positif (montant qui va sortir)
	DueDay  int     `json:"due_day"`  // jour habituel dans le mois
}

// remaining calcule le « reste à vivre » projeté : ce qu'il reste sur le compte
// une fois retirés les prélèvements récurrents encore attendus d'ici la fin du
// mois. Version « engagements seulement » : on ne soustrait que des
// prélèvements réels identifiés, jamais une estimation de dépenses variables.
// Chaque euro retiré correspond donc à un prélèvement que l'on peut nommer.
func (s *Server) remaining(w http.ResponseWriter, r *http.Request) {
	owner := s.ownerID(r)

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
		writeJSON(w, http.StatusOK, map[string]any{"available": false})
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
	lastDayOfMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location()).Day()

	upcoming := []remainingItem{}
	var committed float64
	seen := map[string]bool{}
	for _, g := range rec {
		if !g.Regular || g.LastDate == "" || seen[g.Label] {
			continue
		}
		last, e := time.Parse("2006-01-02", g.LastDate)
		if e != nil {
			continue
		}
		// Jour habituel du prélèvement, borné au dernier jour du mois courant
		// (ex. un prélèvement « le 31 » tombe le 30 en avril).
		dueDay := last.Day()
		if dueDay > lastDayOfMonth {
			dueDay = lastDayOfMonth
		}
		// On ne compte que ce qui n'est pas encore passé ce mois-ci. Et on évite
		// de recompter un prélèvement déjà survenu ce mois (déjà dans le solde).
		alreadyThisMonth := last.Year() == now.Year() && last.Month() == now.Month()
		if dueDay <= now.Day() || alreadyThisMonth {
			continue
		}
		amt := g.AvgAmount
		if amt <= 0 {
			amt = g.Amount
		}
		upcoming = append(upcoming, remainingItem{Label: cleanMerchant(g.Label), Amount: amt, DueDay: dueDay})
		committed += amt
		seen[g.Label] = true
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"available":     true,
		"balance":       estimated,
		"balance_date":  balanceDate,
		"committed":     committed,
		"remaining":     estimated - committed,
		"upcoming":      upcoming,
		"upcoming_count": len(upcoming),
	})
}
