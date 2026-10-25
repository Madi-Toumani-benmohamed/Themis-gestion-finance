package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"themis/internal/money"
)

func lockTax(ctx context.Context, tx *sql.Tx, owner, id int64) (taxState, error) {
	return scanTax(tx.QueryRowContext(ctx, `SELECT `+taxStateColumns+` FROM taxes t WHERE t.id=$1 AND t.owner_id=$2 FOR UPDATE`, id, owner))
}
func recordTaxChange(ctx context.Context, tx *sql.Tx, owner int64, kind string, before *taxView, after taxView, key string, paymentDate *string, paymentAmount *money.Cents, note string) error {
	var old any
	paidBefore := false
	if before != nil {
		data, err := json.Marshal(before)
		if err != nil {
			return err
		}
		old = string(data)
		paidBefore = before.Paid
	}
	data, err := json.Marshal(after)
	if err != nil {
		return err
	}
	var submission, amount any
	if key != "" {
		submission = key
	}
	if paymentAmount != nil {
		amount = int64(*paymentAmount)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO tax_history(tax_id,resource_id,owner_id,paid_before,paid_after,changed_by,event_type,before_state,after_state,idempotency_key,payment_date,payment_amount_cents,note)
 VALUES($1,$1,$2,$3,$4,$2,$5,$6::jsonb,$7::jsonb,$8,$9::date,$10,$11)`, after.ID, owner, paidBefore, after.Paid, kind, old, string(data), submission, paymentDate, amount, note)
	return err
}
func (s *Server) createTax(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name   string      `json:"name"`
		Amount json.Number `json:"amount"`
		Date   string      `json:"due_date"`
	}
	if err := decodeEngagement(w, r, &in); err != nil {
		writeErr(w, 400, err)
		return
	}
	amount, err := money.Parse(string(in.Amount))
	if err != nil || amount <= 0 {
		writeErr(w, 400, fmt.Errorf("montant positif au centime requis"))
		return
	}
	if _, err = engagementDate(in.Date); err != nil {
		writeErr(w, 400, err)
		return
	}
	name := strings.TrimSpace(in.Name)
	if name == "" || len(name) > 200 {
		writeErr(w, 400, fmt.Errorf("nom requis, 200 caractères maximum"))
		return
	}
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		engagementHTTPError(w, err)
		return
	}
	defer tx.Rollback()
	t := taxState{Name: name, Amount: amount, DueDate: in.Date, PaidAmount: sql.NullInt64{Int64: 0, Valid: true}}
	err = tx.QueryRowContext(r.Context(), `INSERT INTO taxes(owner_id,name,amount,due_date,paid_amount_cents) VALUES($1,$2,$3::numeric,$4::date,0) RETURNING id`, s.ownerID(r), name, amount.String(), in.Date).Scan(&t.ID)
	if err != nil {
		engagementHTTPError(w, err)
		return
	}
	out, err := buildTax(t, time.Now())
	if err != nil {
		engagementHTTPError(w, err)
		return
	}
	if err = recordTaxChange(r.Context(), tx, s.ownerID(r), "created", nil, out, "", nil, nil, ""); err != nil {
		engagementHTTPError(w, err)
		return
	}
	if err = tx.Commit(); err != nil {
		engagementHTTPError(w, err)
		return
	}
	writeJSON(w, 201, out)
}
func applyTaxPatch(t *taxState, patch map[string]json.RawMessage, now time.Time) (string, error) {
	if len(patch) == 0 {
		return "", fmt.Errorf("aucun champ à modifier")
	}
	before, err := buildTax(*t, now)
	if err != nil {
		return "", err
	}
	paid, err := engagementAmount(string(before.PaidAmount), false)
	if err != nil {
		return "", err
	}
	moneyChanged := false
	originalAmount := t.Amount
	note := ""
	for key, raw := range patch {
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return "", fmt.Errorf("valeur nulle interdite pour %s", key)
		}
		switch key {
		case "name":
			if json.Unmarshal(raw, &t.Name) != nil {
				return "", fmt.Errorf("nom invalide")
			}
			t.Name = strings.TrimSpace(t.Name)
		case "amount":
			var n json.Number
			if json.Unmarshal(raw, &n) != nil {
				return "", fmt.Errorf("montant invalide")
			}
			t.Amount, err = money.Parse(string(n))
			if err != nil {
				return "", err
			}
			moneyChanged = t.Amount != originalAmount
		case "due_date":
			if json.Unmarshal(raw, &t.DueDate) != nil {
				return "", fmt.Errorf("date invalide")
			}
		case "paid", "paid_amount": // Applied below, after the amount, regardless of map order.
		case "overdue_confirmed":
			if json.Unmarshal(raw, &t.OverdueConfirmed) != nil {
				return "", fmt.Errorf("confirmation invalide")
			}
		case "note":
			if json.Unmarshal(raw, &note) != nil {
				return "", fmt.Errorf("note invalide")
			}
		default:
			return "", fmt.Errorf("champ non modifiable : %s", key)
		}
	}
	if _, a := patch["paid"]; a {
		if _, b := patch["paid_amount"]; b {
			return "", fmt.Errorf("choisir un statut payé ou un montant déclaré")
		}
	}
	if raw, ok := patch["paid"]; ok {
		var flag bool
		if json.Unmarshal(raw, &flag) != nil {
			return "", fmt.Errorf("statut payé invalide")
		}
		paid = 0
		if flag {
			paid = t.Amount
		}
		moneyChanged = true
	}
	if raw, ok := patch["paid_amount"]; ok {
		v, e := engagementOptionalAmount(raw)
		if e != nil || v == nil {
			return "", fmt.Errorf("montant payé invalide")
		}
		paid = *v
		moneyChanged = true
	}
	if strings.TrimSpace(t.Name) == "" || len(t.Name) > 200 || t.Amount <= 0 {
		return "", fmt.Errorf("nom et montant positif requis")
	}
	if _, err = engagementDate(t.DueDate); err != nil {
		return "", err
	}
	if paid > t.Amount {
		return "", fmt.Errorf("le montant payé dépasse le montant de l’engagement")
	}
	if moneyChanged {
		t.PaidAmount = sql.NullInt64{Int64: int64(paid), Valid: true}
		t.Paid = paid == t.Amount
	}
	if t.OverdueConfirmed && (t.Paid || t.DueDate >= engagementToday(now).Format("2006-01-02")) {
		if _, explicit := patch["overdue_confirmed"]; explicit {
			return "", fmt.Errorf("un retard ne peut être confirmé que pour une échéance passée restant à payer")
		}
		t.OverdueConfirmed = false
	}
	return engagementNote(note)
}
func saveTaxState(ctx context.Context, tx *sql.Tx, owner int64, t taxState) error {
	var paidAmount any
	if t.PaidAmount.Valid {
		paidAmount = t.PaidAmount.Int64
	}
	_, err := tx.ExecContext(ctx, `UPDATE taxes SET name=$1,amount=$2::numeric,due_date=$3::date,paid=$4,paid_amount_cents=$5,overdue_confirmed=$6 WHERE id=$7 AND owner_id=$8`, t.Name, t.Amount.String(), t.DueDate, t.Paid, paidAmount, t.OverdueConfirmed, t.ID, owner)
	return err
}
func (s *Server) patchTax(w http.ResponseWriter, r *http.Request) {
	id, err := engagementPathID(r, "id")
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	var patch map[string]json.RawMessage
	if err = decodeEngagement(w, r, &patch); err != nil {
		writeErr(w, 400, err)
		return
	}
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		engagementHTTPError(w, err)
		return
	}
	defer tx.Rollback()
	owner := s.ownerID(r)
	t, err := lockTax(r.Context(), tx, owner, id)
	if err != nil {
		engagementHTTPError(w, err)
		return
	}
	if t.Archived.Valid {
		writeErr(w, 409, fmt.Errorf("échéance archivée"))
		return
	}
	now := time.Now()
	before, err := buildTax(t, now)
	if err != nil {
		engagementHTTPError(w, err)
		return
	}
	if _, payment := patch["payment_amount"]; payment {
		s.patchTaxPayment(w, r, tx, t, before, patch, now)
		return
	}
	note, err := applyTaxPatch(&t, patch, now)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	after, err := buildTax(t, now)
	if err != nil {
		engagementHTTPError(w, err)
		return
	}
	oldData, _ := json.Marshal(before)
	newData, _ := json.Marshal(after)
	if !bytes.Equal(oldData, newData) {
		if err = saveTaxState(r.Context(), tx, owner, t); err != nil {
			engagementHTTPError(w, err)
			return
		}
		if err = recordTaxChange(r.Context(), tx, owner, "updated", &before, after, "", nil, nil, note); err != nil {
			engagementHTTPError(w, err)
			return
		}
	}
	if err = tx.Commit(); err != nil {
		engagementHTTPError(w, err)
		return
	}
	writeJSON(w, 200, after)
}
func (s *Server) patchTaxPayment(w http.ResponseWriter, r *http.Request, tx *sql.Tx, t taxState, before taxView, patch map[string]json.RawMessage, now time.Time) {
	var in struct {
		Amount json.Number `json:"payment_amount"`
		Date   string      `json:"payment_date"`
		Note   string      `json:"note"`
		Key    string      `json:"idempotency_key"`
	}
	data, err := json.Marshal(patch)
	if err != nil {
		engagementHTTPError(w, err)
		return
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&in); err != nil {
		writeErr(w, 400, fmt.Errorf("un versement ne peut pas être combiné avec une autre modification"))
		return
	}
	amount, err := engagementAmount(string(in.Amount), true)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	date, err := engagementDate(in.Date)
	if err != nil || date.After(engagementToday(now)) {
		writeErr(w, 400, fmt.Errorf("date de versement passée ou du jour requise"))
		return
	}
	note, err := engagementNote(in.Note)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	if err = engagementSubmission(in.Key); err != nil {
		writeErr(w, 400, err)
		return
	}
	var resource, oldAmount int64
	var oldDate, oldNote string
	err = tx.QueryRowContext(r.Context(), `SELECT resource_id,payment_amount_cents,payment_date::text,note FROM tax_history WHERE owner_id=$1 AND idempotency_key=$2`, s.ownerID(r), in.Key).Scan(&resource, &oldAmount, &oldDate, &oldNote)
	if err == nil {
		if resource != t.ID || oldAmount != int64(amount) || oldDate != in.Date || oldNote != note {
			writeErr(w, 409, fmt.Errorf("identifiant de soumission déjà utilisé"))
			return
		}
		if err = tx.Commit(); err != nil {
			engagementHTTPError(w, err)
			return
		}
		writeJSON(w, 200, map[string]any{"tax": before, "replayed": true})
		return
	}
	if !errors.Is(err, sql.ErrNoRows) {
		engagementHTTPError(w, err)
		return
	}
	paid, err := engagementAmount(string(before.PaidAmount), false)
	if err != nil {
		engagementHTTPError(w, err)
		return
	}
	sum, err := engagementAdd(paid, amount)
	if err != nil || sum > t.Amount {
		writeErr(w, 409, fmt.Errorf("le versement dépasse le solde déclaré"))
		return
	}
	t.PaidAmount = sql.NullInt64{Int64: int64(sum), Valid: true}
	t.Paid = sum == t.Amount
	if t.Paid {
		t.OverdueConfirmed = false
	}
	after, err := buildTax(t, now)
	if err != nil {
		engagementHTTPError(w, err)
		return
	}
	if err = saveTaxState(r.Context(), tx, s.ownerID(r), t); err != nil {
		engagementHTTPError(w, err)
		return
	}
	if err = recordTaxChange(r.Context(), tx, s.ownerID(r), "payment", &before, after, in.Key, &in.Date, &amount, note); err != nil {
		engagementHTTPError(w, err)
		return
	}
	if err = tx.Commit(); err != nil {
		engagementHTTPError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"tax": after, "replayed": false})
}
func (s *Server) deleteTax(w http.ResponseWriter, r *http.Request) {
	id, err := engagementPathID(r, "id")
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		engagementHTTPError(w, err)
		return
	}
	defer tx.Rollback()
	t, err := lockTax(r.Context(), tx, s.ownerID(r), id)
	if err != nil {
		engagementHTTPError(w, err)
		return
	}
	now := time.Now()
	before, err := buildTax(t, now)
	if err != nil {
		engagementHTTPError(w, err)
		return
	}
	if !t.Archived.Valid {
		t.Archived = sql.NullString{String: now.UTC().Format(time.RFC3339), Valid: true}
		_, err = tx.ExecContext(r.Context(), `UPDATE taxes SET archived_at=$1::timestamptz WHERE id=$2 AND owner_id=$3`, t.Archived.String, id, s.ownerID(r))
		if err != nil {
			engagementHTTPError(w, err)
			return
		}
		after, e := buildTax(t, now)
		if e != nil {
			engagementHTTPError(w, e)
			return
		}
		if err = recordTaxChange(r.Context(), tx, s.ownerID(r), "archived", &before, after, "", nil, nil, ""); err != nil {
			engagementHTTPError(w, err)
			return
		}
	}
	if err = tx.Commit(); err != nil {
		engagementHTTPError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"status": "archivé", "history_preserved": true})
}
