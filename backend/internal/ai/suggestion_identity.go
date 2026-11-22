package ai

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// SuggestionIdentity defines the stable v1 semantic identity used by durable
// rejections. Context eligibility is checked separately; model and attempt time
// never alter this identity. The canonical contract remains unchanged from 1A.
func SuggestionIdentity(owner, transaction, domain int64, tags []string) (string, error) {
	if owner <= 0 || transaction <= 0 || domain <= 0 {
		return "", fmt.Errorf("identité de proposition invalide")
	}
	seen := map[string]bool{}
	canonical := []string{}
	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == "" || len(tag) > 40 {
			return "", fmt.Errorf("tag de proposition invalide")
		}
		if !seen[tag] {
			canonical = append(canonical, tag)
			seen[tag] = true
		}
	}
	if len(canonical) > 4 {
		return "", fmt.Errorf("trop de tags proposés")
	}
	sort.Strings(canonical)
	data, _ := json.Marshal(struct {
		Version                    int
		Owner, Transaction, Domain int64
		Tags                       []string
	}{1, owner, transaction, domain, canonical})
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}
