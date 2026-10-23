package api

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func reliabilityTestDate(t *testing.T, s string) time.Time {
	t.Helper()
	d, e := time.Parse("2006-01-02", s)
	if e != nil {
		t.Fatal(e)
	}
	return d
}

func TestUncoveredStatementDates(t *testing.T) {
	d := func(s string) time.Time { return reliabilityTestDate(t, s) }
	tests := []struct {
		name  string
		spans []heroDateSpan
		want  []reliabilityGap
	}{
		{"no data", nil, []reliabilityGap{{"2026-09-01", "2026-09-30"}}},
		{"statement on 21", []heroDateSpan{{d("2026-08-22"), d("2026-09-21")}}, []reliabilityGap{{"2026-09-22", "2026-09-30"}}},
		{"successive cycle unsorted", []heroDateSpan{{d("2026-09-22"), d("2026-10-21")}, {d("2026-08-22"), d("2026-09-21")}}, []reliabilityGap{}},
		{"interior and edges", []heroDateSpan{{d("2026-09-02"), d("2026-09-10")}, {d("2026-09-12"), d("2026-09-29")}}, []reliabilityGap{{"2026-09-01", "2026-09-01"}, {"2026-09-11", "2026-09-11"}, {"2026-09-30", "2026-09-30"}}},
		{"overlap", []heroDateSpan{{d("2026-08-01"), d("2026-09-22")}, {d("2026-09-15"), d("2026-10-22")}}, []reliabilityGap{}},
		{"out of range", []heroDateSpan{{d("2026-08-01"), d("2026-08-31")}, {d("2026-10-01"), d("2026-10-31")}}, []reliabilityGap{{"2026-09-01", "2026-09-30"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uncoveredStatementDates(tt.spans, d("2026-09-01"), d("2026-09-30"))
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("gaps = %v, want %v", got, tt.want)
			}
			if heroSpansCover(tt.spans, d("2026-09-01"), d("2026-09-30")) != (len(got) == 0) {
				t.Fatal("hero disagrees with reliability")
			}
		})
	}
}

func TestReliabilityCurrentWindow(t *testing.T) {
	now := reliabilityTestDate(t, "2026-10-23")
	first, last, current, err := reliabilityWindow("2026-10", now)
	if err != nil || !current || first.Day() != 1 || last.Day() != 23 {
		t.Fatal("current month must stop today")
	}
	_, last, current, err = reliabilityWindow("2024-02", now)
	if err != nil || current || last.Day() != 29 {
		t.Fatal("leap month incorrect")
	}
	for _, month := range []string{"2026-11", "2026-13", "2026-1", "not-a-month", "0000-01"} {
		if _, _, _, err := reliabilityWindow(month, now); err == nil {
			t.Errorf("accepted invalid month %q", month)
		}
	}
}

func TestReliabilityAccountEvidence(t *testing.T) {
	first, last := reliabilityTestDate(t, "2026-10-01"), reliabilityTestDate(t, "2026-10-23")
	a := reliabilityAccount{LastImport: "2026-10-23", LastTransaction: "2026-10-23", Statements: []reliabilityStatement{}}
	assessReliabilityAccount(&a, first, last)
	if a.Status != "no_statement" || len(a.Gaps) != 1 {
		t.Fatal("recent metadata must not establish coverage")
	}
	a.Statements = []reliabilityStatement{{From: "2026-10-01", To: "2026-10-31", Validated: false}}
	assessReliabilityAccount(&a, first, last)
	if a.Status != "to_review" || a.ToReview != 1 || len(a.Gaps) != 1 {
		t.Fatal("unvalidated statement established coverage")
	}
	a.Statements = append(a.Statements, reliabilityStatement{From: "2026-10-01", To: "2026-10-23", Validated: true}, reliabilityStatement{Validated: true})
	assessReliabilityAccount(&a, first, last)
	if a.Status != "documented" || a.ToReview != 2 || a.UnknownPeriods != 1 {
		t.Fatal("coverage and review warnings should be independent")
	}
}

func TestReliabilityRejectsInvalidMonthBeforeDB(t *testing.T) {
	srv := &Server{}
	w := httptest.NewRecorder()
	srv.reliability(w, httptest.NewRequest("GET", "/api/reliability?month=invalid", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status %d", w.Code)
	}
}
