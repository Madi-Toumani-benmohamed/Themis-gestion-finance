package intelligence

import (
	"testing"
	"time"
)

func TestScoreUsesConfiguredWeightsAndKeepsConfidenceSeparate(t *testing.T) {
	f := Factors{Impact: 80, Evidence: 100, Novelty: 50, Context: 20, Action: 70}
	score, ok := Score(f, DefaultWeights, true, "complete")
	if !ok || score != 69 {
		t.Fatalf("Score() = %d, %v; want 69, true", score, ok)
	}
}

func TestScoreCapsClaimsWithoutAdequateEvidence(t *testing.T) {
	f := Factors{Impact: 100, Evidence: 100, Novelty: 100, Context: 100, Action: 100}
	for _, tc := range []struct {
		name     string
		evidence bool
		coverage string
	}{
		{name: "missing evidence", evidence: false, coverage: "complete"},
		{name: "partial coverage", evidence: true, coverage: "partial"},
		{name: "unknown coverage", evidence: true, coverage: "unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			score, ok := Score(f, DefaultWeights, tc.evidence, tc.coverage)
			if !ok || score != 49 {
				t.Fatalf("Score() = %d, %v; want 49, true", score, ok)
			}
		})
	}
}

func TestScoreRejectsInvalidInputs(t *testing.T) {
	if _, ok := Score(Factors{Impact: 101}, DefaultWeights, true, "complete"); ok {
		t.Fatal("invalid factor accepted")
	}
	if _, ok := Score(Factors{}, Weights{Impact: 30, Evidence: 25, Novelty: 20, Context: 15, Action: 9}, true, "complete"); ok {
		t.Fatal("weights not summing to 100 accepted")
	}
}

func TestPrioritizeDeduplicatesHidesAndBounds(t *testing.T) {
	now := time.Now()
	items := []Discovery{
		{ID: "a", ObservationID: "a-old", RelevanceScore: 70, Confidence: 40, State: "active", UpdatedAt: now.Add(-time.Hour)},
		{ID: "a", ObservationID: "a-new", RelevanceScore: 75, Confidence: 40, State: "active", UpdatedAt: now},
		{ID: "b", ObservationID: "b", RelevanceScore: 90, Confidence: 90, State: "hidden"},
		{ID: "c", ObservationID: "c", RelevanceScore: 60, Confidence: 60, State: "active"},
		{ID: "invalid", RelevanceScore: 101, Confidence: 100, State: "active"},
	}
	got := Prioritize(items, 2)
	if len(got) != 2 || got[0].ID != "a" || got[0].ObservationID != "a-new" || got[1].ID != "c" {
		t.Fatalf("Prioritize() = %#v", got)
	}
	if got := Prioritize(items, 0); len(got) != 0 {
		t.Fatalf("zero limit returned %d items", len(got))
	}
}
