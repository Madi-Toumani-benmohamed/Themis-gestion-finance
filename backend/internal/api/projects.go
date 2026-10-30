package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"themis/internal/money"
	"time"
	"unicode/utf8"
)

type projectError struct {
	status  int
	message string
}

func (e *projectError) Error() string { return e.message }
func projectBad(message string) error { return &projectError{400, message} }
func projectHTTPError(w http.ResponseWriter, err error) {
	var e *projectError
	if errors.As(err, &e) {
		writeErr(w, e.status, e)
		return
	}
	if errors.Is(err, sql.ErrNoRows) {
		writeErr(w, 404, fmt.Errorf("projet ou transaction introuvable"))
		return
	}
	writeErr(w, 500, fmt.Errorf("données du projet indisponibles ; aucune modification confirmée"))
}
func projectID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, projectBad("identifiant de projet invalide")
	}
	return id, nil
}
func projectDecode(w http.ResponseWriter, r *http.Request, in any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 65536))
	dec.DisallowUnknownFields()
	if err := dec.Decode(in); err != nil {
		return projectBad("requête invalide")
	}
	var extra any
	if err := dec.Decode(&extra); err == nil || !errors.Is(err, io.EOF) {
		return projectBad("requête invalide")
	}
	return nil
}

type financialProject struct {
	ID          int64        `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Budget      *int64       `json:"budget_cents"`
	Start       string       `json:"start_date"`
	End         string       `json:"end_date"`
	Status      string       `json:"status"`
	Created     string       `json:"created_at"`
	Updated     string       `json:"updated_at"`
	Cost        int64        `json:"cost_cents"`
	Count       int          `json:"transaction_count"`
	Suggestions int          `json:"suggestion_count"`
	Remaining   *int64       `json:"remaining_cents"`
	Overrun     *int64       `json:"overrun_cents"`
	Percent     *json.Number `json:"consumed_pct"`
}

const projectColumns = `p.id,p.name,p.description,p.budget_cents,COALESCE(p.start_date::text,''),COALESCE(p.end_date::text,''),p.status,p.created_at::text,p.updated_at::text`

func scanProject(row interface{ Scan(...any) error }, p *financialProject) error {
	return row.Scan(&p.ID, &p.Name, &p.Description, &p.Budget, &p.Start, &p.End, &p.Status, &p.Created, &p.Updated)
}

type projectInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Budget      *int64 `json:"budget_cents"`
	Start       string `json:"start_date"`
	End         string `json:"end_date"`
	Status      string `json:"status"`
}

func (in *projectInput) validate() error {
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	if utf8.RuneCountInString(in.Name) < 1 || utf8.RuneCountInString(in.Name) > 120 || utf8.RuneCountInString(in.Description) > 2000 {
		return projectBad("nom ou description invalide")
	}
	if in.Budget != nil && (*in.Budget < 0 || *in.Budget > 999999999999) {
		return projectBad("budget invalide : centimes entiers positifs attendus")
	}
	for _, d := range []string{in.Start, in.End} {
		if d != "" {
			t, e := time.Parse("2006-01-02", d)
			if e != nil || t.Format("2006-01-02") != d || t.Year() < 1 {
				return projectBad("date invalide")
			}
		}
	}
	if in.Start != "" && in.End != "" && in.End < in.Start {
		return projectBad("fin antérieure au début")
	}
	if in.Status == "" {
		in.Status = "active"
	}
	if in.Status != "active" && in.Status != "completed" && in.Status != "archived" {
		return projectBad("statut invalide")
	}
	return nil
}
func projectHistory(r *http.Request, tx *sql.Tx, owner, id int64, transaction any, event string, before, after any) error {
	a, e := json.Marshal(after)
	if e != nil {
		return e
	}
	b, e := json.Marshal(before)
	if e != nil {
		return e
	}
	_, e = tx.ExecContext(r.Context(), `INSERT INTO project_history(owner_id,project_id,transaction_id,event,before_state,after_state) VALUES($1,$2,$3,$4,$5,$6)`, owner, id, transaction, event, string(b), string(a))
	return e
}
func (s *Server) saveProject(w http.ResponseWriter, r *http.Request) {
	var in projectInput
	if e := projectDecode(w, r, &in); e != nil {
		projectHTTPError(w, e)
		return
	}
	if e := in.validate(); e != nil {
		projectHTTPError(w, e)
		return
	}
	owner := s.ownerID(r)
	id := int64(0)
	var err error
	if r.Method == http.MethodPut {
		id, err = projectID(r)
		if err != nil {
			projectHTTPError(w, err)
			return
		}
	}
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		projectHTTPError(w, err)
		return
	}
	defer tx.Rollback()
	var before *financialProject
	if id > 0 {
		before = &financialProject{}
		err = scanProject(tx.QueryRowContext(r.Context(), `SELECT `+projectColumns+` FROM financial_projects p WHERE p.id=$1 AND p.owner_id=$2 FOR UPDATE`, id, owner), before)
		if err != nil {
			projectHTTPError(w, err)
			return
		}
	}
	if before != nil {
		sameBudget := before.Budget == nil && in.Budget == nil || before.Budget != nil && in.Budget != nil && *before.Budget == *in.Budget
		if sameBudget && before.Name == in.Name && before.Description == in.Description && before.Start == in.Start && before.End == in.End && before.Status == in.Status {
			writeJSON(w, 200, map[string]any{"id": id})
			return
		}
	}
	if id == 0 {
		err = tx.QueryRowContext(r.Context(), `INSERT INTO financial_projects(owner_id,name,description,budget_cents,start_date,end_date,status) VALUES($1,$2,$3,$4,NULLIF($5,'')::date,NULLIF($6,'')::date,$7) RETURNING id`, owner, in.Name, in.Description, in.Budget, in.Start, in.End, in.Status).Scan(&id)
	} else {
		_, err = tx.ExecContext(r.Context(), `UPDATE financial_projects SET name=$3,description=$4,budget_cents=$5,start_date=NULLIF($6,'')::date,end_date=NULLIF($7,'')::date,status=$8,updated_at=now() WHERE id=$1 AND owner_id=$2`, id, owner, in.Name, in.Description, in.Budget, in.Start, in.End, in.Status)
	}
	if err == nil {
		err = projectHistory(r, tx, owner, id, nil, "project_saved", before, in)
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		projectHTTPError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"id": id})
}

func projectMetrics(p *financialProject) {
	if p.Budget == nil {
		return
	}
	remaining := *p.Budget - p.Cost
	over := int64(0)
	if remaining < 0 {
		over = -remaining
	}
	p.Remaining = &remaining
	p.Overrun = &over
	p.Percent = analysisPercent(money.Cents(p.Cost), money.Cents(*p.Budget))
}

// Same predicate for all views. Decisions are unique, so joins never multiply costs.
const projectEligible = `t.amount<0 AND ` + financialExpenseSQLT + ` AND (p.start_date IS NULL OR t.tx_date>=p.start_date) AND (p.end_date IS NULL OR t.tx_date<=p.end_date) AND t.tx_date<=$3::date`

func projectPage(r *http.Request) (int, error) {
	v := r.URL.Query().Get("offset")
	if v == "" {
		return 0, nil
	}
	n, e := strconv.Atoi(v)
	if e != nil || n < 0 || n > 10000000 {
		return 0, projectBad("pagination invalide")
	}
	return n, nil
}
func (s *Server) projects(w http.ResponseWriter, r *http.Request) {
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
	rows, e := s.DB.QueryContext(r.Context(), `SELECT `+projectColumns+`,COALESCE(-SUM(t.amount) FILTER(WHERE pt.state='confirmed' AND `+projectEligible+`),0)::text,
 count(t.id) FILTER(WHERE pt.state='confirmed' AND `+projectEligible+`),count(t.id) FILTER(WHERE pt.state='suggested' AND `+projectEligible+`)
 FROM financial_projects p LEFT JOIN project_transactions pt ON pt.project_id=p.id AND pt.owner_id=p.owner_id
 LEFT JOIN transactions t ON t.id=pt.transaction_id AND t.owner_id=p.owner_id
 WHERE p.owner_id=$1 AND p.id IN (SELECT id FROM financial_projects WHERE owner_id=$1 ORDER BY id DESC LIMIT $2 OFFSET $4) GROUP BY p.id ORDER BY p.id DESC`, s.ownerID(r), 101, last, offset)
	if e != nil {
		projectHTTPError(w, e)
		return
	}
	defer rows.Close()
	out := []financialProject{}
	for rows.Next() {
		var p financialProject
		var raw string
		e = rows.Scan(&p.ID, &p.Name, &p.Description, &p.Budget, &p.Start, &p.End, &p.Status, &p.Created, &p.Updated, &raw, &p.Count, &p.Suggestions)
		if e != nil {
			projectHTTPError(w, e)
			return
		}
		c, e := parseAnalysisAmount(raw)
		if e != nil {
			projectHTTPError(w, e)
			return
		}
		p.Cost = int64(c)
		projectMetrics(&p)
		out = append(out, p)
	}
	if e = rows.Err(); e != nil {
		projectHTTPError(w, e)
		return
	}
	more := len(out) > 100
	if more {
		out = out[:100]
	}
	writeJSON(w, 200, map[string]any{"projects": out, "query_to": last, "has_more": more, "offset": offset, "additive": false, "coverage": "provisional"})
}
