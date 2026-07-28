package api

import (
	"fmt"

	"github.com/lib/pq"
	"net/http"
	"sort"
	"strings"
	"time"
)

// ── Vague 2 : repérer tout seul, sans rien affirmer à tort ──
// Détections purement déterministes sur les données déjà là. Chaque découverte
// est formulée au conditionnel : thémis signale un motif possible, il ne
// tranche pas. L'utilisateur juge.

type discovery struct {
	Kind    string `json:"kind"` // duplicate_subs | price_creep | seasonal | merchant_drift
	Title   string `json:"title"`
	Detail  string `json:"detail"`
	Tone    string `json:"tone"` // info | attention
	Amount  float64 `json:"amount,omitempty"`
	Query   string `json:"query,omitempty"` // pour rebondir vers les transactions
	// Scope indique comment interpréter le clic « voir les transactions » :
	//   "month" → filtrer sur le mois de référence (analyse mensuelle)
	//   ""      → tout l'historique du libellé/tag (analyse de fond)
	Scope   string `json:"scope,omitempty"`
	Month   string `json:"month,omitempty"` // YYYY-MM quand Scope == "month"
}

// segments de services connus : plusieurs abonnements d'un même segment
// suggèrent un possible doublon (plusieurs plateformes vidéo, musique…).
var serviceSegments = map[string][]string{
	"vidéo à la demande": {"Netflix", "Disney+", "Canal+", "Amazon Prime", "OCS", "Paramount", "Apple TV"},
	"musique":            {"Spotify", "Deezer", "Apple", "YouTube"},
	"stockage cloud":     {"Google", "Dropbox", "iCloud", "Microsoft"},
}

// discoveries produit les découvertes de la vague 2.
func (s *Server) discoveries(w http.ResponseWriter, r *http.Request) {
	owner := s.ownerID(r)
	out := []discovery{}

	rec, err := computeRecurring(s.DB, owner, 2)
	if err == nil {
		out = append(out, detectDuplicateSubs(rec)...)
		out = append(out, detectPriceCreep(rec)...)
	}
	out = append(out, s.detectSeasonal(owner)...)
	out = append(out, s.detectTagSurge(owner)...)
	out = append(out, s.detectTagProject(owner)...)
	drifts := s.detectMerchantDrift(owner)
	// Dédup : si un marchand est déjà signalé « en dérive » (tendance de fond),
	// on retire son signal « prix qui grimpe » — le drift est plus parlant et
	// éviter le doublon fait partie de ne pas crier au loup.
	driftNames := map[string]bool{}
	for _, d := range drifts {
		driftNames[strings.ToLower(d.Query)] = true
	}
	filtered := out[:0]
	for _, d := range out {
		if d.Kind == "price_creep" && driftNames[strings.ToLower(d.Query)] {
			continue
		}
		filtered = append(filtered, d)
	}
	out = append(filtered, drifts...)

	writeJSON(w, http.StatusOK, map[string]any{"discoveries": out})
}

// detectDuplicateSubs : plusieurs abonnements actifs du même segment.
func detectDuplicateSubs(rec []recurring) []discovery {
	// abonnements « actifs » = récurrents mensuels de montant modéré
	active := map[string]float64{} // marchand -> montant mensuel
	for _, g := range rec {
		if len(g.Months) >= 2 && g.AvgAmount > 0 && g.AvgAmount < 60 {
			active[strings.ToLower(g.Label)] = g.AvgAmount
		}
	}
	var out []discovery
	for segment, members := range serviceSegments {
		found := []string{}
		total := 0.0
		for _, m := range members {
			for lbl, amt := range active {
				if strings.Contains(lbl, strings.ToLower(m)) {
					found = append(found, m)
					total += amt
					break
				}
			}
		}
		if len(found) >= 2 {
			sort.Strings(found)
			out = append(out, discovery{
				Kind:  "duplicate_subs",
				Title: fmt.Sprintf("Plusieurs abonnements %s", segment),
				Detail: fmt.Sprintf("Tu sembles avoir %d abonnements %s actifs (%s), soit environ %s par mois. Un possible doublon à regarder.",
					len(found), segment, strings.Join(found, ", "), euro(total)),
				Tone: "attention", Amount: total,
			})
		}
	}
	return out
}

// detectPriceCreep : abonnement dont le dernier montant dépasse nettement la
// moyenne (hausse de tarif passée inaperçue). Complète l'anomalie existante en
// se concentrant sur les petits abonnements réguliers.
func detectPriceCreep(rec []recurring) []discovery {
	var out []discovery
	for _, g := range rec {
		if len(g.Months) < 3 || g.AvgAmount <= 0 || g.AvgAmount > 100 {
			continue
		}
		last := g.Amount
		if last > g.AvgAmount*1.15 && last-g.AvgAmount >= 1.5 {
			out = append(out, discovery{
				Kind:  "price_creep",
				Title: fmt.Sprintf("%s a peut-être augmenté", g.Label),
				Detail: fmt.Sprintf("Le dernier prélèvement (%s) dépasse la moyenne habituelle (%s). Le tarif a peut-être changé.",
					euro(last), euro(g.AvgAmount)),
				Tone: "attention", Amount: last, Query: g.Label,
				Scope: "month", Month: lastMonthOf(g.LastDate),
			})
		}
	}
	return out
}

