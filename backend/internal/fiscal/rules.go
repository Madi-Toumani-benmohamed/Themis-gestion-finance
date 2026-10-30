// Package fiscal implements a deliberately bounded, deterministic income-tax estimate.
package fiscal

import "fmt"

type Rules struct {
	IncomeYear     int      `json:"income_year"`
	AssessmentYear int      `json:"assessment_year"`
	Version        string   `json:"version"`
	Status         string   `json:"status"`
	Verified       string   `json:"verified_on"`
	Bounds         []int64  `json:"bracket_bounds_euros"`
	Rates          []int64  `json:"bracket_rates_pct"`
	SalaryMin      int64    `json:"salary_min_euros"`
	SalaryMax      int64    `json:"salary_max_euros"`
	HalfPartCap    int64    `json:"half_part_cap_euros"`
	IsolatedCap    int64    `json:"isolated_cap_euros"`
	SingleDiscount int64    `json:"single_discount_euros"`
	JointDiscount  int64    `json:"joint_discount_euros"`
	Sources        []string `json:"sources"`
}

func ForYear(year int) (Rules, error) {
	r := Rules{IncomeYear: year, AssessmentYear: year + 1, Status: "official", Verified: "2026-10-30", Rates: []int64{0, 11, 30, 41, 45}}
	switch year {
	case 2024:
		r.Version = "FR-IR-2024-2025.1"
		r.Bounds = []int64{0, 11497, 29315, 83823, 180294}
		r.SalaryMin = 504
		r.SalaryMax = 14426
		r.HalfPartCap = 1791
		r.IsolatedCap = 4224
		r.SingleDiscount = 889
		r.JointDiscount = 1470
	case 2025:
		r.Version = "FR-IR-2025-2026.1"
		r.Bounds = []int64{0, 11600, 29579, 84577, 181917}
		r.SalaryMin = 509
		r.SalaryMax = 14555
		r.HalfPartCap = 1807
		r.IsolatedCap = 4262
		r.SingleDiscount = 897
		r.JointDiscount = 1483
	default:
		return Rules{}, unsupported("Règles non disponibles pour cette année de revenus ; aucun barème d'une autre année n'est appliqué.")
	}
	r.Sources = []string{fmt.Sprintf("https://www.impots.gouv.fr/www2/fichiers/documentation/brochure/ir_%d/pdf_integral/Brochure-IR-%d.pdf", year+1, year+1), fmt.Sprintf("https://simulateur-ir-ifi.impots.gouv.fr/calcul_impot/%d/aides/frais.htm", year+1)}
	return r, nil
}
func Available() []Rules {
	out := []Rules{}
	for _, y := range []int{2024, 2025} {
		r, _ := ForYear(y)
		out = append(out, r)
	}
	return out
}
