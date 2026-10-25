package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"themis/internal/money"
)

type reimbursementPage struct {
	Cases  []reimbursementCaseView `json:"cases"`
	Total  int                     `json:"total"`
	Limit  int                     `json:"limit"`
	Offset int                     `json:"offset"`
}

func reimbursementPagination(r *http.Request) (int, int, error) {
	limit, offset := 100, 0
	for key, target := range map[string]*int{"limit": &limit, "offset": &offset} {
		if raw := r.URL.Query().Get(key); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil || n < 0 || (key == "limit" && (n < 1 || n > 100)) || (key == "offset" && n > 1000000) {
				return 0, 0, reimbursementBad("pagination invalide")
			}
			*target = n
		}
	}
	return limit, offset, nil
}

func (s *Server) listReimbursements(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := reimbursementPagination(r)
	if err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	owner := s.ownerID(r)
	var transactionID int64
	if raw := r.URL.Query().Get("transaction_id"); raw != "" {
		transactionID, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || transactionID <= 0 {
			reimbursementHTTPError(w, reimbursementBad("transaction_id invalide"))
			return
		}
	}
	var total int
	err = s.DB.QueryRowContext(r.Context(), `SELECT count(*) FROM reimbursement_cases c
	 WHERE c.owner_id=$1 AND ($2=0 OR c.credit_transaction_id=$2 OR EXISTS(
	  SELECT 1 FROM reimbursement_allocations a WHERE a.owner_id=c.owner_id AND a.case_id=c.id AND a.expense_transaction_id=$2))`, owner, transactionID).Scan(&total)
	if err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	rows, err := s.DB.QueryContext(r.Context(), `SELECT c.id FROM reimbursement_cases c
	 WHERE c.owner_id=$1 AND ($2=0 OR c.credit_transaction_id=$2 OR EXISTS(
	  SELECT 1 FROM reimbursement_allocations a WHERE a.owner_id=c.owner_id AND a.case_id=c.id AND a.expense_transaction_id=$2))
	 ORDER BY c.updated_at DESC,c.id DESC LIMIT $3 OFFSET $4`, owner, transactionID, limit, offset)
	if err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err = rows.Scan(&id); err != nil {
			break
		}
		ids = append(ids, id)
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	out := reimbursementPage{Cases: []reimbursementCaseView{}, Total: total, Limit: limit, Offset: offset}
	for _, id := range ids {
		item, readErr := readReimbursementCase(r.Context(), s.DB, owner, id)
		if readErr != nil {
			reimbursementHTTPError(w, readErr)
			return
		}
		out.Cases = append(out.Cases, item)
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, out)
}

func (s *Server) getReimbursement(w http.ResponseWriter, r *http.Request) {
	id, err := engagementPathID(r, "id")
	if err != nil {
		reimbursementHTTPError(w, reimbursementBad("identifiant de dossier invalide"))
		return
	}
	out, err := readReimbursementCase(r.Context(), s.DB, s.ownerID(r), id)
	if err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, out)
}

