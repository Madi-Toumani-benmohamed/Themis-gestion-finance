package categorize

import "testing"

func TestRulePatternSemantics(t *testing.T) {
	for _, tc := range []struct {
		pattern, label string
		want           bool
	}{
		{"PAYPAL", "CARTE Paypal TEST", true},
		{"A_B%", "CARTE A_B% TEST", true},
		{"A_B%", "CARTE AXXB TEST", false},
		{"re:^PRLV (EDF|FREE)$", "prlv edf", true},
		{"re:^PRLV (EDF|FREE)$", "CB EDF", false},
	} {
		match, err := CompilePattern(tc.pattern)
		if err != nil || match(tc.label) != tc.want {
			t.Fatalf("pattern %q label %q: err=%v", tc.pattern, tc.label, err)
		}
	}
	if _, err := CompilePattern("re:["); err == nil {
		t.Fatal("invalid regex accepted")
	}
}
