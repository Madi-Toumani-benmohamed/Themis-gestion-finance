package api

import "themis/internal/classification"

func classificationVersion(owner, id int64, domain *int64, source string, tags []string, revision ...int64) string {
	var rev int64
	if len(revision) > 0 {
		rev = revision[0]
	}
	return classification.Version(owner, id, classification.State{Domain: domain, Source: source, Tags: tags, Revision: rev})
}
