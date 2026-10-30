package fiscal

import (
	"testing"
	"themis/internal/money"
)

func finalInput(year int, family string, euros int64) Input {
	return Input{IncomeYear: year, Family: family, Basis: "final_taxable", IncomePeriod: "annual", ObservedMonths: 12, Income: money.Cents(euros * 100), Deduction: "standard", ScopeConfirmed: true}
}
func must(t *testing.T, in Input) Result {
	t.Helper()
	r, e := Compute(in)
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func TestOfficialReferences(t *testing.T) {
	// Service-Public F1419 (15 April 2026): single RNI30000, gross2103.99.
	r := must(t, finalInput(2025, "single", 30000))
	if r.Tax != 210400 || r.GrossTax != 210400 || r.MarginalRate != 30 {
		t.Fatalf("F1419: %+v", r)
	}
	// DGFiP brochure2026 p371: married, 2 children, RNI54000 -> gross2112.
	in := finalInput(2025, "married", 54000)
	in.Children = 2
	r = must(t, in)
	if r.HalfParts != 6 || r.GrossTax != 211200 {
		t.Fatalf("brochure p371 %+v", r)
	}
	// CGI barème2025, taxable R30000: 17818*11% + 685*30%=2165.48.
	r = must(t, finalInput(2024, "single", 30000))
	if r.GrossTax != 216500 {
		t.Fatal("2024 rules", r)
	}
}
func TestBracketAndRoundingLimits(t *testing.T) {
	for _, tc := range []struct{ income, tax int64 }{{0, 0}, {11600, 0}, {17595, 0}, {30000, 2104}, {29579, 1976}, {84577, 18477}, {181917, 58386}} {
		r := must(t, finalInput(2025, "single", tc.income))
		if int64(r.Tax) != tc.tax*100 {
			t.Fatalf("income%d got%d want%d", tc.income, r.Tax, tc.tax*100)
		}
	}
	a := finalInput(2025, "single", 30000)
	a.Income += 49
	r1 := must(t, a)
	a.Income++
	r2 := must(t, a)
	if r1.Taxable != 3000000 || r2.Taxable != 3000100 {
		t.Fatal("legal base rounding", r1.Taxable, r2.Taxable)
	}
	if euro(159730) != 1597 || euro(159750) != 1598 || euro(159775) != 1598 {
		t.Fatal("DGFiP legal rounding examples")
	}
}
func TestSalaryHouseholdsAndDeduction(t *testing.T) {
	in := finalInput(2025, "single", 30000)
	in.Basis = "salary"
	r := must(t, in)
	if r.SalaryDeduction != 300000 || r.Taxable != 2700000 || r.Discount != 13000 || r.Tax != 156400 {
		t.Fatalf("salary %+v", r)
	}
	in.Income = 10000
	r = must(t, in)
	if r.SalaryDeduction != 10000 || r.Tax != 0 {
		t.Fatal("minimum cannot exceed income")
	}
	in.Income = 20000000
	r = must(t, in)
	if r.SalaryDeduction != 1455500 {
		t.Fatal("salary deduction cap", r.SalaryDeduction)
	}
	in = finalInput(2025, "married", 30000)
	in.Basis = "salary"
	in.SpouseIncome = 3000000
	r = must(t, in)
	if r.Taxable != 5400000 || r.HalfParts != 4 {
		t.Fatal("separate spouse deduction", r)
	}
	in.Deduction = "actual"
	in.ActualCosts = 400000
	in.SpouseActualCosts = 100000
	r = must(t, in)
	if r.Taxable != 5500000 {
		t.Fatal("actual expenses", r.Taxable)
	}
	in = finalInput(2025, "single", 120000)
	in.Children = 1
	r = must(t, in)
	if !r.FamilyCapApplied || r.HalfParts != 3 {
		t.Fatal("family cap missing")
	}
	base := must(t, finalInput(2025, "single", 120000))
	if base.GrossTax-r.GrossTax != 180700 {
		t.Fatal("half-part cap", base.GrossTax, r.GrossTax)
	}
	in.Isolated = true
	isolated := must(t, in)
	if base.GrossTax-isolated.GrossTax != 426200 || isolated.HalfParts != 4 {
		t.Fatal("isolated cap", isolated)
	}
	in = finalInput(2025, "married", 70000)
	in.Children = 3
	if must(t, in).HalfParts != 8 {
		t.Fatal("third child full part")
	}
}
func TestWithholdingAndProjections(t *testing.T) {
	in := finalInput(2025, "single", 30000)
	r := must(t, in)
	if r.Settlement != nil || r.ObservedPaid != nil {
		t.Fatal("unknown PAS fabricated")
	}
	paid := money.Cents(100000)
	in.WithholdingObserved = &paid
	in.WithholdingYear = 2025
	in.WithholdingMonths = 12
	r = must(t, in)
	if *r.Settlement != 110400 {
		t.Fatal("balance", *r.Settlement)
	}
	paid = 300000
	r = must(t, in)
	if *r.Settlement != -89600 {
		t.Fatal("refund", *r.Settlement)
	}
	in.WithholdingMonths = 6
	r = must(t, in)
	if r.Settlement != nil || r.SettlementStatus != "incomplete" {
		t.Fatal("partial PAS treated as final")
	}
	future := money.Cents(10000)
	in.FutureMonthlyWithholding = &future
	r = must(t, in)
	if *r.ProjectedPaid != 60000 || *r.Settlement != -149600 || *r.ObservedPaid != 300000 {
		t.Fatal("observed and projected mixed", r)
	}
	in.WithholdingYear = 2024
	if _, e := Compute(in); e == nil {
		t.Fatal("mixed income years accepted")
	}
	in = finalInput(2025, "single", 15000)
	in.Basis = "salary"
	in.IncomePeriod = "cumulative"
	in.ObservedMonths = 6
	if _, e := Compute(in); e == nil {
		t.Fatal("incomplete income silently annualized")
	}
	in.ProjectIncome = true
	r = must(t, in)
	if !r.Projection || r.AnnualIncome != 3000000 {
		t.Fatal("explicit projection", r)
	}
	in.IncomePeriod = "monthly"
	in.Income = 250000
	r = must(t, in)
	if r.AnnualIncome != 3000000 {
		t.Fatal("monthly projection")
	}
}
func TestUnsupportedAndInvalid(t *testing.T) {
	tests := []func(*Input){func(i *Input) { i.IncomeYear = 2026 }, func(i *Input) { i.ScopeConfirmed = false }, func(i *Input) { i.SpecialCases = []string{"alternating_custody"} }, func(i *Input) { i.Family = "widowed"; i.Children = 1 }, func(i *Input) { i.Income = -1 }, func(i *Input) { i.Income = 25000100 }, func(i *Input) { i.SpouseIncome = 1 }, func(i *Input) { i.ActualCosts = 1 }, func(i *Input) { i.Children = 11 }, func(i *Input) { i.Isolated = true }}
	for n, change := range tests {
		in := finalInput(2025, "single", 30000)
		change(&in)
		if _, e := Compute(in); e == nil {
			t.Fatalf("invalid%d accepted", n)
		}
	}
}

// DGFiP brochure2026 p371: gross2140, decote515, net1625.
func TestOfficialJointDiscountExample(t *testing.T) {
	in := finalInput(2025, "married", 42654)
	r := must(t, in)
	if r.GrossTax != 214000 || r.Discount != 51500 || r.Tax != 162500 {
		t.Fatal(r)
	}
}

func TestFinalHouseholdBaseRejectsDoubleCounting(t *testing.T) {
	in := finalInput(2025, "married", 54000)
	in.SpouseIncome = 100
	if _, e := Compute(in); e == nil {
		t.Fatal("household final base counted spouse twice")
	}
}
