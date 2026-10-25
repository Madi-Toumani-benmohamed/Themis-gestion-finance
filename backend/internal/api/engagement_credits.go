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

	"github.com/lib/pq"
	"themis/internal/money"
)

func engagementHTTPError(w http.ResponseWriter, err error) {
	var databaseError *pq.Error
	if errors.As(err, &databaseError) && databaseError.Code == "23505" {
		writeErr(w, 409, fmt.Errorf("soumission déjà utilisée ; actualiser l’historique"))
		return
	}
	if errors.Is(err, sql.ErrNoRows) {
		writeErr(w, 404, fmt.Errorf("engagement introuvable"))
		return
	}
	writeErr(w, 500, fmt.Errorf("opération indisponible ; aucune modification confirmée"))
}
func lockCredit(ctx context.Context, tx *sql.Tx, owner, id int64) (creditState, error) {
	// Separate statements: after waiting for the lock, the aggregate must use a
	// fresh READ COMMITTED snapshot, including the preceding writer's payments.
	var locked int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM credits WHERE id=$1 AND owner_id=$2 FOR UPDATE`, id, owner).Scan(&locked); err != nil {
		return creditState{}, err
	}
	return scanCredit(tx.QueryRowContext(ctx, `SELECT `+creditStateColumns+` FROM credits c WHERE c.id=$1 AND c.owner_id=$2`, id, owner))
}
func recordCreditChange(ctx context.Context, tx *sql.Tx, owner int64, kind string, before *creditView, after creditView) error {
	var old any
	if before != nil {
		data, err := json.Marshal(before)
		if err != nil {
			return err
		}
		old = string(data)
	}
	data, err := json.Marshal(after)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO credit_history(credit_id,resource_id,owner_id,event_type,before_state,after_state) VALUES($1,$1,$2,$3,$4::jsonb,$5::jsonb)`, after.ID, owner, kind, old, string(data))
	return err
}
func validCreditState(c creditState, now time.Time) error {
	if strings.TrimSpace(c.Name) == "" || len(c.Name) > 200 {
		return fmt.Errorf("nom requis, 200 caractères maximum")
	}
	if c.Monthly <= 0 || c.Monthly > 999999999999 {
		return fmt.Errorf("mensualité positive au centime requise")
	}
	if len(c.Note) > 2000 {
		return fmt.Errorf("note trop longue")
	}
	if c.Archived.Valid {
		return fmt.Errorf("crédit archivé")
	}
	v := buildCredit(c, now)
	if v.Status == "invalid" {
		return fmt.Errorf("%s", v.Issue)
	}
	return nil
}
func (s *Server) createCredit(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name    string          `json:"name"`
		Monthly json.Number     `json:"monthly_amount"`
		Months  int             `json:"months_total"`
		Start   string          `json:"start_month"`
		Note    string          `json:"note"`
		Opening json.RawMessage `json:"opening_paid"`
	}
	if err := decodeEngagement(w, r, &in); err != nil {
		writeErr(w, 400, err)
		return
	}
	monthly, err := money.Parse(string(in.Monthly))
	if err != nil {
		writeErr(w, 400, fmt.Errorf("mensualité positive exacte requise"))
		return
	}
	opening, err := engagementOptionalAmount(in.Opening)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	c := creditState{Name: strings.TrimSpace(in.Name), Monthly: monthly, Months: in.Months, StartMonth: in.Start, Note: strings.TrimSpace(in.Note)}
	if opening != nil {
		c.Opening = sql.NullInt64{Int64: int64(*opening), Valid: true}
	}
	now := time.Now()
	if err = validCreditState(c, now); err != nil {
		writeErr(w, 400, err)
		return
	}
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		engagementHTTPError(w, err)
		return
	}
	defer tx.Rollback()
	var baseline any
	if c.Opening.Valid {
		baseline = c.Opening.Int64
	}
	err = tx.QueryRowContext(r.Context(), `INSERT INTO credits(owner_id,name,monthly_amount,months_total,start_month,note,opening_paid_cents) VALUES($1,$2,$3::numeric,$4,$5,$6,$7) RETURNING id`, s.ownerID(r), c.Name, c.Monthly.String(), c.Months, c.StartMonth, c.Note, baseline).Scan(&c.ID)
	if err != nil {
		engagementHTTPError(w, err)
		return
	}
	out := buildCredit(c, now)
	if err = recordCreditChange(r.Context(), tx, s.ownerID(r), "created", nil, out); err != nil {
		engagementHTTPError(w, err)
		return
	}
	if err = tx.Commit(); err != nil {
		engagementHTTPError(w, err)
		return
	}
	writeJSON(w, 201, out)
}
func applyCreditPatch(c *creditState, patch map[string]json.RawMessage, now time.Time) error {
	if len(patch) == 0 {
		return fmt.Errorf("aucun champ à modifier")
	}
	for key, raw := range patch {
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) && key != "opening_paid" {
			return fmt.Errorf("valeur nulle interdite pour %s", key)
		}
		switch key {
		case "name":
			if json.Unmarshal(raw, &c.Name) != nil {
				return fmt.Errorf("nom invalide")
			}
			c.Name = strings.TrimSpace(c.Name)
		case "monthly_amount":
			var n json.Number
			if json.Unmarshal(raw, &n) != nil {
				return fmt.Errorf("mensualité invalide")
			}
			value, err := money.Parse(string(n))
			if err != nil {
				return err
			}
			c.Monthly = value
		case "months_total":
			if json.Unmarshal(raw, &c.Months) != nil {
				return fmt.Errorf("durée invalide")
			}
		case "start_month":
			if json.Unmarshal(raw, &c.StartMonth) != nil {
				return fmt.Errorf("mois invalide")
			}
		case "note":
			if json.Unmarshal(raw, &c.Note) != nil {
				return fmt.Errorf("note invalide")
			}
			c.Note = strings.TrimSpace(c.Note)
		case "opening_paid":
			value, err := engagementOptionalAmount(raw)
			if err != nil {
				return err
			}
			c.Opening = sql.NullInt64{}
			if value != nil {
				c.Opening = sql.NullInt64{Int64: int64(*value), Valid: true}
			}
		case "closed":
			var closed bool
			if json.Unmarshal(raw, &closed) != nil {
				return fmt.Errorf("clôture invalide")
			}
			if closed {
				if !c.Closed.Valid {
					c.Closed = sql.NullString{String: now.UTC().Format(time.RFC3339), Valid: true}
				}
			} else {
				c.Closed = sql.NullString{}
			}
		default:
			return fmt.Errorf("champ non modifiable : %s", key)
		}
	}
	return validCreditState(*c, now)
}
func (s *Server) updateCredit(w http.ResponseWriter, r *http.Request) {
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
	c, err := lockCredit(r.Context(), tx, s.ownerID(r), id)
	if err != nil {
		engagementHTTPError(w, err)
		return
	}
	now := time.Now()
	before := buildCredit(c, now)
	if c.Archived.Valid {
		writeErr(w, 409, fmt.Errorf("crédit archivé"))
		return
	}
	if err = applyCreditPatch(&c, patch, now); err != nil {
		writeErr(w, 400, err)
		return
	}
	after := buildCredit(c, now)
	oldData, _ := json.Marshal(before)
	newData, _ := json.Marshal(after)
	if !bytes.Equal(oldData, newData) {
		var baseline, closed any
		if c.Opening.Valid {
			baseline = c.Opening.Int64
		}
		if c.Closed.Valid {
			closed = c.Closed.String
		}
		_, err = tx.ExecContext(r.Context(), `UPDATE credits SET name=$1,monthly_amount=$2::numeric,months_total=$3,start_month=$4,note=$5,opening_paid_cents=$6,closed_at=$7::timestamptz WHERE id=$8 AND owner_id=$9`, c.Name, c.Monthly.String(), c.Months, c.StartMonth, c.Note, baseline, closed, c.ID, s.ownerID(r))
		if err != nil {
			engagementHTTPError(w, err)
			return
		}
		if err = recordCreditChange(r.Context(), tx, s.ownerID(r), "updated", &before, after); err != nil {
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
func (s *Server) deleteCredit(w http.ResponseWriter, r *http.Request) {
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
	c, err := lockCredit(r.Context(), tx, s.ownerID(r), id)
	if err != nil {
		engagementHTTPError(w, err)
		return
	}
	now := time.Now()
	before := buildCredit(c, now)
	if !c.Archived.Valid {
		c.Archived = sql.NullString{String: now.UTC().Format(time.RFC3339), Valid: true}
		_, err = tx.ExecContext(r.Context(), `UPDATE credits SET archived_at=$1::timestamptz WHERE id=$2 AND owner_id=$3`, c.Archived.String, id, s.ownerID(r))
		if err != nil {
			engagementHTTPError(w, err)
			return
		}
		if err = recordCreditChange(r.Context(), tx, s.ownerID(r), "archived", &before, buildCredit(c, now)); err != nil {
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
