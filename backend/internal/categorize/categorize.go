package categorize

import (
	"database/sql"
	"regexp"
	"strings"
)

type rule struct {
	pattern    string
	re         *regexp.Regexp
	domainID int64
}

type Engine struct{ rules []rule }

// Load charge les règles depuis la base. Les motifs préfixés "re:" sont des regex,
// les autres des sous-chaînes insensibles à la casse.
func Load(db *sql.DB, ownerID int64) (*Engine, error) {
	rows, err := db.Query(`SELECT pattern, domain_id FROM rules WHERE owner_id = $1 ORDER BY length(pattern) DESC`, ownerID)
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
		r := rule{pattern: strings.ToUpper(p), domainID: cid}
		if after, ok := strings.CutPrefix(p, "re:"); ok {
			if re, err := regexp.Compile("(?i)" + after); err == nil {
				r.re = re
			}
		}
		e.rules = append(e.rules, r)
	}
	return e, rows.Err()
}

// Match retourne le domaine du premier motif correspondant (les plus longs d'abord).
func (e *Engine) Match(label string) (int64, bool) {
	u := strings.ToUpper(label)
	for _, r := range e.rules {
		if r.re != nil {
			if r.re.MatchString(label) {
				return r.domainID, true
			}
		} else if strings.Contains(u, r.pattern) {
			return r.domainID, true
		}
	}
	return 0, false
}
