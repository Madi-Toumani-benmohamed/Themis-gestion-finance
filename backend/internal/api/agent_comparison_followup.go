package api

import (
	"context"
	"database/sql"
	"regexp"
	"strings"
	"time"
)

// Only the server creates this context after an unfiltered comparison query.
// Dates are retained, never cached amounts; each follow-up executes a new read.
type agentComparisonContext struct {
	A         agentPeriod `json:"period_a"`
	B         agentPeriod `json:"period_b"`
	Direction string      `json:"direction"`
	Scope     string      `json:"scope"`
}

const agentComparisonUnfiltered = "unfiltered_months"

var agentComparisonDirectionFollowup = regexp.MustCompile(`^(?:et\s+)?(?:(?:pour|concernant)\s+)?(?:(?:mes|les)\s+)?(revenus|entrees(?: d'argent)?|depenses)(?:\s+(?:sur|pour|dans)\s+(?:les memes periodes|ces deux periodes))?\s*[?.!]*$`)

func agentComparisonReference(c *agentComparisonContext) string {
	if c == nil || c.Scope != agentComparisonUnfiltered {
		return ""
	}
	return strings.Join([]string{c.A.From, c.A.To, c.B.From, c.B.To}, "|")
}
func agentComparisonTurnComplete(ctx context.Context, reader analysisReader, conv int64) (bool, error) {
	var role string
	err := reader.QueryRowContext(ctx, `SELECT role FROM chat_messages WHERE conversation_id=$1 ORDER BY id DESC LIMIT 1`, conv).Scan(&role)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil && role == "assistant", err
}
func agentComparisonFollowup(text string, previous agentContext, now time.Time) (string, bool) {
	c := previous.Comparison
	if c == nil || c.Scope != agentComparisonUnfiltered || (c.Direction != "income" && c.Direction != "expense") || previous.Intent != c.Direction+"_comparison" || previous.Period.From != c.A.From || previous.Period.To != c.A.To {
		return "", false
	}
	// Treat old/corrupt/unknown contexts conservatively. A comparison never
	// silently accepts a future date, another kind of scope or arbitrary range.
	for _, p := range []agentPeriod{c.A, c.B} {
		first, e := time.ParseInLocation("2006-01-02", p.From, now.Location())
		last, f := time.ParseInLocation("2006-01-02", p.To, now.Location())
		if e != nil || f != nil || !agentPeriodValid(p) || first.Day() != 1 || first.Year() != last.Year() || first.Month() != last.Month() || last.After(now) {
			return "", false
		}
	}
	if c.A.From == c.B.From && c.A.To == c.B.To {
		return "", false
	}
	m := agentComparisonDirectionFollowup.FindStringSubmatch(agentTemporalText(agentText(text)))
	if m == nil {
		return "", false
	}
	if m[1] == "depenses" {
		return "expense", true
	}
	return "income", true
}
