package api

import (
	"testing"
	"time"
)

func TestAgentRelativePeriods(t *testing.T) {
	loc, _ := agentLocation("Europe/Paris")
	now := time.Date(2026, 11, 17, 23, 30, 0, 0, loc)
	previous := agentPeriod{From: "2026-09-01", To: "2026-09-30"}
	tests := []struct{ text, from, to string }{
		{"donne moi ma top dépense du mois", "2026-11-01", "2026-11-17"},
		{"ce mois-ci", "2026-11-01", "2026-11-17"}, {"mois dernier", "2026-10-01", "2026-10-31"}, {"et le mois précédent ?", "2026-08-01", "2026-08-31"},
		{"cette semaine", "2026-11-16", "2026-11-17"}, {"la semaine dernière", "2026-11-09", "2026-11-15"}, {"hier", "2026-11-16", "2026-11-16"}, {"avant-hier", "2026-11-15", "2026-11-15"},
		{"cette année", "2026-01-01", "2026-11-17"}, {"l’année dernière", "2025-01-01", "2025-12-31"}, {"novembre", "2026-11-01", "2026-11-17"}, {"février 2024", "2024-02-01", "2024-02-29"},
		{"les 7 derniers jours", "2026-11-11", "2026-11-17"}, {"ces 6 derniers mois", "2026-05-17", "2026-11-17"}, {"Préparer mon budget du mois prochain", "2026-11-01", "2026-11-17"}, {"ces 7 jours", "2026-11-11", "2026-11-17"}, {"ces 3 derniers mois", "2026-08-17", "2026-11-17"}, {"du 2026-10-03 au 2026-10-25", "2026-10-03", "2026-10-25"},
		{"depuis 2026-10-03", "2026-10-03", "2026-11-17"}, {"2026-09", "2026-09-01", "2026-09-30"}, {"cette sélection", "2026-07-01", "2026-07-31"},
	}
	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			p, e := resolveAgentPeriod(tt.text, now, previous, func() string {
				if tt.text == "cette sélection" {
					return "2026-07"
				}
				return ""
			}())
			if e != nil || p.From != tt.from || p.To != tt.to {
				t.Fatalf("got %+v %v, expected %s/%s", p, e, tt.from, tt.to)
			}
		})
	}
	for _, s := range []string{"2026-02-30", "2026-13", "du 2026-10-31 au 2026-10-01", "décembre 2026", "depuis mon salaire"} {
		if _, e := resolveAgentPeriod(s, now, agentPeriod{}, ""); e == nil {
			t.Errorf("expected clarification: %s", s)
		}
	}
	p, e := resolveAgentPeriod("sans les virements", now, previous, "")
	if e != nil || p.From != previous.From {
		t.Fatalf("follow-up lost period: %+v", p)
	}
}
func TestAgentCalendarBoundaries(t *testing.T) {
	loc, _ := agentLocation("Europe/Paris")
	for _, tt := range []struct{ day, text, from, to string }{{"2027-01-01", "mois dernier", "2026-12-01", "2026-12-31"}, {"2026-03-31", "ces 3 derniers mois", "2025-12-31", "2026-03-31"}, {"2026-03-29", "cette semaine", "2026-03-23", "2026-03-29"}} {
		now, _ := time.ParseInLocation("2006-01-02", tt.day, loc)
		p, e := resolveAgentPeriod(tt.text, now, agentPeriod{}, "")
		if e != nil || p.From != tt.from || p.To != tt.to {
			t.Fatalf("%s: %+v %v", tt.day, p, e)
		}
	}
	utc := time.Date(2026, 10, 31, 23, 30, 0, 0, time.UTC)
	p, e := resolveAgentPeriod("ce mois", utc.In(loc), agentPeriod{}, "")
	if e != nil || p.From != "2026-11-01" {
		t.Fatalf("timezone boundary: %+v %v", p, e)
	}
	if _, e = agentLocation("../../etc/passwd"); e == nil {
		t.Fatal("invalid timezone accepted")
	}
}

func TestAgentQualifiedTopUsesPlanner(t *testing.T) {
	for _, text := range []string{"ma top dépense chez TEST", "ma plus grosse dépense alimentaire du mois", "ma top dépense supérieure à 200 euros"} {
		if agentIntent(text, "") == "top_expense" {
			t.Fatalf("qualifier discarded: %s", text)
		}
	}
	if agentIntent("donne moi ma top dépense du mois", "") != "top_expense" {
		t.Fatal("simple request not recognized")
	}
}
