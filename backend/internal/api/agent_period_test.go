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
	for _, text := range []string{"ma top dépense de 100 €", "ma top dépense de 99,99", "ma top dépense chez TEST", "ma plus grosse dépense alimentaire du mois", "ma top dépense supérieure à 200 euros"} {
		if agentIntent(text, "") == "top_expense" {
			t.Fatalf("qualifier discarded: %s", text)
		}
	}
	if agentIntent("donne moi ma top dépense du mois", "") != "top_expense" {
		t.Fatal("simple request not recognized")
	}
}

func TestAgentNaturalCalendar(t *testing.T) {
	loc, _ := agentLocation("Europe/Paris")
	now := time.Date(2026, 11, 19, 16, 0, 0, 0, loc)
	previous := agentPeriod{From: "2024-02-01", To: "2024-02-29"}
	for _, tt := range []struct{ text, from, to string }{
		{"ma top dépense du 1er au 15 novembre 2026", "2026-11-01", "2026-11-15"},
		{"entre le 3 et le 9 novembre", "2026-11-03", "2026-11-09"},
		{"du 3 novembre au 9", "2026-11-03", "2026-11-09"},
		{"depuis le 3 novembre", "2026-11-03", "2026-11-19"},
		{"le 3 novembre 2025", "2025-11-03", "2025-11-03"},
		{"du 3 novembre au 9 novembre 2025", "2025-11-03", "2025-11-09"},
		{"du 28 décembre au 3 janvier 2026", "2025-12-28", "2026-01-03"},
		{"de septembre à octobre 2025", "2025-09-01", "2025-10-31"},
		{"compare octobre et septembre", "2026-10-01", "2026-10-31"},
		{"du 03/11/2026 au 09/11/2026", "2026-11-03", "2026-11-09"},
		{"du 2026-11-03 au 9 novembre 2026", "2026-11-03", "2026-11-09"},
		{"novembre dernier", "2025-11-01", "2025-11-30"},
		{"décembre dernier", "2025-12-01", "2025-12-31"},
		{"le 3 novembre dernier", "2026-11-03", "2026-11-03"},
		{"la même période l’an dernier", "2023-02-01", "2023-02-28"},
		{"et l’année précédente ?", "2023-01-01", "2023-12-31"},
		{"les\tdeux\ndernières\tsemaines", "2026-11-06", "2026-11-19"},
		{"ces deux dernières semaines", "2026-11-06", "2026-11-19"},
		{"les trente derniers jours", "2026-10-21", "2026-11-19"},
		{"les deux dernières années", "2024-11-19", "2026-11-19"},
		{"depuis 2024", "2024-01-01", "2026-11-19"},
		{"depuis l’année 2025", "2025-01-01", "2026-11-19"},
		{"depuis 2025-12-01", "2025-12-01", "2026-11-19"},
		{"depuis 2025-12", "2025-12-01", "2026-11-19"},
		{"depuis 2024 jusqu’au 3 novembre 2025", "2024-01-01", "2025-11-03"},
		{"les dix derniers jours", "2026-11-10", "2026-11-19"},
		{"les 2 dernières semaines", "2026-11-06", "2026-11-19"},
		{"plus de 2000 euros en novembre", "2026-11-01", "2026-11-19"},
		{"novembre 2000 euros", "2026-11-01", "2026-11-19"},
		{"et en novembre ?", "2024-11-01", "2024-11-30"},
		{"compare ce mois au mois dernier", "2026-11-01", "2026-11-19"},
		{"ma top dépense de 2025", "2025-01-01", "2025-12-31"},
		{"bilan en 2025", "2025-01-01", "2025-12-31"},
	} {
		t.Run(tt.text, func(t *testing.T) {
			p, e := resolveAgentPeriod(tt.text, now, previous, "")
			if e != nil || p.From != tt.from || p.To != tt.to {
				t.Fatalf("%+v %v, expected %s/%s", p, e, tt.from, tt.to)
			}
		})
	}
	for _, text := range []string{"depuis 2000", "les 100 dernières années", "le 0 novembre", "le 123 novembre", "du 0 au 15 novembre", "00/11/2026", "les 1000 derniers jours", "les 1000 derniers mois", "les 100 dernières semaines", "le 31 novembre", "le 29 février 2025", "du 15 au 3 novembre", "31/02/2026", "du 01/01/2000 au 19 novembre 2026", "du 1er au 5 décembre 2026"} {
		if p, e := resolveAgentPeriod(text, now, previous, ""); e == nil {
			t.Fatalf("invalid date accepted %s: %+v", text, p)
		}
	}
	p, e := resolveAgentPeriod("et la semaine précédente ?", now, agentPeriod{From: "2026-10-05", To: "2026-10-11"}, "")
	if e != nil || p.From != "2026-09-28" || p.To != "2026-10-04" {
		t.Fatalf("contextual week: %+v %v", p, e)
	}
	p, e = resolveAgentPeriod("plus de 2000 euros", now, agentPeriod{}, "")
	if e != nil || p.From != "2026-11-01" {
		t.Fatalf("amount mistaken for year: %+v %v", p, e)
	}
}
