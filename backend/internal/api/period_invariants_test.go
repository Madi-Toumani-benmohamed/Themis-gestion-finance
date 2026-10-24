package api

import "testing"

func TestPeriodRequiresDocumentaryEvidence(t *testing.T) {
	for _, lastTx := range []string{"", "2026-01-31"} {
		info := periodFromReliability(reliabilityReport{Month: "2026-01"}, lastTx, 31, false, 1)
		if info.Completeness != "unknown" || info.Status == "complete" || info.CoverageRatio != 0 {
			t.Fatalf("absence de preuve: %+v", info)
		}
	}
	report := reliabilityReport{Month: "2026-01", Accounts: []reliabilityAccount{{Status: "documented"}}}
	if info := periodFromReliability(report, "", 31, false, 1); info.Completeness != "complete" {
		t.Fatalf("mois documenté sans transaction: %+v", info)
	}
	info := periodFromReliability(report, "2026-01-20", 31, true, 21)
	if info.Completeness != "partial" || info.Status != "current" {
		t.Fatalf("mois avant salaire: %+v", info)
	}
	report.Accounts = append(report.Accounts, reliabilityAccount{Status: "no_statement"})
	if info := periodFromReliability(report, "2026-01-31", 31, false, 1); info.Completeness == "complete" {
		t.Fatal("compte manquant ignoré")
	}
}
