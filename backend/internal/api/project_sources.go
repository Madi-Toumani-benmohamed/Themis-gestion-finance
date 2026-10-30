package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

func (s *Server) projectTransactions(w http.ResponseWriter, r *http.Request) {
	id, e := projectID(r)
	if e != nil {
		projectHTTPError(w, e)
		return
	}
	owner := s.ownerID(r)
	state := r.URL.Query().Get("state")
	if state == "" {
		state = "confirmed"
	}
	if state != "confirmed" && state != "suggested" && state != "rejected" {
		projectHTTPError(w, projectBad("état invalide"))
		return
	}
	_, _, last, e := folderWindow(r)
	if e != nil {
		projectHTTPError(w, projectBad(e.Error()))
		return
	}
	offset, e := projectPage(r)
	if e != nil {
		projectHTTPError(w, e)
		return
	}
	tx, e := s.DB.BeginTx(r.Context(), &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if e != nil {
		projectHTTPError(w, e)
		return
	}
	defer tx.Rollback()
	var exists bool
	e = tx.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM financial_projects WHERE owner_id=$1 AND id=$2)`, owner, id).Scan(&exists)
	if e != nil {
		projectHTTPError(w, e)
		return
	}
	if !exists {
		projectHTTPError(w, sql.ErrNoRows)
		return
	}
	rows, e := tx.QueryContext(r.Context(), `SELECT t.id,t.tx_date::text,COALESCE(NULLIF(t.merchant,''),NULLIF(t.display_label,''),t.label),t.amount::text,pt.state,pt.reason,`+projectEligible+`,COALESCE(d.name,'Sans catégorie')
 FROM financial_projects p JOIN project_transactions pt ON pt.project_id=p.id AND pt.owner_id=p.owner_id JOIN transactions t ON t.id=pt.transaction_id AND t.owner_id=p.owner_id LEFT JOIN domains d ON d.id=t.domain_id AND d.owner_id=t.owner_id
 WHERE p.owner_id=$1 AND p.id=$2 AND pt.state=$4 AND t.tx_date<=$3::date AND ($5='' OR COALESCE(d.name,'Sans catégorie')=$5)
 ORDER BY t.tx_date DESC,t.id DESC LIMIT 51 OFFSET $6`, owner, id, last, state, r.URL.Query().Get("category"), offset)
	if e != nil {
		projectHTTPError(w, e)
		return
	}
	out := []map[string]any{}
	for rows.Next() {
		var tid int64
		var date, label, raw, current, cat string
		var reason json.RawMessage
		var eligible bool
		if e = rows.Scan(&tid, &date, &label, &raw, &current, &reason, &eligible, &cat); e != nil {
			rows.Close()
			projectHTTPError(w, e)
			return
		}
		c, e := parseAnalysisAmount(raw)
		if e != nil {
			rows.Close()
			projectHTTPError(w, e)
			return
		}
		out = append(out, map[string]any{"id": tid, "date": date, "label": label, "amount_cents": int64(c), "state": current, "reason": reason, "included": current == "confirmed" && eligible, "eligible": eligible, "category": cat})
	}
	e = rows.Err()
	rows.Close()
	if e == nil {
		e = tx.Commit()
	}
	if e != nil {
		projectHTTPError(w, e)
		return
	}
	more := len(out) > 50
	if more {
		out = out[:50]
	}
	writeJSON(w, 200, map[string]any{"transactions": out, "has_more": more, "offset": offset, "query_to": last})
}
func (s *Server) projectHistoryHTTP(w http.ResponseWriter, r *http.Request) {
	id, e := projectID(r)
	if e != nil {
		projectHTTPError(w, e)
		return
	}
	offset, e := projectPage(r)
	if e != nil {
		projectHTTPError(w, e)
		return
	}
	var exists bool
	e = s.DB.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM financial_projects WHERE owner_id=$1 AND id=$2)`, s.ownerID(r), id).Scan(&exists)
	if e != nil {
		projectHTTPError(w, e)
		return
	}
	if !exists {
		projectHTTPError(w, sql.ErrNoRows)
		return
	}
	rows, e := s.DB.QueryContext(r.Context(), `SELECT h.id,h.transaction_id,h.event,h.before_state,h.after_state,h.changed_at::text FROM project_history h WHERE h.owner_id=$1 AND h.project_id=$2 ORDER BY h.id DESC LIMIT 51 OFFSET $3`, s.ownerID(r), id, offset)
	if e != nil {
		projectHTTPError(w, e)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var hid int64
		var tid *int64
		var event, date string
		var before, after json.RawMessage
		if e = rows.Scan(&hid, &tid, &event, &before, &after, &date); e != nil {
			projectHTTPError(w, e)
			return
		}
		out = append(out, map[string]any{"id": hid, "transaction_id": tid, "event": event, "before": before, "after": after, "changed_at": date})
	}
	if e = rows.Err(); e != nil {
		projectHTTPError(w, e)
		return
	}
	more := len(out) > 50
	if more {
		out = out[:50]
	}
	writeJSON(w, 200, map[string]any{"history": out, "has_more": more})
}
