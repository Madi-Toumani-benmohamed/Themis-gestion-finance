package api

import (
	"sort"
	"themis/internal/categorize"
)

// This is a state precondition, not a persisted revision or an authorization
// token. It cannot detect an ABA change; a durable revision needs sprint 1B.
func classificationVersion(owner, id int64, domain *int64, source string, tags []string) string {
	normalized := categorize.NormalizeTags(tags)
	sort.Strings(normalized)
	return agentHash(struct {
		Owner, ID int64
		Domain    *int64
		Source    string
		Tags      []string
	}{owner, id, domain, source, normalized})
}
