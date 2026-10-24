package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"themis/internal/money"
)

// A transaction belongs to at most one envelope: its explicit category first,
// otherwise its parent. Domain creation limits the taxonomy to two levels.
// This exact predicate is also used by the source-transaction endpoint.
const budgetAssignmentSQL = `(SELECT nearest.id FROM budgets nearest
 JOIN domains target ON target.id=nearest.domain_id AND target.owner_id=nearest.owner_id
 WHERE nearest.owner_id=t.owner_id AND (nearest.domain_id=t.domain_id OR nearest.domain_id=(
 SELECT parent_id FROM domains own_domain WHERE own_domain.id=t.domain_id AND own_domain.owner_id=t.owner_id))
 ORDER BY CASE WHEN nearest.domain_id=t.domain_id THEN 0 ELSE 1 END,nearest.id LIMIT 1)`

// Savings are identified by the existing normalizer, internal transfers by
// category flags. Neither is consumption of an expense envelope.
const budgetExpenseSQL = financialExpenseSQLT

func budgetSourcePredicate(parameter string) string {
	return budgetAssignmentSQL + `=` + parameter + ` AND (` + budgetExpenseSQL + ` OR EXISTS (
 SELECT 1 FROM budgets source_budget JOIN domains own_domain ON own_domain.id=source_budget.domain_id AND own_domain.owner_id=source_budget.owner_id
 LEFT JOIN domains parent_domain ON parent_domain.id=own_domain.parent_id AND parent_domain.owner_id=own_domain.owner_id
 WHERE source_budget.id=` + parameter + ` AND source_budget.owner_id=t.owner_id
 AND (own_domain.is_transfer OR COALESCE(parent_domain.is_transfer,false))))`
}

type budgetStatus struct {
	ID                   int64        `json:"id"`
	DomainID             int64        `json:"domain_id"`
	DomainName           string       `json:"domain_name"`
	MonthlyLimit         json.Number  `json:"monthly_limit"`
	Spent                json.Number  `json:"spent"`
	Ratio                float64      `json:"ratio"`
	Projected            *json.Number `json:"projected"`
	ProjectionAvailable  bool         `json:"projection_available"`
	Scope                string       `json:"scope"`
	Remaining            json.Number  `json:"remaining"`
	Exceeded             bool         `json:"exceeded"`
	Status               string       `json:"status"`
	DebitTotal           json.Number  `json:"debit_total"`
	CreditTotal          json.Number  `json:"credit_total"`
	TransactionCount     int          `json:"transaction_count"`
	Month                string       `json:"month"`
	Period               periodInfo   `json:"period"`
	QueryFrom            string       `json:"query_from"`
	QueryTo              string       `json:"query_to"`
	limit, spent, credit money.Cents
}
type budgetSummary struct {
	Allocated          json.Number `json:"allocated"`
	Spent              json.Number `json:"spent"`
	Remaining          json.Number `json:"remaining"`
	OverCount          int         `json:"over_count"`
	ConsumptionRatio   *float64    `json:"consumption_ratio"`
	ObservedExpenses   json.Number `json:"observed_expenses"`
	CoveredExpenses    json.Number `json:"covered_expenses"`
	CoverageRatio      *float64    `json:"coverage_ratio"`
	UncategorizedSpent json.Number `json:"uncategorized_spent"`
	UncategorizedCount int         `json:"uncategorized_count"`
	CreditsObserved    json.Number `json:"credits_observed"`
	SavingsOutflows    json.Number `json:"savings_outflows"`
	TransferOutflows   json.Number `json:"transfer_outflows"`
}
type budgetReport struct {
	Month     string            `json:"month"`
	Period    periodInfo        `json:"period"`
	Coverage  reliabilityReport `json:"coverage"`
	Budgets   []budgetStatus    `json:"budgets"`
	Summary   budgetSummary     `json:"summary"`
	QueryFrom string            `json:"query_from"`
	QueryTo   string            `json:"query_to"`
}

func budgetNumber(c money.Cents) json.Number { return json.Number(c.String()) }

