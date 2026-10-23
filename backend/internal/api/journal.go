package api

import (
	"fmt"
	"net/http"
	"sort"
	"time"
)

// event est une entrée du journal financier : un fait daté, porteur de sens,
// éventuellement assorti d'une action. Le journal est le cœur de thémis —
// l'utilisateur lit une chronologie plutôt que d'interpréter des tableaux.
type event struct {
	Kind         string  `json:"kind"`       // salary | bill_paid | anomaly | win | upcoming | ...
	Date         string  `json:"date"`       // ISO YYYY-MM-DD
	Tone         string  `json:"tone"`       // positive | neutral | attention
	Importance   int     `json:"importance"` // 0..100 : sert au tri et au filtrage du bruit
	Title        string  `json:"title"`
	MerchantKind string  `json:"merchant_kind,omitempty"`
	Detail       string  `json:"detail"`
	Amount       float64 `json:"amount,omitempty"`
	Action       *evtAct `json:"action,omitempty"`
}

type evtAct struct {
	Label string `json:"label"`
	Kind  string `json:"kind"` // navigate | create_rule | ...
	Arg   string `json:"arg,omitempty"`
}

// journalFeed produit la chronologie d'événements récents et à venir. Il
// agrège des sources déjà existantes (anomalies, récurrences) et en ajoute
// deux peu coûteuses (victoires, échéances à venir). L'enjeu n'est pas la
// quantité mais la pertinence : on trie par importance et on plafonne.
func (s *Server) journalFeed(w http.ResponseWriter, r *http.Request) {
	owner := s.ownerID(r)

	// Mois de référence : paramètre 'month' (YYYY-MM) ou mois le plus récent contenant des transactions.
	month := r.URL.Query().Get("month")
	var ref string
	if month != "" {
		ref = month
	} else if s.DB.QueryRow(`SELECT to_char(max(tx_date),'YYYY-MM') FROM transactions WHERE owner_id = $1`, owner).Scan(&ref); ref == "" {
		writeJSON(w, http.StatusOK, map[string]any{"events": []event{}})
		return
	}

	events := []event{}

	events = append(events, s.eventsFromTransactions(owner, ref)...)
	events = append(events, s.eventsFromAnomalies(owner)...)
	events = append(events, s.eventsUpcoming(owner)...)
	events = append(events, s.eventsWins(owner, ref)...)

	// Tri : les faits les plus significatifs en tête (importance), puis, à
	// importance égale, le plus récent d'abord. Un journal met en avant ce qui
	// compte, pas seulement ce qui est récent.
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].Importance != events[j].Importance {
			return events[i].Importance > events[j].Importance
		}
		return events[i].Date > events[j].Date
	})

	// Plafond : un journal lisible, pas un mur. 25 entrées max.
	if len(events) > 25 {
		events = events[:25]
	}

	writeJSON(w, http.StatusOK, map[string]any{"events": events})
}

// eventsFromTransactions : le revenu affiché est le CUMUL des transactions que
// l'utilisateur a explicitement taguées « salaire » sur le mois de référence.
// On ne devine plus à partir du signe (un crédit peut être un remboursement, un
// virement d'un proche…) : seule l'intention posée par le tag fait foi. C'est
// fiable et juste, au prix d'exiger que les salaires soient tagués.
func (s *Server) eventsFromTransactions(owner int64, month string) []event {
	out := []event{}

	// Cumul des montants (positifs) tagués « salaire » sur ce mois. Le tag est
	// comparé sans tenir compte de la casse ; on accepte « salaire »/« salaires ».
	var total float64
	var n int
	var lastDate string
	err := s.DB.QueryRow(`
		SELECT COALESCE(SUM(t.amount),0), COUNT(*), COALESCE(MAX(t.tx_date)::text,'')
		FROM transactions t
		JOIN transaction_tags tt ON tt.transaction_id = t.id
		JOIN tags tg ON tg.id = tt.tag_id
		WHERE t.owner_id = $1
		  AND t.amount > 0
		  AND to_char(t.tx_date,'YYYY-MM') = $2
		  AND lower(tg.name) IN ('salaire','salaires')`,
		owner, month).Scan(&total, &n, &lastDate)
	if err != nil || n == 0 || total <= 0 {
		return out // aucun salaire tagué ce mois-ci : pas d'événement (plutôt qu'un chiffre faux)
	}

	detail := "Salaire perçu ce mois-ci."
	if n > 1 {
		detail = fmt.Sprintf("Cumul de %d versements de salaire ce mois-ci.", n)
	}
	out = append(out, event{
		Kind: "income", Date: lastDate, Tone: "positive", Importance: 70,
		Title: "Salaire du mois", Detail: detail, Amount: total,
	})
	return out
}

