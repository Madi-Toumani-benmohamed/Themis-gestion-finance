package api

import (
	"testing"
	"time"
)

func TestHeroDiagnosis(t *testing.T) {
	tests := []struct {
		name              string
		current, complete bool
		net               float64
		points            []heroPoint
		want              string
	}{
		{"salary not yet recorded", true, false, -859.28, nil, "neutral"},
		{"positive provisional flows", true, false, 100, nil, "neutral"},
		{"current real budget alert", true, false, -859, []heroPoint{{Tone: "attention", Kind: "budget_over"}}, "attention"},
		{"current critical alert", true, false, 100, []heroPoint{{Tone: "alert"}}, "alert"},
		{"incomplete negative history", false, false, -1552, nil, "neutral"},
		{"incomplete preserves alerts", false, false, -1552, []heroPoint{{Tone: "alert"}}, "alert"},
		{"complete negative history", false, true, -1552, nil, "alert"},
		{"complete positive history", false, true, 200, nil, "positive"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := heroDiagnosis(tt.current, tt.complete, tt.net, tt.points)
			if got != tt.want {
				t.Fatalf("tone = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestHeroSpansCover(t *testing.T) {
	date := func(s string) time.Time {
		d, e := time.Parse("2006-01-02", s)
		if e != nil {
			t.Fatal(e)
		}
		return d
	}
	first, last := date("2026-09-01"), date("2026-09-30")
	tests := []struct {
		name  string
		spans []heroDateSpan
		want  bool
	}{
		{"no evidence", nil, false},
		{"statement ends on 21", []heroDateSpan{{date("2026-08-22"), date("2026-09-21")}}, false},
		{"successive statements", []heroDateSpan{{date("2026-08-22"), date("2026-09-21")}, {date("2026-09-22"), date("2026-10-21")}}, true},
		{"gap", []heroDateSpan{{first, date("2026-09-21")}, {date("2026-09-23"), last}}, false},
		{"missing start", []heroDateSpan{{date("2026-09-02"), last}}, false},
		{"full month", []heroDateSpan{{first, last}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := heroSpansCover(tt.spans, first, last); got != tt.want {
				t.Fatalf("covered = %v, want %v", got, tt.want)
			}
		})
	}
}
