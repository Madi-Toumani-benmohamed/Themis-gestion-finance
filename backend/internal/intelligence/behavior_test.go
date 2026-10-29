package intelligence

import (
	"math"
	"testing"

	"themis/internal/money"
)

func TestDetectProgressiveTrendRequiresSixConsecutiveGrowingMonths(t *testing.T) {
	series := []MonthlyAmount{
		{Month: "2026-01", Amount: 10000},
		{Month: "2026-02", Amount: 12000},
		{Month: "2026-03", Amount: 14000},
		{Month: "2026-04", Amount: 16000},
		{Month: "2026-05", Amount: 18000},
		{Month: "2026-06", Amount: 20000},
	}
	trend, ok := DetectProgressiveTrend(series)
	if !ok || trend.DeltaCents != 10000 || trend.ChangeBPS != 10000 || trend.Points != 6 {
		t.Fatalf("trend = %+v, %v", trend, ok)
	}
	if _, ok := DetectProgressiveTrend(series[1:]); ok {
		t.Fatal("five observations were accepted")
	}
	series[4].Amount = 12500
	if _, ok := DetectProgressiveTrend(series); ok {
		t.Fatal("a declining month was accepted as progressive")
	}
}

func TestDetectProgressiveTrendRejectsGapsAndSmallChange(t *testing.T) {
	series := []MonthlyAmount{
		{Month: "2026-01", Amount: 100000},
		{Month: "2026-02", Amount: 101000},
		{Month: "2026-03", Amount: 102000},
		{Month: "2026-04", Amount: 103000},
		{Month: "2026-05", Amount: 104000},
		{Month: "2026-06", Amount: 105000},
	}
	if _, ok := DetectProgressiveTrend(series); ok {
		t.Fatal("small relative change was accepted")
	}
	series[3].Month = "2026-05"
	if _, ok := DetectProgressiveTrend(series); ok {
		t.Fatal("gapped calendar series was accepted")
	}
}

func TestDetectProgressiveTrendSaturatesExtremePercentageWithoutOverflow(t *testing.T) {
	series := []MonthlyAmount{{Month: "2026-01", Amount: 1}, {Month: "2026-02", Amount: 2}, {Month: "2026-03", Amount: 3}, {Month: "2026-04", Amount: 4}, {Month: "2026-05", Amount: 5}, {Month: "2026-06", Amount: money.Cents(math.MaxInt64)}}
	trend, ok := DetectProgressiveTrend(series)
	if !ok || trend.ChangeBPS != math.MaxInt64 {
		t.Fatalf("extreme trend overflowed: %+v %v", trend, ok)
	}
}

func TestExplainVariationReconcilesPositiveAndNegativeContributors(t *testing.T) {
	rows := []MonthlySubjectAmount{}
	for _, month := range []string{"2026-01", "2026-02", "2026-03"} {
		rows = append(rows,
			MonthlySubjectAmount{Month: month, Key: "food", Label: "Courses", Amount: money.Cents(10000)},
			MonthlySubjectAmount{Month: month, Key: "travel", Label: "Transport", Amount: money.Cents(5000)},
		)
	}
	for _, month := range []string{"2026-04", "2026-05", "2026-06"} {
		rows = append(rows,
			MonthlySubjectAmount{Month: month, Key: "food", Label: "Courses", Amount: money.Cents(20000)},
			MonthlySubjectAmount{Month: month, Key: "travel", Label: "Transport", Amount: money.Cents(2500)},
		)
	}
	variation, ok := ExplainVariation(rows, []string{"2026-01", "2026-02", "2026-03"}, []string{"2026-04", "2026-05", "2026-06"})
	if !ok || variation.BaselineCents != 45000 || variation.CurrentCents != 67500 || variation.Delta != 22500 {
		t.Fatalf("variation = %+v, %v", variation, ok)
	}
	if len(variation.Contributions) != 2 || variation.Contributions[0].Key != "food" || variation.Contributions[0].DeltaCents != 30000 || variation.Contributions[1].DeltaCents != -7500 {
		t.Fatalf("contributions do not reconcile: %+v", variation.Contributions)
	}
}

func TestExplainVariationRejectsDuplicateAndOverlappingPeriods(t *testing.T) {
	periods := []string{"2026-01", "2026-02", "2026-03"}
	rows := []MonthlySubjectAmount{{Month: "2026-01", Key: "food", Amount: 100}}
	if _, ok := ExplainVariation(rows, periods, []string{"2026-03", "2026-04", "2026-05"}); ok {
		t.Fatal("overlapping windows were accepted")
	}
	rows = append(rows, rows[0])
	if _, ok := ExplainVariation(rows, periods, []string{"2026-04", "2026-05", "2026-06"}); ok {
		t.Fatal("duplicate subject/month aggregate was accepted")
	}
}

func TestDetectFrequencyChangeRequiresRepeatedRecentActivity(t *testing.T) {
	previous := []MonthlyCount{{Month: "2026-01", Count: 1}, {Month: "2026-02", Count: 1}, {Month: "2026-03", Count: 1}}
	current := []MonthlyCount{{Month: "2026-04", Count: 2}, {Month: "2026-05", Count: 2}, {Month: "2026-06", Count: 2}}
	change, ok := DetectFrequencyChange(previous, current)
	if !ok || change.PreviousCount != 3 || change.CurrentCount != 6 || change.Delta != 3 || change.ActiveMonths != 3 {
		t.Fatalf("frequency change = %+v, %v", change, ok)
	}
	current[0].Count, current[1].Count = 0, 0
	current[2].Count = 6
	if _, ok := DetectFrequencyChange(previous, current); ok {
		t.Fatal("one month spike was mistaken for a habit change")
	}
}
