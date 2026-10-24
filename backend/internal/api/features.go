package api

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"themis/internal/money"
	"time"
)

// ---------- Dépenses récurrentes / abonnements ----------

type recurring struct {
	Label             string    `json:"label"`      // libellé normalisé représentatif
	Amount            float64   `json:"amount"`     // dernier montant (positif)
	AvgAmount         float64   `json:"avg_amount"` // moyenne (positif)
	Months            []string  `json:"months"`     // mois où la dépense apparaît
	LastDate          string    `json:"last_date"`
	DomainID          *int64    `json:"domain_id"`
	Regular           bool      `json:"regular"`                 // cadence mensuelle documentée, nature abonnement/charge fixe
	MerchantKind      string    `json:"merchant_kind,omitempty"` // savings | tax | benefit | health
	Key               string    `json:"key,omitempty"`
	Nature            string    `json:"nature,omitempty"`
	Confidence        string    `json:"confidence,omitempty"`
	Frequency         *string   `json:"frequency"`
	EstimatedNextDate *string   `json:"estimated_next_date"`
	Amounts           []float64 `json:"-"` // montants ordonnés par date (usage interne : insights)
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

// listRecurring preserves the historical array contract. Regular now requires
// documented monthly cadence and an eligible economic nature; avg_amount is
// explicitly a per-transaction mean, never a monthly total.
func (s *Server) listRecurring(w http.ResponseWriter, r *http.Request) {
	minMonths := 2
	if v, err := strconv.Atoi(r.URL.Query().Get("min_months")); err == nil && v > 1 {
		minMonths = v
	}
	month := r.URL.Query().Get("month")
	if _, _, _, err := reliabilityWindow(month, time.Now()); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	out, err := computeRecurringForMonth(s.DB, s.ownerID(r), minMonths, month)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// Compatibility adapter: amounts retain the legacy JSON shape, but detection
// and economic nature now come from the same deterministic source as Budgets.
func computeRecurring(db *sql.DB, ownerID int64, minMonths int) ([]recurring, error) {
	return computeRecurringForMonth(db, ownerID, minMonths, "")
}
func computeRecurringForMonth(db *sql.DB, ownerID int64, minMonths int, month string) ([]recurring, error) {
	server := Server{DB: db}
	report, err := server.loadRecurringReview(context.Background(), ownerID, month, time.Now())
	if err != nil {
		return nil, err
	}
	return legacyRecurringItems(report.Items, minMonths)
}
func legacyRecurringItems(items []recurringReviewItem, minMonths int) ([]recurring, error) {
	out := []recurring{}
	for _, item := range items {
		if len(item.Months) < minMonths || item.Status == "ignored" || item.Nature == "transfer_savings" || (item.protected && item.Nature == "variable_recurring") || item.Direction == "credit" {
			continue
		}
		last, err := money.Parse(string(item.LastAmount))
		if err != nil {
			return nil, err
		}
		average, err := money.Parse(string(item.AveragePerTransaction))
		if err != nil {
			return nil, err
		}
		amounts := make([]float64, len(item.SourceAmounts))
		for i, amount := range item.SourceAmounts {
			amounts[i] = float64(amount) / 100
		}
		regular := item.Confidence == "probable" && item.Frequency != nil && *item.Frequency == "monthly" && (item.Nature == "subscription" || item.Nature == "fixed_charge")
		out = append(out, recurring{Key: item.Key, Label: item.Label, Amount: float64(last) / 100, AvgAmount: float64(average) / 100, Months: item.Months, LastDate: item.LastDate, DomainID: item.DomainID, Regular: regular, MerchantKind: item.MerchantKind, Amounts: amounts, Nature: item.Nature, Confidence: item.Confidence, Frequency: item.Frequency, EstimatedNextDate: item.EstimatedNextDate})
	}
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
	view := r.URL.Query().Get("view")
	if view != "" && view != "report" {
		writeErr(w, 400, fmt.Errorf("vue invalide"))
		return
	}
	f, err := parseTransactionFilter(r.URL.Query(), s.ownerID(r), view == "report", time.Now())
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	if f.budgetID != 0 {
		exists, err := s.transactionBudgetExists(r.Context(), s.ownerID(r), f.budgetID)
		if err != nil {
			writeErr(w, 500, fmt.Errorf("budget indisponible"))
			return
		}
		if !exists {
			writeErr(w, 404, fmt.Errorf("budget indisponible"))
			return
		}
	}
	// A single SQL snapshot, identical filters and no page limit: exporting page
	// two still exports the whole explicit scope, never a silent first 1000 rows.
	rows, err := s.DB.QueryContext(r.Context(), `SELECT t.tx_date::text,t.label,t.amount::text,COALESCE(d.name,'')
 FROM transactions t LEFT JOIN domains d ON d.id=t.domain_id AND d.owner_id=t.owner_id
 WHERE `+f.where+` ORDER BY t.tx_date,t.id`, f.args...)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Errorf("export indisponible"))
		return
	}
	defer rows.Close()
	// Stage with mode0600 and remove on every exit. A database/scan/disk error
	// must return an error, rather than a successful but incomplete financial CSV.
	staged, err := os.CreateTemp("", "themis-export-*.csv")
	if err != nil {
		writeErr(w, 500, fmt.Errorf("export indisponible"))
		return
	}
	defer os.Remove(staged.Name())
	defer staged.Close()
	cw := csv.NewWriter(staged)
	cw.Comma = ';' // convention FR : Excel/LibreOffice ouvrent directement
	if err = cw.Write([]string{"date", "libelle", "montant", "categorie"}); err != nil {
		writeErr(w, 500, fmt.Errorf("export interrompu"))
		return
	}
	for rows.Next() {
		var date, label, amount, cat string
		if err := rows.Scan(&date, &label, &amount, &cat); err != nil {
			writeErr(w, 500, fmt.Errorf("export interrompu"))
			return
		}
		exact, err := money.Parse(amount)
		if err != nil {
			writeErr(w, 500, fmt.Errorf("montant exporté invalide"))
			return
		}
		if err = cw.Write([]string{date, csvSafe(label), strings.Replace(exact.String(), ".", ",", 1), csvSafe(cat)}); err != nil {
			writeErr(w, 500, fmt.Errorf("export interrompu"))
			return
		}
	}
	if rows.Err() != nil {
		writeErr(w, 500, fmt.Errorf("export interrompu"))
		return
	}
	rows.Close()
	cw.Flush()
	if cw.Error() != nil {
		writeErr(w, 500, fmt.Errorf("export interrompu"))
		return
	}
	info, err := staged.Stat()
	if err != nil {
		writeErr(w, 500, fmt.Errorf("export interrompu"))
		return
	}
	if _, err = staged.Seek(0, io.SeekStart); err != nil {
		writeErr(w, 500, fmt.Errorf("export interrompu"))
		return
	}
	name := "transactions"
	if month := r.URL.Query().Get("month"); month != "" {
		name += "-" + month
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`.csv"`)
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
	_, _ = io.Copy(w, staged) // A transport failure cannot be retried after headers.
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
