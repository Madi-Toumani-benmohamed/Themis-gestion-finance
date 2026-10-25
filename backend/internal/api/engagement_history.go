package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

type engagementChange struct {
	ID           int64           `json:"id"`
	Event        string          `json:"event_type"`
	Before       json.RawMessage `json:"before_state"`
	After        json.RawMessage `json:"after_state"`
	Date         string          `json:"changed_at"`
	Note         string          `json:"note"`
	PaymentDate  *string         `json:"payment_date,omitempty"`
	PaymentCents *int64          `json:"payment_amount_cents,omitempty"`
}

func engagementHistoryPage(r *http.Request) (int, error) {
	raw := r.URL.Query().Get("offset")
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 || n > 1000000 {
		return 0, fmt.Errorf("pagination invalide")
	}
	return n, nil
}
func (s *Server) creditHistory(w http.ResponseWriter, r *http.Request) {
	s.engagementHistory(w, r, true)
}
func (s *Server) taxHistory(w http.ResponseWriter, r *http.Request) { s.engagementHistory(w, r, false) }
func (s *Server) engagementHistory(w http.ResponseWriter, r *http.Request, credit bool) {
	id, err := engagementPathID(r, "id")
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	offset, err := engagementHistoryPage(r)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	tx, err := s.DB.BeginTx(r.Context(), &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		engagementHTTPError(w, err)
		return
	}
	defer tx.Rollback()
	table := "taxes"
	if credit {
		table = "credits"
	}
	owner := s.ownerID(r)
	var owned int64
	if err = tx.QueryRowContext(r.Context(), `SELECT id FROM `+table+` WHERE id=$1 AND owner_id=$2`, id, owner).Scan(&owned); err != nil {
		engagementHTTPError(w, err)
		return
	}
	out := map[string]any{"changes": []engagementChange{}, "offset": offset, "limit": 50}
	changes, more, err := readEngagementChanges(r.Context(), tx, owner, id, offset, credit)
	if err != nil {
		engagementHTTPError(w, err)
		return
	}
	out["changes"] = changes
	out["has_more"] = more
	if credit {
		rows, e := tx.QueryContext(r.Context(), `SELECT `+creditPaymentColumns+` FROM credit_payments WHERE credit_id=$1 AND owner_id=$2 ORDER BY id DESC LIMIT 51 OFFSET $3`, id, owner, offset)
		if e != nil {
			engagementHTTPError(w, e)
			return
		}
		payments := []creditPaymentView{}
		for rows.Next() {
			p, _, e := scanPayment(rows)
			if e != nil {
				rows.Close()
				engagementHTTPError(w, e)
				return
			}
			payments = append(payments, p)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			engagementHTTPError(w, e)
			return
		}
		out["payments_has_more"] = len(payments) > 50
		if len(payments) > 50 {
			payments = payments[:50]
		}
		out["payments"] = payments
	}
	if err = tx.Commit(); err != nil {
		engagementHTTPError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, out)
}
func readEngagementChanges(ctx context.Context, tx *sql.Tx, owner, id int64, offset int, credit bool) ([]engagementChange, bool, error) {
	query := `SELECT id,event_type,before_state::text,after_state::text,changed_at::text,note,payment_date::text,payment_amount_cents FROM tax_history WHERE owner_id=$1 AND resource_id=$2 ORDER BY id DESC LIMIT 51 OFFSET $3`
	if credit {
		query = `SELECT id,event_type,before_state::text,after_state::text,changed_at::text,''::text,NULL::text,NULL::bigint FROM credit_history WHERE owner_id=$1 AND resource_id=$2 ORDER BY id DESC LIMIT 51 OFFSET $3`
	}
	rows, err := tx.QueryContext(ctx, query, owner, id, offset)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	out := []engagementChange{}
	for rows.Next() {
		var c engagementChange
		var before, after, date sql.NullString
		var cents sql.NullInt64
		if err = rows.Scan(&c.ID, &c.Event, &before, &after, &c.Date, &c.Note, &date, &cents); err != nil {
			return nil, false, err
		}
		if before.Valid {
			c.Before = json.RawMessage(before.String)
		}
		if after.Valid {
			c.After = json.RawMessage(after.String)
		}
		c.PaymentDate = engagementString(date)
		if cents.Valid {
			v := cents.Int64
			c.PaymentCents = &v
		}
		out = append(out, c)
	}
	if err = rows.Err(); err != nil {
		return nil, false, err
	}
	more := len(out) > 50
	if more {
		out = out[:50]
	}
	return out, more, nil
}
