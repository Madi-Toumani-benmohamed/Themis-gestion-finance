package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
	"themis/internal/money"
)

type reimbursementError struct {
	status  int
	message string
}

func (e *reimbursementError) Error() string { return e.message }

func reimbursementHTTPError(w http.ResponseWriter, err error) {
	var appErr *reimbursementError
	if errors.As(err, &appErr) {
		writeErr(w, appErr.status, fmt.Errorf("%s", appErr.message))
		return
	}
	if errors.Is(err, sql.ErrNoRows) {
		writeErr(w, 404, fmt.Errorf("remboursement ou transaction introuvable"))
		return
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && (pqErr.Code == "23503" || pqErr.Code == "23505" || pqErr.Code == "23514") {
		writeErr(w, 409, fmt.Errorf("modification incompatible avec les rapprochements existants"))
		return
	}
	writeErr(w, 500, fmt.Errorf("opération de remboursement indisponible ; aucune modification confirmée"))
}

func reimbursementBad(message string) error {
	return &reimbursementError{status: 400, message: message}
}
func reimbursementConflict(message string) error {
	return &reimbursementError{status: 409, message: message}
}
func reimbursementNotFound() error {
	return &reimbursementError{status: 404, message: "remboursement ou transaction introuvable"}
}

type reimbursementSnapshot struct {
	TransactionID int64  `json:"transaction_id"`
	Date          string `json:"date"`
	AmountCents   int64  `json:"amount_cents"`
	Label         string `json:"label"`
	DisplayLabel  string `json:"display_label"`
	Merchant      string `json:"merchant"`
	DomainName    string `json:"domain_name,omitempty"`
}

type reimbursementAllocationView struct {
	ID                   int64                 `json:"id"`
	ExpenseTransactionID *int64                `json:"expense_transaction_id,omitempty"`
	Expense              reimbursementSnapshot `json:"expense"`
	AmountCents          int64                 `json:"amount_cents"`
	CreatedAt            string                `json:"created_at"`
	VoidedAt             *string               `json:"voided_at,omitempty"`
	VoidReason           string                `json:"void_reason,omitempty"`
}

type reimbursementCaseView struct {
	ID                  int64                         `json:"id"`
	CreditTransactionID *int64                        `json:"credit_transaction_id,omitempty"`
	Credit              reimbursementSnapshot         `json:"credit"`
	Kind                string                        `json:"kind"`
	Status              string                        `json:"status"`
	Note                string                        `json:"note"`
	AllocatedCents      int64                         `json:"allocated_cents"`
	UnallocatedCents    int64                         `json:"unallocated_cents"`
	CreatedAt           string                        `json:"created_at"`
	UpdatedAt           string                        `json:"updated_at"`
	Allocations         []reimbursementAllocationView `json:"allocations"`
}

type eligibleReimbursementExpense struct {
	ExpenseSnapshot
	RemainingCents int64 `json:"remaining_cents"`
}

// ExpenseSnapshot is intentionally public to the JSON encoder through the
// embedded fields, while keeping financial values in integer cents.
type ExpenseSnapshot struct {
	TransactionID int64  `json:"transaction_id"`
	Date          string `json:"date"`
	AmountCents   int64  `json:"amount_cents"`
	Label         string `json:"label"`
	DisplayLabel  string `json:"display_label"`
	Merchant      string `json:"merchant"`
	DomainName    string `json:"domain_name,omitempty"`
}

type reimbursementQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func readReimbursementSnapshot(ctx context.Context, q reimbursementQuerier, owner, id int64) (reimbursementSnapshot, bool, error) {
	var snapshot reimbursementSnapshot
	var amountRaw string
	var eligible bool
	err := q.QueryRowContext(ctx, `SELECT t.id,t.tx_date::text,t.amount::text,t.label,
	 COALESCE(t.display_label,''),COALESCE(t.merchant,''),COALESCE(d.name,''),`+financialExpenseSQLT+`
	 FROM transactions t LEFT JOIN domains d ON d.id=t.domain_id AND d.owner_id=t.owner_id
	 WHERE t.id=$1 AND t.owner_id=$2`, id, owner).Scan(&snapshot.TransactionID, &snapshot.Date, &amountRaw, &snapshot.Label, &snapshot.DisplayLabel, &snapshot.Merchant, &snapshot.DomainName, &eligible)
	if err != nil {
		return snapshot, false, err
	}
	amount, err := money.Parse(amountRaw)
	if err != nil {
		return snapshot, false, err
	}
	snapshot.AmountCents = int64(amount)
	return snapshot, eligible, nil
}

func encodeReimbursementSnapshot(snapshot reimbursementSnapshot) (string, error) {
	b, err := json.Marshal(snapshot)
	return string(b), err
}

func decodeReimbursementSnapshot(raw string) (reimbursementSnapshot, error) {
	var snapshot reimbursementSnapshot
	err := json.Unmarshal([]byte(raw), &snapshot)
	return snapshot, err
}

func reimbursementCents(raw string) (int64, error) {
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || v < 0 || v > int64(engagementMaxCents) {
		return 0, fmt.Errorf("agrégat de remboursement hors limites")
	}
	return v, nil
}

func lockReimbursementTransactions(ctx context.Context, tx *sql.Tx, owner int64, ids ...int64) error {
	unique := map[int64]bool{}
	ordered := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id > 0 && !unique[id] {
			unique[id] = true
			ordered = append(ordered, id)
		}
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	if len(ordered) == 0 {
		return reimbursementNotFound()
	}
	rows, err := tx.QueryContext(ctx, `SELECT id FROM transactions WHERE owner_id=$1 AND id=ANY($2) ORDER BY id FOR UPDATE`, owner, pq.Array(ordered))
	if err != nil {
		return err
	}
	locked := 0
	for rows.Next() {
		var id int64
		if err = rows.Scan(&id); err != nil {
			break
		}
		locked++
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		return err
	}
	if locked != len(ordered) {
		return reimbursementNotFound()
	}
	return nil
}

func recordReimbursementHistory(ctx context.Context, tx *sql.Tx, owner, actor, caseID int64, allocationID *int64, event string, before, after any, note string) error {
	var beforeJSON any
	if before != nil {
		b, err := json.Marshal(before)
		if err != nil {
			return err
		}
		beforeJSON = string(b)
	}
	afterBytes, err := json.Marshal(after)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO reimbursement_history(owner_id,case_id,allocation_id,event_type,before_state,after_state,actor_id,note)
	 VALUES($1,$2,$3,$4,$5::jsonb,$6::jsonb,$7,$8)`, owner, caseID, allocationID, event, beforeJSON, string(afterBytes), actor, note)
	return err
}

func updateReimbursementStatus(ctx context.Context, tx *sql.Tx, owner, caseID int64) error {
	var amountRaw string
	var allocatedRaw string
	if err := tx.QueryRowContext(ctx, `SELECT (c.credit_snapshot->>'amount_cents')::bigint,
	 COALESCE((SELECT sum(a.amount_cents) FROM reimbursement_allocations a WHERE a.owner_id=c.owner_id AND a.case_id=c.id AND a.voided_at IS NULL),0)::text
	 FROM reimbursement_cases c WHERE c.id=$1 AND c.owner_id=$2 FOR UPDATE`, caseID, owner).Scan(&amountRaw, &allocatedRaw); err != nil {
		return err
	}
	credit, err := reimbursementCents(amountRaw)
	if err != nil || credit <= 0 {
		return fmt.Errorf("montant source du dossier invalide")
	}
	allocated, err := reimbursementCents(allocatedRaw)
	if err != nil || allocated > credit {
		return fmt.Errorf("allocations incohérentes")
	}
	status := "unmatched"
	if allocated > 0 {
		status = "partial"
		if allocated == credit {
			status = "allocated"
		}
	}
	_, err = tx.ExecContext(ctx, `UPDATE reimbursement_cases SET status=$3,updated_at=now() WHERE id=$1 AND owner_id=$2 AND status<>'voided'`, caseID, owner, status)
	return err
}

func readReimbursementCase(ctx context.Context, q reimbursementQuerier, owner, caseID int64) (reimbursementCaseView, error) {
	out := reimbursementCaseView{Allocations: []reimbursementAllocationView{}}
	var creditID sql.NullInt64
	var creditRaw string
	var allocatedRaw string
	err := q.QueryRowContext(ctx, `SELECT c.id,c.credit_transaction_id,c.credit_snapshot::text,c.kind,c.status,c.note,
	 COALESCE((SELECT sum(a.amount_cents) FROM reimbursement_allocations a WHERE a.owner_id=c.owner_id AND a.case_id=c.id AND a.voided_at IS NULL),0)::text,
	 c.created_at::text,c.updated_at::text
	 FROM reimbursement_cases c WHERE c.id=$1 AND c.owner_id=$2`, caseID, owner).Scan(&out.ID, &creditID, &creditRaw, &out.Kind, &out.Status, &out.Note, &allocatedRaw, &out.CreatedAt, &out.UpdatedAt)
	if err != nil {
		return out, err
	}
	if creditID.Valid {
		id := creditID.Int64
		out.CreditTransactionID = &id
	}
	out.Credit, err = decodeReimbursementSnapshot(creditRaw)
	if err != nil {
		return out, err
	}
	out.AllocatedCents, err = reimbursementCents(allocatedRaw)
	if err != nil || out.AllocatedCents > out.Credit.AmountCents {
		return out, fmt.Errorf("allocations incohérentes")
	}
	out.UnallocatedCents = out.Credit.AmountCents - out.AllocatedCents
	rows, err := q.QueryContext(ctx, `SELECT a.id,a.expense_transaction_id,a.expense_snapshot::text,a.amount_cents,
	 a.created_at::text,a.voided_at::text,a.void_reason
	 FROM reimbursement_allocations a WHERE a.case_id=$1 AND a.owner_id=$2 ORDER BY a.id`, caseID, owner)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var allocation reimbursementAllocationView
		var expenseID sql.NullInt64
		var expenseRaw string
		var amount int64
		var voided sql.NullString
		if err = rows.Scan(&allocation.ID, &expenseID, &expenseRaw, &amount, &allocation.CreatedAt, &voided, &allocation.VoidReason); err != nil {
			return out, err
		}
		allocation.Expense, err = decodeReimbursementSnapshot(expenseRaw)
		if err != nil || amount <= 0 {
			return out, fmt.Errorf("allocation incohérente")
		}
		if expenseID.Valid {
			id := int64(expenseID.Int64)
			allocation.ExpenseTransactionID = &id
		}
		allocation.AmountCents = amount
		allocation.VoidedAt = reimbursementTimestamp(voided)
		out.Allocations = append(out.Allocations, allocation)
	}
	if err = rows.Err(); err != nil {
		return out, err
	}
	return out, nil
}

func validateReimbursementNote(raw string, max int) (string, error) {
	note := strings.TrimSpace(raw)
	if len(note) > max {
		return "", reimbursementBad("note trop longue")
	}
	return note, nil
}

func reimbursementTimestamp(raw sql.NullString) *string {
	if !raw.Valid {
		return nil
	}
	return &raw.String
}

func voidTime() time.Time { return time.Now().UTC() }
