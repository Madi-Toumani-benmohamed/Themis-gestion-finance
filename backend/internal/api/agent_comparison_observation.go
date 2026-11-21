package api

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var agentComparisonSuffix = regexp.MustCompile(`(?:avec(?: le)?|au|par rapport au) mois (?:dernier|precedent)\s*[?.!]*$`)
var agentComparisonPrefix = regexp.MustCompile(`^(?:et\s+)?(?:(?:peux[- ]tu|pourrais[- ]tu)\s+)?(?:compare|comparer|comparaison)\b\s*`)
var agentComparisonMonth = regexp.MustCompile(`\b(?:janvier|fevrier|mars|avril|mai|juin|juillet|aout|septembre|octobre|novembre|decembre|\d{4}-\d{2})\b`)

type agentComparisonRequest struct {
	Base, Direction string
	CalendarLast    bool
}

func parseAgentComparison(text, previous string) (agentComparisonRequest, bool) {
	s := agentTemporalText(agentText(text))
	loc := agentComparisonSuffix.FindStringIndex(s)
	if loc == nil {
		return agentComparisonRequest{}, false
	}
	base := strings.TrimSpace(s[:loc[0]])
	if agentComparisonPrefix.MatchString(base) {
		base = agentComparisonPrefix.ReplaceAllString(base, "")
	} else if !strings.HasPrefix(s[loc[0]:], "par rapport au") || (base != "et" && base != "") {
		return agentComparisonRequest{}, false
	}
	if agentContains(base, "top", "gros", "grand", "important", "selection", "operation", "transaction", "celle", "celui", "cette depense") {
		return agentComparisonRequest{}, false
	}
	// A request containing several baseline months/years belongs to the planner.
	for _, pattern := range []*regexp.Regexp{agentComparisonMonth, agentYear} {
		seen := map[string]bool{}
		for _, m := range pattern.FindAllString(base, -1) {
			seen[m] = true
		}
		if len(seen) > 1 {
			return agentComparisonRequest{}, false
		}
	}
	hasIncome := agentContains(base, "revenu", "entree", "recu", "encaisse", "percu")
	hasExpense := agentContains(base, "depens", "sortie")
	direction := ""
	if hasIncome && agentSimpleIncome(base) {
		direction = "income"
	} else if hasExpense && agentSimpleSpending(base) {
		direction = "expense"
	} else if !hasIncome && !hasExpense && (agentSimpleIncome(base) || agentSimpleSpending(base)) {
		if previous == "income" {
			direction = "income"
		} else if previous == "expenses" {
			direction = "expense"
		}
	}
	return agentComparisonRequest{Base: base, Direction: direction, CalendarLast: strings.Contains(s[loc[0]:], "dernier")}, direction != ""
}

