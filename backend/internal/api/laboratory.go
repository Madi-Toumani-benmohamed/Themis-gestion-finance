package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type laboratoryCell struct {
	Input  string `json:"input"`
	Format string `json:"format"`
}
type laboratoryCalculation struct {
	Expression string `json:"expression"`
	Result     string `json:"result"`
	Created    string `json:"created_at"`
}
type laboratoryContent struct {
	Rows             int                       `json:"rows"`
	Columns          int                       `json:"columns"`
	Cells            map[string]laboratoryCell `json:"cells"`
	Expression       string                    `json:"expression"`
	History          []laboratoryCalculation   `json:"history"`
	SourceExpression string                    `json:"source_expression"`
	SourceResult     string                    `json:"source_result"`
	SourceOperands   []string                  `json:"source_operands"`
}
type laboratoryInput struct {
	ClientKey     string            `json:"client_key"`
	Title         string            `json:"title"`
	Type          string            `json:"type"`
	SchemaVersion int               `json:"schema_version"`
	Revision      int64             `json:"revision"`
	Content       laboratoryContent `json:"content"`
}
type laboratoryDocument struct {
	ID            int64           `json:"id"`
	ClientKey     string          `json:"client_key"`
	Title         string          `json:"title"`
	Type          string          `json:"type"`
	Content       json.RawMessage `json:"content,omitempty"`
	SchemaVersion int             `json:"schema_version"`
	Revision      int64           `json:"revision"`
	Created       string          `json:"created_at"`
	Updated       string          `json:"updated_at"`
	Replayed      bool            `json:"replayed"`
}

var laboratoryAddress = regexp.MustCompile(`^([A-Z])([1-9][0-9]{0,2})$`)
var laboratoryUUID = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func validateLaboratory(in *laboratoryInput) ([]byte, error) {
	in.Title = strings.TrimSpace(in.Title)
	if utf8.RuneCountInString(in.Title) < 1 || utf8.RuneCountInString(in.Title) > 120 || !laboratoryUUID.MatchString(in.ClientKey) || in.SchemaVersion != 1 || (in.Type != "scratchpad" && in.Type != "workbook") {
		return nil, errors.New("document, titre ou version de schéma invalide")
	}
	c := in.Content
	if c.Rows < 2 || c.Rows > 100 || c.Columns < 2 || c.Columns > 26 || c.Cells == nil || len(c.Cells) > 1000 || c.History == nil || len(c.History) > 50 || c.SourceOperands == nil || len(c.SourceOperands) > 128 {
		return nil, errors.New("taille de feuille ou historique hors limites")
	}
	for address, cell := range c.Cells {
		a := laboratoryAddress.FindStringSubmatch(address)
		if a == nil {
			return nil, errors.New("adresse de cellule invalide")
		}
		row, _ := strconv.Atoi(a[2])
		if row > c.Rows || int(a[1][0]-'A') >= c.Columns || utf8.RuneCountInString(cell.Input) > 512 || (cell.Format != "number" && cell.Format != "currency" && cell.Format != "percent") {
			return nil, errors.New("cellule invalide ou hors feuille")
		}
	}
	if utf8.RuneCountInString(c.Expression) > 512 || utf8.RuneCountInString(c.SourceExpression) > 512 || utf8.RuneCountInString(c.SourceResult) > 512 {
		return nil, errors.New("expression ou résultat trop long")
	}
	for _, operand := range c.SourceOperands {
		if utf8.RuneCountInString(operand) > 128 {
			return nil, errors.New("opérande trop long")
		}
	}
	for _, h := range c.History {
		_, e := time.Parse(time.RFC3339Nano, h.Created)
		if e != nil || utf8.RuneCountInString(h.Expression) > 512 || utf8.RuneCountInString(h.Result) > 512 {
			return nil, errors.New("historique de calcul invalide")
		}
	}
	data, e := json.Marshal(c)
	if e != nil || len(data) > 200000 {
		return nil, errors.New("document trop volumineux (200 ko maximum)")
	}
	return data, nil
}
func laboratoryDecode(w http.ResponseWriter, r *http.Request, in any) error {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 262144))
	d.DisallowUnknownFields()
	if d.Decode(in) != nil || d.Decode(new(any)) != io.EOF {
		return errors.New("document JSON invalide ou trop volumineux")
	}
	return nil
}
func laboratoryID(r *http.Request) (int64, error) {
	id, e := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if e != nil || id <= 0 || id > 9007199254740991 {
		return 0, errors.New("identifiant de document invalide")
	}
	return id, nil
}
func laboratoryError(w http.ResponseWriter, e error) {
	if errors.Is(e, sql.ErrNoRows) {
		writeErr(w, 404, errors.New("document introuvable"))
		return
	}
	writeErr(w, 500, errors.New("Laboratoire indisponible ; le brouillon courant doit être conservé"))
}

const laboratoryColumns = `id,client_key::text,title,type,content,schema_version,revision,created_at::text,updated_at::text`

