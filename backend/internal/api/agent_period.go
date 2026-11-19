package api

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"
)

type agentPeriod struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Label  string `json:"label"`
	Origin string `json:"origin"`
}
type agentPage struct {
	FiltersRequireSelection bool    `json:"filters_require_selection,omitempty"`
	Page                    string  `json:"page,omitempty"`
	Month                   string  `json:"month,omitempty"`
	TransactionIDs          []int64 `json:"transaction_ids,omitempty"`
	Tag                     string  `json:"tag,omitempty"`
}
type agentContext struct {
	DirectSpending bool        `json:"direct_spending,omitempty"`
	TargetMonth    string      `json:"target_month,omitempty"`
	FocusedIDs     []int64     `json:"focused_ids,omitempty"`
	Timezone       string      `json:"timezone"`
	Today          string      `json:"today"`
	Period         agentPeriod `json:"period"`
	Intent         string      `json:"intent,omitempty"`
	Page           agentPage   `json:"page_context"`
}

var agentISODate = regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}\b`)
var agentISOMonth = regexp.MustCompile(`\b\d{4}-\d{2}\b`)
var agentYear = regexp.MustCompile(`\b(20\d{2}|19\d{2})\b`)
var agentDays = regexp.MustCompile(`(?:ces|les|depuis(?: les)?) (\d+)(?: derniers?)? jours?|(\d+) derniers? jours?|derniers? (\d+) jours?`)
var agentShortRangeNumberPrefix = regexp.MustCompile(strings.TrimSuffix(agentShortRangeStart.String(), "$"))
var agentPriceQualifier = regexp.MustCompile(`[€$]|\b(?:euros?|eur)\b|\d+[.,]\d+|\b(?:plus|moins)\s+de\s+\d|[<>≤≥]`)
var agentMonths = regexp.MustCompile(`(\d+) derniers mois`)

func agentText(s string) string {
	return strings.ToLower(strings.NewReplacer("é", "e", "è", "e", "ê", "e", "ë", "e", "à", "a", "â", "a", "î", "i", "ï", "i", "ô", "o", "ù", "u", "û", "u", "ç", "c", "’", "'").Replace(strings.ToLower(s)))
}
func agentContains(s string, words ...string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}
func agentLocation(zone string) (*time.Location, error) {
	if zone == "" {
		zone = "Europe/Paris"
	}
	if len(zone) > 80 {
		return nil, fmt.Errorf("fuseau horaire invalide")
	}
	l, e := time.LoadLocation(zone)
	if e != nil {
		return nil, fmt.Errorf("fuseau horaire invalide")
	}
	return l, nil
}
func agentMonthRange(t time.Time, now time.Time, origin string) agentPeriod {
	a := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	b := a.AddDate(0, 1, -1)
	if b.After(now) {
		b = now
	}
	return agentPeriod{a.Format("2006-01-02"), b.Format("2006-01-02"), a.Format("2006-01"), origin}
}
func resolveAgentPeriod(text string, now time.Time, previous agentPeriod, selected string) (agentPeriod, error) {
	s := agentTemporalText(agentText(text))
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	makeRange := func(a, b time.Time, label, origin string) (agentPeriod, error) {
		if a.After(b) || a.After(today) {
			return agentPeriod{}, fmt.Errorf("la période demandée est future ou inversée")
		}
		if b.After(today) {
			b = today
		}
		p := agentPeriod{a.Format("2006-01-02"), b.Format("2006-01-02"), label, origin}
		if !agentPeriodValid(p) {
			return agentPeriod{}, fmt.Errorf("période trop large : dix ans maximum")
		}
		return p, nil
	}
	if p, matched, err := agentFrenchCalendar(s, now, previous); matched {
		return p, err
	}
	if p, matched, err := agentContextualCalendar(s, now, previous); matched {
		return p, err
	}
	if agentContains(s, "mois prochain") {
		if agentContains(s, "budget", "prepar", "objectif", "mission") {
			p := agentMonthRange(first, today, "planning_baseline")
			return p, nil
		}
		return agentPeriod{}, fmt.Errorf("le mois prochain n’est pas encore observé ; souhaites-tu préparer un scénario ?")
	}
	dates := agentISODate.FindAllString(s, -1)
	if len(dates) == 4 && agentContains(s, "compare", "comparaison") {
		dates = dates[:2]
	} else if len(dates) > 2 {
		return agentPeriod{}, fmt.Errorf("plusieurs périodes : préciser la comparaison")
	}
	if len(dates) > 0 {
		a, e := time.ParseInLocation("2006-01-02", dates[0], now.Location())
		if e != nil {
			return agentPeriod{}, fmt.Errorf("date invalide")
		}
		b := today
		if len(dates) == 2 {
			b, e = time.ParseInLocation("2006-01-02", dates[1], now.Location())
			if e != nil {
				return agentPeriod{}, fmt.Errorf("date invalide")
			}
		} else if !strings.Contains(s, "depuis") {
			b = a
		}
		return makeRange(a, b, "Dates demandées", "explicit")
	}
	if agentContains(s, "depuis mon salaire", "depuis le salaire", "depuis ma paie", "depuis mon dernier salaire") {
		return agentPeriod{}, fmt.Errorf("quel versement correspond au salaire à utiliser comme point de départ ?")
	}
	if m := agentISOMonth.FindString(s); m != "" {
		t, e := time.ParseInLocation("2006-01", m, now.Location())
		if e != nil {
			return agentPeriod{}, fmt.Errorf("mois invalide")
		}
		if strings.Contains(s, "depuis") {
			return makeRange(t, today, "Depuis le mois demandé", "explicit")
		}
		p := agentMonthRange(t, today, "explicit")
		if p.From > p.To {
			return agentPeriod{}, fmt.Errorf("la période demandée est future")
		}
		return p, nil
	}
	if agentContains(s, "avant-hier") {
		return makeRange(today.AddDate(0, 0, -2), today.AddDate(0, 0, -2), "Avant-hier", "relative")
	}
	if strings.Contains(s, "hier") {
		return makeRange(today.AddDate(0, 0, -1), today.AddDate(0, 0, -1), "Hier", "relative")
	}
	if agentContains(s, "aujourd'hui", "aujourdhui") {
		return makeRange(today, today, "Aujourd’hui", "relative")
	}
	if strings.Contains(s, "semaine") {
		offset := (int(today.Weekday()) + 6) % 7
		a := today.AddDate(0, 0, -offset)
		b := today
		if agentContains(s, "derniere", "precedente") {
			a = a.AddDate(0, 0, -7)
			b = a.AddDate(0, 0, 6)
		}
		return makeRange(a, b, "Semaine demandée (lundi–dimanche)", "relative")
	}
	if m := agentDays.FindStringSubmatch(s); m != nil {
		raw := ""
		for _, v := range m[1:] {
			if v != "" {
				raw = v
				break
			}
		}
		n, _ := strconv.Atoi(raw)
		if n < 1 || n > 366 {
			return agentPeriod{}, fmt.Errorf("nombre de jours invalide")
		}
		return makeRange(today.AddDate(0, 0, 1-n), today, "Derniers jours, aujourd’hui inclus", "relative")
	}
	if agentContains(s, "annee derniere", "an dernier", "annee precedente") {
		a := time.Date(now.Year()-1, 1, 1, 0, 0, 0, 0, now.Location())
		return makeRange(a, a.AddDate(1, 0, -1), "Année civile précédente", "relative")
	}
	if agentContains(s, "cette annee", "cette année") {
		return makeRange(time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location()), today, "Cette année", "relative")
	}
	if m := agentMonths.FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		if n < 1 || n > 120 {
			return agentPeriod{}, fmt.Errorf("fenêtre mensuelle invalide : 1 à 120 mois")
		}
		a := first.AddDate(0, -n, 0)
		day := today.Day()
		if last := a.AddDate(0, 1, -1).Day(); day > last {
			day = last
		}
		return makeRange(a.AddDate(0, 0, day-1), today, "Mois glissants demandés", "relative")
	}
	if agentContains(s, "compare", "comparaison") && agentContains(s, "ce mois") {
		return agentMonthRange(first, today, "relative"), nil
	}

	if agentContains(s, "mois dernier", "dernier mois") {
		return agentMonthRange(first.AddDate(0, -1, 0), today, "relative"), nil
	}
	if agentContains(s, "mois precedent", "mois d'avant", "et avant") {
		base := first
		if previous.From != "" {
			if t, e := time.ParseInLocation("2006-01-02", previous.From, now.Location()); e == nil {
				base = t
			}
		}
		return agentMonthRange(base.AddDate(0, -1, 0), today, "conversation"), nil
	}
	if agentContains(s, "ce mois", "du mois", "mois en cours") {
		return agentMonthRange(first, today, "relative"), nil
	}

	if y := agentRequestedYear(s); y != "" {
		year, _ := strconv.Atoi(y)
		a := time.Date(year, 1, 1, 0, 0, 0, 0, now.Location())
		return makeRange(a, a.AddDate(1, 0, -1), y, "explicit")
	}
	if selected != "" {
		if t, e := time.ParseInLocation("2006-01", selected, now.Location()); e == nil {
			p := agentMonthRange(t, today, "page")
			if p.From > p.To {
				return agentPeriod{}, fmt.Errorf("mois affiché futur")
			}
			return p, nil
		}
	}
	if previous.From != "" {
		previous.Origin = "conversation"
		return previous, nil
	}
	return agentMonthRange(first, today, "default"), nil
}
func agentIntent(text, previous string) string {
	s := agentText(text)
	if agentContains(s, "compare", "comparaison") {
		return ""
	}
	if agentContains(s, " chez ", " categorie ", "detaille celle") {
		return ""
	}
	if agentContains(s, "plus grosse depense", "plus grande depense", "top depense", "depense la plus importante", "plus gros achat") {
		if agentSimpleTop(s) {
			return "top_expense"
		}
		return ""
	}
	if agentContains(s, "combien", "total", "montant", "somme") && agentContains(s, "depens", "sorties") {
		return "expenses"
	}
	if agentContains(s, "categorie", "ou je depense", "ou va mon argent") {
		return "categories"
	}
	if agentContains(s, "et le mois", "et avant", "et en", "et hier", "et aujourd", "et cette semaine", "et cette annee", "et ce mois", "meme periode", "sans les virements", "detaille celle", "et pour") {
		if previous == "top_expense" && !agentSimpleTop(s) {
			return ""
		}
		return previous
	}
	return ""
}
func agentPeriodValid(p agentPeriod) bool {
	a, e := time.Parse("2006-01-02", p.From)
	b, f := time.Parse("2006-01-02", p.To)
	return e == nil && f == nil && !a.After(b) && b.Sub(a) <= 3660*24*time.Hour
}

// A qualified request (merchant, category, amount, etc.) must use the tools
// planner rather than silently discarding a qualifier in the fast path.
func agentSimpleTop(text string) bool {
	text = agentTemporalText(agentText(text))
	if agentPriceQualifier.MatchString(text) {
		return false
	}
	// Numbers are accepted only in recognized calendar expressions. A bare
	// amount or ranking constraint must stay with the planner.
	residue := text
	for _, pattern := range []*regexp.Regexp{agentISODate, agentNumericDate, agentISOMonth, agentNamedDate, agentYear, agentDays, agentMonths, agentRollingWeeks, agentRollingYears} {
		residue = pattern.ReplaceAllString(residue, " ")
	}
	if agentNamedDate.MatchString(text) {
		residue = agentShortRangeNumberPrefix.ReplaceAllString(residue, " ")
	}
	if regexp.MustCompile(`\d`).MatchString(residue) {
		return false
	}
	allowed := map[string]bool{}
	for _, word := range strings.Fields("donne moi ma mon mes la le les un une top plus grosse gros grande depense importante operation transaction transactions achat du de des ce mois ci en pour au aux et dernier derniers precedent precedente cette semaine annee aujourd hui aujourdhui hier avant er quelle quel est sont s il te plait montre affiche peux tu pourrais janvier fevrier mars avril mai juin juillet aout septembre octobre novembre decembre depuis jours ces ici selection operations celle meme sans virements") {
		allowed[word] = true
	}
	for _, word := range regexp.MustCompile(`[a-z]+`).FindAllString(text, -1) {
		if !allowed[word] {
			return false
		}
	}
	return true
}