// detectSeasonal : dépenses qui reviennent à date fixe chaque année (assurance
// annuelle, impôts…) et qui approchent. Purement basé sur l'historique.
func (s *Server) detectSeasonal(owner int64) []discovery {
	rows, err := s.DB.Query(`
		SELECT COALESCE(NULLIF(merchant,''), display_label, label) AS m,
		       EXTRACT(MONTH FROM tx_date)::int AS mois,
		       -amount AS amt, tx_date::text
		FROM transactions
		WHERE owner_id = $1 AND amount < 0
		  AND (domain_id IS NULL OR domain_id NOT IN (SELECT id FROM domains WHERE is_transfer))
		  AND -amount >= 80
		ORDER BY m, tx_date`, owner)
	if err != nil {
		return nil
	}
	defer rows.Close()
	// pour chaque marchand : les mois (de l'année) où il apparaît, sur des
	// années différentes → candidat saisonnier.
	type occ struct {
		months map[int]bool
		years  map[string]bool
		amt    float64
		lastM  int
	}
	byM := map[string]*occ{}
	for rows.Next() {
		var m, dstr string
		var mois int
		var amt float64
		if rows.Scan(&m, &mois, &amt, &dstr) != nil {
			continue
		}
		o := byM[m]
		if o == nil {
			o = &occ{months: map[int]bool{}, years: map[string]bool{}}
			byM[m] = o
		}
		o.months[mois] = true
		if len(dstr) >= 4 {
			o.years[dstr[:4]] = true
		}
		o.amt = amt
		o.lastM = mois
	}
	now := time.Now()
	var out []discovery
	for name, o := range byM {
		// saisonnier = revient sur >=2 années, concentré sur 1-2 mois de l'année
		if len(o.years) >= 2 && len(o.months) <= 2 {
			// approche : le mois saisonnier est le mois prochain ?
			next := int(now.Month())%12 + 1
			for mo := range o.months {
				if mo == next {
					out = append(out, discovery{
						Kind:  "seasonal",
						Title: fmt.Sprintf("%s revient bientôt", name),
						Detail: fmt.Sprintf("Cette dépense (environ %s) est apparue à cette période les années précédentes. Elle pourrait revenir le mois prochain.",
							euro(o.amt)),
						Tone: "info", Amount: o.amt, Query: name,
					})
					break
				}
			}
		}
	}
	return out
}

// detectMerchantDrift : marchand chez qui la dépense augmente régulièrement sur
// les derniers mois (dérive douce, invisible au quotidien).
func (s *Server) detectMerchantDrift(owner int64) []discovery {
	rows, err := s.DB.Query(`
		SELECT m, arr FROM (
			SELECT COALESCE(NULLIF(merchant,''), display_label, label) AS m,
			       array_agg(monthly ORDER BY ym) AS arr
			FROM (
				SELECT COALESCE(NULLIF(merchant,''), display_label, label) AS merchant,
				       display_label, label,
				       to_char(tx_date,'YYYY-MM') AS ym,
				       SUM(-amount) AS monthly
				FROM transactions
				WHERE owner_id = $1 AND amount < 0
				  AND (domain_id IS NULL OR domain_id NOT IN (SELECT id FROM domains WHERE is_transfer))
				  AND tx_date >= CURRENT_DATE - INTERVAL '4 months'
				GROUP BY 1,2,3, to_char(tx_date,'YYYY-MM')
			) q
			GROUP BY m
		) z
		WHERE array_length(arr,1) >= 3`, owner)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []discovery
	for rows.Next() {
		var name string
		var vals []float64
		if rows.Scan(&name, pq.Array(&vals)) != nil {
			continue
		}
		if len(vals) < 3 {
			continue
		}
		// strictement croissant sur les 3 derniers mois, et hausse ≥ 40 %
		strictlyUp := true
		for i := 1; i < len(vals); i++ {
			if vals[i] <= vals[i-1] {
				strictlyUp = false
				break
			}
		}
		if strictlyUp && vals[0] > 0 && (vals[len(vals)-1]-vals[0])/vals[0] >= 0.4 {
			out = append(out, discovery{
				Kind:  "merchant_drift",
				Title: fmt.Sprintf("Tes dépenses chez %s augmentent", name),
				Detail: fmt.Sprintf("De %s à %s par mois sur les derniers mois. Une tendance qui monte, à surveiller si ce n'est pas voulu.",
					euro(vals[0]), euro(vals[len(vals)-1])),
				Tone: "attention", Amount: vals[len(vals)-1], Query: name,
			})
		}
	}
	return out
}