func (s *Server) createReimbursement(w http.ResponseWriter, r *http.Request) {
	var in struct {
		CreditTransactionID int64  `json:"credit_transaction_id"`
		Kind                string `json:"kind"`
		Note                string `json:"note"`
	}
	if err := decodeEngagement(w, r, &in); err != nil {
		reimbursementHTTPError(w, reimbursementBad(err.Error()))
		return
	}
	if in.CreditTransactionID <= 0 || (in.Kind != "personal_expense" && in.Kind != "merchant_refund") {
		reimbursementHTTPError(w, reimbursementBad("transaction créditrice et type de remboursement requis"))
		return
	}
	note, err := validateReimbursementNote(in.Note, 2000)
	if err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	owner := s.ownerID(r)
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	defer tx.Rollback()
	if err = lockReimbursementTransactions(r.Context(), tx, owner, in.CreditTransactionID); err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	snapshot, eligible, err := readReimbursementSnapshot(r.Context(), tx, owner, in.CreditTransactionID)
	if err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	if snapshot.AmountCents <= 0 {
		reimbursementHTTPError(w, reimbursementBad("la source du remboursement doit être un crédit strictement positif"))
		return
	}
	if !eligible {
		reimbursementHTTPError(w, reimbursementBad("un transfert interne ou un mouvement d’épargne ne peut pas être qualifié comme remboursement"))
		return
	}
	snapshotJSON, err := encodeReimbursementSnapshot(snapshot)
	if err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	var existingID int64
	var existingKind, existingNote string
	err = tx.QueryRowContext(r.Context(), `SELECT id,kind,note FROM reimbursement_cases WHERE owner_id=$1 AND credit_transaction_id=$2 FOR UPDATE`, owner, in.CreditTransactionID).Scan(&existingID, &existingKind, &existingNote)
	if err == nil {
		if existingKind != in.Kind || existingNote != note {
			reimbursementHTTPError(w, reimbursementConflict("ce crédit est déjà qualifié différemment"))
			return
		}
		if err = tx.Commit(); err != nil {
			reimbursementHTTPError(w, err)
			return
		}
		out, readErr := readReimbursementCase(r.Context(), s.DB, owner, existingID)
		if readErr != nil {
			reimbursementHTTPError(w, readErr)
			return
		}
		writeJSON(w, 200, out)
		return
	}
	if err != sql.ErrNoRows {
		reimbursementHTTPError(w, err)
		return
	}
	var id int64
	err = tx.QueryRowContext(r.Context(), `INSERT INTO reimbursement_cases(owner_id,credit_transaction_id,credit_snapshot,kind,note,created_by)
	 VALUES($1,$2,$3::jsonb,$4,$5,$1) RETURNING id`, owner, in.CreditTransactionID, snapshotJSON, in.Kind, note).Scan(&id)
	if err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	state := map[string]any{"credit_transaction_id": in.CreditTransactionID, "credit": snapshot, "kind": in.Kind, "status": "unmatched", "note": note}
	if err = recordReimbursementHistory(r.Context(), tx, owner, owner, id, nil, "case_created", nil, state, note); err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	if err = tx.Commit(); err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	out, err := readReimbursementCase(r.Context(), s.DB, owner, id)
	if err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	writeJSON(w, 201, out)
}

func (s *Server) eligibleReimbursementExpenses(w http.ResponseWriter, r *http.Request) {
	caseID, err := strconv.ParseInt(r.URL.Query().Get("case_id"), 10, 64)
	if err != nil || caseID <= 0 {
		reimbursementHTTPError(w, reimbursementBad("case_id requis"))
		return
	}
	limit, offset, err := reimbursementPagination(r)
	if err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	owner := s.ownerID(r)
	var creditID sql.NullInt64
	var status string
	if err = s.DB.QueryRowContext(r.Context(), `SELECT credit_transaction_id,status FROM reimbursement_cases WHERE id=$1 AND owner_id=$2`, caseID, owner).Scan(&creditID, &status); err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	if status == "voided" || !creditID.Valid {
		reimbursementHTTPError(w, reimbursementConflict("ce dossier ne peut plus recevoir d’allocation"))
		return
	}
	args := []any{owner, creditID.Int64}
	base := ` FROM transactions t LEFT JOIN domains d ON d.id=t.domain_id AND d.owner_id=t.owner_id
	 WHERE t.owner_id=$1 AND t.id<>$2 AND t.amount<0 AND ` + financialExpenseSQLT + ` AND
	 (-t.amount*100)::bigint>COALESCE((SELECT sum(a.amount_cents) FROM reimbursement_allocations a JOIN reimbursement_cases c ON c.id=a.case_id AND c.owner_id=a.owner_id
	 WHERE a.owner_id=t.owner_id AND a.expense_transaction_id=t.id AND a.voided_at IS NULL AND c.status<>'voided'),0)`
	var total int
	if err = s.DB.QueryRowContext(r.Context(), `SELECT count(*)`+base, args...).Scan(&total); err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	args = append(args, limit, offset)
	rows, err := s.DB.QueryContext(r.Context(), `SELECT t.id,t.tx_date::text,t.amount::text,t.label,COALESCE(t.display_label,''),COALESCE(t.merchant,''),COALESCE(d.name,''),
	 COALESCE((SELECT sum(a.amount_cents) FROM reimbursement_allocations a JOIN reimbursement_cases c ON c.id=a.case_id AND c.owner_id=a.owner_id
	 WHERE a.owner_id=t.owner_id AND a.expense_transaction_id=t.id AND a.voided_at IS NULL AND c.status<>'voided'),0)::text`+base+
		` ORDER BY t.tx_date DESC,t.id DESC LIMIT $3 OFFSET $4`, args...)
	if err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	out := struct {
		Expenses []eligibleReimbursementExpense `json:"expenses"`
		Total    int                            `json:"total"`
		Limit    int                            `json:"limit"`
		Offset   int                            `json:"offset"`
	}{Expenses: []eligibleReimbursementExpense{}, Total: total, Limit: limit, Offset: offset}
	for rows.Next() {
		var item eligibleReimbursementExpense
		var amountRaw, allocatedRaw string
		if err = rows.Scan(&item.TransactionID, &item.Date, &amountRaw, &item.Label, &item.DisplayLabel, &item.Merchant, &item.DomainName, &allocatedRaw); err != nil {
			break
		}
		amount, parseErr := money.Parse(amountRaw)
		if parseErr != nil || amount >= 0 {
			err = fmt.Errorf("dépense source incohérente")
			break
		}
		allocated, parseErr := reimbursementCents(allocatedRaw)
		if parseErr != nil {
			err = parseErr
			break
		}
		item.AmountCents = int64(amount)
		item.RemainingCents = int64(-amount) - allocated
		out.Expenses = append(out.Expenses, item)
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, out)
}

