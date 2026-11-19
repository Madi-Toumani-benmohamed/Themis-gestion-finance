package api

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var agentMonthNames = []string{"janvier", "fevrier", "mars", "avril", "mai", "juin", "juillet", "aout", "septembre", "octobre", "novembre", "decembre"}
var agentNamedDate = regexp.MustCompile(`\b(?:(\d+)(?:er)?\s+(?:de\s+)?)?(janvier|fevrier|mars|avril|mai|juin|juillet|aout|septembre|octobre|novembre|decembre)(?:\s+((?:19|20)\d{2}))?\b`)
var agentNumericDate = regexp.MustCompile(`\b(\d{1,2})/(\d{1,2})/((?:19|20)\d{2})\b`)
var agentShortRangeStart = regexp.MustCompile(`\b(?:du|entre(?: le)?)\s+(\d+)(?:er)?\s+(?:au|et(?: le)?)\s+$`)
var agentShortRangeEnd = regexp.MustCompile(`^\s+(?:au|et(?: le)?)\s+(\d+)(?:er)?\b`)
var agentYearContext = regexp.MustCompile(`\b(?:en|de|annee|bilan(?: de)?|pour)\s+((?:19|20)\d{2})\b`)
var agentCurrencySuffix = regexp.MustCompile(`^\s*(?:€|euros?\b|eur\b)`)
var agentCalendarFollowup = regexp.MustCompile(`^(?:et(?: en| pour)?|puis|idem|meme chose)\b`)
var agentSinceYear = regexp.MustCompile(`\bdepuis(?: l'annee)?\s+((?:19|20)\d{2})\b`)
var agentRollingYears = regexp.MustCompile(`\b(\d+) dernieres? annees?\b`)
var agentRollingWeeks = regexp.MustCompile(`\b(\d+) dernieres? semaines?\b`)

var agentTemporalQuantity = regexp.MustCompile(`\b(un|une|deux|trois|quatre|cinq|six|sept|huit|neuf|dix|douze|vingt|trente|quarante|cinquante|soixante)\s+((?:derniers?|dernieres?)\s+(?:jours?|semaines?|mois|annees?))\b`)

func agentTemporalText(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	quantities := map[string]string{"un": "1", "une": "1", "deux": "2", "trois": "3", "quatre": "4", "cinq": "5", "six": "6", "sept": "7", "huit": "8", "neuf": "9", "dix": "10", "douze": "12", "vingt": "20", "trente": "30", "quarante": "40", "cinquante": "50", "soixante": "60"}
	return agentTemporalQuantity.ReplaceAllStringFunc(s, func(match string) string {
		parts := strings.SplitN(match, " ", 2)
		return quantities[parts[0]] + " " + parts[1]
	})
}

type agentCalendarDate struct{ start, end, day, month, year int }

func agentCalendarDay(year, month, day int, zone *time.Location) (time.Time, error) {
	d := time.Date(year, time.Month(month), day, 0, 0, 0, 0, zone)
	if d.Year() != year || int(d.Month()) != month || d.Day() != day {
		return time.Time{}, fmt.Errorf("date invalide : vérifier le jour, le mois et l’année")
	}
	return d, nil
}
func agentShiftYear(day time.Time, delta int) time.Time {
	y, m := day.Year()+delta, day.Month()
	last := time.Date(y, m+1, 0, 0, 0, 0, 0, day.Location()).Day()
	n := day.Day()
	if n > last {
		n = last
	}
	return time.Date(y, m, n, 0, 0, 0, 0, day.Location())
}
func agentCalendarRange(a, b, now time.Time, label, origin string) (agentPeriod, bool, error) {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if a.After(b) || a.After(today) {
		return agentPeriod{}, true, fmt.Errorf("la période demandée est future ou inversée")
	}
	if b.After(today) {
		b = today
	}
	p := agentPeriod{From: a.Format("2006-01-02"), To: b.Format("2006-01-02"), Label: label, Origin: origin}
	if !agentPeriodValid(p) {
		return agentPeriod{}, true, fmt.Errorf("période trop large : dix ans maximum")
	}
	return p, true, nil
}

