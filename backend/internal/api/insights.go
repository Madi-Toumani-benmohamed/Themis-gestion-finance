package api

import (
	"fmt"
	"math"
	"net/http"
	"time"
)

// Un insight est quelque chose que l'utilisateur aurait dû penser à vérifier —
// thémis le fait à sa place. severity : "info" | "warn" | "alert".
type insight struct {
	Kind     string  `json:"kind"`
	Severity string  `json:"severity"`
	Title    string  `json:"title"`
	Detail   string  `json:"detail,omitempty"`
	Count    int     `json:"count,omitempty"`
	Amount   float64 `json:"amount,omitempty"`
}

func fmtEUR(v float64) string {
	return fmt.Sprintf("%.2f €", v)
}

const insightCategorySpikeSQL = `WITH monthly AS (
			SELECT to_char(t.tx_date, 'YYYY-MM') AS m, t.domain_id, -SUM(t.amount) AS spent
			FROM transactions t
			WHERE t.amount < 0 AND t.domain_id IS NOT NULL AND t.owner_id = $1` + notTransferT + `
			GROUP BY 1, 2
		), last AS (SELECT max(m) AS m FROM monthly)
		SELECT c.name, cur.spent, COALESCE(avg(hist.spent), 0)
		FROM monthly cur
		JOIN last ON cur.m = last.m
		JOIN domains c ON c.id = cur.domain_id AND c.owner_id=$1
		LEFT JOIN monthly hist ON hist.domain_id = cur.domain_id AND hist.m < cur.m
		GROUP BY c.name, cur.spent
		HAVING count(hist.m) >= 2`

// listInsights agrège tout ce qui mérite l'attention : relevé du mois manquant,
// transactions à trier, abonnements en hausse, catégories en forte hausse,
// budgets dépassés. Une liste vide = tout est en ordre.
func (s *Server) listInsights(w http.ResponseWriter, r *http.Request) {
	owner := s.ownerID(r)
	out := []insight{}
	now := time.Now()

	// 1. Relevé du mois précédent manquant (grâce de 5 jours en début de mois :
	//    les banques émettent les relevés dans les premiers jours)
	if now.Day() >= 5 {
		prevMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
		var covered bool
		err := s.DB.QueryRow(`SELECT EXISTS(
			SELECT 1 FROM statements WHERE owner_id = $2 AND period_end >= $1)`, prevMonthStart, owner).Scan(&covered)
		var any bool
		_ = s.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM statements WHERE owner_id = $1)`, owner).Scan(&any)
		if err == nil && any && !covered {
			out = append(out, insight{
				Kind: "missing_statement", Severity: "warn",
				Title:  "Relevé du mois dernier non déposé",
				Detail: "Dépose-le dans la fente en haut de page — deux minutes, et tout le reste est automatique.",
			})
		}
	}

	// 2. Transactions à trier
	var uncat int
	if err := s.DB.QueryRow(`SELECT count(*) FROM transactions WHERE domain_id IS NULL AND owner_id = $1`, owner).Scan(&uncat); err == nil && uncat > 0 {
		out = append(out, insight{
			Kind: "uncategorized", Severity: "info",
			Title:  fmt.Sprintf("%d transaction%s à trier", uncat, Plural(uncat)),
			Detail: "Onglet Transactions — ou laisse l'IA les classer si elle est configurée.",
			Count:  uncat,
		})
	}

	// 3. Abonnements en hausse : dernier montant > précédent (seuil 0,30 € et 2 %)
	if recs, err := computeRecurring(s.DB, s.ownerID(r), 2); err == nil {
		for _, rc := range recs {
			if !rc.Regular || len(rc.Amounts) < 2 {
				continue
			}
			last := rc.Amounts[len(rc.Amounts)-1]
			prev := rc.Amounts[len(rc.Amounts)-2]
			if last-prev >= 0.30 && (last-prev)/prev >= 0.02 {
				out = append(out, insight{
					Kind: "subscription_increase", Severity: "alert",
					Title:  fmt.Sprintf("Abonnement en hausse : %s", shorten(rc.Label, 40)),
					Detail: fmt.Sprintf("%s → %s par mois (+%s). Négociation ou résiliation ?", fmtEUR(prev), fmtEUR(last), fmtEUR(last-prev)),
					Amount: last - prev,
				})
			}
		}
	}

	// 4. Catégorie en forte hausse : dernier mois complet vs moyenne des 3 précédents
	rows, err := s.DB.Query(insightCategorySpikeSQL, owner)
	if err == nil {
		for rows.Next() {
			var name string
			var cur, avg float64
			if rows.Scan(&name, &cur, &avg) == nil && avg > 0 && cur > avg*1.5 && cur-avg > 50 {
				out = append(out, insight{
					Kind: "category_spike", Severity: "warn",
					Title:  fmt.Sprintf("%s : %s ce mois-ci", name, fmtEUR(cur)),
					Detail: fmt.Sprintf("+%d %% par rapport à ton habitude (%s en moyenne).", int(math.Round((cur-avg)/avg*100)), fmtEUR(avg)),
					Amount: cur - avg,
				})
			}
		}
		rows.Close()
	}

	// 5. Dépassements réellement observés, même périmètre que Budgets.
	report, budgetErr := s.loadBudgetReport(r.Context(), owner, now.Format("2006-01"), now)
	if budgetErr == nil {
		for _, budget := range report.Budgets {
			if budget.Scope != "expense" || !budget.Exceeded {
				continue
			}
			out = append(out, insight{Kind: "budget_overrun", Severity: "alert", Title: fmt.Sprintf("Budget %s dépassé", budget.DomainName), Detail: fmt.Sprintf("%s € de débits importés sur %s € (+%s €).", budget.Spent, budget.MonthlyLimit, (budget.spent - budget.limit).String()), Amount: float64(budget.spent-budget.limit) / 100})
		}
	}

	writeJSON(w, http.StatusOK, out)
}

func shorten(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
