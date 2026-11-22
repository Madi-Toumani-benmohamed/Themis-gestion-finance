package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"themis/internal/classification"
)

// renameTag preserves a stable ID and its decisions. Name collisions require
// a separate, explicit merge policy and are refused rather than merged silently.
func (s *Server) renameTag(w http.ResponseWriter, r *http.Request) {
	owner := s.ownerID(r)
	old := strings.TrimSpace(r.PathValue("name"))
	var in struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	neo := strings.TrimSpace(in.Name)
	if old == "" || neo == "" {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("nom courant et nouveau nom requis"))
		return
	}
	if strings.EqualFold(old, neo) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "merged": false})
		return
	}

	tx, err := s.DB.Begin()
	if err != nil {
		classificationFailure(w, err)
		return
	}
	defer tx.Rollback()

	if _, err = tx.ExecContext(r.Context(), `SELECT set_config('themis.classification_origin','dictionary',true),set_config('themis.classification_actor',$1,true),set_config('themis.classification_operation',$2,true)`, fmt.Sprint(owner), classification.OperationID()); err != nil {
		classificationFailure(w, err)
		return
	}
	var oldID int64
	if err := tx.QueryRow(`SELECT id FROM tags WHERE owner_id = $1 AND name = $2 FOR UPDATE`, owner, old).Scan(&oldID); err != nil {
		writeErr(w, http.StatusNotFound, fmt.Errorf("tag introuvable"))
		return
	}

	// Refuse implicit identity changes through a name collision.
	var newID int64
	merged := false
	err = tx.QueryRow(`SELECT id FROM tags WHERE owner_id = $1 AND name = $2`, owner, neo).Scan(&newID)
	if err == nil && newID != oldID {
		writeErr(w, 409, fmt.Errorf("ce nom est déjà utilisé ; la fusion de tags n'est pas disponible avec les protections de classification"))
		return
	} else {
		if err != sql.ErrNoRows {
			classificationFailure(w, err)
			return
		}
		// simple renommage
		if _, err := tx.Exec(`UPDATE tags SET name = $1 WHERE id = $2`, neo, oldID); err != nil {
			classificationFailure(w, err)
			return
		}
	}

	// Mettre à jour les règles qui citent l'ancien nom (rules.tags = CSV de noms).
	if err := s.rewriteRuleTags(tx, owner, old, neo); err != nil {
		classificationFailure(w, err)
		return
	}

	if err := tx.Commit(); err != nil {
		classificationFailure(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "merged": merged})
}

// deleteTag supprime un tag et toutes ses liaisons (les transactions perdent ce
// tag mais ne sont pas supprimées). Le nom est aussi retiré des règles.
func (s *Server) deleteTag(w http.ResponseWriter, r *http.Request) {
	owner := s.ownerID(r)
	name := strings.TrimSpace(r.PathValue("name"))
	if name == "" {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("nom requis"))
		return
	}
	tx, err := s.DB.Begin()
	if err != nil {
		classificationFailure(w, err)
		return
	}
	defer tx.Rollback()

	// Referenced or explicitly protected tags are refused by the shared guard.
	res, err := tx.Exec(`DELETE FROM tags WHERE owner_id = $1 AND name = $2`, owner, name)
	if err != nil {
		classificationFailure(w, err)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeErr(w, http.StatusNotFound, fmt.Errorf("tag introuvable"))
		return
	}
	// retirer le nom des règles
	if err := s.rewriteRuleTags(tx, owner, name, ""); err != nil {
		classificationFailure(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		classificationFailure(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// rewriteRuleTags met à jour la colonne CSV rules.tags : remplace `old` par
// `neo`, ou le retire si `neo` est vide. Dédoublonne au passage.
func (s *Server) rewriteRuleTags(tx dbExec, owner int64, old, neo string) error {
	rows, err := tx.Query(`SELECT id, tags FROM rules WHERE owner_id = $1 AND tags <> ''`, owner)
	if err != nil {
		return err
	}
	type ru struct {
		id   int64
		tags string
	}
	var rules []ru
	for rows.Next() {
		var x ru
		if rows.Scan(&x.id, &x.tags) == nil {
			rules = append(rules, x)
		}
	}
	rows.Close()
	for _, x := range rules {
		parts := strings.Split(x.tags, ",")
		var kept []string
		seen := map[string]bool{}
		changed := false
		for _, p := range parts {
			t := strings.TrimSpace(p)
			if t == "" {
				continue
			}
			if strings.EqualFold(t, old) {
				changed = true
				t = neo // remplacement (ou suppression si neo == "")
			}
			if t == "" {
				continue
			}
			key := strings.ToLower(t)
			if seen[key] {
				continue
			}
			seen[key] = true
			kept = append(kept, t)
		}
		if changed {
			if _, err := tx.Exec(`UPDATE rules SET tags = $1 WHERE id = $2`, strings.Join(kept, ","), x.id); err != nil {
				return err
			}
		}
	}
	return nil
}

// dbExec : sous-ensemble commun à *sql.DB et *sql.Tx utilisé par rewriteRuleTags.
type dbExec interface {
	Query(query string, args ...any) (*sql.Rows, error)
	Exec(query string, args ...any) (sql.Result, error)
}
