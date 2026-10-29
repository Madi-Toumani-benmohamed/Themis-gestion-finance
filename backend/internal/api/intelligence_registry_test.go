package api

import (
	"testing"

	"themis/internal/intelligence"
	"themis/internal/money"
)

func TestRecurringDiscoveryCarriesExactEvidenceAndUnits(t *testing.T) {
	freq := "monthly"
	item := recurringReviewItem{
		Key: "stable-key", Label: "Marchand synthétique", Direction: "debit", Confidence: "probable",
		ConfidenceReason: "trois occurrences cohérentes sur un intervalle documenté", CoverageComplete: true,
		OccurrenceCount: 3, MonthsCount: 3, Months: []string{"2026-07", "2026-08", "2026-09"},
		TransactionIDs: []int64{11, 12, 13}, SourceAmounts: []money.Cents{1299, 1300, 1301},
		TypicalAmount: "13.00", Frequency: &freq, FirstDate: "2026-07-10", LastDate: "2026-09-10",
	}
	discovery := recurringDiscovery(7, item)
	if discovery.OwnerID != 7 || discovery.Kind != "recurring" || discovery.SubjectKey != item.Key || discovery.Confidence != 85 {
		t.Fatalf("identity/confidence mismatch: %+v", discovery)
	}
	if discovery.Metrics[1].Kind != "money_cents" || discovery.Metrics[1].Value != 1300 || discovery.Metrics[0].Value != 3 {
		t.Fatalf("typed metrics mismatch: %+v", discovery.Metrics)
	}
	if discovery.Coverage.Status != "complete" || len(discovery.Evidence.TransactionIDs) != 3 || discovery.Period != "2026-09" {
		t.Fatalf("evidence/coverage mismatch: %+v", discovery)
	}
	if medianCents([]money.Cents{1, 2}) != 2 {
		t.Fatal("even median must use the established half-up cent rounding")
	}
}

func TestRecurringDiscoveryDoesNotPromoteUnknownCoverage(t *testing.T) {
	item := recurringReviewItem{
		Key: "stable-key", Label: "Marchand synthétique", Direction: "debit", Confidence: "insufficient",
		ConfidenceReason: "couverture documentaire inconnue", CoverageComplete: false,
		OccurrenceCount: 2, MonthsCount: 2, TransactionIDs: []int64{21, 22},
		SourceAmounts: []money.Cents{50000, 50000}, TypicalAmount: "500.00",
	}
	discovery := recurringDiscovery(7, item)
	if discovery.Confidence >= 50 || discovery.RelevanceScore > 49 || discovery.Coverage.Status != "unknown" {
		t.Fatalf("unknown coverage was overstated: %+v", discovery)
	}
}

func TestRecurringImpactScoreUsesExplicitCentBoundaries(t *testing.T) {
	for _, tc := range []struct {
		cents money.Cents
		want  int
	}{
		{cents: 0, want: 0},
		{cents: 2499, want: 15},
		{cents: 2500, want: 30},
		{cents: 10000, want: 50},
		{cents: 50000, want: 75},
		{cents: 200000, want: 100},
	} {
		if got := recurringImpactScore(tc.cents); got != tc.want {
			t.Errorf("recurringImpactScore(%d) = %d, want %d", tc.cents, got, tc.want)
		}
	}
}

func TestAnalysisSignalAdapterKeepsTypedMetricsAndCoverage(t *testing.T) {
	signal := map[string]any{
		"kind": "merchant_drift", "title": "Hausse observée", "detail": "Quatre périodes comparées.",
		"metrics": []intelligence.Metric{{Name: "observed_increase_cents", Kind: intelligence.MetricMoneyCents, Value: 40000}},
		"source":  map[string]any{"merchant": "Synthetic Shop", "periods": []string{"2026-06", "2026-07", "2026-08", "2026-09"}},
	}
	discovery, err := discoveryFromAnalysisSignal(3, "2026-09", signal)
	if err != nil {
		t.Fatal(err)
	}
	if discovery.Coverage.Status != "complete" || discovery.ImpactScore != 50 || discovery.Metrics[0].Value != 40000 {
		t.Fatalf("signal facts were not preserved: %+v", discovery)
	}
	if discovery.SubjectKey == "Synthetic Shop" {
		t.Fatal("subject key exposed merchant text")
	}
}

func TestSeasonalSignalRemainsHypothesisWithUnknownCoverage(t *testing.T) {
	signal := map[string]any{
		"kind": "seasonal", "title": "Motif annuel possible", "detail": "Aucune échéance certaine.",
		"source": map[string]any{"merchant": "Synthetic Shop", "periods": []string{"2024-12", "2025-12"}},
	}
	discovery, err := discoveryFromAnalysisSignal(3, "2026-11", signal)
	if err != nil {
		t.Fatal(err)
	}
	if discovery.Coverage.Status != "unknown" || discovery.Confidence != 45 || discovery.RelevanceScore > 49 {
		t.Fatalf("seasonal hypothesis overstated: %+v", discovery)
	}
}
