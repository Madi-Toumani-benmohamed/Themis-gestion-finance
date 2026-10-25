package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
	"themis/internal/models"
	"themis/internal/money"
)

type transactionFilter struct {
	where         string
	args          []any
	limit, offset int
	from, to      string
	budgetID      int64
}

// All views use the same predicates. Correlated tag predicates avoid multiplying
// transactions with several tags before counting or aggregating money.
func parseTransactionFilter(v url.Values, owner int64, report bool, now time.Time) (transactionFilter, error) {
	f := transactionFilter{where: "t.owner_id=$1", args: []any{owner}, limit: 1000}
	if report {
		f.limit = 100
	}
	add := func(value any) string { f.args = append(f.args, value); return fmt.Sprintf("$%d", len(f.args)) }
	bad := func() (transactionFilter, error) { return f, fmt.Errorf("filtres de transactions invalides") }
	for key, target := range map[string]*int{"limit": &f.limit, "offset": &f.offset} {
		if raw := v.Get(key); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil || n < 0 || (key == "limit" && (n < 1 || n > 1000)) || (key == "offset" && n > 10000000) {
				return bad()
			}
			*target = n
		}
	}
	if raw := v.Get("ids"); raw != "" {
		ids, err := parseTransactionSourceIDs(raw)
		if err != nil {
			return f, err
		}
		f.where += " AND t.id=ANY(" + add(pq.Array(ids)) + ")"
	}
	if report && ((v.Get("from") == "") != (v.Get("to") == "")) {
		return bad()
	}
	if v.Get("all") != "" && v.Get("all") != "1" {
		return bad()
	}
	month := v.Get("month")
	if report && month == "" && v.Get("from") == "" && v.Get("to") == "" && v.Get("ids") == "" && v.Get("all") != "1" {
		month = now.Format("2006-01")
	}
	if month != "" {
		first, err := time.Parse("2006-01", month)
		if err != nil || first.Year() < 1 || first.Format("2006-01") != month {
			return bad()
		}
		f.from = first.Format("2006-01-02")
		f.to = first.AddDate(0, 1, -1).Format("2006-01-02")
		if report && month == now.Format("2006-01") {
			f.to = now.Format("2006-01-02")
		}
	}
	for _, key := range []string{"from", "to"} {
		if raw := v.Get(key); raw != "" {
			day, err := time.Parse("2006-01-02", raw)
			if err != nil || day.Year() < 1 || day.Format("2006-01-02") != raw {
				return bad()
			}
			if key == "from" && (f.from == "" || raw > f.from) {
				f.from = raw
			}
			if key == "to" && (f.to == "" || raw < f.to) {
				f.to = raw
			}
		}
	}
	if f.from != "" && f.to != "" && f.from > f.to {
		return bad()
	}
	if raw := v.Get("budget_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 || month == "" {
			return bad()
		}
		_, last, _, err := reliabilityWindow(month, now)
		if err != nil {
			return f, err
		}
		// Budget source drilldowns must keep the budget service's exact window,
		// including the current-month bound, also for legacy array consumers.
		if lastDay := last.Format("2006-01-02"); f.to > lastDay {
			f.to = lastDay
		}
		f.budgetID = id
		f.where += " AND " + budgetSourcePredicate(add(id))
	}
	if f.from != "" && f.to != "" && f.from > f.to {
		return bad()
	}
	if f.from != "" {
		f.where += " AND t.tx_date >= " + add(f.from) + "::date"
	}
	if f.to != "" {
		f.where += " AND t.tx_date <= " + add(f.to) + "::date"
	}
	if raw := v.Get("account"); raw != "" {
		if raw == "none" {
			f.where += ` AND NOT EXISTS(SELECT 1 FROM statements st JOIN accounts a ON a.id=st.account_id AND a.owner_id=st.owner_id WHERE st.id=t.statement_id AND st.owner_id=t.owner_id)`
		} else {
			id, err := strconv.ParseInt(raw, 10, 64)
			if err != nil || id <= 0 {
				return bad()
			}
			f.where += ` AND EXISTS(SELECT 1 FROM statements st JOIN accounts a ON a.id=st.account_id AND a.owner_id=st.owner_id WHERE st.id=t.statement_id AND st.owner_id=t.owner_id AND a.id=` + add(id) + `)`
		}
	}
	if raw := v.Get("domain"); raw != "" {
		if raw == "none" {
			f.where += ` AND (t.domain_id IS NULL OR NOT EXISTS(SELECT 1 FROM domains d WHERE d.id=t.domain_id AND d.owner_id=t.owner_id))`
		} else {
			id, err := strconv.ParseInt(raw, 10, 64)
			if err != nil || id <= 0 {
				return bad()
			}
			p := add(id)
			f.where += ` AND EXISTS(SELECT 1 FROM domains d WHERE d.owner_id=t.owner_id AND d.id=t.domain_id AND (d.id=` + p + ` OR d.parent_id=` + p + `))`
		}
	}
	if raw := v.Get("merchant"); strings.TrimSpace(raw) != "" {
		if len(raw) > 1000 {
			return bad()
		}
		f.where += " AND COALESCE(NULLIF(t.merchant,''),NULLIF(t.display_label,''),t.label)=" + add(raw)
	}
	if raw := strings.TrimSpace(v.Get("tag")); raw != "" {
		if raw == "__none__" {
			f.where += ` AND NOT EXISTS(SELECT 1 FROM transaction_tags tt JOIN tags tg ON tg.id=tt.tag_id AND tg.owner_id=t.owner_id WHERE tt.transaction_id=t.id)`
		} else {
			f.where += ` AND EXISTS(SELECT 1 FROM transaction_tags tt JOIN tags tg ON tg.id=tt.tag_id AND tg.owner_id=t.owner_id WHERE tt.transaction_id=t.id AND tg.name=` + add(strings.ToLower(raw)) + `)`
		}
	}
	switch v.Get("type") {
	case "":
	case "debit":
		f.where += " AND t.amount<0 AND " + financialExpenseSQLT
	case "credit":
		f.where += " AND t.amount>0 AND " + financialExpenseSQLT
	case "neutral":
		f.where += " AND t.amount=0 AND " + financialExpenseSQLT
	case "transfer":
		f.where += " AND NOT " + financialExpenseSQLT
	default:
		return bad()
	}
	if search := strings.TrimSpace(v.Get("q")); search != "" {
		// ILIKE wildcards are escaped: a user query is text, not a SQL pattern.
		escaped := strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(search)
		p := add("%" + escaped + "%")
		predicate := `t.label ILIKE ` + p + ` OR t.display_label ILIKE ` + p + ` OR t.merchant ILIKE ` + p + ` OR EXISTS(SELECT 1 FROM transaction_tags tt JOIN tags tg ON tg.id=tt.tag_id AND tg.owner_id=t.owner_id WHERE tt.transaction_id=t.id AND tg.name ILIKE ` + p + `)`
		if amount, err := money.Parse(strings.ReplaceAll(strings.TrimSpace(strings.TrimSuffix(search, "€")), ",", ".")); err == nil {
			predicate += " OR t.amount=" + add(amount.String()) + "::numeric"
		}
		f.where += " AND (" + predicate + ")"
	}
	return f, nil
}