func (s *Server) createReimbursementAllocation(w http.ResponseWriter, r *http.Request) {
	caseID, err := engagementPathID(r, "id")
	if err != nil {
		reimbursementHTTPError(w, reimbursementBad("identifiant de dossier invalide"))
		return
	}
	key := r.Header.Get("Idempotency-Key")
	if err = engagementSubmission(key); err != nil {
		reimbursementHTTPError(w, reimbursementBad(err.Error()))
		return
	}
	var in struct {
		ExpenseTransactionID int64 `json:"expense_transaction_id"`
		AmountCents          int64 `json:"amount_cents"`
	}
	if err = decodeEngagement(w, r, &in); err != nil {
		reimbursementHTTPError(w, reimbursementBad(err.Error()))
		return
	}
	if in.ExpenseTransactionID <= 0 || in.AmountCents <= 0 || in.AmountCents > int64(engagementMaxCents) {
		reimbursementHTTPError(w, reimbursementBad("dépense et montant positif en centimes requis"))
		return
	}
	owner := s.ownerID(r)
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	defer tx.Rollback()
	var creditID sql.NullInt64
	var caseStatus string
	err = tx.QueryRowContext(r.Context(), `SELECT credit_transaction_id,status FROM reimbursement_cases WHERE id=$1 AND owner_id=$2 FOR UPDATE`, caseID, owner).Scan(&creditID, &caseStatus)
	if err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	if caseStatus == "voided" || caseStatus == "needs_review" || !creditID.Valid {
		reimbursementHTTPError(w, reimbursementConflict("ce dossier ne peut plus recevoir d’allocation"))
		return
	}
	var oldID, oldCase, oldExpense sql.NullInt64
	var oldAmount sql.NullInt64
	var oldSnapshot string
	err = tx.QueryRowContext(r.Context(), `SELECT id,case_id,expense_transaction_id,amount_cents,expense_snapshot::text FROM reimbursement_allocations WHERE owner_id=$1 AND idempotency_key=$2`, owner, key).Scan(&oldID, &oldCase, &oldExpense, &oldAmount, &oldSnapshot)
	if err == nil {
		old, decodeErr := decodeReimbursementSnapshot(oldSnapshot)
		if decodeErr != nil {
			reimbursementHTTPError(w, decodeErr)
			return
		}
		if oldCase.Int64 != caseID || old.TransactionID != in.ExpenseTransactionID || oldAmount.Int64 != in.AmountCents {
			reimbursementHTTPError(w, reimbursementConflict("clé d’idempotence déjà utilisée pour une autre allocation"))
			return
		}
		if err = tx.Commit(); err != nil {
			reimbursementHTTPError(w, err)
			return
		}
		out, readErr := readReimbursementCase(r.Context(), s.DB, owner, caseID)
		if readErr != nil {
			reimbursementHTTPError(w, readErr)
			return
		}
		writeJSON(w, 200, out)
		return
	}
	if err != sql.ErrNoRows {
		reimbursementHTTPError(w, err)
		return
	}
	if err = lockReimbursementTransactions(r.Context(), tx, owner, creditID.Int64, in.ExpenseTransactionID); err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	credit, creditEligible, err := readReimbursementSnapshot(r.Context(), tx, owner, creditID.Int64)
	if err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	expense, expenseEligible, err := readReimbursementSnapshot(r.Context(), tx, owner, in.ExpenseTransactionID)
	if err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	if credit.AmountCents <= 0 || expense.AmountCents >= 0 {
		reimbursementHTTPError(w, reimbursementBad("un crédit positif et un débit négatif sont requis"))
		return
	}
	if !creditEligible || !expenseEligible {
		reimbursementHTTPError(w, reimbursementBad("les transferts internes et mouvements d’épargne ne peuvent pas être rapprochés"))
		return
	}
	var caseAllocatedRaw, expenseAllocatedRaw string
	if err = tx.QueryRowContext(r.Context(), `SELECT COALESCE(sum(amount_cents),0)::text FROM reimbursement_allocations WHERE owner_id=$1 AND case_id=$2 AND voided_at IS NULL`, owner, caseID).Scan(&caseAllocatedRaw); err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	if err = tx.QueryRowContext(r.Context(), `SELECT COALESCE(sum(a.amount_cents),0)::text FROM reimbursement_allocations a JOIN reimbursement_cases c ON c.id=a.case_id AND c.owner_id=a.owner_id
	 WHERE a.owner_id=$1 AND a.expense_transaction_id=$2 AND a.voided_at IS NULL AND c.status<>'voided'`, owner, in.ExpenseTransactionID).Scan(&expenseAllocatedRaw); err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	caseAllocated, err := reimbursementCents(caseAllocatedRaw)
	if err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	expenseAllocated, err := reimbursementCents(expenseAllocatedRaw)
	if err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	if caseAllocated > credit.AmountCents || expenseAllocated > -expense.AmountCents || in.AmountCents > credit.AmountCents-caseAllocated || in.AmountCents > -expense.AmountCents-expenseAllocated {
		reimbursementHTTPError(w, reimbursementConflict("le montant dépasse le crédit disponible ou le reste remboursable de la dépense"))
		return
	}
	expenseJSON, err := encodeReimbursementSnapshot(expense)
	if err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	var allocationID int64
	err = tx.QueryRowContext(r.Context(), `INSERT INTO reimbursement_allocations(owner_id,case_id,expense_transaction_id,expense_snapshot,amount_cents,idempotency_key,created_by)
	 VALUES($1,$2,$3,$4::jsonb,$5,$6,$1) RETURNING id`, owner, caseID, in.ExpenseTransactionID, expenseJSON, in.AmountCents, key).Scan(&allocationID)
	if err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	if err = updateReimbursementStatus(r.Context(), tx, owner, caseID); err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	allocation := map[string]any{"id": allocationID, "expense_transaction_id": in.ExpenseTransactionID, "amount_cents": in.AmountCents, "expense": expense}
	if err = recordReimbursementHistory(r.Context(), tx, owner, owner, caseID, &allocationID, "allocation_created", nil, allocation, ""); err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	if err = tx.Commit(); err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	out, err := readReimbursementCase(r.Context(), s.DB, owner, caseID)
	if err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	writeJSON(w, 201, out)
}

