package api

import (
	"encoding/json"
	"testing"
	"themis/internal/money"
)

func TestAnalysisRoundingAndReconciliation(t *testing.T) {
	if _, err := parseAnalysisAmount("90071992547409.92"); err == nil {
		t.Fatal("unsafe JSON monetary aggregate accepted")
	}
	for _, tc := range []struct {
		values   []money.Cents
		div      int
		expected []money.Cents
	}{{[]money.Cents{1, 1, 1}, 3, []money.Cents{1, 0, 0}}, {[]money.Cents{100, 101, 102}, 3, []money.Cents{33, 34, 34}}, {[]money.Cents{0, 0}, 3, []money.Cents{0, 0}}} {
		got := analysisApportion(tc.values, tc.div)
		for i, want := range tc.expected {
			if got[i] != want {
				t.Fatalf("%v/%d => %v expected %v", tc.values, tc.div, got, tc.expected)
			}
		}
	}
	for _, tc := range []struct {
		n, d money.Cents
		want string
	}{{1, 3, "33.33"}, {-1, 3, "-33.33"}, {2, 3, "66.67"}, {1, 20000, "0.01"}} {
		if got := analysisPercent(tc.n, tc.d); got == nil || string(*got) != tc.want {
			t.Fatalf("ratio %d/%d = %v", tc.n, tc.d, got)
		}
	}
	if analysisPercent(0, 0) != nil {
		t.Fatal("unknown ratio replaced by zero")
	}
	if got := analysisMean(94766, 3); got != "315.89" {
		t.Fatal(got)
	}
}
func TestAnalysisEvidenceRejectsInventedFacts(t *testing.T) {
	report := analysisReport{Available: true, Month: "2020-01", Income: "3249.47", Expenses: "4197.14", Period: periodInfo{Completeness: "partial"}, Categories: []analysisCategory{{Name: "<untrusted category>", Spent: "0.01", Count: 1}}}
	facts := analysisFacts(report)
	for _, raw := range []string{`{"observation_ids":["invented"]}`, `{"observation_ids":["flows"],"narrative":"invented amount"}`, `{"observation_ids":["flows","flows"]}`, `{"observation_ids":[]}`, `{"observation_ids":["flows"]} {}`, `free text`} {
		if _, err := selectAnalysisEvidence(raw, facts); err == nil {
			t.Fatalf("unsafe accepted: %s", raw)
		}
	}
	selected, err := selectAnalysisEvidence(`{"observation_ids":["flows"]}`, facts)
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != 3 || selected[0].ID != "coverage" || selected[2].ID != "timing" {
		t.Fatalf("missing context: %+v", selected)
	}
	raw, _ := json.Marshal(selected)
	if !json.Valid(raw) {
		t.Fatal("invalid evidence")
	}
}