type transactionReportRow struct {
	ID                   int64                  `json:"id"`
	StatementID          *int64                 `json:"statement_id,omitempty"`
	Date                 string                 `json:"date"`
	Label                string                 `json:"label"`
	Display              string                 `json:"display_label"`
	Merchant             string                 `json:"merchant"`
	Kind                 string                 `json:"kind,omitempty"`
	Amount               json.Number            `json:"amount"`
	AmountCents          money.Cents            `json:"amount_cents"`
	DomainID             *int64                 `json:"domain_id"`
	Tags                 []string               `json:"tags"`
	AccountID            *int64                 `json:"account_id"`
	AccountName          string                 `json:"account_name"`
	StatementFilename    string                 `json:"statement_filename"`
	StatementValidated   *bool                  `json:"statement_validated"`
	StatementPeriodStart *string                `json:"statement_period_start"`
	StatementPeriodEnd   *string                `json:"statement_period_end"`
	FinancialType        string                 `json:"financial_type"`
	ClassificationSource string                 `json:"classification_source"`
	Suggestion           *transactionSuggestion `json:"suggestion"`
}
type transactionSuggestion struct {
	DomainID int64    `json:"domain_id"`
	Tags     []string `json:"tags"`
	Source   string   `json:"source"`
}
type transactionReportSummary struct {
	Expenses           json.Number `json:"expenses"`
	Income             json.Number `json:"income"`
	Net                json.Number `json:"net"`
	TransferDebits     json.Number `json:"transfer_debits"`
	TransferCredits    json.Number `json:"transfer_credits"`
	UncategorizedCount int64       `json:"uncategorized_count"`
}
type transactionReport struct {
	Transactions []transactionReportRow   `json:"transactions"`
	Total        int64                    `json:"total"`
	Limit        int                      `json:"limit"`
	Offset       int                      `json:"offset"`
	Summary      transactionReportSummary `json:"summary"`
	QueryFrom    string                   `json:"query_from"`
	QueryTo      string                   `json:"query_to"`
}