func finishBudgetStatus(b *budgetStatus) {
	b.MonthlyLimit = budgetNumber(b.limit)
	b.Spent = budgetNumber(b.spent)
	b.DebitTotal = b.Spent
	b.CreditTotal = budgetNumber(b.credit)
	b.Remaining = budgetNumber(b.limit - b.spent)
	b.Ratio = float64(b.spent) / float64(b.limit)
	b.Exceeded = b.spent > b.limit
	b.Status = "available"
	if b.Exceeded {
		b.Status = "over"
	} else if b.spent == b.limit {
		b.Status = "reached"
	}
}

// loadBudgetReport reads limits and source transactions in a single read-only
// snapshot, so edits during a request cannot make the envelope sum diverge.
func (s *Server) loadBudgetReport(ctx context.Context, owner int64, month string, now time.Time) (budgetReport, error) {
	first, last, current, err := reliabilityWindow(month, now)
	report := budgetReport{Month: first.Format("2006-01"), QueryFrom: first.Format("2006-01-02"), QueryTo: last.Format("2006-01-02"), Budgets: []budgetStatus{}}
	if err != nil {
		return report, err
	}
	report.Coverage, err = s.loadReliability(ctx, owner, report.Month, now)
	if err != nil {
		return report, err
	}
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return report, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT b.id,b.domain_id,d.name,b.monthly_limit::text,
 (d.is_transfer OR COALESCE(parent.is_transfer,false))
 FROM budgets b JOIN domains d ON d.id=b.domain_id AND d.owner_id=b.owner_id
 LEFT JOIN domains parent ON parent.id=d.parent_id AND parent.owner_id=d.owner_id
 WHERE b.owner_id=$1 ORDER BY b.id`, owner)
	if err != nil {
		return report, err
	}
	indices := map[int64]int{}
	for rows.Next() {
		var b budgetStatus
		var limit string
		var transfer bool
		if err = rows.Scan(&b.ID, &b.DomainID, &b.DomainName, &limit, &transfer); err != nil {
			break
		}
		b.limit, err = money.Parse(limit)
		if err != nil || b.limit <= 0 {
			err = fmt.Errorf("plafond budgétaire incohérent")
			break
		}
		b.Month = report.Month
		b.QueryFrom = report.QueryFrom
		b.QueryTo = report.QueryTo
		b.Scope = "expense"
		if transfer {
			b.Scope = "transfer"
		}
		indices[b.ID] = len(report.Budgets)
		report.Budgets = append(report.Budgets, b)
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		return report, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT t.tx_date::text,t.amount::text,`+budgetAssignmentSQL+`,
 `+budgetExpenseSQL+`,(t.domain_id IS NULL OR NOT EXISTS(SELECT 1 FROM domains own_domain WHERE own_domain.id=t.domain_id AND own_domain.owner_id=t.owner_id)),COALESCE(t.label_kind,'')='savings'
 FROM transactions t WHERE t.owner_id=$1 AND t.tx_date >= $2 AND t.tx_date < $3`, owner, first, last.AddDate(0, 0, 1))
	if err != nil {
		return report, err
	}
	var observed, covered, uncategorized, credits, savings, transfers money.Cents
	var lastTransaction string
	for rows.Next() {
		var date, rawAmount string
		var assigned sql.NullInt64
		var expense, uncategorizedDomain, saving bool
		if err = rows.Scan(&date, &rawAmount, &assigned, &expense, &uncategorizedDomain, &saving); err != nil {
			break
		}
		var amount money.Cents
		amount, err = money.Parse(rawAmount)
		if err != nil {
			break
		}
		if date > lastTransaction {
			lastTransaction = date
		}
		if amount < 0 && !expense {
			if saving {
				savings -= amount
			} else {
				transfers -= amount
			}
		}
		if expense {
			if amount < 0 {
				observed -= amount
				if uncategorizedDomain {
					uncategorized -= amount
					report.Summary.UncategorizedCount++
				}
			}
			if amount > 0 {
				credits += amount
			}
		}
		i, assignedToBudget := indices[assigned.Int64]
		if !assigned.Valid || !assignedToBudget {
			continue
		}
		b := &report.Budgets[i]
		if b.Scope == "expense" && !expense {
			continue
		}
		b.TransactionCount++
		if amount < 0 {
			b.spent -= amount
			if b.Scope == "expense" {
				covered -= amount
			}
		}
		if amount > 0 {
			b.credit += amount
		}
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		return report, err
	}
	if err = tx.Commit(); err != nil {
		return report, err
	}
	report.Period = periodFromReliability(report.Coverage, lastTransaction, first.AddDate(0, 1, -1).Day(), current, now.Day())
	var allocated, spent money.Cents
	for i := range report.Budgets {
		b := &report.Budgets[i]
		b.Period = report.Period
		finishBudgetStatus(b)
		if b.Scope == "expense" {
			allocated += b.limit
			spent += b.spent
			if b.Exceeded {
				report.Summary.OverCount++
			}
		}
	}
	sort.SliceStable(report.Budgets, func(i, j int) bool {
		a, b := report.Budgets[i], report.Budgets[j]
		if a.Scope != b.Scope {
			return a.Scope == "expense"
		}
		if a.Ratio != b.Ratio {
			return a.Ratio > b.Ratio
		}
		return a.ID < b.ID
	})
	report.Summary.Allocated = budgetNumber(allocated)
	report.Summary.Spent = budgetNumber(spent)
	report.Summary.Remaining = budgetNumber(allocated - spent)
	report.Summary.ObservedExpenses = budgetNumber(observed)
	report.Summary.CoveredExpenses = budgetNumber(covered)
	report.Summary.UncategorizedSpent = budgetNumber(uncategorized)
	report.Summary.CreditsObserved = budgetNumber(credits)
	report.Summary.SavingsOutflows = budgetNumber(savings)
	report.Summary.TransferOutflows = budgetNumber(transfers)
	if allocated > 0 {
		v := float64(spent) / float64(allocated)
		report.Summary.ConsumptionRatio = &v
	}
	if observed > 0 {
		v := float64(covered) / float64(observed)
		report.Summary.CoverageRatio = &v
	}
	return report, nil
}

