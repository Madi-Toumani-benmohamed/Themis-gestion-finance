package categorize

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
)

type rule struct {
	Rule
	matches func(string) bool
}

type Engine struct {
	rules   []rule
	invalid []InvalidRule
}

// CompilePattern is shared by imports and explicit rule reapplication. Plain
// patterns are literal substrings, including SQL wildcard characters % and _.
func CompilePattern(pattern string) (func(string) bool, error) {
	if expression, ok := strings.CutPrefix(pattern, "re:"); ok {
		re, err := regexp.Compile("(?i)" + expression)
		if err != nil {
			return nil, fmt.Errorf("expression régulière invalide")
		}
		return re.MatchString, nil
	}
	if strings.TrimSpace(pattern) == "" {
		return nil, fmt.Errorf("motif vide")
	}
	upper := strings.ToUpper(pattern)
	return func(label string) bool { return strings.Contains(strings.ToUpper(label), upper) }, nil
}

// Load charge les règles depuis la base. Les motifs préfixés "re:" sont des regex,
// les autres des sous-chaînes insensibles à la casse.
func Load(db *sql.DB, ownerID int64) (*Engine, error) {
	return LoadFrom(context.Background(), db, ownerID)
}

type Reader interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func LoadFrom(ctx context.Context, db Reader, ownerID int64) (*Engine, error) {
	rows, err := db.QueryContext(ctx, `SELECT r.id,r.pattern,r.domain_id,COALESCE(r.tags,'') FROM rules r JOIN domains d ON d.id=r.domain_id AND d.owner_id=r.owner_id WHERE r.owner_id = $1 ORDER BY length(r.pattern) DESC,r.id`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var specs []Rule
	for rows.Next() {
		var spec Rule
		var tags string
		if err := rows.Scan(&spec.ID, &spec.Pattern, &spec.DomainID, &tags); err != nil {
			return nil, err
		}
		spec.Tags = NormalizeTags(strings.Split(tags, ","))
		specs = append(specs, spec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return New(specs), nil
}

// Match retourne le domaine du premier motif correspondant (les plus longs d'abord).
func (e *Engine) Match(label string) (int64, bool) {
	for _, r := range e.rules {
		if r.matches(label) {
			return r.DomainID, true
		}
	}
	return 0, false
}
