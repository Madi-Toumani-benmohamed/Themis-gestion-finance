package categorize

import (
	"sort"
	"strings"
	"unicode/utf8"
)

type Rule struct {
	ID       int64    `json:"id"`
	Pattern  string   `json:"pattern"`
	DomainID int64    `json:"domain_id"`
	Tags     []string `json:"tags"`
}
type InvalidRule struct {
	ID     int64  `json:"id"`
	Reason string `json:"reason"`
}
type Evaluation struct {
	Winner   *Rule         `json:"winner"`
	Matching []Rule        `json:"matching"`
	Invalid  []InvalidRule `json:"invalid_rules"`
	Conflict bool          `json:"different_domains"`
}
type Input struct {
	DomainLocked, TagsLocked, ProtectionsKnown bool
	RemovedTags                                []string
	ID                                         int64
	Label                                      string
	DomainID                                   *int64
	Source                                     string
	Tags                                       []string
}
type Plan struct {
	TransactionID int64 `json:"transaction_id,omitempty"`
	Evaluation
	ProposedDomain        *int64   `json:"proposed_domain"`
	ProposedTags          []string `json:"proposed_tags"`
	SetDomain             bool     `json:"set_domain"`
	AddTags               []string `json:"add_tags"`
	ManualDomainProtected bool     `json:"manual_domain_protected"`
	TagProtectionKnown    bool     `json:"tag_protection_known"`
	LockStateKnown        bool     `json:"lock_state_known"`
	Explanation           string   `json:"explanation"`
}

func NormalizeTags(in []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, s := range in {
		s = strings.ToLower(strings.TrimSpace(s))
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
func New(specs []Rule) *Engine {
	ordered := append([]Rule{}, specs...)
	sort.SliceStable(ordered, func(i, j int) bool {
		a, b := utf8.RuneCountInString(ordered[i].Pattern), utf8.RuneCountInString(ordered[j].Pattern)
		if a != b {
			return a > b
		}
		return ordered[i].ID < ordered[j].ID
	})
	e := &Engine{invalid: []InvalidRule{}}
	for _, r := range ordered {
		r.Tags = NormalizeTags(r.Tags)
		match, err := CompilePattern(r.Pattern)
		if err != nil {
			e.invalid = append(e.invalid, InvalidRule{ID: r.ID, Reason: "invalid_pattern"})
			continue
		}
		e.rules = append(e.rules, rule{Rule: r, matches: match})
	}
	return e
}
func (e *Engine) Evaluate(label string) Evaluation {
	v := Evaluation{Matching: []Rule{}, Invalid: append([]InvalidRule{}, e.invalid...)}
	for _, r := range e.rules {
		if !r.matches(label) {
			continue
		}
		spec := r.Rule
		spec.Tags = append([]string{}, spec.Tags...)
		v.Matching = append(v.Matching, spec)
		if v.Winner == nil {
			winner := spec
			v.Winner = &winner
		} else if spec.DomainID != v.Winner.DomainID {
			v.Conflict = true
		}
	}
	return v
}
func (e *Engine) Plan(in Input) Plan {
	p := Plan{TransactionID: in.ID, Evaluation: e.Evaluate(in.Label), ProposedTags: []string{}, AddTags: []string{}, ManualDomainProtected: in.Source == "manual", Explanation: "no_matching_rule"}
	// Known metadata is supplied by the repository; legacy provenance is not inferred.
	p.TagProtectionKnown = in.ProtectionsKnown
	p.LockStateKnown = in.ProtectionsKnown
	if p.Winner == nil {
		return p
	}
	domain := p.Winner.DomainID
	p.ProposedDomain = &domain
	p.ProposedTags = append([]string{}, p.Winner.Tags...)
	p.SetDomain = in.DomainID == nil && in.Source != "manual" && !in.DomainLocked
	p.Explanation = "winning_rule"
	if in.DomainID == nil && in.Source == "manual" {
		p.Explanation = "manual_unclassified_preserved"
		return p
	}
	if in.DomainID != nil && *in.DomainID != domain {
		p.Explanation = "existing_domain_preserved"
		return p
	}
	if in.DomainID != nil && in.Source == "manual" {
		p.Explanation = "manual_domain_preserved_tags_proposed"
	}
	if in.TagsLocked {
		p.Explanation = "tags_locked"
		return p
	}
	if in.DomainLocked {
		p.Explanation = "domain_locked"
	}
	blocked := map[string]bool{}
	for _, tag := range NormalizeTags(in.RemovedTags) {
		blocked[tag] = true
	}
	existing := map[string]bool{}
	for _, tag := range NormalizeTags(in.Tags) {
		existing[tag] = true
	}
	for _, tag := range p.ProposedTags {
		if !existing[tag] && !blocked[tag] {
			p.AddTags = append(p.AddTags, tag)
		}
	}
	return p
}