func (s *Server) budgetReportHandler(w http.ResponseWriter, r *http.Request) {
	month := r.URL.Query().Get("month")
	if _, _, _, err := reliabilityWindow(month, time.Now()); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	report, err := s.loadBudgetReport(r.Context(), s.ownerID(r), month, time.Now())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Errorf("lecture des budgets indisponible"))
		return
	}
	writeJSON(w, http.StatusOK, report)
}
func (s *Server) listBudgets(w http.ResponseWriter, r *http.Request) {
	month := r.URL.Query().Get("month")
	if _, _, _, err := reliabilityWindow(month, time.Now()); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	report, err := s.loadBudgetReport(r.Context(), s.ownerID(r), month, time.Now())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Errorf("lecture des budgets indisponible"))
		return
	}
	writeJSON(w, http.StatusOK, report.Budgets)
}
func (s *Server) upsertBudget(w http.ResponseWriter, r *http.Request) {
	var in struct {
		DomainID     int64       `json:"domain_id"`
		MonthlyLimit json.Number `json:"monthly_limit"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("budget JSON invalide"))
		return
	}
	limit, err := money.Parse(string(in.MonthlyLimit))
	if err != nil || limit <= 0 || in.DomainID <= 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("catégorie et plafond positif au centime requis"))
		return
	}
	result, err := s.DB.ExecContext(r.Context(), `INSERT INTO budgets(domain_id,monthly_limit,owner_id)
 SELECT id,$2::numeric,$3 FROM domains WHERE id=$1 AND owner_id=$3
 ON CONFLICT(owner_id,domain_id) DO UPDATE SET monthly_limit=EXCLUDED.monthly_limit`, in.DomainID, limit.String(), s.ownerID(r))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Errorf("enregistrement du budget impossible"))
		return
	}
	n, err := result.RowsAffected()
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	if n == 0 {
		writeErr(w, http.StatusNotFound, fmt.Errorf("catégorie indisponible"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
func (s *Server) deleteBudget(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("identifiant de budget invalide"))
		return
	}
	result, err := s.DB.ExecContext(r.Context(), `DELETE FROM budgets WHERE id=$1 AND owner_id=$2`, id, s.ownerID(r))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Errorf("suppression du budget impossible"))
		return
	}
	n, err := result.RowsAffected()
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	if n == 0 {
		writeErr(w, http.StatusNotFound, fmt.Errorf("budget indisponible"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
