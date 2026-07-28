package api

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ---------- Budgets ----------

type budgetStatus struct {
	ID           int64   `json:"id"`
	DomainID   int64   `json:"domain_id"`
	DomainName string  `json:"domain_name"`
	MonthlyLimit float64 `json:"monthly_limit"`
	Spent        float64 `json:"spent"`   // dépenses du mois courant (positif)
	Ratio        float64 `json:"ratio"`   // spent / limit
	Projected    float64 `json:"projected"` // projection fin de mois au rythme actuel
}

// listBudgets retourne les budgets avec la consommation du mois demandé
// (?month=YYYY-MM, défaut : mois courant).
func (s *Server) listBudgets(w http.ResponseWriter, r *http.Request) {
	month := r.URL.Query().Get("month")
	if !reMonth.MatchString(month) {
		month = time.Now().Format("2006-01")
	}
	rows, err := s.DB.Query(`
		SELECT b.id, b.domain_id, d.name, b.monthly_limit,
		       COALESCE(-SUM(t.amount) FILTER (WHERE t.amount < 0), 0) AS spent
		FROM budgets b
		JOIN domains d ON d.id = b.domain_id
		LEFT JOIN transactions t
		       ON t.domain_id = b.domain_id AND t.owner_id = b.owner_id
		      AND to_char(t.tx_date, 'YYYY-MM') = $1
		WHERE b.owner_id = $2
		GROUP BY b.id, b.domain_id, d.name, b.monthly_limit
		ORDER BY COALESCE(-SUM(t.amount) FILTER (WHERE t.amount < 0), 0) / b.monthly_limit DESC`, month, s.ownerID(r))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	// projection : dépense actuelle extrapolée au prorata des jours écoulés
	// (uniquement si le mois demandé est le mois courant)
	prorata := 1.0
	if month == time.Now().Format("2006-01") {
		now := time.Now()
		daysIn := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
		prorata = float64(daysIn) / float64(now.Day())
	}

	out := []budgetStatus{}
	for rows.Next() {
		var b budgetStatus
		if err := rows.Scan(&b.ID, &b.DomainID, &b.DomainName, &b.MonthlyLimit, &b.Spent); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		b.Ratio = b.Spent / b.MonthlyLimit
		b.Projected = math.Round(b.Spent*prorata*100) / 100
		out = append(out, b)
	}
	writeJSON(w, http.StatusOK, out)
}

