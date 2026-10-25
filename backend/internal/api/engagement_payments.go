package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"themis/internal/money"
)

type creditPaymentView struct {
	AmountCents int64       `json:"amount_cents"`
	ID          int64       `json:"id"`
	Amount      json.Number `json:"amount"`
	Date        string      `json:"payment_date"`
	Note        string      `json:"note"`
	Created     string      `json:"created_at"`
	Voided      *string     `json:"voided_at"`
	VoidNote    string      `json:"void_note"`
}

func scanPayment(row engagementScanner) (creditPaymentView, money.Cents, error) {
	var p creditPaymentView
	var cents int64
	var voided sql.NullString
	err := row.Scan(&p.ID, &cents, &p.Date, &p.Note, &p.Created, &voided, &p.VoidNote)
	if err != nil {
		return p, 0, err
	}
	if cents <= 0 || cents > int64(engagementMaxCents) {
		return p, 0, fmt.Errorf("versement hors limites")
	}
	amount := money.Cents(cents)
	p.AmountCents = cents
	p.Amount = engagementNumber(amount)
	p.Voided = engagementString(voided)
	return p, amount, nil
}

const creditPaymentColumns = `id,amount_cents,payment_date::text,note,created_at::text,voided_at::text,void_note`

func (s *Server) recordCreditPayment(w http.ResponseWriter, r *http.Request) {
	id, err := engagementPathID(r, "id")
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	var in struct {
		Amount json.Number `json:"amount"`
		Date   string      `json:"payment_date"`
		Note   string      `json:"note"`
		Key    string      `json:"idempotency_key"`
	}
	if err = decodeEngagement(w, r, &in); err != nil {
		writeErr(w, 400, err)
		return
	}
	amount, err := engagementAmount(string(in.Amount), true)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	date, err := engagementDate(in.Date)
	if err != nil || date.After(engagementToday(time.Now())) {
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
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		engagementHTTPError(w, err)
		return
	}
	defer tx.Rollback()
	owner := s.ownerID(r)
	c, err := lockCredit(r.Context(), tx, owner, id)
	if err != nil {
		engagementHTTPError(w, err)
		return
	}
	old, oldAmount, e := scanPayment(tx.QueryRowContext(r.Context(), `SELECT `+creditPaymentColumns+` FROM credit_payments WHERE credit_id=$1 AND owner_id=$2 AND idempotency_key=$3`, id, owner, in.Key))
	if e == nil {
		if old.Voided != nil {
			writeErr(w, 409, fmt.Errorf("ce versement a été annulé ; une nouvelle soumission distincte est requise"))
			return
		}
		if oldAmount != amount || old.Date != in.Date || old.Note != note {
			writeErr(w, 409, fmt.Errorf("identifiant de soumission déjà utilisé pour un autre versement"))
			return
		}
		if err = tx.Commit(); err != nil {
			engagementHTTPError(w, err)
			return
		}
		writeJSON(w, 200, map[string]any{"payment": old, "replayed": true})
		return
	}
	if !errors.Is(e, sql.ErrNoRows) {
		engagementHTTPError(w, e)
		return
	}
	if c.Archived.Valid || c.Closed.Valid {
		writeErr(w, 409, fmt.Errorf("crédit archivé ou clôturé"))
		return
	}
	view := buildCredit(c, time.Now())
	if view.Status == "invalid" {
		writeErr(w, 409, fmt.Errorf("échéancier incohérent : vérifier les données avant un paiement"))
		return
	}
	total, err := engagementProduct(c.Monthly, c.Months)
	if err != nil {
		writeErr(w, 409, err)
		return
	}
	paid := c.Recorded
	if c.Opening.Valid {
		paid, err = engagementAdd(paid, money.Cents(c.Opening.Int64))
		if err != nil {
			writeErr(w, 409, err)
			return
		}
	}
	after, err := engagementAdd(paid, amount)
	if err != nil || after > total {
		writeErr(w, 409, fmt.Errorf("le versement dépasse le solde de l’échéancier déclaré"))
		return
	}
	payment, _, err := scanPayment(tx.QueryRowContext(r.Context(), `INSERT INTO credit_payments(credit_id,owner_id,amount_cents,payment_date,note,idempotency_key) VALUES($1,$2,$3,$4::date,$5,$6) RETURNING `+creditPaymentColumns, id, owner, int64(amount), in.Date, note, in.Key))
	if err != nil {
		engagementHTTPError(w, err)
		return
	}
	beforeView := buildCredit(c, time.Now())
	c.Recorded += amount
	c.PaymentCount++
	if err = recordCreditChange(r.Context(), tx, owner, "payment", &beforeView, buildCredit(c, time.Now())); err != nil {
		engagementHTTPError(w, err)
		return
	}
	if err = tx.Commit(); err != nil {
		engagementHTTPError(w, err)
		return
	}
	writeJSON(w, 201, map[string]any{"payment": payment, "replayed": false})
}
func (s *Server) voidCreditPayment(w http.ResponseWriter, r *http.Request) {
	id, err := engagementPathID(r, "id")
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	paymentID, err := engagementPathID(r, "payment_id")
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	var in struct {
		Note string `json:"note"`
	}
	if err = decodeEngagement(w, r, &in); err != nil {
		writeErr(w, 400, err)
		return
	}
	note, err := engagementNote(in.Note)
	if err != nil || note == "" {
		writeErr(w, 400, fmt.Errorf("motif d’annulation requis"))
		return
	}
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		engagementHTTPError(w, err)
		return
	}
	defer tx.Rollback()
	owner := s.ownerID(r)
	c, err := lockCredit(r.Context(), tx, owner, id)
	if err != nil {
		engagementHTTPError(w, err)
		return
	}
	if c.Closed.Valid || c.Archived.Valid {
		writeErr(w, 409, fmt.Errorf("rouvrir le crédit avant de corriger un versement"))
		return
	}
	p, _, err := scanPayment(tx.QueryRowContext(r.Context(), `SELECT `+creditPaymentColumns+` FROM credit_payments WHERE id=$1 AND credit_id=$2 AND owner_id=$3`, paymentID, id, owner))
	if err != nil {
		engagementHTTPError(w, err)
		return
	}
	if p.Voided == nil {
		beforeView := buildCredit(c, time.Now())
		paymentCents := money.Cents(p.AmountCents)
		p, _, err = scanPayment(tx.QueryRowContext(r.Context(), `UPDATE credit_payments SET voided_at=now(),void_note=$1 WHERE id=$2 AND credit_id=$3 AND owner_id=$4 RETURNING `+creditPaymentColumns, note, paymentID, id, owner))
		if err != nil {
			engagementHTTPError(w, err)
			return
		}
		c.Recorded -= paymentCents
		c.PaymentCount--
		if err = recordCreditChange(r.Context(), tx, owner, "payment_voided", &beforeView, buildCredit(c, time.Now())); err != nil {
			engagementHTTPError(w, err)
			return
		}

	}
	if err = tx.Commit(); err != nil {
		engagementHTTPError(w, err)
		return
	}
	writeJSON(w, 200, p)
}
