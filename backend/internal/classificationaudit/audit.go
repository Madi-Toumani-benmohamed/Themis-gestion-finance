// Package classificationaudit measures existing assignments without applying
// rules. Banking text stays in process memory and is never part of the report.
package classificationaudit

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"themis/internal/categorize"
	"themis/internal/money"
)

const MaxRows = 50000
const MaxMatches = 5000000

type Domain struct {
	ID, Parent int64
	Name       string
	Transfer   bool
}
type Tag struct {
	ID   int64
	Name string
}
type Rule struct {
	ID, Domain    int64
	Pattern, Tags string
}
type Transaction struct {
	ID, Domain                            int64
	Label, Merchant, Kind, Source, Amount string
	Tags                                  []int64
}
type Dataset struct {
	Domains      []Domain
	Tags         []Tag
	Rules        []Rule
	Transactions []Transaction
}
type Flow struct {
	DebitCents  string `json:"debit_cents"`
	CreditCents string `json:"credit_cents"`
	NetCents    string `json:"net_cents"`
}
type DomainMetric struct {
	Ref          string `json:"ref"`
	Parent       string `json:"parent,omitempty"`
	Transfer     bool   `json:"is_transfer"`
	Transactions int    `json:"transactions"`
	Flow         Flow   `json:"bank_flow"`
}
type TagMetric struct {
	Ref                         string         `json:"ref"`
	Transactions                int            `json:"transactions"`
	Domains                     map[string]int `json:"domains"`
	SingleMerchant              bool           `json:"single_nonempty_merchant"`
	UnknownMerchantTransactions int            `json:"unknown_merchant_transactions"`
	DomainNameDuplicate         bool           `json:"domain_name_candidate"`
	StructuredCandidate         bool           `json:"structured_attribute_candidate"`
}
type RuleMetric struct {
	Ref                string `json:"ref"`
	Domain             string `json:"domain,omitempty"`
	Priority           int    `json:"priority"`
	Executable         bool   `json:"executable"`
	InvalidPattern     bool   `json:"invalid_pattern"`
	LiteralSQLWildcard bool   `json:"literal_sql_wildcard"`
	Regex              bool   `json:"regex_skipped_by_legacy_coherence"`
	Matches            int    `json:"matches"`
	FirstMatches       int    `json:"first_matches"`
}
type Pair struct {
	A             string `json:"a"`
	B             string `json:"b"`
	Count         int    `json:"transactions"`
	Contradictory bool   `json:"different_domains,omitempty"`
	Systematic    bool   `json:"always_together,omitempty"`
}
type Report struct {
	Version                                                                     int            `json:"version"`
	Scope                                                                       string         `json:"scope"`
	Transactions                                                                int            `json:"transactions"`
	Domains                                                                     []DomainMetric `json:"domains"`
	Tags                                                                        []TagMetric    `json:"tags"`
	Rules                                                                       []RuleMetric   `json:"rules"`
	RulePairs                                                                   []Pair         `json:"observed_rule_overlaps"`
	TagPairs                                                                    []Pair         `json:"observed_tag_pairs"`
	SourceCounts                                                                map[string]int `json:"classification_sources"`
	BankFlow                                                                    Flow           `json:"bank_flow"`
	UnassignedFlow                                                              Flow           `json:"unassigned_bank_flow"`
	InvalidDomainFlow                                                           Flow           `json:"invalid_domain_bank_flow"`
	DomainTotal, DomainUsed, DomainParents, DomainChildren                      int
	TagTotal, TagUsed                                                           int
	WithoutDomain, InvalidDomainLinks, InvalidTagLinks                          int
	ConflictingTransactions, ManualDisagreements, ManualUnassignedMatches       int
	LosingRuleAddsTags, RuleTagsRepeated, TagLexicalGroups, DomainLexicalGroups int
	SavingsWithoutTransferFlag, TransferFlagWithoutSavings                      int
}
type sums struct{ debit, credit int64 }