// upsertBudget crée ou met à jour le budget d'une catégorie.
func (s *Server) upsertBudget(w http.ResponseWriter, r *http.Request) {
	var in struct {
		DomainID     int64   `json:"domain_id"`
		MonthlyLimit float64 `json:"monthly_limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if in.DomainID == 0 || in.MonthlyLimit <= 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("domain_id et monthly_limit (> 0) requis"))
		return
	}
	_, err := s.DB.Exec(`
		INSERT INTO budgets (domain_id, monthly_limit, owner_id) VALUES ($1, $2, $3)
		ON CONFLICT (owner_id, domain_id) DO UPDATE SET monthly_limit = EXCLUDED.monthly_limit`,
		in.DomainID, in.MonthlyLimit, s.ownerID(r))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) deleteBudget(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if _, err := s.DB.Exec(`DELETE FROM budgets WHERE id = $1 AND owner_id = $2`, id, s.ownerID(r)); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ---------- Dépenses récurrentes / abonnements ----------

type recurring struct {
	Label      string    `json:"label"`      // libellé normalisé représentatif
	Amount     float64   `json:"amount"`     // dernier montant (positif)
	AvgAmount  float64   `json:"avg_amount"` // moyenne (positif)
	Months     []string  `json:"months"`     // mois où la dépense apparaît
	LastDate   string    `json:"last_date"`
	DomainID *int64    `json:"domain_id"`
	Regular    bool      `json:"regular"`    // montant stable (< 15 % d'écart) → abonnement probable
	MerchantKind string  `json:"merchant_kind,omitempty"` // savings | tax | benefit | health
	Amounts    []float64 `json:"-"`          // montants ordonnés par date (usage interne : insights)
}

var reRecNoise = regexp.MustCompile(`\d{2}[./]\d{2}(?:[./]\d{2,4})?|\d{4,}|\b\d+[hH]\d{2}\b`)

// recStopwords : préfixes et mots outils bancaires qui ne portent pas
// l'identité du marchand.
var recStopwords = map[string]bool{
	"PRLV": true, "PRELEVEMENT": true, "CB": true, "CARTE": true,
	"VIREMENT": true, "VIR": true, "INST": true, "WEB": true,
	"RET": true, "DAB": true, "PAIEMENT": true, "ACHAT": true,
	"VERS": true, "DE": true, "DU": true, "M.": true, "MR": true, "MME": true,
}

var reCardRef = regexp.MustCompile(`^X\d+$`)

// normalizeRecLabel réduit un libellé à l'identité du marchand : sans
// préfixes bancaires, dates, numéros de carte ni références, puis les deux
// premiers mots significatifs. Ainsi « PRLV FREE TELECOM ABONNEMENT » et
// « Prlv Free Telecom Free Hautdebit 1478… » donnent la même clé.
func normalizeRecLabel(label string) string {
	l := strings.ToUpper(label)
	l = reRecNoise.ReplaceAllString(l, "")
	kept := []string{}
	for _, f := range strings.Fields(l) {
		if recStopwords[f] || reCardRef.MatchString(f) || len(f) < 2 {
			continue
		}
		kept = append(kept, f)
		if len(kept) == 2 {
			break
		}
	}
	return strings.Join(kept, " ")
}

// listRecurring détecte les dépenses récurrentes : même libellé normalisé
// présent sur au moins minMonths mois distincts. Celles au montant stable
// (écart max < 15 %) sont marquées comme abonnements probables.
func (s *Server) listRecurring(w http.ResponseWriter, r *http.Request) {
	minMonths := 2
	if v, err := strconv.Atoi(r.URL.Query().Get("min_months")); err == nil && v > 1 {
		minMonths = v
	}
	out, err := computeRecurring(s.DB, s.ownerID(r), minMonths)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// computeRecurring regroupe les dépenses des 12 derniers mois par marchand,
// pour un utilisateur donné.
func computeRecurring(db *sql.DB, ownerID int64, minMonths int) ([]recurring, error) {
	rows, err := db.Query(`
		SELECT t.tx_date::text, t.label, t.amount, t.domain_id, t.merchant, t.display_label, t.label_kind
		FROM transactions t
		WHERE t.amount < 0 AND t.owner_id = $1 AND (t.domain_id IS NULL OR t.domain_id NOT IN (SELECT id FROM domains WHERE is_transfer)) 
		  AND t.tx_date >= now() - INTERVAL '12 months'
		ORDER BY t.tx_date`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type acc struct {
		label      string
		kind       string
		amounts    []float64
		months     map[string]bool
		lastDate   string
		lastAmount float64
		domainID *int64
	}
	groups := map[string]*acc{}
	for rows.Next() {
		var date, label, merchant, display, lkind string
		var amount float64
		var catID *int64
		if err := rows.Scan(&date, &label, &amount, &catID, &merchant, &display, &lkind); err != nil {
			return nil, err
		}
		// Le marchand normalisé regroupe mieux que le libellé brut : la même
		// enseigne avec des références de carte ou dates différentes devient
		// UN seul flux récurrent.
		key := merchant
		if key == "" {
			key = normalizeRecLabel(label)
		}
		if key == "" {
			continue
		}
		g, ok := groups[key]
		if !ok {
			lbl := display
			if lbl == "" {
				lbl = label
			}
			g = &acc{label: lbl, kind: lkind, months: map[string]bool{}}
			groups[key] = g
		}
		g.amounts = append(g.amounts, -amount)
		g.months[date[:7]] = true
		g.lastDate = date
		g.lastAmount = -amount
		if catID != nil {
			g.domainID = catID
		}
	}

	out := []recurring{}
	for _, g := range groups {
		if len(g.months) < minMonths {
			continue
		}
		sum, min, max := 0.0, math.MaxFloat64, 0.0
		for _, a := range g.amounts {
			sum += a
			if a < min {
				min = a
			}
			if a > max {
				max = a
			}
		}
		months := make([]string, 0, len(g.months))
		for m := range g.months {
			months = append(months, m)
		}
		sortStrings(months)
		out = append(out, recurring{
			Label:      g.label,
			MerchantKind: g.kind,
			Amount:     g.lastAmount,
			AvgAmount:  math.Round(sum/float64(len(g.amounts))*100) / 100,
			Months:     months,
			LastDate:   g.lastDate,
			DomainID: g.domainID,
			Regular:    max > 0 && (max-min)/max < 0.15,
			Amounts:    g.amounts,
		})
	}
	// abonnements d'abord, puis par montant décroissant
	sortRecurring(out)
	return out, nil
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

func sortRecurring(rs []recurring) {
	for i := 1; i < len(rs); i++ {
		for j := i; j > 0; j-- {
			a, b := rs[j], rs[j-1]
			if (a.Regular && !b.Regular) || (a.Regular == b.Regular && a.AvgAmount > b.AvgAmount) {
				rs[j], rs[j-1] = rs[j-1], rs[j]
			} else {
				break
			}
		}
	}
}

// ---------- Export CSV ----------

// exportTransactions produit un CSV (mêmes filtres que /api/transactions).
func (s *Server) exportTransactions(w http.ResponseWriter, r *http.Request) {
	q := `SELECT t.tx_date::text, t.label, t.amount, COALESCE(d.name, '')
	      FROM transactions t LEFT JOIN domains d ON d.id = t.domain_id WHERE t.owner_id = $1`
	args := []any{s.ownerID(r)}
	if m := r.URL.Query().Get("month"); reMonth.MatchString(m) {
		args = append(args, m)
		q += fmt.Sprintf(" AND to_char(t.tx_date, 'YYYY-MM') = $%d", len(args))
	}
	q += " ORDER BY t.tx_date, t.id"

	rows, err := s.DB.Query(q, args...)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	name := "transactions"
	if m := r.URL.Query().Get("month"); m != "" {
		name += "-" + m
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`.csv"`)

	cw := csv.NewWriter(w)
	cw.Comma = ';' // convention FR : Excel/LibreOffice ouvrent directement
	_ = cw.Write([]string{"date", "libelle", "montant", "categorie"})
	for rows.Next() {
		var date, label, cat string
		var amount float64
		if err := rows.Scan(&date, &label, &amount, &cat); err != nil {
			return
		}
		_ = cw.Write([]string{date, csvSafe(label), strings.Replace(fmt.Sprintf("%.2f", amount), ".", ",", 1), csvSafe(cat)})
	}
	cw.Flush()
}

// csvSafe neutralise l'injection de formule CSV : un tableur (Excel,
// LibreOffice) interprète une cellule commençant par = + - @ comme une formule.
// On préfixe ces cellules d'une apostrophe pour les forcer en texte.
func csvSafe(s string) string {
	if s == "" {
		return s
	}
	switch s[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + s
	}
	return s
}
