package money

import "testing"

func TestExactCents(t *testing.T) {
	a, _ := Parse("0.10")
	b, _ := Parse("0.20")
	if (a + b).String() != "0.30" {
		t.Fatal("addition inexacte")
	}
	for _, s := range []string{"0.001", "NaN", "Inf", "10000000000", "1e3", "--1", "-+1", "+-1", "1."} {
		if _, err := Parse(s); err == nil {
			t.Fatalf("accepté %q", s)
		}
	}
	for _, s := range []string{"0.01", "-1234.56", "9999999999.99"} {
		v, e := Parse(s)
		if e != nil || v.String() != s {
			t.Fatalf("conversion %s: %v", s, e)
		}
	}
}
