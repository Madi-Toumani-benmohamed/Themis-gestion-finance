package categorize

import (
	"reflect"
	"testing"
)

func TestSharedEvaluationAndWinningTags(t *testing.T) {
	specs := []Rule{{ID: 2, Pattern: "UBER", DomainID: 2, Tags: []string{"VTC", "commun"}}, {ID: 1, Pattern: "UBER EATS", DomainID: 1, Tags: []string{"Livraison", "commun", "livraison"}}, {ID: 3, Pattern: "re:[", DomainID: 1}}
	e := New(specs)
	p := e.Plan(Input{ID: 8, Label: "uber eats", Source: "unknown", Tags: []string{"manuel", "historique"}})
	if p.Winner == nil || p.Winner.ID != 1 || len(p.Matching) != 2 || len(p.Invalid) != 1 || !p.Conflict || !p.SetDomain || p.TransactionID != 8 {
		t.Fatal(p)
	}
	if !reflect.DeepEqual(p.AddTags, []string{"livraison", "commun"}) || len(specs[0].Tags) != 2 {
		t.Fatal(p.AddTags)
	}
	if d, ok := e.Match("UBER EATS"); !ok || d != *p.ProposedDomain {
		t.Fatal("import diverges from plan")
	}
	// Returned metadata must not mutate the engine or another request.
	p.Winner.Tags[0] = "tampered"
	p.Matching[0].Tags[0] = "tampered"
	if e.Plan(Input{Label: "UBER EATS"}).ProposedTags[0] != "livraison" {
		t.Fatal("mutable engine metadata")
	}
	for _, tags := range [][]string{nil, {"commun"}} {
		q := New([]Rule{{ID: 1, Pattern: "UBER EATS", DomainID: 1, Tags: tags}, {ID: 2, Pattern: "UBER", DomainID: 1, Tags: []string{"vtc", "commun"}}}).Plan(Input{Label: "UBER EATS"})
		if q.Conflict || !reflect.DeepEqual(q.ProposedTags, NormalizeTags(tags)) {
			t.Fatal(q)
		}
	}
}
func TestPriorityRegexUnicodeAndLiteralParity(t *testing.T) {
	for _, specs := range [][]Rule{
		{{ID: 2, Pattern: "éé", DomainID: 2}, {ID: 1, Pattern: "AB", DomainID: 1}},
		{{ID: 1, Pattern: "AB", DomainID: 1}, {ID: 2, Pattern: "éé", DomainID: 2}},
	} {
		if p := New(specs).Plan(Input{Label: "AB ÉÉ"}); p.Winner.ID != 1 {
			t.Fatal(p)
		}
	}
	e := New([]Rule{{ID: 1, Pattern: "A_B%", DomainID: 1}, {ID: 2, Pattern: "re:^PRLV (EDF|FREE)$", DomainID: 2}})
	for _, tc := range []struct {
		label string
		want  int64
	}{{"a_b%", 1}, {"AXXB", 0}, {"prlv edf", 2}, {"CB PRLV EDF", 0}} {
		p := e.Plan(Input{Label: tc.label})
		if tc.want == 0 {
			if p.Winner != nil {
				t.Fatal(p)
			}
		} else if p.Winner == nil || p.Winner.DomainID != tc.want {
			t.Fatal(p)
		}
	}
}
func TestManualPoliciesAndUnrepresentableProtections(t *testing.T) {
	a, b := int64(1), int64(2)
	e := New([]Rule{{ID: 1, Pattern: "TEST", DomainID: 1, Tags: []string{"new", "kept"}}})
	for _, tc := range []struct {
		name, source string
		domain       *int64
		set          bool
		add          int
	}{{"manual different", "manual", &b, false, 0}, {"manual unclassified", "manual", nil, false, 0}, {"old NULL", "unknown", nil, true, 2}, {"automatic existing", "rule", &a, false, 2}, {"manual same domain, tags still eligible", "manual", &a, false, 2}} {
		t.Run(tc.name, func(t *testing.T) {
			p := e.Plan(Input{Label: "TEST", Source: tc.source, DomainID: tc.domain, Tags: []string{"historical", "manual"}})
			if p.SetDomain != tc.set || len(p.AddTags) != tc.add {
				t.Fatal(p)
			}
			if p.LockStateKnown || p.TagProtectionKnown {
				t.Fatal("invented persisted protection")
			}
		})
	}
	// A removed tag and a never-present tag have the same persisted state.
	// Without tombstones/independent locks the engine cannot claim protection.
	removed := e.Plan(Input{Label: "TEST", Source: "unknown", Tags: []string{"kept"}})
	if !reflect.DeepEqual(removed.AddTags, []string{"new"}) || removed.TagProtectionKnown {
		t.Fatal("limit hidden")
	}
}