func (s sums) flow() Flow {
	return Flow{fmt.Sprint(s.debit), fmt.Sprint(s.credit), fmt.Sprint(s.credit - s.debit)}
}
func key(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
func Analyze(data Dataset) (Report, error) {
	r := Report{Version: 1, Scope: "one owner; all imported dates; observed matches only; local sensitive aggregates", Domains: []DomainMetric{}, Tags: []TagMetric{}, Rules: []RuleMetric{}, RulePairs: []Pair{}, TagPairs: []Pair{}, SourceCounts: map[string]int{}}
	if len(data.Domains)+len(data.Tags)+len(data.Rules)+len(data.Transactions) > MaxRows {
		return r, fmt.Errorf("audit size limit")
	}
	if len(data.Rules) > 0 && len(data.Transactions) > MaxMatches/len(data.Rules) {
		return r, fmt.Errorf("audit work limit")
	}
	// Copy before sorting: this tool must not mutate even its caller's input.
	domains := append([]Domain{}, data.Domains...)
	tags := append([]Tag{}, data.Tags...)
	rules := append([]Rule{}, data.Rules...)
	sort.Slice(domains, func(i, j int) bool { return domains[i].ID < domains[j].ID })
	sort.Slice(tags, func(i, j int) bool { return tags[i].ID < tags[j].ID })
	sort.Slice(rules, func(i, j int) bool {
		a, b := utf8.RuneCountInString(rules[i].Pattern), utf8.RuneCountInString(rules[j].Pattern)
		if a != b {
			return a > b
		}
		return rules[i].ID < rules[j].ID
	})
	di := map[int64]int{}
	ti := map[int64]int{}
	dn := map[string]int{}
	tn := map[string]int{}
	flows := map[int64]sums{}
	for i, d := range domains {
		di[d.ID] = i
		dn[key(d.Name)]++
		r.Domains = append(r.Domains, DomainMetric{Ref: fmt.Sprintf("domain-%d", i+1), Transfer: d.Transfer, Flow: sums{}.flow()})
	}
	dref := func(id int64) string {
		if i, ok := di[id]; ok {
			return r.Domains[i].Ref
		}
		return "unassigned-or-invalid"
	}
	for i, d := range domains {
		if d.Parent != 0 {
			r.DomainChildren++
			r.Domains[i].Parent = dref(d.Parent)
		}
	}
	parents := map[int64]bool{}
	for _, d := range domains {
		if _, ok := di[d.Parent]; ok {
			parents[d.Parent] = true
		}
	}
	r.DomainParents = len(parents)
	merchants := map[int64]map[string]bool{}
	for i, t := range tags {
		ti[t.ID] = i
		tn[key(t.Name)]++
		merchants[t.ID] = map[string]bool{}
		r.Tags = append(r.Tags, TagMetric{Ref: fmt.Sprintf("tag-%d", i+1), Domains: map[string]int{}, DomainNameDuplicate: dn[key(t.Name)] > 0, StructuredCandidate: structuredTag(t.Name)})
	}
	for _, n := range tn {
		if n > 1 {
			r.TagLexicalGroups++
		}
	}
	for _, n := range dn {
		if n > 1 {
			r.DomainLexicalGroups++
		}
	}
	matchers := make([]func(string) bool, len(rules))
	repeated := map[string]int{}
	for i, ru := range rules {
		if len(ru.Pattern) > 16384 || len(ru.Tags) > 16384 {
			return r, fmt.Errorf("audit field limit")
		}
		m, e := categorize.CompilePattern(ru.Pattern)
		_, valid := di[ru.Domain]
		metric := RuleMetric{Ref: fmt.Sprintf("rule-%d", i+1), Domain: dref(ru.Domain), Priority: i + 1, Executable: e == nil && valid, InvalidPattern: e != nil, Regex: strings.HasPrefix(ru.Pattern, "re:")}
		metric.LiteralSQLWildcard = !metric.Regex && strings.ContainsAny(ru.Pattern, "%_")
		if metric.Executable {
			matchers[i] = m
			for tag := range ruleTags(ru.Tags) {
				repeated[tag]++
			}
		}
		r.Rules = append(r.Rules, metric)
	}
	for _, n := range repeated {
		if n > 1 {
			r.RuleTagsRepeated++
		}
	}
	rp := map[[2]int]int{}
	tp := map[[2]int]int{}
	pairWork := 0
	var bank, unassigned, invalid sums
	for _, t := range data.Transactions {
		if len(t.Label) > 16384 || len(t.Merchant) > 16384 || len(t.Tags) > 1000 {
			return r, fmt.Errorf("audit field limit")
		}
		c, e := money.Parse(t.Amount)
		if e != nil {
			return r, fmt.Errorf("invalid monetary data")
		}
		add := func(s *sums) {
			if c < 0 {
				s.debit -= int64(c)
			} else {
				s.credit += int64(c)
			}
		}
		add(&bank)
		r.Transactions++
		source := t.Source
		if source != "unknown" && source != "manual" && source != "rule" && source != "ai" {
			source = "other"
		}
		r.SourceCounts[source]++
		idx, valid := di[t.Domain]
		if t.Domain == 0 {
			r.WithoutDomain++
			add(&unassigned)
		} else if !valid {
			r.InvalidDomainLinks++
			add(&invalid)
		}
		if valid {
			r.Domains[idx].Transactions++
			s := flows[t.Domain]
			if c < 0 {
				s.debit -= int64(c)
			} else {
				s.credit += int64(c)
			}
			flows[t.Domain] = s
		}
		transfer := valid && domains[idx].Transfer
		if valid {
			if p, ok := di[domains[idx].Parent]; ok {
				transfer = transfer || domains[p].Transfer
			}
		}
		if t.Kind == "savings" && !transfer {
			r.SavingsWithoutTransferFlag++
		}
		if t.Kind != "savings" && transfer {
			r.TransferFlagWithoutSavings++
		}
		seen := map[int]bool{}
		present := []int{}
		for _, id := range t.Tags {
			j, ok := ti[id]
			if !ok {
				r.InvalidTagLinks++
				continue
			}
			if seen[j] {
				continue
			}
			seen[j] = true
			present = append(present, j)
			m := &r.Tags[j]
			m.Transactions++
			m.Domains[dref(t.Domain)]++
			if t.Merchant == "" {
				m.UnknownMerchantTransactions++
			} else {
				merchants[id][t.Merchant] = true
			}
		}
		sort.Ints(present)
		for a := 0; a < len(present); a++ {
			for b := a + 1; b < len(present); b++ {
				pairWork++
				if pairWork > MaxMatches {
					return r, fmt.Errorf("audit pair limit")
				}
				tp[[2]int{present[a], present[b]}]++
			}
		}
		matched := []int{}
		domainSet := map[int64]bool{}
		extra := false
		winningTags := map[string]bool{}
		for i, m := range matchers {
			if m == nil || !m(t.Label) {
				continue
			}
			r.Rules[i].Matches++
			matched = append(matched, i)
			domainSet[rules[i].Domain] = true
			if len(matched) == 1 {
				r.Rules[i].FirstMatches++
				winningTags = ruleTags(rules[i].Tags)
			} else {
				for name := range ruleTags(rules[i].Tags) {
					if !winningTags[name] {
						extra = true
					}
				}
			}
		}
		if len(domainSet) > 1 {
			r.ConflictingTransactions++
		}
		if extra {
			r.LosingRuleAddsTags++
		}
		if len(matched) > 0 && t.Source == "manual" {
			if t.Domain == 0 {
				r.ManualUnassignedMatches++
			} else if t.Domain != rules[matched[0]].Domain {
				r.ManualDisagreements++
			}
		}
		for a := 0; a < len(matched); a++ {
			for b := a + 1; b < len(matched); b++ {
				pairWork++
				if pairWork > MaxMatches {
					return r, fmt.Errorf("audit pair limit")
				}
				rp[[2]int{matched[a], matched[b]}]++
			}
		}
	}
	for id, i := range di {
		r.Domains[i].Flow = flows[id].flow()
		if r.Domains[i].Transactions > 0 {
			r.DomainUsed++
		}
	}
	for id, i := range ti {
		m := &r.Tags[i]
		m.SingleMerchant = m.Transactions > 0 && m.UnknownMerchantTransactions == 0 && len(merchants[id]) == 1
		if m.Transactions > 0 {
			r.TagUsed++
		}
	}
	for _, p := range sortedPairs(rp) {
		r.RulePairs = append(r.RulePairs, Pair{A: r.Rules[p[0]].Ref, B: r.Rules[p[1]].Ref, Count: rp[p], Contradictory: rules[p[0]].Domain != rules[p[1]].Domain})
	}
	for _, p := range sortedPairs(tp) {
		n := tp[p]
		r.TagPairs = append(r.TagPairs, Pair{A: r.Tags[p[0]].Ref, B: r.Tags[p[1]].Ref, Count: n, Systematic: n == r.Tags[p[0]].Transactions && n == r.Tags[p[1]].Transactions})
	}
	r.DomainTotal = len(domains)
	r.TagTotal = len(tags)
	r.BankFlow = bank.flow()
	r.UnassignedFlow = unassigned.flow()
	r.InvalidDomainFlow = invalid.flow()
	return r, nil
}
func ruleTags(s string) map[string]bool {
	out := map[string]bool{}
	for _, n := range strings.Split(s, ",") {
		if n = key(n); n != "" {
			out[n] = true
		}
	}
	return out
}
func sortedPairs(m map[[2]int]int) [][2]int {
	out := make([][2]int, 0, len(m))
	for p := range m {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i][0] != out[j][0] {
			return out[i][0] < out[j][0]
		}
		return out[i][1] < out[j][1]
	})
	return out
}
func structuredTag(s string) bool {
	switch key(s) {
	case "revenu", "dépense", "depense", "transfert", "épargne", "epargne", "remboursement", "abonnement", "mensuel", "credit", "débit", "debit", "virement":
		return true
	}
	return false
}