// detectTagSurge : un tag RÉCURRENT (présent sur plusieurs mois) dont le total
// du mois de référence dépasse nettement sa moyenne habituelle. Répond au vécu
// (« restau », « livraison ») mieux qu'une catégorie.
func (s *Server) detectTagSurge(owner int64) []discovery {
	var ref string
	if s.DB.QueryRow(`SELECT to_char(max(tx_date),'YYYY-MM') FROM transactions WHERE owner_id=$1`, owner).Scan(&ref); ref == "" {
		return nil
	}
	rows, err := s.DB.Query(`
		WITH tagmonth AS (
			SELECT tg.name AS tag, to_char(t.tx_date,'YYYY-MM') AS ym, SUM(-t.amount) AS spent
			FROM transactions t
			JOIN transaction_tags tt ON tt.transaction_id = t.id
			JOIN tags tg ON tg.id = tt.tag_id
			WHERE t.owner_id = $1 AND t.amount < 0
			  AND (t.domain_id IS NULL OR t.domain_id NOT IN (SELECT id FROM domains WHERE is_transfer))
			GROUP BY tg.name, to_char(t.tx_date,'YYYY-MM')
		)
		SELECT tag,
		       MAX(spent) FILTER (WHERE ym = $2) AS cur,
		       AVG(spent) FILTER (WHERE ym <> $2) AS avg_prev,
		       COUNT(*)   FILTER (WHERE ym <> $2) AS prev_months
		FROM tagmonth
		GROUP BY tag
		HAVING MAX(spent) FILTER (WHERE ym = $2) IS NOT NULL
		   AND COUNT(*) FILTER (WHERE ym <> $2) >= 2`, owner, ref)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []discovery
	for rows.Next() {
		var tag string
		var cur, avgPrev float64
		var prevMonths int
		if rows.Scan(&tag, &cur, &avgPrev, &prevMonths) != nil {
			continue
		}
		// hausse nette : > 60 % au-dessus de l'habitude et écart significatif
		if avgPrev > 0 && cur > avgPrev*1.6 && cur-avgPrev >= 50 {
			pct := int((cur - avgPrev) / avgPrev * 100)
			out = append(out, discovery{
				Kind:  "tag_surge",
				Title: fmt.Sprintf("Beaucoup de « %s » ce mois-ci", tag),
				Detail: fmt.Sprintf("Tu as %s de dépenses taguées « %s » ce mois, contre %s en moyenne les mois précédents (+%d%%).",
					euro(cur), tag, euro(avgPrev), pct),
				Tone: "attention", Amount: cur, Query: tag, Scope: "month", Month: ref,
			})
		}
	}
	return out
}

// detectTagProject : un tag PONCTUEL (concentré sur une courte période, puis
// plus rien) → un bilan de projet, pas une alerte. « Ton projet travaux
// totalise X sur N semaines. »
func (s *Server) detectTagProject(owner int64) []discovery {
	rows, err := s.DB.Query(`
		SELECT tg.name,
		       SUM(-t.amount)                              AS total,
		       MIN(t.tx_date)                              AS first_d,
		       MAX(t.tx_date)                              AS last_d,
		       COUNT(DISTINCT to_char(t.tx_date,'YYYY-MM')) AS months_span,
		       COUNT(*)                                    AS n
		FROM transactions t
		JOIN transaction_tags tt ON tt.transaction_id = t.id
		JOIN tags tg ON tg.id = tt.tag_id
		WHERE t.owner_id = $1 AND t.amount < 0
		  AND (t.domain_id IS NULL OR t.domain_id NOT IN (SELECT id FROM domains WHERE is_transfer))
		GROUP BY tg.name
		HAVING SUM(-t.amount) >= 200
		   AND COUNT(*) >= 3
		   AND MAX(t.tx_date) < CURRENT_DATE - INTERVAL '10 days'   -- terminé récemment
		   AND MAX(t.tx_date) > CURRENT_DATE - INTERVAL '90 days'   -- mais pas trop vieux
		   AND COUNT(DISTINCT to_char(t.tx_date,'YYYY-MM')) <= 3`, owner)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []discovery
	for rows.Next() {
		var tag string
		var total float64
		var firstD, lastD time.Time
		var monthsSpan, n int
		if rows.Scan(&tag, &total, &firstD, &lastD, &monthsSpan, &n) != nil {
			continue
		}
		days := int(lastD.Sub(firstD).Hours()/24) + 1
		span := fmt.Sprintf("%d jours", days)
		if days >= 14 {
			span = fmt.Sprintf("%d semaines", (days+3)/7)
		}
		out = append(out, discovery{
			Kind:  "tag_project",
			Title: fmt.Sprintf("Bilan « %s »", tag),
			Detail: fmt.Sprintf("Ce tag totalise %s sur %s (%d dépenses). Il semble terminé — voici son coût complet.",
				euro(total), span, n),
			Tone: "info", Amount: total, Query: tag,
		})
	}
	return out
}

// lastMonthOf extrait le mois (YYYY-MM) d'une date YYYY-MM-DD.
func lastMonthOf(d string) string {
	if len(d) >= 7 {
		return d[:7]
	}
	return ""
}
