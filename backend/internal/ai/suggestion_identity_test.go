package ai

import "testing"

func TestRejectedSuggestionIdentityContract(t *testing.T) {
	a, e := SuggestionIdentity(1, 2, 3, []string{" FACTURE ", "commun", "facture"})
	if e != nil {
		t.Fatal(e)
	}
	b, e := SuggestionIdentity(1, 2, 3, []string{"commun", "facture"})
	if e != nil || a != b || len(a) != 64 {
		t.Fatal("tag order/case changes semantic identity", e)
	}
	for _, ids := range [][3]int64{{2, 2, 3}, {1, 3, 3}, {1, 2, 4}} {
		other, e := SuggestionIdentity(ids[0], ids[1], ids[2], []string{"commun", "facture"})
		if e != nil || other == a {
			t.Fatal("scope collision", e)
		}
	}
	other, e := SuggestionIdentity(1, 2, 3, []string{"autre"})
	if e != nil || other == a {
		t.Fatal("tags not included", e)
	}
	if _, e = SuggestionIdentity(0, 2, 3, nil); e == nil {
		t.Fatal("missing owner")
	}
	if _, e = SuggestionIdentity(1, 2, 3, []string{"a", "b", "c", "d", "e"}); e == nil {
		t.Fatal("invalid proposal cardinality")
	}
	// Rerunning a provider/import does not receive a new identity. No claim
	// is made that the current schema stores this identity or rejection state.
	again, _ := SuggestionIdentity(1, 2, 3, []string{"facture", "commun"})
	if again != a {
		t.Fatal("identity not stable")
	}
}
