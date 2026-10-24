package api

import (
	"fmt"
	"strconv"
	"strings"
)

// Only explicit positive source identifiers; owner isolation is still enforced
// by the main transaction query. The cap matches the existing endpoint limit.
func parseTransactionSourceIDs(raw string) ([]int64, error) {
	parts := strings.Split(raw, ",")
	if len(parts) > 1000 {
		return nil, fmt.Errorf("1000 identifiants sources maximum")
	}
	ids := make([]int64, 0, len(parts))
	seen := map[int64]bool{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("identifiant source invalide")
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return nil, fmt.Errorf("identifiant source invalide")
			}
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("identifiant source invalide")
		}
		if !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	return ids, nil
}
