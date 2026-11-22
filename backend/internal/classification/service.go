// Package classification owns the transaction boundary for human decisions,
// replay and proposal contexts. PostgreSQL primitives enforce revisions and
// atomic history for every domain/tag writer, including imports and dictionaries.
package classification

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/lib/pq"
	"themis/internal/categorize"
)

var ErrConflict = errors.New("la classification a changé ; recharge cette opération avant d'enregistrer")
var ErrNotFound = errors.New("opération introuvable")

type Reader interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}
type State struct {
	Domain       *int64   `json:"domain_id"`
	Source       string   `json:"source"`
	Tags         []string `json:"tags"`
	TagIDs       []int64  `json:"tag_ids"`
	Revision     int64    `json:"revision,string"`
	Generation   int64    `json:"context_generation"`
	DomainLocked bool     `json:"domain_locked"`
	TagsLocked   bool     `json:"tags_locked"`
	Configured   bool     `json:"configured"`
}

func Read(ctx context.Context, q Reader, owner, id int64) (State, error) {
	var s State
	e := q.QueryRowContext(ctx, `SELECT t.domain_id,t.classification_source,
 COALESCE((SELECT array_agg(g.name ORDER BY g.name) FROM transaction_tags tt JOIN tags g ON g.id=tt.tag_id AND g.owner_id=t.owner_id WHERE tt.transaction_id=t.id),'{}'),
 COALESCE((SELECT array_agg(g.id ORDER BY g.id) FROM transaction_tags tt JOIN tags g ON g.id=tt.tag_id AND g.owner_id=t.owner_id WHERE tt.transaction_id=t.id),'{}'),
 COALESCE(c.revision,0),COALESCE(c.context_generation,0),COALESCE(c.domain_locked,false),COALESCE(c.tags_locked,false),c.transaction_id IS NOT NULL
 FROM transactions t LEFT JOIN classification_state c ON c.owner_id=t.owner_id AND c.transaction_id=t.id WHERE t.owner_id=$1 AND t.id=$2`, owner, id).Scan(&s.Domain, &s.Source, pq.Array(&s.Tags), pq.Array(&s.TagIDs), &s.Revision, &s.Generation, &s.DomainLocked, &s.TagsLocked, &s.Configured)
	if e == sql.ErrNoRows {
		return s, ErrNotFound
	}
	return s, e
}
func Hash(v any) string {
	data, _ := json.Marshal(v)
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
func LegacyVersion(owner, id int64, domain *int64, source string, tags []string) string {
	n := categorize.NormalizeTags(tags)
	sort.Strings(n)
	return Hash(struct {
		Owner, ID int64
		Domain    *int64
		Source    string
		Tags      []string
	}{owner, id, domain, source, n})
}
func Version(owner, id int64, s State) string {
	return Hash(struct {
		State    string
		Revision int64
	}{LegacyVersion(owner, id, s.Domain, s.Source, s.Tags), s.Revision})
}
func OperationID() string {
	var b [16]byte
	if _, e := rand.Read(b[:]); e != nil {
		panic("secure operation identity unavailable")
	}
	return hex.EncodeToString(b[:])
}
func DatabaseError(e error) error {
	var pe *pq.Error
	if errors.As(e, &pe) {
		if pe.Code == "40001" || pe.Code == "40P01" {
			return ErrConflict
		}
		if pe.Code == "P0002" {
			return ErrNotFound
		}
	}
	return e
}

// Replay is resolved before the version check, so retrying a committed request
// does not apply it twice even though its original expected revision is old.
func Begin(ctx context.Context, tx *sql.Tx, owner, id int64, origin, operation string, payload any, expected *int64, version string) (bool, error) {
	if operation == "" {
		operation = OperationID()
	}
	if len(operation) > 100 || strings.TrimSpace(operation) == "" {
		return false, fmt.Errorf("identifiant d'opération invalide")
	}
	var replay bool
	e := tx.QueryRowContext(ctx, `SELECT classification_begin($1,$2,$3,$4,$5,$6)`, owner, id, origin, operation, Hash(payload), expected).Scan(&replay)
	if e != nil {
		return false, DatabaseError(e)
	}
	if replay {
		return true, nil
	}
	if version != "" {
		s, e := Read(ctx, tx, owner, id)
		if e != nil {
			return false, e
		}
		if version != Version(owner, id, s) && !(s.Revision == 0 && !s.Configured && version == LegacyVersion(owner, id, s.Domain, s.Source, s.Tags)) {
			return false, ErrConflict
		}
	}
	return false, nil
}

// Automatic gives all effective changes in an import/rule batch one operation
// identity. Existing deduplication and effective-delta guards provide replay safety.
func Automatic(ctx context.Context, tx *sql.Tx, origin string) error {
	if origin != "import" && origin != "rule" {
		return fmt.Errorf("origine automatique invalide")
	}
	_, e := tx.ExecContext(ctx, `SELECT set_config('themis.classification_origin',$1,true),set_config('themis.classification_operation',$2,true),set_config('themis.classification_actor','',true),set_config('themis.classification_rule','',true),set_config('themis.classification_suggestion','',true)`, origin, OperationID())
	return e
}

// Rule identifies the actual winning rule in subsequent journal entries.
func Rule(ctx context.Context, tx *sql.Tx, id int64) error {
	value := ""
	if id > 0 {
		value = fmt.Sprint(id)
	}
	_, e := tx.ExecContext(ctx, `SELECT set_config('themis.classification_rule',$1,true)`, value)
	return e
}
func Finish(ctx context.Context, tx *sql.Tx, owner, id int64) error {
	_, e := tx.ExecContext(ctx, `SELECT classification_finish($1,$2)`, owner, id)
	return DatabaseError(e)
}
func Locks(ctx context.Context, tx *sql.Tx, owner, id int64, domain, tags *bool) error {
	_, e := tx.ExecContext(ctx, `SELECT classification_set_locks($1,$2,$3,$4)`, owner, id, domain, tags)
	return DatabaseError(e)
}

func Context(ctx context.Context, q Reader, owner, id int64) (State, string, error) {
	s, e := Read(ctx, q, owner, id)
	if e != nil {
		return s, "", e
	}
	var label string
	if e = q.QueryRowContext(ctx, `SELECT label FROM transactions WHERE owner_id=$1 AND id=$2`, owner, id).Scan(&label); e != nil {
		return s, "", e
	}
	engine, e := categorize.LoadFrom(ctx, q, owner)
	if e != nil {
		return s, "", e
	}
	type dependency struct {
		ID, Version, Domain int64
		Pattern             string
		Tags                []string
	}
	deps := []dependency{}
	ids := []int64{}
	matched := engine.Evaluate(label).Matching
	for _, r := range matched {
		ids = append(ids, r.ID)
	}
	versions := map[int64]int64{}
	if len(ids) > 0 {
		rows, e := q.QueryContext(ctx, `SELECT rule_id,revision FROM classification_rule_versions WHERE owner_id=$1 AND rule_id=ANY($2)`, owner, pq.Array(ids))
		if e != nil {
			return s, "", e
		}
		for rows.Next() {
			var rid, rev int64
			if e = rows.Scan(&rid, &rev); e != nil {
				rows.Close()
				return s, "", e
			}
			versions[rid] = rev
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return s, "", e
		}
	}
	for _, r := range matched {
		tags := categorize.NormalizeTags(r.Tags)
		sort.Strings(tags)
		pattern := r.Pattern
		if !strings.HasPrefix(pattern, "re:") {
			pattern = strings.ToUpper(pattern)
		}
		deps = append(deps, dependency{r.ID, versions[r.ID], r.DomainID, pattern, tags})
	}
	return s, Hash(struct {
		Policy       int
		Generation   int64
		Dependencies []dependency
	}{1, s.Generation, deps}), nil
}
func Reevaluate(ctx context.Context, tx *sql.Tx, owner, id int64) error {
	_, e := tx.ExecContext(ctx, `SELECT classification_record($1,$2,'reevaluation',classification_snapshot($1,$2),'explicit_reevaluation',NULL,true)`, owner, id)
	if e != nil {
		return DatabaseError(e)
	}
	_, e = tx.ExecContext(ctx, `DELETE FROM transaction_suggestions WHERE owner_id=$1 AND transaction_id=$2`, owner, id)
	return e
}
func Reject(ctx context.Context, tx *sql.Tx, owner, id int64, identity, key string) error {
	s, e := Read(ctx, tx, owner, id)
	if e != nil {
		return e
	}
	r, e := tx.ExecContext(ctx, `INSERT INTO classification_rejections(owner_id,transaction_id,identity,context_key,revision,origin) VALUES($1,$2,$3,$4,$5,'manual') ON CONFLICT DO NOTHING`, owner, id, identity, key, s.Revision+1)
	if e != nil {
		return e
	}
	n, e := r.RowsAffected()
	if e != nil {
		return e
	}
	if n > 0 {
		if _, e = tx.ExecContext(ctx, `SELECT set_config('themis.classification_suggestion',$1,true)`, identity); e != nil {
			return e
		}
		_, e = tx.ExecContext(ctx, `SELECT classification_record($1,$2,'suggestion_rejected',classification_snapshot($1,$2),'explicit_suggestion_rejection',NULL,false)`, owner, id)
	}
	if e != nil {
		return e
	}
	_, e = tx.ExecContext(ctx, `DELETE FROM transaction_suggestions WHERE transaction_id=$1 AND owner_id=$2`, id, owner)
	return e
}
