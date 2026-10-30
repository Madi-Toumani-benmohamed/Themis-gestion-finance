package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type projectRule struct {
	ID       int64  `json:"id"`
	Kind     string `json:"kind"`
	TagID    *int64 `json:"tag_id"`
	DomainID *int64 `json:"domain_id"`
	Match    string `json:"match_text"`
	From     string `json:"from_date"`
	To       string `json:"to_date"`
	Active   bool   `json:"active"`
}

func (rule *projectRule) validate() error {
	rule.Match = strings.TrimSpace(rule.Match)
	a, e := time.Parse("2006-01-02", rule.From)
	if e != nil || a.Year() < 1 || a.Format("2006-01-02") != rule.From {
		return projectBad("date de début de règle invalide")
	}
	b, e := time.Parse("2006-01-02", rule.To)
	if e != nil || b.Format("2006-01-02") != rule.To || b.Before(a) || b.Sub(a).Hours()/24 > 3660 {
		return projectBad("règle : période requise de 0 à 3660 jours")
	}
	switch rule.Kind {
	case "tag":
		if rule.TagID == nil || *rule.TagID <= 0 || rule.DomainID != nil || rule.Match != "" {
			return projectBad("tag de règle invalide")
		}
	case "category":
		if rule.DomainID == nil || *rule.DomainID <= 0 || rule.TagID != nil || rule.Match != "" {
			return projectBad("catégorie de règle invalide")
		}
	case "merchant", "label":
		if rule.TagID != nil || rule.DomainID != nil || utf8.RuneCountInString(rule.Match) < 3 || utf8.RuneCountInString(rule.Match) > 200 {
			return projectBad("critère textuel invalide (3 à 200 caractères)")
		}
	default:
		return projectBad("type de règle invalide")
	}
	return nil
}
func projectLock(ctx context.Context, tx *sql.Tx, owner, id int64) error {
	var status string
	if e := tx.QueryRowContext(ctx, `SELECT status FROM financial_projects WHERE owner_id=$1 AND id=$2 FOR UPDATE`, owner, id).Scan(&status); e != nil {
		return e
	}
	if status == "archived" {
		return &projectError{409, "projet archivé : réactivez-le avant modification"}
	}
	return nil
}
func (s *Server) projectRules(w http.ResponseWriter, r *http.Request) {
	id, e := projectID(r)
	if e != nil {
		projectHTTPError(w, e)
		return
	}
	owner := s.ownerID(r)
	if r.Method == http.MethodGet {
		var exists bool
		e = s.DB.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM financial_projects WHERE owner_id=$1 AND id=$2)`, owner, id).Scan(&exists)
		if e != nil {
			projectHTTPError(w, e)
			return
		}
		if !exists {
			projectHTTPError(w, sql.ErrNoRows)
			return
		}
		rows, e := s.DB.QueryContext(r.Context(), `SELECT id,kind,tag_id,domain_id,match_text,from_date::text,to_date::text,active FROM project_rules WHERE owner_id=$1 AND project_id=$2 AND active ORDER BY id`, owner, id)
		if e != nil {
			projectHTTPError(w, e)
			return
		}
		defer rows.Close()
		out := []projectRule{}
		for rows.Next() {
			var rule projectRule
			if e = rows.Scan(&rule.ID, &rule.Kind, &rule.TagID, &rule.DomainID, &rule.Match, &rule.From, &rule.To, &rule.Active); e != nil {
				projectHTTPError(w, e)
				return
			}
			out = append(out, rule)
		}
		if e = rows.Err(); e != nil {
			projectHTTPError(w, e)
			return
		}
		writeJSON(w, 200, out)
		return
	}
	var rule projectRule
	if r.Method == http.MethodPost {
		if e = projectDecode(w, r, &rule); e == nil {
			e = rule.validate()
		}
		if e != nil {
			projectHTTPError(w, e)
			return
		}
	}
	tx, e := s.DB.BeginTx(r.Context(), nil)
	if e != nil {
		projectHTTPError(w, e)
		return
	}
	defer tx.Rollback()
	if e = projectLock(r.Context(), tx, owner, id); e != nil {
		projectHTTPError(w, e)
		return
	}
	if r.Method == http.MethodDelete {
		rid, err := strconv.ParseInt(r.PathValue("rule_id"), 10, 64)
		if err != nil || rid <= 0 {
			projectHTTPError(w, projectBad("règle invalide"))
			return
		}
		var active bool
		if err := tx.QueryRowContext(r.Context(), `SELECT active FROM project_rules WHERE owner_id=$1 AND project_id=$2 AND id=$3`, owner, id, rid).Scan(&active); err != nil {
			projectHTTPError(w, err)
			return
		}
		if !active {
			writeJSON(w, 200, map[string]any{"id": rid, "active": false})
			return
		}
		result, err := tx.ExecContext(r.Context(), `UPDATE project_rules SET active=false WHERE owner_id=$1 AND project_id=$2 AND id=$3 AND active`, owner, id, rid)
		if err != nil {
			projectHTTPError(w, err)
			return
		}
		n, _ := result.RowsAffected()
		if n == 0 {
			projectHTTPError(w, sql.ErrNoRows)
			return
		}
		rule.ID = rid
		rule.Active = false
	} else {
		var valid bool
		if rule.TagID != nil {
			e = tx.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM tags WHERE owner_id=$1 AND id=$2)`, owner, *rule.TagID).Scan(&valid)
		} else if rule.DomainID != nil {
			e = tx.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM domains WHERE owner_id=$1 AND id=$2)`, owner, *rule.DomainID).Scan(&valid)
		} else {
			valid = true
		}
		if e != nil {
			projectHTTPError(w, e)
			return
		}
		if !valid {
			projectHTTPError(w, projectBad("critère de règle introuvable"))
			return
		}
		e = tx.QueryRowContext(r.Context(), `SELECT id FROM project_rules WHERE owner_id=$1 AND project_id=$2 AND kind=$3 AND tag_id IS NOT DISTINCT FROM $4 AND domain_id IS NOT DISTINCT FROM $5 AND match_text=$6 AND from_date=$7::date AND to_date=$8::date AND active`, owner, id, rule.Kind, rule.TagID, rule.DomainID, rule.Match, rule.From, rule.To).Scan(&rule.ID)
		if e == nil {
			rule.Active = true
			writeJSON(w, 200, rule)
			return
		}
		if e != sql.ErrNoRows {
			projectHTTPError(w, e)
			return
		}
		var n int
		e = tx.QueryRowContext(r.Context(), `SELECT count(*) FROM project_rules WHERE owner_id=$1 AND project_id=$2 AND active`, owner, id).Scan(&n)
		if e != nil {
			projectHTTPError(w, e)
			return
		}
		if n >= 50 {
			projectHTTPError(w, projectBad("limite de 50 règles actives par projet"))
			return
		}
		e = tx.QueryRowContext(r.Context(), `INSERT INTO project_rules(owner_id,project_id,kind,tag_id,domain_id,match_text,from_date,to_date) VALUES($1,$2,$3,$4,$5,$6,$7::date,$8::date) ON CONFLICT DO NOTHING RETURNING id`, owner, id, rule.Kind, rule.TagID, rule.DomainID, rule.Match, rule.From, rule.To).Scan(&rule.ID)
		if e == sql.ErrNoRows {
			e = tx.QueryRowContext(r.Context(), `SELECT id FROM project_rules WHERE owner_id=$1 AND project_id=$2 AND kind=$3 AND tag_id IS NOT DISTINCT FROM $4 AND domain_id IS NOT DISTINCT FROM $5 AND match_text=$6 AND from_date=$7::date AND to_date=$8::date AND active`, owner, id, rule.Kind, rule.TagID, rule.DomainID, rule.Match, rule.From, rule.To).Scan(&rule.ID)
		}
		if e != nil {
			projectHTTPError(w, e)
			return
		}
		rule.Active = true
	}
	if e = projectHistory(r, tx, owner, id, nil, "rule_saved", nil, rule); e == nil {
		e = tx.Commit()
	}
	if e != nil {
		projectHTTPError(w, e)
		return
	}
	writeJSON(w, 200, rule)
}

// Only persisted transactions participate. Rejected or confirmed decisions are
// never overwritten, including after import replay or a new matching rule.
func (s *Server) refreshProjectSuggestions(ctx context.Context, owner, project int64) (int64, error) {
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return 0, e
	}
	defer tx.Rollback()
	// Serialize with project edits and decisions, in stable order for worker replicas.
	rows, e := tx.QueryContext(ctx, `SELECT id FROM financial_projects WHERE owner_id=$1 AND status='active' AND ($2::bigint=0 OR id=$2) ORDER BY id FOR UPDATE`, owner, project)
	if e != nil {
		return 0, e
	}
	for rows.Next() {
		var id int64
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return 0, e
		}
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return 0, e
	}
	result, e := tx.ExecContext(ctx, `INSERT INTO project_transactions(owner_id,project_id,transaction_id,state,reason)
 SELECT DISTINCT ON(p.id,t.id) p.owner_id,p.id,t.id,'suggested',jsonb_build_object('kind',r.kind,'rule_id',r.id,'tag_id',r.tag_id,'tag_name',(SELECT name FROM tags WHERE id=r.tag_id AND owner_id=r.owner_id),'domain_id',r.domain_id,'domain_name',(SELECT name FROM domains WHERE id=r.domain_id AND owner_id=r.owner_id),'match_text',r.match_text,'from_date',r.from_date,'to_date',r.to_date)
 FROM financial_projects p JOIN project_rules r ON r.project_id=p.id AND r.owner_id=p.owner_id AND r.active
 JOIN transactions t ON t.owner_id=p.owner_id AND t.tx_date BETWEEN r.from_date AND r.to_date
 WHERE p.owner_id=$1 AND ($2::bigint=0 OR p.id=$2) AND p.status='active' AND t.tx_date<=CURRENT_DATE
 AND NOT EXISTS(SELECT 1 FROM project_transactions decided WHERE decided.project_id=p.id AND decided.transaction_id=t.id) AND t.amount<0 AND `+financialExpenseSQLT+` AND (p.start_date IS NULL OR t.tx_date>=p.start_date) AND (p.end_date IS NULL OR t.tx_date<=p.end_date)
 AND ((r.kind='tag' AND EXISTS(SELECT 1 FROM transaction_tags tt JOIN tags tg ON tg.id=tt.tag_id AND tg.owner_id=t.owner_id WHERE tt.transaction_id=t.id AND tg.id=r.tag_id))
 OR (r.kind='category' AND t.domain_id=r.domain_id) OR (r.kind='merchant' AND lower(t.merchant)=lower(r.match_text)) OR (r.kind='label' AND strpos(lower(t.label),lower(r.match_text))>0))
 ORDER BY p.id,t.id,r.id ON CONFLICT(project_id,transaction_id) DO NOTHING`, owner, project)
	if e != nil {
		return 0, e
	}
	n, e := result.RowsAffected()
	if e != nil {
		return 0, e
	}
	if e = tx.Commit(); e != nil {
		return 0, e
	}
	return n, nil
}
func (s *Server) suggestProject(w http.ResponseWriter, r *http.Request) {
	id, e := projectID(r)
	if e != nil {
		projectHTTPError(w, e)
		return
	}
	var status string
	e = s.DB.QueryRowContext(r.Context(), `SELECT status FROM financial_projects WHERE owner_id=$1 AND id=$2`, s.ownerID(r), id).Scan(&status)
	if e != nil {
		projectHTTPError(w, e)
		return
	}
	if status != "active" {
		projectHTTPError(w, &projectError{409, "seuls les projets actifs recherchent de nouvelles suggestions"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	n, e := s.refreshProjectSuggestions(ctx, s.ownerID(r), id)
	if e != nil {
		projectHTTPError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"inserted": n})
}
func (s *Server) decideProjectTransactions(w http.ResponseWriter, r *http.Request) {
	id, e := projectID(r)
	if e != nil {
		projectHTTPError(w, e)
		return
	}
	owner := s.ownerID(r)
	var in struct {
		IDs   []int64 `json:"transaction_ids"`
		State string  `json:"state"`
	}
	if e = projectDecode(w, r, &in); e != nil {
		projectHTTPError(w, e)
		return
	}
	if len(in.IDs) == 0 || len(in.IDs) > 100 || (in.State != "confirmed" && in.State != "rejected") {
		projectHTTPError(w, projectBad("1 à 100 transactions et un état confirmed ou rejected requis"))
		return
	}
	tx, e := s.DB.BeginTx(r.Context(), nil)
	if e != nil {
		projectHTTPError(w, e)
		return
	}
	defer tx.Rollback()
	if e = projectLock(r.Context(), tx, owner, id); e != nil {
		projectHTTPError(w, e)
		return
	}
	seen := map[int64]bool{}
	for _, tid := range in.IDs {
		if tid <= 0 {
			projectHTTPError(w, projectBad("transaction invalide"))
			return
		}
		if seen[tid] {
			continue
		}
		seen[tid] = true
		var eligible bool
		var sourceAmount, sourceDate, sourceLabel string
		e = tx.QueryRowContext(r.Context(), `SELECT `+projectEligible+`,t.amount::text,t.tx_date::text,t.label FROM transactions t CROSS JOIN financial_projects p WHERE t.id=$4 AND t.owner_id=$1 AND p.id=$2 AND p.owner_id=$1 FOR SHARE OF t`, owner, id, time.Now().Format("2006-01-02"), tid).Scan(&eligible, &sourceAmount, &sourceDate, &sourceLabel)
		if e != nil {
			projectHTTPError(w, e)
			return
		}
		if in.State == "confirmed" && !eligible {
			projectHTTPError(w, projectBad("transaction hors période, crédit, transfert ou épargne : confirmation impossible"))
			return
		}
		var before string
		var reason json.RawMessage
		e = tx.QueryRowContext(r.Context(), `SELECT state,reason FROM project_transactions WHERE owner_id=$1 AND project_id=$2 AND transaction_id=$3`, owner, id, tid).Scan(&before, &reason)
		if e != nil && e != sql.ErrNoRows {
			projectHTTPError(w, e)
			return
		}
		if before == in.State {
			continue
		}
		if in.State == "rejected" && before == "" {
			projectHTTPError(w, sql.ErrNoRows)
			return
		}
		if len(reason) == 0 {
			reason = json.RawMessage(`{"kind":"manual"}`)
		}
		_, e = tx.ExecContext(r.Context(), `INSERT INTO project_transactions(owner_id,project_id,transaction_id,state,reason) VALUES($1,$2,$3,$4,$5) ON CONFLICT(project_id,transaction_id) DO UPDATE SET state=EXCLUDED.state,updated_at=now()`, owner, id, tid, in.State, string(reason))
		if e == nil {
			e = projectHistory(r, tx, owner, id, tid, "decision", before, map[string]any{"state": in.State, "transaction_id": tid, "amount": sourceAmount, "date": sourceDate, "label": sourceLabel})
		}
		if e != nil {
			projectHTTPError(w, e)
			return
		}
	}
	if e = tx.Commit(); e != nil {
		projectHTTPError(w, e)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}
