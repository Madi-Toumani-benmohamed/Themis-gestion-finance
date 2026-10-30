package intelligence

import (
	"math/big"
	"sort"
	"time"

	"themis/internal/money"
)

type MonthlyAmount struct {
	Month  string
	Amount money.Cents
}

type ProgressiveTrend struct {
	FromMonth  string
	ToMonth    string
	FirstCents money.Cents
	LastCents  money.Cents
	DeltaCents money.Cents
	ChangeBPS  int64
	Points     int
}

// DetectProgressiveTrend requires six consecutive monthly observations, at
// least four upward steps, and both a 30% increase and a 100-euro minimum.
// This is a conservative signal, not a seasonal or causal explanation.
func DetectProgressiveTrend(series []MonthlyAmount) (ProgressiveTrend, bool) {
	if len(series) != 6 {
		return ProgressiveTrend{}, false
	}
	upwardSteps := 0
	for i, point := range series {
		month, err := time.Parse("2006-01", point.Month)
		if err != nil || month.Format("2006-01") != point.Month || point.Amount < 0 {
			return ProgressiveTrend{}, false
		}
		if i > 0 {
			previous, _ := time.Parse("2006-01", series[i-1].Month)
			if month.Year() != previous.AddDate(0, 1, 0).Year() || month.Month() != previous.AddDate(0, 1, 0).Month() || point.Amount < series[i-1].Amount {
				return ProgressiveTrend{}, false
			}
			if point.Amount > series[i-1].Amount {
				upwardSteps++
			}
		}
	}
	first, last := series[0].Amount, series[len(series)-1].Amount
	delta := last - first
	if upwardSteps < 4 || delta < 10000 || (first > 0 && !ratioAtLeast(delta, first, 30, 100)) {
		return ProgressiveTrend{}, false
	}
	changeBPS := int64(0)
	if first > 0 {
		numerator := new(big.Int).Mul(big.NewInt(int64(delta)), big.NewInt(10000))
		numerator.Quo(numerator, big.NewInt(int64(first)))
		if numerator.IsInt64() {
			changeBPS = numerator.Int64()
		} else {
			changeBPS = int64(^uint64(0) >> 1)
		}
	}
	return ProgressiveTrend{FromMonth: series[0].Month, ToMonth: series[5].Month, FirstCents: first, LastCents: last, DeltaCents: delta, ChangeBPS: changeBPS, Points: len(series)}, true
}

func ratioAtLeast(value, base money.Cents, numerator, denominator int64) bool {
	left := new(big.Int).Mul(big.NewInt(int64(value)), big.NewInt(denominator))
	right := new(big.Int).Mul(big.NewInt(int64(base)), big.NewInt(numerator))
	return left.Cmp(right) >= 0
}

type MonthlySubjectAmount struct {
	Month  string
	Key    string
	Label  string
	Amount money.Cents
}

type Contribution struct {
	Key        string
	Label      string
	DeltaCents money.Cents
}

type Variation struct {
	BaselineCents money.Cents
	CurrentCents  money.Cents
	Delta         money.Cents
	Contributions []Contribution
}

// ExplainVariation compares two equal sets of documented months. The returned
// subject deltas must reconcile exactly to the total average change.
func ExplainVariation(rows []MonthlySubjectAmount, baselineMonths, currentMonths []string) (Variation, bool) {
	if len(baselineMonths) != 3 || len(currentMonths) != 3 || !validMonthSet(baselineMonths) || !validMonthSet(currentMonths) {
		return Variation{}, false
	}
	months := map[string]bool{}
	for _, month := range baselineMonths {
		months[month] = true
	}
	for _, month := range currentMonths {
		if months[month] {
			return Variation{}, false
		}
		months[month] = true
	}
	bySubject := map[string]map[string]money.Cents{}
	labels := map[string]string{}
	totals := map[string]money.Cents{}
	for _, row := range rows {
		if !months[row.Month] || row.Key == "" || row.Amount < 0 {
			return Variation{}, false
		}
		if bySubject[row.Key] == nil {
			bySubject[row.Key] = map[string]money.Cents{}
		}
		if _, duplicate := bySubject[row.Key][row.Month]; duplicate {
			return Variation{}, false
		}
		bySubject[row.Key][row.Month] = row.Amount
		labels[row.Key] = row.Label
		totals[row.Month] += row.Amount
	}
	baselineTotal, currentTotal := money.Cents(0), money.Cents(0)
	for _, month := range baselineMonths {
		baselineTotal += totals[month]
	}
	for _, month := range currentMonths {
		currentTotal += totals[month]
	}
	contributions := make([]Contribution, 0, len(bySubject))
	for key, monthly := range bySubject {
		before, after := money.Cents(0), money.Cents(0)
		for _, month := range baselineMonths {
			before += monthly[month]
		}
		for _, month := range currentMonths {
			after += monthly[month]
		}
		contributions = append(contributions, Contribution{Key: key, Label: labels[key], DeltaCents: after - before})
	}
	sort.Slice(contributions, func(i, j int) bool {
		if contributions[i].DeltaCents != contributions[j].DeltaCents {
			return contributions[i].DeltaCents > contributions[j].DeltaCents
		}
		return contributions[i].Key < contributions[j].Key
	})
	var reconciled money.Cents
	for _, contribution := range contributions {
		reconciled += contribution.DeltaCents
	}
	delta := currentTotal - baselineTotal
	if reconciled != delta {
		return Variation{}, false
	}
	return Variation{BaselineCents: baselineTotal, CurrentCents: currentTotal, Delta: delta, Contributions: contributions}, true
}

type MonthlyCount struct {
	Month string
	Count int64
}

type FrequencyChange struct {
	PreviousCount int64
	CurrentCount  int64
	Delta         int64
	ActiveMonths  int
}

// DetectFrequencyChange needs three complete months on both sides, at least
// three additional operations and activity in at least two recent months.
func DetectFrequencyChange(previous, current []MonthlyCount) (FrequencyChange, bool) {
	if len(previous) != 3 || len(current) != 3 || !validCountWindow(previous) || !validCountWindow(current) {
		return FrequencyChange{}, false
	}
	prevSum, currentSum, active := int64(0), int64(0), 0
	for _, point := range previous {
		prevSum += point.Count
	}
	for _, point := range current {
		currentSum += point.Count
		if point.Count > 0 {
			active++
		}
	}
	minimumRelativeIncrease := prevSum/2 + prevSum%2
	if active < 2 || currentSum < prevSum || currentSum-prevSum < 3 || currentSum-prevSum < minimumRelativeIncrease {
		return FrequencyChange{}, false
	}
	return FrequencyChange{PreviousCount: prevSum, CurrentCount: currentSum, Delta: currentSum - prevSum, ActiveMonths: active}, true
}

func validMonthSet(months []string) bool {
	seen := map[string]bool{}
	for _, value := range months {
		parsed, err := time.Parse("2006-01", value)
		if err != nil || parsed.Format("2006-01") != value || seen[value] {
			return false
		}
		seen[value] = true
	}
	return true
}

func validCountWindow(window []MonthlyCount) bool {
	months := make([]string, len(window))
	for i, point := range window {
		if point.Count < 0 {
			return false
		}
		months[i] = point.Month
		if i > 0 {
			previous, err := time.Parse("2006-01", window[i-1].Month)
			current, currentErr := time.Parse("2006-01", point.Month)
			if err != nil || currentErr != nil || current.Year() != previous.AddDate(0, 1, 0).Year() || current.Month() != previous.AddDate(0, 1, 0).Month() {
				return false
			}
		}
	}
	return validMonthSet(months)
}
