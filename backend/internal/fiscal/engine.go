package fiscal

import (
	"fmt"
	"math/big"
	"themis/internal/money"
)

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string   { return e.Message }
func invalid(s string) error     { return &Error{"invalid_input", s} }
func unsupported(s string) error { return &Error{"unsupported", s} }

type Input struct {
	IncomeYear               int          `json:"income_year"`
	Family                   string       `json:"family"`
	Children                 int          `json:"children"`
	Isolated                 bool         `json:"isolated_parent"`
	ScopeConfirmed           bool         `json:"scope_confirmed"`
	SpecialCases             []string     `json:"special_cases"`
	Basis                    string       `json:"basis"`         // salary (payslip net fiscal) or final_taxable (already after deductions)
	IncomePeriod             string       `json:"income_period"` // annual, monthly, cumulative
	ObservedMonths           int          `json:"observed_months"`
	ProjectIncome            bool         `json:"project_income"`
	Income                   money.Cents  `json:"income_cents"`
	SpouseIncome             money.Cents  `json:"spouse_income_cents"`
	Deduction                string       `json:"deduction"` // standard or actual
	ActualCosts              money.Cents  `json:"actual_costs_cents"`
	SpouseActualCosts        money.Cents  `json:"spouse_actual_costs_cents"`
	WithholdingYear          int          `json:"withholding_year"`
	WithholdingObserved      *money.Cents `json:"withholding_observed_cents"`
	WithholdingMonths        int          `json:"withholding_months"`
	FutureMonthlyWithholding *money.Cents `json:"future_monthly_withholding_cents"`
}
type Step struct {
	Label string      `json:"label"`
	Cents money.Cents `json:"amount_cents"`
}
type Result struct {
	Rules            Rules        `json:"rules"`
	HalfParts        int          `json:"half_parts"`
	AnnualIncome     money.Cents  `json:"annual_income_cents"`
	SalaryDeduction  money.Cents  `json:"salary_deduction_cents"`
	Taxable          money.Cents  `json:"taxable_cents"`
	GrossTax         money.Cents  `json:"gross_tax_cents"`
	FamilyCapApplied bool         `json:"family_cap_applied"`
	Discount         money.Cents  `json:"discount_cents"`
	Tax              money.Cents  `json:"tax_cents"`
	AverageRate      string       `json:"average_rate_pct"`
	MarginalRate     int64        `json:"marginal_rate_pct"`
	ObservedPaid     *money.Cents `json:"observed_paid_cents"`
	ProjectedPaid    *money.Cents `json:"projected_future_paid_cents"`
	GapBeforeFuture  *money.Cents `json:"gap_before_future_cents"`
	Settlement       *money.Cents `json:"settlement_cents"`
	Projection       bool         `json:"income_projected"`
	SettlementStatus string       `json:"settlement_status"`
	Steps            []Step       `json:"steps"`
	Assumptions      []string     `json:"assumptions"`
	Limitations      []string     `json:"limitations"`
}