// Compare complete past months as months. For a partial month, align the day
// numbers and clamp to the preceding month's last day (including leap years).
func agentPreviousComparisonMonth(p agentPeriod, now time.Time, calendarLast bool) (agentPeriod, bool) {
	a, e := time.ParseInLocation("2006-01-02", p.From, now.Location())
	b, f := time.ParseInLocation("2006-01-02", p.To, now.Location())
	if e != nil || f != nil || a.Day() != 1 || a.Year() != b.Year() || a.Month() != b.Month() || b.Before(a) || b.After(now) {
		return agentPeriod{}, false
	}
	start := a.AddDate(0, -1, 0)
	if calendarLast {
		start = time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location())
	}
	last := start.AddDate(0, 1, -1)
	fullPast := p.To == a.AddDate(0, 1, -1).Format("2006-01-02") && a.Format("2006-01") < now.Format("2006-01")
	end := last
	if !fullPast {
		day := b.Day()
		if day > last.Day() {
			day = last.Day()
		}
		end = start.AddDate(0, 0, day-1)
	}
	if p.From == start.Format("2006-01-02") && p.To == end.Format("2006-01-02") {
		return agentPeriod{}, false
	}
	label := "Mois précédent, fenêtre alignée"
	if calendarLast {
		label = "Mois dernier, horloge serveur"
	}
	return agentPeriod{From: start.Format("2006-01-02"), To: end.Format("2006-01-02"), Origin: "comparison", Label: label}, true
}
func agentComparisonFullPast(p agentPeriod, now time.Time) bool {
	first, err := time.ParseInLocation("2006-01-02", p.From, now.Location())
	return err == nil && first.Day() == 1 && p.To == first.AddDate(0, 1, -1).Format("2006-01-02") && p.From[:7] < now.Format("2006-01")
}
func (s *Server) agentObservedComparison(ctx context.Context, owner int64, a, b agentPeriod, direction string, now time.Time) (*chatResult, error) {
	started := time.Now()
	tr := s.agentCompare(ctx, owner, map[string]any{"start_date_a": a.From, "end_date_a": a.To, "start_date_b": b.From, "end_date_b": b.To, "direction": direction}, a)
	if tr.Error != "" {
		return nil, fmt.Errorf("observations de comparaison indisponibles")
	}
	decoder := json.NewDecoder(strings.NewReader(tr.Content))
	decoder.UseNumber()
	var source map[string]any
	if decoder.Decode(&source) != nil {
		return nil, fmt.Errorf("source de comparaison illisible")
	}
	var parsed struct {
		CountA     int    `json:"count_a"`
		CountB     int    `json:"count_b"`
		Comparable bool   `json:"comparable"`
		Reason     string `json:"reason"`
	}
	if json.Unmarshal([]byte(tr.Content), &parsed) != nil {
		return nil, fmt.Errorf("comparaison illisible")
	}
	// The shared tool evaluates coverage in server time. Preserve the user's
	// local month boundary too: a current local month is never a finished month.
	if parsed.Comparable && (!agentComparisonFullPast(a, now) || !agentComparisonFullPast(b, now)) {
		parsed.Comparable = false
		parsed.Reason = "Une fenêtre est en cours ou partielle : la couverture comparable n’est pas établie."
		source["comparable"] = false
		source["reason"] = parsed.Reason
		for _, key := range []string{"income_change", "expense_change", "expense_change_percent"} {
			delete(source, key)
		}
	}
	key, label, change := "expenses", "dépenses enregistrées", "expense_change"
	if direction == "income" {
		key, label, change = "income", "entrées d’argent enregistrées", "income_change"
		delete(source, "expenses_a")
		delete(source, "expenses_b")
		delete(source, "expense_change")
		delete(source, "expense_change_percent")
	} else {
		delete(source, "income_a")
		delete(source, "income_b")
		delete(source, "income_change")
	}
	observation := func(p agentPeriod, count int, field string) (string, error) {
		if count == 0 {
			return "Du " + agentFrenchDate(p.From) + " au " + agentFrenchDate(p.To) + " : aucune opération correspondante enregistrée ; cela ne prouve pas un montant bancaire nul.", nil
		}
		n, ok := source[field].(json.Number)
		if !ok {
			return "", fmt.Errorf("montant source absent")
		}
		amount, e := parseAnalysisAmount(n.String())
		if e != nil || amount <= 0 {
			return "", fmt.Errorf("montant source invalide")
		}
		return fmt.Sprintf("Du %s au %s : %s € sur %d opération(s).", agentFrenchDate(p.From), agentFrenchDate(p.To), strings.ReplaceAll(amount.String(), ".", ","), count), nil
	}
	first, e := observation(a, parsed.CountA, key+"_a")
	if e != nil {
		return nil, e
	}
	second, e := observation(b, parsed.CountB, key+"_b")
	if e != nil {
		return nil, e
	}
	result := &chatResult{Status: "ok", Answer: "Voici les observations sur les " + label + ", avec leurs périodes exactes :\n" + first + "\n" + second, Evidence: []any{map[string]any{"type": "tool", "tool": "compare_periods", "data": source}}, Steps: []agentStep{{Name: "compare_periods", Status: "success", Duration: time.Since(started).Milliseconds(), From: a.From, To: a.To}}, DataQuality: map[string]any{"status": "unknown", "comparison_comparable": parsed.Comparable, "reason": parsed.Reason}}
	if parsed.CountA == 0 || parsed.CountB == 0 {
		result.Status = "insufficient_data"
	}
	if !parsed.Comparable {
		result.Answer += "\nLa couverture comparable n’est pas établie : ces observations ne suffisent pas à conclure à une hausse ou une baisse. Aucun pourcentage d’évolution n’est calculé."
	} else {
		n, ok := source[change].(json.Number)
		if !ok {
			return nil, fmt.Errorf("écart documenté absent")
		}
		amount, e := parseAnalysisAmount(n.String())
		if e != nil {
			return nil, e
		}
		result.Answer += "\nLes deux mois civils terminés sont documentés. Écart de la première période par rapport à la seconde : " + strings.ReplaceAll(amount.String(), ".", ",") + " €."
	}
	if b.Label == "Mois dernier, horloge serveur" {
		result.Answer += "\n« Le mois dernier » désigne ici le mois civil précédent à la date du jour."
	}
	result.Answer += "\nLes virements internes et l’épargne sont exclus. Les jours sont alignés lorsqu’une fenêtre est partielle, dans la limite du calendrier du mois précédent."
	if direction == "income" {
		result.Answer += " Ces crédits peuvent inclure des remboursements ; ils ne constituent pas un revenu fiscal calculé."
	}
	return result, nil
}
