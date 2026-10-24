package categorize

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
)

type rule struct {
	matches  func(string) bool
	domainID int64
}

type Engine struct{ rules []rule }

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
	rows, err := db.Query(`SELECT r.pattern, r.domain_id FROM rules r JOIN domains d ON d.id=r.domain_id AND d.owner_id=r.owner_id WHERE r.owner_id = $1 ORDER BY length(r.pattern) DESC,r.id`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	e := &Engine{}
	for rows.Next() {
		var p string
		var cid int64
		if err := rows.Scan(&p, &cid); err != nil {
			return nil, err
		}
		matches, err := CompilePattern(p)
		// Old invalid expressions were inert; do not block unrelated imports.
		// Rule editing/reapplication returns a validation error for correction.
		if err != nil {
			continue
		}
		e.rules = append(e.rules, rule{matches: matches, domainID: cid})
	}
	return e, rows.Err()
}

// Match retourne le domaine du premier motif correspondant (les plus longs d'abord).
func (e *Engine) Match(label string) (int64, bool) {
	for _, r := range e.rules {
		if r.matches(label) {
			return r.domainID, true
		}
	}
	return 0, false
}