func scanLaboratory(row interface{ Scan(...any) error }, out *laboratoryDocument) error {
	return row.Scan(&out.ID, &out.ClientKey, &out.Title, &out.Type, &out.Content, &out.SchemaVersion, &out.Revision, &out.Created, &out.Updated)
}
func (s *Server) laboratoryList(w http.ResponseWriter, r *http.Request) {
	offset := 0
	if raw := r.URL.Query().Get("offset"); raw != "" {
		v, e := strconv.Atoi(raw)
		if e != nil || v < 0 || v > 1000000 {
			writeErr(w, 400, errors.New("pagination invalide"))
			return
		}
		offset = v
	}
	rows, e := s.DB.QueryContext(r.Context(), `SELECT id,client_key::text,title,type,schema_version,revision,created_at::text,updated_at::text FROM laboratory_documents WHERE owner_id=$1 ORDER BY updated_at DESC,id DESC LIMIT 51 OFFSET $2`, s.ownerID(r), offset)
	if e != nil {
		laboratoryError(w, e)
		return
	}
	defer rows.Close()
	out := []laboratoryDocument{}
	for rows.Next() {
		var d laboratoryDocument
		if e = rows.Scan(&d.ID, &d.ClientKey, &d.Title, &d.Type, &d.SchemaVersion, &d.Revision, &d.Created, &d.Updated); e != nil {
			laboratoryError(w, e)
			return
		}
		out = append(out, d)
	}
	if e = rows.Err(); e != nil {
		laboratoryError(w, e)
		return
	}
	more := len(out) > 50
	if more {
		out = out[:50]
	}
	writeJSON(w, 200, map[string]any{"documents": out, "has_more": more})
}
func (s *Server) laboratoryGet(w http.ResponseWriter, r *http.Request) {
	id, e := laboratoryID(r)
	if e != nil {
		writeErr(w, 400, e)
		return
	}
	var out laboratoryDocument
	e = scanLaboratory(s.DB.QueryRowContext(r.Context(), `SELECT `+laboratoryColumns+` FROM laboratory_documents WHERE id=$1 AND owner_id=$2`, id, s.ownerID(r)), &out)
	if e != nil {
		laboratoryError(w, e)
		return
	}
	writeJSON(w, 200, out)
}
func (s *Server) laboratorySave(w http.ResponseWriter, r *http.Request) {
	var in laboratoryInput
	if e := laboratoryDecode(w, r, &in); e != nil {
		writeErr(w, 400, e)
		return
	}
	data, e := validateLaboratory(&in)
	if e != nil {
		writeErr(w, 400, e)
		return
	}
	owner := s.ownerID(r)
	var out laboratoryDocument
	if r.Method == http.MethodPost {
		if in.Revision != 0 {
			writeErr(w, 400, errors.New("un nouveau document commence sans révision serveur"))
			return
		}
		e = scanLaboratory(s.DB.QueryRowContext(r.Context(), `INSERT INTO laboratory_documents(owner_id,client_key,title,type,content,schema_version) VALUES($1,$2::uuid,$3,$4,$5,$6) ON CONFLICT(owner_id,client_key) DO NOTHING RETURNING `+laboratoryColumns, owner, in.ClientKey, in.Title, in.Type, string(data), in.SchemaVersion), &out)
		status := 201
		if e == sql.ErrNoRows {
			e = scanLaboratory(s.DB.QueryRowContext(r.Context(), `SELECT `+laboratoryColumns+` FROM laboratory_documents WHERE owner_id=$1 AND client_key=$2::uuid`, owner, in.ClientKey), &out)
			out.Replayed = true
			status = 200
		}
		if e != nil {
			laboratoryError(w, e)
			return
		}
		writeJSON(w, status, out)
		return
	}
	id, e := laboratoryID(r)
	if e != nil {
		writeErr(w, 400, e)
		return
	}
	if in.Revision < 1 || in.Revision >= 9007199254740991 {
		writeErr(w, 400, errors.New("révision serveur requise"))
		return
	}
	e = scanLaboratory(s.DB.QueryRowContext(r.Context(), `UPDATE laboratory_documents SET title=$4,type=$5,content=$6,schema_version=$7,revision=revision+1,updated_at=now() WHERE id=$1 AND owner_id=$2 AND revision=$3 AND client_key=$8::uuid RETURNING `+laboratoryColumns, id, owner, in.Revision, in.Title, in.Type, string(data), in.SchemaVersion, in.ClientKey), &out)
	if e == sql.ErrNoRows {
		s.laboratoryConflict(w, r, id)
		return
	}
	if e != nil {
		laboratoryError(w, e)
		return
	}
	writeJSON(w, 200, out)
}
func (s *Server) laboratoryConflict(w http.ResponseWriter, r *http.Request, id int64) {
	var exists bool
	e := s.DB.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM laboratory_documents WHERE id=$1 AND owner_id=$2)`, id, s.ownerID(r)).Scan(&exists)
	if e != nil {
		laboratoryError(w, e)
		return
	}
	if !exists {
		laboratoryError(w, sql.ErrNoRows)
		return
	}
	writeErr(w, 409, errors.New("conflit de révision : document modifié dans un autre onglet ; conservez le brouillon et rechargez la version serveur ou enregistrez une copie"))
}
func (s *Server) laboratoryDelete(w http.ResponseWriter, r *http.Request) {
	id, e := laboratoryID(r)
	if e != nil {
		writeErr(w, 400, e)
		return
	}
	var in struct {
		Revision int64 `json:"revision"`
	}
	if e = laboratoryDecode(w, r, &in); e != nil || in.Revision < 1 || in.Revision > 9007199254740991 {
		writeErr(w, 400, fmt.Errorf("révision serveur requise pour supprimer"))
		return
	}
	result, e := s.DB.ExecContext(r.Context(), `DELETE FROM laboratory_documents WHERE id=$1 AND owner_id=$2 AND revision=$3`, id, s.ownerID(r), in.Revision)
	if e != nil {
		laboratoryError(w, e)
		return
	}
	n, e := result.RowsAffected()
	if e != nil {
		laboratoryError(w, e)
		return
	}
	if n == 0 {
		s.laboratoryConflict(w, r, id)
		return
	}
	w.WriteHeader(204)
}
