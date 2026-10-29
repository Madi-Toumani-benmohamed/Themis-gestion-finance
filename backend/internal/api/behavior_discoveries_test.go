package api

import (
	"testing"

	"themis/internal/intelligence"
	"themis/internal/money"
)

func TestBuildBehaviorDiscoveriesUsesSixMonthExactEvidence(t *testing.T) {
	months := []string{"2026-01", "2026-02", "2026-03", "2026-04", "2026-05", "2026-06"}
	facts := []behaviorFacts{}
	for i, month := range months {
		facts = append(facts, behaviorFacts{id: int64(i + 1), month: month, merchant: "Marché Synthétique", category: "Alimentation", amount: money.Cents((i + 1) * 5000), weekend: i >= 3})
	}
	discoveries := buildBehaviorDiscoveries(9, "2026-06", months, facts)
	byKind := map[string]bool{}
	for _, d := range discoveries {
		byKind[d.Kind] = true
		if d.OwnerID != 9 || d.Coverage.Status != "complete" || d.Coverage.Periods != 6 || len(d.Evidence.TransactionIDs) == 0 {
			t.Fatalf("coverage/owner/evidence mismatch: %+v", d)
		}
		if d.SubjectKey == "Marché Synthétique" || d.SubjectKey == "Alimentation" {
			t.Fatalf("raw subject leaked into stable key: %+v", d)
		}
	}
	if !byKind["progressive_trend"] {
		t.Fatalf("expected progressive trend, got %+v", discoveries)
	}
	for _, d := range discoveries {
		if d.Kind == "spend_variation_contributors" {
			var sum int64
			for _, m := range d.Metrics {
				if m.Name == "top_contributors_cents" || m.Name == "other_contributors_cents" {
					sum += m.Value
				}
			}
			var delta int64
			for _, m := range d.Metrics {
				if m.Name == "observed_change_cents" {
					delta = m.Value
				}
			}
			if sum != delta {
				t.Fatalf("contributions do not reconcile: sum=%d delta=%d", sum, delta)
			}
		}
	}
}

func TestBuildBehaviorDiscoveriesDoesNotPromoteSingleMonthFrequencySpike(t *testing.T) {
	months := []string{"2026-01", "2026-02", "2026-03", "2026-04", "2026-05", "2026-06"}
	facts := []behaviorFacts{}
	for i, month := range months {
		if i == 5 {
			for j := 0; j < 10; j++ {
				facts = append(facts, behaviorFacts{id: int64(100 + j), month: month, merchant: "Shop", category: "Loisirs", amount: 100, weekend: true})
			}
		}
	}
	for _, d := range buildBehaviorDiscoveries(1, "2026-06", months, facts) {
		if d.Kind == "weekend_frequency_change" {
			t.Fatalf("single-month spike should not be a habit change: %+v", d)
		}
	}
}

func TestBuildBehaviorDiscoveriesDetectsSustainedMerchantFrequencyChange(t *testing.T) {
	months := []string{"2026-01", "2026-02", "2026-03", "2026-04", "2026-05", "2026-06"}
	facts := []behaviorFacts{}
	id := int64(1)
	for i, month := range months {
		repetitions := 1
		if i >= 3 {
			repetitions = 2
		}
		for j := 0; j < repetitions; j++ {
			facts = append(facts, behaviorFacts{id: id, month: month, merchant: "Shop synthétique", categoryKey: "10", category: "Catégorie", amount: 100})
			id++
		}
	}
	for _, d := range buildBehaviorDiscoveries(4, "2026-06", months, facts) {
		if d.Kind == "purchase_frequency_change" {
			if d.Metrics[0].Value != 3 || d.Metrics[1].Value != 6 || d.Metrics[2].Value != 3 {
				t.Fatalf("frequency facts mismatch: %+v", d.Metrics)
			}
			return
		}
	}
	t.Fatal("sustained merchant frequency change was not detected")
}

func TestProgressiveMerchantTrendSubsumesLegacyDriftSignal(t *testing.T) {
	discoveries := []intelligence.Discovery{{Kind: "progressive_trend", Title: "Hausse progressive : Shop synthétique", Evidence: intelligence.Evidence{Source: "behavior_facts_v1:merchant"}}}
	signals := []map[string]any{
		{"kind": "merchant_drift", "source": map[string]any{"merchant": "Shop synthétique"}},
		{"kind": "tag_surge", "source": map[string]any{"tag": "Projet"}},
	}
	got := removeSubsumedMerchantDrifts(signals, discoveries)
	if len(got) != 1 || got[0]["kind"] != "tag_surge" {
		t.Fatalf("duplicate merchant signal was not removed: %+v", got)
	}
}