func roundPositive(r *big.Rat) int64 {
	n := new(big.Int).Mul(r.Num(), big.NewInt(2))
	n.Add(n, r.Denom())
	return n.Quo(n, new(big.Int).Mul(r.Denom(), big.NewInt(2))).Int64()
}
func euro(c money.Cents) int64 { return (int64(c) + 50) / 100 }
func rat(n, d int64) *big.Rat  { return big.NewRat(n, d) }
func progressive(base int64, halfParts int, r Rules) (*big.Rat, int64) {
	total := new(big.Rat)
	rate := int64(0)
	parts := rat(int64(halfParts), 2)
	for i, lower := range r.Bounds {
		portion := new(big.Rat).Sub(rat(base, 1), new(big.Rat).Mul(rat(lower, 1), parts))
		if portion.Sign() <= 0 {
			continue
		}
		rate = r.Rates[i]
		if i+1 < len(r.Bounds) {
			width := new(big.Rat).Mul(rat(r.Bounds[i+1]-lower, 1), parts)
			if portion.Cmp(width) > 0 {
				portion = width
			}
		}
		total.Add(total, new(big.Rat).Mul(portion, rat(r.Rates[i], 100)))
	}
	return total, rate
}
func salaryDeduction(salary, actual int64, kind string, r Rules) int64 {
	if kind == "actual" {
		return actual
	}
	if salary == 0 {
		return 0
	}
	d := roundPositive(rat(salary, 10))
	if d < r.SalaryMin {
		d = r.SalaryMin
	}
	if d > r.SalaryMax {
		d = r.SalaryMax
	}
	if d > salary {
		d = salary
	}
	return d
}
func Compute(in Input) (Result, error) {
	out := Result{Steps: []Step{}, Assumptions: []string{}, Limitations: []string{"Estimation non officielle ; ne remplace pas une déclaration ni un avis d'impôt.", "Sans réductions/crédits, revenus spéciaux, déficits, abattement d'âge/invalidité, CEHR/CDHR ou avances de crédits d'impôt."}, SettlementStatus: "unknown"}
	r, e := ForYear(in.IncomeYear)
	if e != nil {
		return out, e
	}
	out.Rules = r
	if !in.ScopeConfirmed || len(in.SpecialCases) > 0 {
		return out, unsupported("Confirmez le périmètre : résident fiscal métropolitain, moins de 65 ans, revenus ordinaires, sans situation particulière ni réduction/crédit applicable.")
	}
	joint := in.Family == "married" || in.Family == "pacs"
	if !joint && in.Family != "single" && in.Family != "divorced" && in.Family != "widowed" {
		return out, invalid("Situation familiale invalide.")
	}
	if in.Children < 0 || in.Children > 10 {
		return out, invalid("Nombre d'enfants invalide.")
	}
	if in.Family == "widowed" && in.Children > 0 {
		return out, unsupported("Le veuvage avec enfant et ses avantages complémentaires ne sont pas pris en charge.")
	}
	if in.Isolated && (joint || in.Children == 0 || in.Family == "widowed") {
		return out, invalid("La case parent isolé nécessite un parent célibataire/divorcé vivant seul avec un enfant à charge exclusive.")
	}
	if in.Basis != "salary" && in.Basis != "final_taxable" {
		return out, invalid("Indiquez la base : salaire net fiscal ou revenu net imposable final.")
	}
	for _, v := range []money.Cents{in.Income, in.SpouseIncome, in.ActualCosts, in.SpouseActualCosts} {
		if v < 0 || v > 50000000 {
			return out, invalid("Montant négatif ou hors périmètre de cette V1.")
		}
	}
	if !joint && (in.SpouseIncome != 0 || in.SpouseActualCosts != 0) {
		return out, invalid("Le revenu du conjoint nécessite une imposition commune.")
	}
	annual := func(v money.Cents) money.Cents { return v }
	switch in.IncomePeriod {
	case "annual":
		if in.ObservedMonths != 12 {
			return out, invalid("Une saisie annuelle complète doit couvrir douze mois.")
		}
	case "monthly":
		if !in.ProjectIncome {
			return out, invalid("L'annualisation mensuelle nécessite une hypothèse de projection explicite.")
		}
		out.Projection = true
		annual = func(v money.Cents) money.Cents { return v * 12 }
		out.Assumptions = append(out.Assumptions, "Revenu mensuel constant annualisé sur douze mois.")
	case "cumulative":
		if in.ObservedMonths < 1 || in.ObservedMonths > 12 {
			return out, invalid("Période observée : 1 à 12 mois requis.")
		}
		if in.ObservedMonths < 12 && !in.ProjectIncome {
			return out, invalid("Revenus incomplets : fournissez un revenu annuel ou autorisez explicitement une projection.")
		}
		out.Projection = in.ObservedMonths < 12
		annual = func(v money.Cents) money.Cents {
			return money.Cents(roundPositive(rat(int64(v)*12, int64(in.ObservedMonths))))
		}
		out.Assumptions = append(out.Assumptions, fmt.Sprintf("Cumul sur %d mois annualisé à rythme constant.", in.ObservedMonths))
	default:
		return out, invalid("Période de revenus invalide.")
	}
	a, b := euro(annual(in.Income)), euro(annual(in.SpouseIncome))
	limit := int64(250000)
	if joint {
		limit = 500000
	}
	if a+b > limit {
		return out, unsupported("Hauts revenus et contributions complémentaires non pris en charge.")
	}
	if in.Deduction != "standard" && in.Deduction != "actual" {
		return out, invalid("Choisissez frais forfaitaires ou frais réels.")
	}
	if in.Basis == "final_taxable" && in.SpouseIncome != 0 {
		return out, invalid("Le revenu net imposable final concerne le foyer entier ; ne rajoutez pas un second revenu du conjoint.")
	}
	if in.Basis == "final_taxable" && (in.ActualCosts != 0 || in.SpouseActualCosts != 0 || in.Deduction != "standard") {
		return out, invalid("Le revenu net imposable final est déjà après déductions ; aucun second abattement autorisé.")
	}
	if in.Deduction == "standard" && (in.ActualCosts != 0 || in.SpouseActualCosts != 0) {
		return out, invalid("Frais réels et forfait ne peuvent pas être cumulés.")
	}
	costsA, costsB := euro(in.ActualCosts), euro(in.SpouseActualCosts)
	if costsA > a || costsB > b {
		return out, unsupported("Les frais réels supérieurs au salaire et déficits ne sont pas pris en charge.")
	}
	deduction := int64(0)
	if in.Basis == "salary" {
		deduction = salaryDeduction(a, costsA, in.Deduction, r) + salaryDeduction(b, costsB, in.Deduction, r)
	}
	taxable := a + b - deduction
	out.AnnualIncome = money.Cents((a + b) * 100)
	out.SalaryDeduction = money.Cents(deduction * 100)
	out.Taxable = money.Cents(taxable * 100)
	adultHalf := 2
	if joint {
		adultHalf = 4
	}
	half := adultHalf
	if in.Children <= 2 {
		half += in.Children
	} else {
		half += 2 + (in.Children-2)*2
	}
	if in.Isolated {
		half++
	}
	out.HalfParts = half
	raw, marginal := progressive(taxable, half, r)
	baseRaw, baseRate := progressive(taxable, adultHalf, r)
	cap := int64(half-adultHalf) * r.HalfPartCap
	if in.Isolated {
		cap = r.IsolatedCap + int64(half-adultHalf-2)*r.HalfPartCap
	}
	floor := new(big.Rat).Sub(baseRaw, rat(cap, 1))
	if floor.Cmp(raw) > 0 {
		raw = floor
		out.FamilyCapApplied = true
		marginal = baseRate
	}
	gross := roundPositive(raw)
	out.GrossTax = money.Cents(gross * 100)
	out.MarginalRate = marginal
	plafond := r.SingleDiscount
	if joint {
		plafond = r.JointDiscount
	}
	discount := roundPositive(maxZero(new(big.Rat).Sub(rat(plafond, 1), new(big.Rat).Mul(rat(gross, 1), rat(4525, 10000)))))
	if discount > gross {
		discount = gross
	}
	out.Discount = money.Cents(discount * 100)
	tax := gross - discount
	if tax < 61 {
		tax = 0
	}
	out.Tax = money.Cents(tax * 100)
	out.AverageRate = "0.00"
	if taxable > 0 {
		rate := roundPositive(rat(tax*10000, taxable))
		out.AverageRate = fmt.Sprintf("%d.%02d", rate/100, rate%100)
	}
	out.Steps = []Step{{"Revenus annualisés, arrondis déclaratifs", out.AnnualIncome}, {"Frais professionnels déduits", out.SalaryDeduction}, {"Revenu net imposable retenu", out.Taxable}, {"Impôt brut après plafonnement familial", out.GrossTax}, {"Décote", out.Discount}, {"Impôt estimé après seuil de recouvrement", out.Tax}}
	if in.WithholdingObserved == nil {
		if in.FutureMonthlyWithholding != nil || in.WithholdingMonths != 0 || in.WithholdingYear != 0 {
			return out, invalid("Un cumul de prélèvement observé est requis avant projection.")
		}
		out.Assumptions = append(out.Assumptions, "Prélèvement à la source non renseigné : aucun solde de régularisation calculé.")
	} else {
		if in.WithholdingYear != in.IncomeYear {
			return out, invalid("Les prélèvements doivent concerner la même année de revenus.")
		}
		if in.WithholdingMonths < 1 || in.WithholdingMonths > 12 || *in.WithholdingObserved < 0 || *in.WithholdingObserved > 50000000 {
			return out, invalid("Cumul observé ou nombre de mois invalide.")
		}
		out.ObservedPaid = in.WithholdingObserved
		gap := out.Tax - *in.WithholdingObserved
		out.GapBeforeFuture = &gap
		if in.WithholdingMonths == 12 {
			if in.FutureMonthlyWithholding != nil {
				return out, invalid("Une année de prélèvements complète ne comporte pas de projection future.")
			}
			out.Settlement = &gap
			out.SettlementStatus = "annual_estimate"
			if out.Projection {
				out.SettlementStatus = "projected"
			}
		} else if in.FutureMonthlyWithholding != nil {
			if *in.FutureMonthlyWithholding < 0 || *in.FutureMonthlyWithholding > 50000000 {
				return out, invalid("Projection de prélèvement hors limites.")
			}
			future := *in.FutureMonthlyWithholding * money.Cents(12-in.WithholdingMonths)
			out.ProjectedPaid = &future
			balance := gap - future
			out.Settlement = &balance
			out.SettlementStatus = "projected"
			out.Assumptions = append(out.Assumptions, "Les prélèvements futurs sont une hypothèse mensuelle constante, distincte des paiements observés.")
		} else {
			out.SettlementStatus = "incomplete"
			out.Assumptions = append(out.Assumptions, "Cumul de PAS partiel : l'écart avant prélèvements futurs n'est pas un solde final.")
		}
	}
	out.Assumptions = append(out.Assumptions, "Salaires ordinaires ou base nette finale déjà déterminée ; enfants mineurs à charge exclusive ; arrondi à l'euro le plus proche, 0,50 € arrondi à l'euro supérieur.")
	return out, nil
}
func maxZero(v *big.Rat) *big.Rat {
	if v.Sign() < 0 {
		return new(big.Rat)
	}
	return v
}