func (s *Server) listTransactions(w http.ResponseWriter, r *http.Request) {
	view := r.URL.Query().Get("view")
	if view != "" && view != "report" {
		writeErr(w, 400, fmt.Errorf("vue invalide"))
		return
	}
	report := view == "report"
	f, err := parseTransactionFilter(r.URL.Query(), s.ownerID(r), report, time.Now())
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
	if report {
		s.writeTransactionReport(w, r, f)
		return
	}
	// Legacy array contract remains unchanged, including its default page size.
	args := append(append([]any{}, f.args...), f.limit, f.offset)
	rows, err := s.DB.QueryContext(r.Context(), `SELECT t.id,t.statement_id,t.tx_date::text,t.label,t.display_label,t.merchant,t.label_kind,t.amount,t.domain_id,
 COALESCE((SELECT array_agg(tg.name ORDER BY tg.name) FROM transaction_tags tt JOIN tags tg ON tg.id=tt.tag_id AND tg.owner_id=t.owner_id WHERE tt.transaction_id=t.id),'{}')
 FROM transactions t WHERE `+f.where+fmt.Sprintf(` ORDER BY t.tx_date DESC,t.id DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		writeErr(w, 500, fmt.Errorf("transactions indisponibles"))
		return
	}
	defer rows.Close()
	out := []models.Transaction{}
	for rows.Next() {
		var t models.Transaction
		if err = rows.Scan(&t.ID, &t.StatementID, &t.Date, &t.Label, &t.Display, &t.Merchant, &t.Kind, &t.Amount, &t.DomainID, pq.Array(&t.Tags)); err != nil {
			writeErr(w, 500, fmt.Errorf("transactions indisponibles"))
			return
		}
		out = append(out, t)
	}
	if rows.Err() != nil {
		writeErr(w, 500, fmt.Errorf("lecture des transactions interrompue"))
		return
	}
	writeJSON(w, 200, out)
}

func (s *Server) transactionBudgetExists(ctx context.Context, owner, budgetID int64) (bool, error) {
	var exists bool
	err := s.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM budgets b JOIN domains d ON d.id=b.domain_id AND d.owner_id=b.owner_id WHERE b.id=$1 AND b.owner_id=$2)`, budgetID, owner).Scan(&exists)
	return exists, err
}

func (s *Server) writeTransactionReport(w http.ResponseWriter, r *http.Request, f transactionFilter) {
	tx, err := s.DB.BeginTx(r.Context(), &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		writeErr(w, 500, fmt.Errorf("transactions indisponibles"))
		return
	}
	defer tx.Rollback()
	out := transactionReport{Transactions: []transactionReportRow{}, Limit: f.limit, Offset: f.offset, QueryFrom: f.from, QueryTo: f.to}
	var expense, income, transferDebit, transferCredit string
	err = tx.QueryRowContext(r.Context(), `SELECT count(*),
 COALESCE(-SUM(t.amount) FILTER(WHERE t.amount<0 AND `+financialExpenseSQLT+`),0)::text,
 COALESCE(SUM(t.amount) FILTER(WHERE t.amount>0 AND `+financialExpenseSQLT+`),0)::text,
 COALESCE(-SUM(t.amount) FILTER(WHERE t.amount<0 AND NOT `+financialExpenseSQLT+`),0)::text,
 COALESCE(SUM(t.amount) FILTER(WHERE t.amount>0 AND NOT `+financialExpenseSQLT+`),0)::text,
 count(*) FILTER(WHERE t.domain_id IS NULL OR NOT EXISTS(SELECT 1 FROM domains d WHERE d.id=t.domain_id AND d.owner_id=t.owner_id))
 FROM transactions t WHERE `+f.where, f.args...).Scan(&out.Total, &expense, &income, &transferDebit, &transferCredit, &out.Summary.UncategorizedCount)
	if err != nil {
		writeErr(w, 500, fmt.Errorf("synthèse des transactions indisponible"))
		return
	}
	// PostgreSQL NUMERIC sums remain exact; parse the aggregates without imposing
	// an individual NUMERIC(12,2) amount limit on a sum of many transactions.
	values := []*json.Number{&out.Summary.Expenses, &out.Summary.Income, &out.Summary.TransferDebits, &out.Summary.TransferCredits}
	var cents [4]money.Cents
	for i, raw := range []string{expense, income, transferDebit, transferCredit} {
		cents[i], err = parseTransactionAggregate(raw)
		if err != nil {
			writeErr(w, 500, fmt.Errorf("montants agrégés hors limites"))
			return
		}
		*values[i] = json.Number(cents[i].String())
	}
	out.Summary.Net = json.Number((cents[1] - cents[0]).String())
	args := append(append([]any{}, f.args...), f.limit, f.offset)
	rows, err := tx.QueryContext(r.Context(), `SELECT t.id,t.statement_id,t.tx_date::text,t.label,t.display_label,t.merchant,t.label_kind,t.amount::text,
 CASE WHEN EXISTS(SELECT 1 FROM domains valid_domain WHERE valid_domain.id=t.domain_id AND valid_domain.owner_id=t.owner_id) THEN t.domain_id ELSE NULL END,
 COALESCE((SELECT array_agg(tg.name ORDER BY tg.name) FROM transaction_tags tt JOIN tags tg ON tg.id=tt.tag_id AND tg.owner_id=t.owner_id WHERE tt.transaction_id=t.id),'{}'),
 a.id,COALESCE(a.name,''),COALESCE(st.filename,''),st.validated,st.period_start::text,st.period_end::text,
 CASE WHEN NOT `+financialExpenseSQLT+` THEN 'transfer' WHEN t.amount<0 THEN 'debit' WHEN t.amount>0 THEN 'credit' ELSE 'neutral' END,t.classification_source,
 sug.domain_id,COALESCE(sug.tags,'{}'),COALESCE(sug.source,'')
 FROM transactions t LEFT JOIN statements st ON st.id=t.statement_id AND st.owner_id=t.owner_id
 LEFT JOIN accounts a ON a.id=st.account_id AND a.owner_id=t.owner_id
 LEFT JOIN transaction_suggestions sug ON sug.transaction_id=t.id AND sug.owner_id=t.owner_id AND EXISTS(SELECT 1 FROM domains suggested_domain WHERE suggested_domain.id=sug.domain_id AND suggested_domain.owner_id=t.owner_id)
 WHERE `+f.where+fmt.Sprintf(` ORDER BY t.tx_date DESC,t.id DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		writeErr(w, 500, fmt.Errorf("transactions indisponibles"))
		return
	}
	for rows.Next() {
		var t transactionReportRow
		var amount string
		var suggestionDomain *int64
		var suggestionTags []string
		var suggestionSource string
		if err = rows.Scan(&t.ID, &t.StatementID, &t.Date, &t.Label, &t.Display, &t.Merchant, &t.Kind, &amount, &t.DomainID, pq.Array(&t.Tags), &t.AccountID, &t.AccountName, &t.StatementFilename, &t.StatementValidated, &t.StatementPeriodStart, &t.StatementPeriodEnd, &t.FinancialType, &t.ClassificationSource, &suggestionDomain, pq.Array(&suggestionTags), &suggestionSource); err != nil {
			rows.Close()
			writeErr(w, 500, fmt.Errorf("transactions indisponibles"))
			return
		}
		t.AmountCents, err = money.Parse(amount)
		if err != nil {
			rows.Close()
			writeErr(w, 500, fmt.Errorf("montant invalide"))
			return
		}
		t.Amount = json.Number(t.AmountCents.String())
		if suggestionDomain != nil {
			t.Suggestion = &transactionSuggestion{DomainID: *suggestionDomain, Tags: suggestionTags, Source: suggestionSource}
		}
		out.Transactions = append(out.Transactions, t)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		writeErr(w, 500, fmt.Errorf("lecture interrompue"))
		return
	}
	if err = tx.Commit(); err != nil {
		writeErr(w, 500, fmt.Errorf("lecture interrompue"))
		return
	}
	writeJSON(w, 200, out)
}

func parseTransactionAggregate(raw string) (money.Cents, error) {
	negative := strings.HasPrefix(raw, "-")
	raw = strings.TrimPrefix(raw, "-")
	parts := strings.Split(raw, ".")
	if len(parts) > 2 || len(parts[0]) == 0 {
		return 0, fmt.Errorf("montant invalide")
	}
	fraction := "00"
	if len(parts) == 2 {
		if len(parts[1]) > 2 || len(parts[1]) < 1 {
			return 0, fmt.Errorf("précision invalide")
		}
		fraction = (parts[1] + "0")[:2]
	}
	digits := parts[0] + fraction
	for _, r := range digits {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("montant invalide")
		}
	}
	n, err := strconv.ParseInt(digits, 10, 64)
	if err != nil {
		return 0, err
	}
	if negative {
		n = -n
	}
	return money.Cents(n), nil
}

// Category and tags form one atomic edit. No raw label, date or amount can be
// written through this endpoint, including through unrecognized JSON fields.
func (s *Server) updateTransaction(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeErr(w, 400, fmt.Errorf("identifiant invalide"))
		return
	}
	var raw map[string]json.RawMessage
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384))
	if err = decoder.Decode(&raw); err != nil || len(raw) == 0 {
		writeErr(w, 400, fmt.Errorf("modification invalide"))
		return
	}
	if decoder.Decode(new(any)) != io.EOF {
		writeErr(w, 400, fmt.Errorf("modification invalide"))
		return
	}
	var domain *int64
	setDomain := false
	var tags []string
	setTags := false
	var addTags, removeTags []string
	discardSuggestion := false
	for key, value := range raw {
		switch key {
		case "add_tags", "remove_tags":
			var names []string
			if string(value) == "null" || json.Unmarshal(value, &names) != nil || len(names) == 0 {
				writeErr(w, 400, fmt.Errorf("tags invalides"))
				return
			}
			names, err = validateTransactionTags(names)
			if err != nil {
				writeErr(w, 400, err)
				return
			}
			if key == "add_tags" {
				addTags = names
			} else {
				removeTags = names
			}
		case "discard_suggestion":
			if err = json.Unmarshal(value, &discardSuggestion); err != nil || !discardSuggestion {
				writeErr(w, 400, fmt.Errorf("décision de suggestion invalide"))
				return
			}
		case "domain_id":
			setDomain = true
			if err = json.Unmarshal(value, &domain); err != nil || (domain != nil && *domain <= 0) {
				writeErr(w, 400, fmt.Errorf("catégorie invalide"))
				return
			}
		case "tags":
			setTags = true
			if string(value) == "null" || json.Unmarshal(value, &tags) != nil {
				writeErr(w, 400, fmt.Errorf("tags invalides"))
				return
			}
			tags, err = validateTransactionTags(tags)
			if err != nil {
				writeErr(w, 400, err)
				return
			}
		default:
			writeErr(w, 400, fmt.Errorf("champ non modifiable"))
			return
		}
	}
	if setTags && (len(addTags) > 0 || len(removeTags) > 0) {
		writeErr(w, 400, fmt.Errorf("choisir remplacement ou ajout/retrait de tags"))
		return
	}
	owner := s.ownerID(r)
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeErr(w, 500, fmt.Errorf("enregistrement impossible"))
		return
	}
	defer tx.Rollback()
	var found int64
	err = tx.QueryRowContext(r.Context(), `SELECT id FROM transactions WHERE id=$1 AND owner_id=$2 FOR UPDATE`, id, owner).Scan(&found)
	if err == sql.ErrNoRows {
		writeErr(w, 404, fmt.Errorf("transaction introuvable"))
		return
	}
	if err != nil {
		writeErr(w, 500, fmt.Errorf("enregistrement impossible"))
		return
	}
	if setDomain {
		if domain != nil {
			var exists bool
			err = tx.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM domains WHERE id=$1 AND owner_id=$2)`, *domain, owner).Scan(&exists)
			if err != nil {
				writeErr(w, 500, fmt.Errorf("catégorie indisponible"))
				return
			}
			if !exists {
				writeErr(w, 400, fmt.Errorf("catégorie indisponible"))
				return
			}
		}
		if _, err = tx.ExecContext(r.Context(), `UPDATE transactions SET domain_id=$1,classification_source='manual' WHERE id=$2 AND owner_id=$3`, domain, id, owner); err != nil {
			writeErr(w, 500, fmt.Errorf("enregistrement impossible"))
			return
		}
	}
	if setDomain || discardSuggestion {
		if _, err = tx.ExecContext(r.Context(), `DELETE FROM transaction_suggestions WHERE transaction_id=$1 AND owner_id=$2`, id, owner); err != nil {
			writeErr(w, 500, fmt.Errorf("enregistrement impossible"))
			return
		}
	}
	if len(addTags) > 0 || len(removeTags) > 0 {
		// Apply tag additions/removals to current data under the row lock, rather
		// than overwriting tags copied from an earlier frontend page snapshot.
		if err = tx.QueryRowContext(r.Context(), `SELECT COALESCE(array_agg(tg.name ORDER BY tg.name),'{}') FROM transaction_tags tt JOIN tags tg ON tg.id=tt.tag_id AND tg.owner_id=$2 WHERE tt.transaction_id=$1`, id, owner).Scan(pq.Array(&tags)); err != nil {
			writeErr(w, 500, fmt.Errorf("tags indisponibles"))
			return
		}
		removed := map[string]bool{}
		for _, name := range removeTags {
			removed[name] = true
		}
		merged := []string{}
		seen := map[string]bool{}
		for _, name := range append(tags, addTags...) {
			if !removed[name] && !seen[name] {
				merged = append(merged, name)
				seen[name] = true
			}
		}
		tags, err = validateTransactionTags(merged)
		if err != nil {
			writeErr(w, 400, err)
			return
		}
		setTags = true
	}
	if setTags {
		if err = replaceTransactionTags(r.Context(), tx, id, owner, tags); err != nil {
			writeErr(w, 500, fmt.Errorf("enregistrement des tags impossible"))
			return
		}
	}
	if err = tx.Commit(); err != nil {
		writeErr(w, 500, fmt.Errorf("enregistrement impossible"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func validateTransactionTags(names []string) ([]string, error) {
	if len(names) > 50 {
		return nil, fmt.Errorf("50 tags maximum")
	}
	out := []string{}
	seen := map[string]bool{}
	for _, raw := range names {
		name := strings.ToLower(strings.TrimSpace(raw))
		if name == "" || len(name) > 40 {
			return nil, fmt.Errorf("chaque tag doit contenir entre 1 et 40 caractères")
		}
		if !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	return out, nil
}
func replaceTransactionTags(ctx context.Context, tx *sql.Tx, id, owner int64, names []string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM transaction_tags WHERE transaction_id=$1`, id); err != nil {
		return err
	}
	for _, name := range names {
		var tag int64
		if err := tx.QueryRowContext(ctx, `INSERT INTO tags(name,owner_id) VALUES($1,$2) ON CONFLICT(owner_id,name) DO UPDATE SET name=EXCLUDED.name RETURNING id`, name, owner).Scan(&tag); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO transaction_tags(transaction_id,tag_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, id, tag); err != nil {
			return err
		}
	}
	return nil
}