func (s *Server) voidReimbursementAllocation(w http.ResponseWriter, r *http.Request) {
	caseID, err := engagementPathID(r, "id")
	if err != nil {
		reimbursementHTTPError(w, reimbursementBad("identifiant de dossier invalide"))
		return
	}
	allocationID, err := engagementPathID(r, "allocation_id")
	if err != nil {
		reimbursementHTTPError(w, reimbursementBad("identifiant d’allocation invalide"))
		return
	}
	var in struct {
		Reason string `json:"reason"`
	}
	if err = decodeEngagement(w, r, &in); err != nil {
		reimbursementHTTPError(w, reimbursementBad(err.Error()))
		return
	}
	reason, err := validateReimbursementNote(in.Reason, 1000)
	if err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	owner := s.ownerID(r)
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	defer tx.Rollback()
	var creditID sql.NullInt64
	var caseStatus string
	if err = tx.QueryRowContext(r.Context(), `SELECT credit_transaction_id,status FROM reimbursement_cases WHERE id=$1 AND owner_id=$2 FOR UPDATE`, caseID, owner).Scan(&creditID, &caseStatus); err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	var expenseID sql.NullInt64
	var amount int64
	var expenseRaw string
	var createdAt string
	var voided sql.NullString
	var oldReason string
	if err = tx.QueryRowContext(r.Context(), `SELECT expense_transaction_id,amount_cents,expense_snapshot::text,created_at::text,voided_at::text,void_reason
	 FROM reimbursement_allocations WHERE id=$1 AND case_id=$2 AND owner_id=$3 FOR UPDATE`, allocationID, caseID, owner).Scan(&expenseID, &amount, &expenseRaw, &createdAt, &voided, &oldReason); err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	if voided.Valid {
		if err = tx.Commit(); err != nil {
			reimbursementHTTPError(w, err)
			return
		}
		w.WriteHeader(204)
		return
	}
	ids := []int64{}
	if creditID.Valid {
		ids = append(ids, creditID.Int64)
	}
	if expenseID.Valid {
		ids = append(ids, expenseID.Int64)
	}
	if len(ids) > 0 {
		if err = lockReimbursementTransactions(r.Context(), tx, owner, ids...); err != nil {
			reimbursementHTTPError(w, err)
			return
		}
	}
	if _, err = tx.ExecContext(r.Context(), `UPDATE reimbursement_allocations SET voided_at=now(),voided_by=$3,void_reason=$4 WHERE id=$1 AND owner_id=$2 AND voided_at IS NULL`, allocationID, owner, owner, reason); err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	expenseSnapshot, err := decodeReimbursementSnapshot(expenseRaw)
	if err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	before := reimbursementAllocationView{ID: allocationID, ExpenseTransactionID: nullableID(expenseID), Expense: expenseSnapshot, AmountCents: amount, CreatedAt: createdAt}
	if voidedAt := reimbursementTimestamp(voided); voidedAt != nil {
		before.VoidedAt = voidedAt
	}
	allocationState := map[string]any{"id": allocationID, "expense_transaction_id": nullableID(expenseID), "expense": expenseSnapshot, "amount_cents": amount, "void_reason": reason}
	if err = recordReimbursementHistory(r.Context(), tx, owner, owner, caseID, &allocationID, "allocation_voided", before, allocationState, reason); err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	if caseStatus != "voided" {
		if err = updateReimbursementStatus(r.Context(), tx, owner, caseID); err != nil {
			reimbursementHTTPError(w, err)
			return
		}
	}
	if err = tx.Commit(); err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	w.WriteHeader(204)
}