// Parse explicit calendar expressions before relative defaults. Years next to
// amounts are never inferred as calendar years. Dates are validated, not rolled
// into the next month by time.Date.
func agentFrenchCalendar(s string, now time.Time, previous agentPeriod) (agentPeriod, bool, error) {
	var dates []agentCalendarDate
	for _, m := range agentNamedDate.FindAllStringSubmatchIndex(s, -1) {
		d := agentCalendarDate{start: m[0], end: m[1]}
		if m[2] >= 0 {
			d.day, _ = strconv.Atoi(s[m[2]:m[3]])
			if d.day < 1 || d.day > 31 {
				return agentPeriod{}, true, fmt.Errorf("jour invalide")
			}
		}
		name := s[m[4]:m[5]]
		for i, n := range agentMonthNames {
			if name == n {
				d.month = i + 1
				break
			}
		}
		if m[6] >= 0 && !agentCurrencySuffix.MatchString(s[m[7]:]) {
			d.year, _ = strconv.Atoi(s[m[6]:m[7]])
		}
		dates = append(dates, d)
	}
	for _, m := range agentNumericDate.FindAllStringSubmatchIndex(s, -1) {
		d := agentCalendarDate{start: m[0], end: m[1]}
		d.day, _ = strconv.Atoi(s[m[2]:m[3]])
		d.month, _ = strconv.Atoi(s[m[4]:m[5]])
		d.year, _ = strconv.Atoi(s[m[6]:m[7]])
		if d.day < 1 || d.day > 31 || d.month < 1 || d.month > 12 {
			return agentPeriod{}, true, fmt.Errorf("date invalide")
		}
		dates = append(dates, d)
	}
	for _, m := range agentSinceYear.FindAllStringSubmatchIndex(s, -1) {
		suffix := s[m[1]:]
		if strings.HasPrefix(suffix, "-") || strings.HasPrefix(suffix, "/") || agentCurrencySuffix.MatchString(suffix) {
			continue
		}
		year, _ := strconv.Atoi(s[m[2]:m[3]])
		dates = append(dates, agentCalendarDate{start: m[0], end: m[1], day: 1, month: 1, year: year})
	}
	if len(dates) == 0 {
		return agentPeriod{}, false, nil
	}
	// Allow mixed ISO/French endpoints without silently dropping either one.
	for _, m := range agentISODate.FindAllStringIndex(s, -1) {
		t, e := time.ParseInLocation("2006-01-02", s[m[0]:m[1]], now.Location())
		if e != nil {
			return agentPeriod{}, true, fmt.Errorf("date invalide")
		}
		dates = append(dates, agentCalendarDate{start: m[0], end: m[1], day: t.Day(), month: int(t.Month()), year: t.Year()})
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i].start < dates[j].start })
	if len(dates) == 1 && dates[0].day > 0 {
		d := dates[0]
		if m := agentShortRangeStart.FindStringSubmatch(s[:d.start]); m != nil {
			n, _ := strconv.Atoi(m[1])
			if n < 1 || n > 31 {
				return agentPeriod{}, true, fmt.Errorf("jour invalide")
			}
			a := d
			a.day = n
			dates = []agentCalendarDate{a, d}
		} else if m := agentShortRangeEnd.FindStringSubmatch(s[d.end:]); m != nil {
			n, _ := strconv.Atoi(m[1])
			if n < 1 || n > 31 {
				return agentPeriod{}, true, fmt.Errorf("jour invalide")
			}
			b := d
			b.day = n
			dates = append(dates, b)
		}
	}
	comparison := agentContains(s, "compare", "comparaison")
	if len(dates) > 2 {
		if comparison && len(dates) == 4 {
			dates = dates[:2]
		} else {
			return agentPeriod{}, true, fmt.Errorf("plusieurs périodes : préciser la comparaison")
		}
	}
	// A year on either endpoint applies to the other endpoint. An omitted year
	// on the first endpoint crosses December/January only when needed.
	defaultYear := now.Year()
	if agentCalendarFollowup.MatchString(strings.TrimSpace(s)) && agentPeriodValid(previous) {
		base, _ := time.Parse("2006-01-02", previous.From)
		defaultYear = base.Year()
	}
	for _, d := range dates {
		if d.year > 0 {
			defaultYear = d.year
			break
		}
	}
	firstYearMissing := dates[0].year == 0
	for i := range dates {
		if dates[i].year == 0 {
			dates[i].year = defaultYear
		}
	}
	if len(dates) == 2 && !comparison && firstYearMissing && dates[0].month > dates[1].month {
		dates[0].year--
	}
	if len(dates) == 1 && firstYearMissing && agentContains(s[dates[0].end:], "dernier", "derniere") {
		d := &dates[0]
		if d.month > int(now.Month()) || (d.month == int(now.Month()) && (d.day == 0 || d.day > now.Day())) {
			d.year--
		}
	}

	aDay := dates[0].day
	if aDay == 0 {
		aDay = 1
	}
	a, e := agentCalendarDay(dates[0].year, dates[0].month, aDay, now.Location())
	if e != nil {
		return agentPeriod{}, true, e
	}
	b := a
	if len(dates) == 2 {
		d := dates[1]
		day := d.day
		if day == 0 {
			day = time.Date(d.year, time.Month(d.month)+1, 0, 0, 0, 0, 0, now.Location()).Day()
		}
		b, e = agentCalendarDay(d.year, d.month, day, now.Location())
		if e != nil {
			return agentPeriod{}, true, e
		}
	} else if agentContains(s[:dates[0].start], "depuis") || agentSinceYear.MatchString(s[dates[0].start:dates[0].end]) {
		b = now
	} else if dates[0].day == 0 {
		b = a.AddDate(0, 1, -1)
	}
	// Two month names in a comparison identify the primary month here; the
	// comparison tool receives the original question and the secondary period.
	if comparison && len(dates) == 2 && dates[0].day == 0 && dates[1].day == 0 {
		b = a.AddDate(0, 1, -1)
	}
	return agentCalendarRange(a, b, now, "Dates demandées", "explicit")
}
func agentRequestedYear(s string) string {
	trimmed := strings.Trim(s, " \t\n?.!")
	if agentYear.FindString(trimmed) == trimmed {
		return trimmed
	}
	for _, m := range agentYearContext.FindAllStringSubmatchIndex(s, -1) {
		if !agentCurrencySuffix.MatchString(s[m[3]:]) {
			return s[m[2]:m[3]]
		}
	}
	return ""
}
func agentContextualCalendar(s string, now time.Time, previous agentPeriod) (agentPeriod, bool, error) {
	if agentContains(s, "meme periode") && agentContains(s, "an dernier", "annee derniere", "annee precedente") {
		a := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		b := now
		if agentPeriodValid(previous) {
			a, _ = time.ParseInLocation("2006-01-02", previous.From, now.Location())
			b, _ = time.ParseInLocation("2006-01-02", previous.To, now.Location())
		}
		return agentCalendarRange(agentShiftYear(a, -1), agentShiftYear(b, -1), now, "Même période un an avant", "conversation")
	}
	if m := agentRollingYears.FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		if n < 1 || n > 10 {
			return agentPeriod{}, true, fmt.Errorf("fenêtre annuelle invalide : 1 à 10 années")
		}
		return agentCalendarRange(agentShiftYear(now, -n), now, now, "Années glissantes demandées", "relative")
	}
	if m := agentRollingWeeks.FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		if n < 1 || n > 52 {
			return agentPeriod{}, true, fmt.Errorf("fenêtre hebdomadaire invalide : 1 à 52 semaines")
		}
		return agentCalendarRange(now.AddDate(0, 0, 1-7*n), now, now, "Semaines glissantes, aujourd’hui inclus", "relative")
	}
	if agentContains(s, "annee precedente") && agentPeriodValid(previous) {
		base, _ := time.ParseInLocation("2006-01-02", previous.From, now.Location())
		a := time.Date(base.Year()-1, 1, 1, 0, 0, 0, 0, now.Location())
		return agentCalendarRange(a, a.AddDate(1, 0, -1), now, "Année civile précédente dans la conversation", "conversation")
	}
	if agentContains(s, "semaine precedente") && agentPeriodValid(previous) {
		base, _ := time.ParseInLocation("2006-01-02", previous.From, now.Location())
		offset := (int(base.Weekday()) + 6) % 7
		a := base.AddDate(0, 0, -offset-7)
		return agentCalendarRange(a, a.AddDate(0, 0, 6), now, "Semaine précédente dans la conversation", "conversation")
	}
	return agentPeriod{}, false, nil
}
