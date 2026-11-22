package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"themis/internal/classification"
)

func classificationFailure(w http.ResponseWriter, e error) {
	e = classification.DatabaseError(e)
	if errors.Is(e, classification.ErrConflict) {
		writeErr(w, 409, e)
	} else if errors.Is(e, classification.ErrNotFound) {
		writeErr(w, 404, e)
	} else {
		writeErr(w, 500, fmt.Errorf("décision de classification non enregistrée"))
	}
}
func (s *Server) classificationHistory(w http.ResponseWriter, r *http.Request) {
	id, e := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if e != nil || id <= 0 {
		writeErr(w, 400, fmt.Errorf("opération invalide"))
		return
	}
	before := int64(9223372036854775807)
	if v := r.URL.Query().Get("before_id"); v != "" {
		before, e = strconv.ParseInt(v, 10, 64)
		if e != nil || before <= 0 {
			writeErr(w, 400, fmt.Errorf("pagination invalide"))
			return
		}
	}
	owner := s.ownerID(r)
	tx, e := s.DB.BeginTx(r.Context(), &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if e != nil {
		classificationFailure(w, e)
		return
	}
	defer tx.Rollback()
	state, e := classification.Read(r.Context(), tx, owner, id)
	if e != nil {
		classificationFailure(w, e)
		return
	}
	rows, e := tx.QueryContext(r.Context(), `SELECT id,revision,occurred_at::text,origin,event_type,rule_id,tag_id,before_state,after_state,reason,actor_id,operation_id,suggestion_identity FROM classification_events WHERE owner_id=$1 AND transaction_id=$2 AND id<$3 ORDER BY id DESC LIMIT 51`, owner, id, before)
	if e != nil {
		classificationFailure(w, e)
		return
	}
	defer rows.Close()
	type event struct {
		ID                 int64           `json:"id"`
		Revision           int64           `json:"revision,string"`
		At                 string          `json:"at"`
		Origin             string          `json:"origin"`
		Type               string          `json:"event_type"`
		RuleID             *int64          `json:"rule_id"`
		TagID              *int64          `json:"tag_id"`
		Before             json.RawMessage `json:"before"`
		After              json.RawMessage `json:"after"`
		Reason             string          `json:"reason"`
		ActorID            *int64          `json:"actor_id"`
		OperationID        string          `json:"operation_id"`
		SuggestionIdentity *string         `json:"suggestion_identity"`
	}

	events := []event{}
	for rows.Next() {
		var x event
		if e = rows.Scan(&x.ID, &x.Revision, &x.At, &x.Origin, &x.Type, &x.RuleID, &x.TagID, &x.Before, &x.After, &x.Reason, &x.ActorID, &x.OperationID, &x.SuggestionIdentity); e != nil {
			classificationFailure(w, e)
			return
		}
		events = append(events, x)
	}
	if e = rows.Err(); e != nil {
		classificationFailure(w, e)
		return
	}
	more := len(events) > 50
	if more {
		events = events[:50]
	}
	rows.Close()
	if e = tx.Commit(); e != nil {
		classificationFailure(w, e)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, map[string]any{"state": state, "events": events, "has_more": more, "historical_provenance": "not_reconstructed"})
}