func (s *Server) voidReimbursement(w http.ResponseWriter, r *http.Request) {
	caseID, err := engagementPathID(r, "id")
	if err != nil {
		reimbursementHTTPError(w, reimbursementBad("identifiant de dossier invalide"))
		return
	}
	var in struct {
		Reason string `json:"reason"`
	}
	if err = decodeEngagement(w, r, &in); err != nil {
		reimbursementHTTPError(w, reimbursementBad(err.Error()))
		return
	}
	reason, err := validateReimbursementNote(in.Reason, 1000)
	if err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	owner := s.ownerID(r)
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	defer tx.Rollback()
	var status string
	var creditID sql.NullInt64
	if err = tx.QueryRowContext(r.Context(), `SELECT status,credit_transaction_id FROM reimbursement_cases WHERE id=$1 AND owner_id=$2 FOR UPDATE`, caseID, owner).Scan(&status, &creditID); err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	if status == "voided" {
		if err = tx.Commit(); err != nil {
			reimbursementHTTPError(w, err)
			return
		}
		w.WriteHeader(204)
		return
	}
	ids := []int64{}
	if creditID.Valid {
		ids = append(ids, creditID.Int64)
	}
	rows, err := tx.QueryContext(r.Context(), `SELECT expense_transaction_id FROM reimbursement_allocations WHERE owner_id=$1 AND case_id=$2 AND voided_at IS NULL ORDER BY expense_transaction_id`, owner, caseID)
	if err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	for rows.Next() {
		var expenseID sql.NullInt64
		if err = rows.Scan(&expenseID); err != nil {
			break
		}
		if expenseID.Valid {
			ids = append(ids, expenseID.Int64)
		}
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	if len(ids) > 0 {
		if err = lockReimbursementTransactions(r.Context(), tx, owner, ids...); err != nil {
			reimbursementHTTPError(w, err)
			return
		}
	}
	rows, err = tx.QueryContext(r.Context(), `SELECT id,expense_transaction_id,expense_snapshot::text,amount_cents,created_at::text FROM reimbursement_allocations
	 WHERE owner_id=$1 AND case_id=$2 AND voided_at IS NULL ORDER BY id FOR UPDATE`, owner, caseID)
	if err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	type activeAllocation struct {
		id       int64
		expense  sql.NullInt64
		snapshot string
		amount   int64
		created  string
	}
	active := []activeAllocation{}
	for rows.Next() {
		var a activeAllocation
		if err = rows.Scan(&a.id, &a.expense, &a.snapshot, &a.amount, &a.created); err != nil {
			break
		}
		active = append(active, a)
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	for _, a := range active {
		if _, err = tx.ExecContext(r.Context(), `UPDATE reimbursement_allocations SET voided_at=now(),voided_by=$3,void_reason=$4 WHERE id=$1 AND owner_id=$2`, a.id, owner, owner, reason); err != nil {
			reimbursementHTTPError(w, err)
			return
		}
		snapshot, decodeErr := decodeReimbursementSnapshot(a.snapshot)
		if decodeErr != nil {
			reimbursementHTTPError(w, decodeErr)
			return
		}
		allocationState := map[string]any{"id": a.id, "expense_transaction_id": nullableID(a.expense), "expense": snapshot, "amount_cents": a.amount, "void_reason": reason}
		if err = recordReimbursementHistory(r.Context(), tx, owner, owner, caseID, &a.id, "allocation_voided", allocationState, allocationState, reason); err != nil {
			reimbursementHTTPError(w, err)
			return
		}
	}
	if _, err = tx.ExecContext(r.Context(), `UPDATE reimbursement_cases SET status='voided',updated_at=now() WHERE id=$1 AND owner_id=$2`, caseID, owner); err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	if err = recordReimbursementHistory(r.Context(), tx, owner, owner, caseID, nil, "case_voided", map[string]any{"status": status}, map[string]any{"status": "voided", "reason": reason}, reason); err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	if err = tx.Commit(); err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	w.WriteHeader(204)
}

func (s *Server) reimbursementHistory(w http.ResponseWriter, r *http.Request) {
	caseID, err := engagementPathID(r, "id")
	if err != nil {
		reimbursementHTTPError(w, reimbursementBad("identifiant de dossier invalide"))
		return
	}
	limit, offset, err := reimbursementPagination(r)
	if err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	owner := s.ownerID(r)
	var exists bool
	if err = s.DB.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM reimbursement_cases WHERE id=$1 AND owner_id=$2)`, caseID, owner).Scan(&exists); err != nil || !exists {
		reimbursementHTTPError(w, reimbursementNotFound())
		return
	}
	rows, err := s.DB.QueryContext(r.Context(), `SELECT id,event_type,before_state::text,after_state::text,actor_id,note,changed_at::text
	 FROM reimbursement_history WHERE owner_id=$1 AND case_id=$2 ORDER BY id DESC LIMIT $3 OFFSET $4`, owner, caseID, limit+1, offset)
	if err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	type historyItem struct {
		ID        int64           `json:"id"`
		Event     string          `json:"event_type"`
		Before    json.RawMessage `json:"before_state,omitempty"`
		After     json.RawMessage `json:"after_state"`
		ActorID   *int64          `json:"actor_id,omitempty"`
		Note      string          `json:"note"`
		ChangedAt string          `json:"changed_at"`
	}
	items := []historyItem{}
	for rows.Next() {
		var item historyItem
		var before, after sql.NullString
		var actor sql.NullInt64
		if err = rows.Scan(&item.ID, &item.Event, &before, &after, &actor, &item.Note, &item.ChangedAt); err != nil {
			break
		}
		if before.Valid {
			item.Before = json.RawMessage(before.String)
		}
		if after.Valid {
			item.After = json.RawMessage(after.String)
		}
		if actor.Valid {
			id := actor.Int64
			item.ActorID = &id
		}
		items = append(items, item)
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		reimbursementHTTPError(w, err)
		return
	}
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, map[string]any{"changes": items, "limit": limit, "offset": offset, "has_more": hasMore})
}

func nullableID(id sql.NullInt64) *int64 {
	if !id.Valid {
		return nil
	}
	v := id.Int64
	return &v
}