// eventsFromAnomalies réutilise le moteur d'anomalies existant : chaque
// anomalie devient un événement daté d'aujourd'hui (elle vient d'être détectée).
func (s *Server) eventsFromAnomalies(owner int64) []event {
	rec, err := computeRecurring(s.DB, owner, 2)
	if err != nil {
		return nil
	}
	anomalies := detectAnomalies(rec)
	today := time.Now().Format("2006-01-02")
	out := make([]event, 0, len(anomalies))
	for _, a := range anomalies {
		tone := "attention"
		imp := 65
		if a.Severity == "info" {
			tone, imp = "neutral", 45
		}
		ev := event{
			Kind: "anomaly", Date: today, Tone: tone, Importance: imp,
			Title: a.Title, Detail: a.Detail, Amount: a.Amount, MerchantKind: a.MerchantKind,
		}
		// Une hausse d'abonnement invite à créer une règle / vérifier.
		if a.Kind == "subscription_up" {
			ev.Action = &evtAct{Label: "Voir les transactions", Kind: "navigate", Arg: "transactions"}
		}
		out = append(out, ev)
	}
	return out
}

// eventsUpcoming : le calendrier à venir. À partir des prélèvements réguliers,
// on projette la prochaine échéance dans les 20 prochains jours. « Vois venir
// ton mois » sans rien saisir.
func (s *Server) eventsUpcoming(owner int64) []event {
	rec, err := computeRecurring(s.DB, owner, 3)
	if err != nil {
		return nil
	}
	now := time.Now()
	out := []event{}
	for _, g := range rec {
		if !g.Regular || g.LastDate == "" {
			continue
		}
		// Même filtre de pertinence que les anomalies : pas de prédiction pour
		// un prélèvement de quelques euros ni pour un faux marchand.
		if looksLikeNoise(g.Label, g.AvgAmount) {
			continue
		}
		last, err := time.Parse("2006-01-02", g.LastDate)
		if err != nil {
			continue
		}
		// Prochaine occurrence estimée = dernière + ~1 mois, ramenée dans le futur.
		next := last.AddDate(0, 1, 0)
		for next.Before(now) {
			next = next.AddDate(0, 1, 0)
		}
		days := int(next.Sub(now).Hours() / 24)
		if days < 0 || days > 20 {
			continue
		}
		out = append(out, event{
			Kind: "upcoming", Date: next.Format("2006-01-02"), Tone: "neutral", Importance: 55,
			Title:  fmt.Sprintf("Habituellement à cette période : %s", g.Label),
			Detail: fmt.Sprintf("Un prélèvement d'environ %s est généralement observé %s, d'après tes relevés.", Euro(g.AvgAmount), relativeDays(days)),
			Amount: -g.AvgAmount,
		})
	}
	return out
}

// eventsWins : les victoires. Contrepoids délibéré à la détection d'anomalies —
// thémis célèbre aussi les réussites, pas seulement les problèmes.
func (s *Server) eventsWins(owner int64, month string) []event {
	out := []event{}
	today := time.Now().Format("2006-01-02")

	// Victoire 1 : le mois de référence dégage un solde positif.
	var income, spent float64
	s.DB.QueryRow(`
		SELECT COALESCE(SUM(amount) FILTER (WHERE amount>0),0),
		       COALESCE(-SUM(amount) FILTER (WHERE amount<0),0)
		FROM transactions WHERE owner_id=$1 AND to_char(tx_date,'YYYY-MM')=$2`+notTransfer, owner, month).Scan(&income, &spent)
	if income > 0 && income > spent {
		out = append(out, event{
			Kind: "win", Date: today, Tone: "positive", Importance: 60,
			Title:  "Mois dans le vert",
			Detail: fmt.Sprintf("Tu as mis de côté %s ce mois-ci. Beau travail.", Euro(income-spent)),
			Amount: income - spent,
		})
	}

	// Victoire 2 : tous les budgets du mois sont respectés (aucun dépassement).
	var overBudget int
	s.DB.QueryRow(`
		SELECT count(*) FROM budgets b
		WHERE b.owner_id = $1 AND (
			SELECT COALESCE(-SUM(t.amount) FILTER (WHERE t.amount<0),0)
			FROM transactions t
			WHERE t.owner_id=$1 AND t.domain_id=b.domain_id
			  AND to_char(t.tx_date,'YYYY-MM')=$2
		) > b.monthly_limit`, owner, month).Scan(&overBudget)
	var totalBudgets int
	s.DB.QueryRow(`SELECT count(*) FROM budgets WHERE owner_id=$1`, owner).Scan(&totalBudgets)
	if totalBudgets > 0 && overBudget == 0 {
		out = append(out, event{
			Kind: "win", Date: today, Tone: "positive", Importance: 58,
			Title:  "Tous les budgets tenus",
			Detail: fmt.Sprintf("Tes %d budget(s) sont respectés ce mois-ci.", totalBudgets),
		})
	}

	return out
}

// relativeDays met en mots un délai court en jours.
func relativeDays(d int) string {
	switch {
	case d <= 0:
		return "aujourd'hui"
	case d == 1:
		return "demain"
	default:
		return fmt.Sprintf("dans %d jours", d)
	}
}
