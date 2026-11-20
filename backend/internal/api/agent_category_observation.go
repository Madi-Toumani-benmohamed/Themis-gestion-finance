package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var agentCategoryVocabulary = regexp.MustCompile(`\bpar categories?\b`)
var agentNamedCategoryAfterGroup = regexp.MustCompile(`\bpar categories?\s+(?:revenus?|depenses?|entrees?)\b`)
var agentCategoryQuestion = regexp.MustCompile(`^(?:(?:et|peux[- ]tu|pourrais[- ]tu)\s+)?(?:comment\s+se\s+repartissent|repartis|repartir|repartition|ventile|ventilation|regroupe|detailler?)\b`)

func agentCategoryScope(text string) bool {
	s := agentTemporalText(agentText(text))
	if agentContains(s, "top", "gros", "grand", "important", "compare") || agentNamedCategoryAfterGroup.MatchString(s) {
		return false
	}
	s = strings.TrimPrefix(s, "ou va mon argent")
	// Remove only the grouping phrase. A second category or grouping remains
	// a constraint for the planner, rather than disappearing from the query.
	s = agentCategoryVocabulary.ReplaceAllString(s, " ")
	s = agentCategoryQuestion.ReplaceAllString(s, " ")
	return agentSimpleIncome(s) || agentSimpleSpending(s)
}
func agentCategoryIntent(text, previous string) string {
	s := agentTemporalText(agentText(text))
	if (!agentCategoryVocabulary.MatchString(s) && !strings.Contains(s, "ou va mon argent")) || !agentCategoryScope(text) {
		return ""
	}
	if agentContains(s, "revenu", "entree", "recu", "encaisse", "percu") || (!agentContains(s, "depens", "sortie", "argent") && (previous == "income" || previous == "income_categories")) {
		return "income_categories"
	}
	return "expense_categories"
}
func agentCategoryDirection(intent string) string {
	if intent == "income_categories" {
		return "income"
	}
	if intent == "expense_categories" {
		return "expense"
	}
	return ""
}
func (s *Server) agentObservedCategories(ctx context.Context, owner int64, c agentContext, text, direction string) (*chatResult, error) {
	started := time.Now()
	tr := agentCategoryBreakdown(ctx, s.DB, owner, map[string]any{"start_date": c.Period.From, "end_date": c.Period.To, "direction": direction}, agentScopedIDs(c, text))
	if tr.Error != "" {
		return nil, fmt.Errorf("répartition indisponible")
	}
	var parsed struct {
		Categories []struct {
			Category string      `json:"category"`
			Amount   json.Number `json:"amount"`
			Count    int         `json:"transaction_count"`
		} `json:"categories"`
		Truncated bool `json:"truncated"`
	}
	if json.Unmarshal([]byte(tr.Content), &parsed) != nil {
		return nil, fmt.Errorf("répartition illisible")
	}
	decoder := json.NewDecoder(strings.NewReader(tr.Content))
	decoder.UseNumber()
	var source map[string]any
	if decoder.Decode(&source) != nil {
		return nil, fmt.Errorf("source illisible")
	}
	var latest sql.NullString
	if err := s.DB.QueryRowContext(ctx, `SELECT max(tx_date)::text FROM transactions WHERE owner_id=$1`, owner).Scan(&latest); err != nil {
		return nil, err
	}
	result := &chatResult{Status: "ok", Evidence: []any{map[string]any{"type": "tool", "tool": "get_category_breakdown", "data": source}}, Steps: []agentStep{{Name: "get_category_breakdown", Status: "success", Duration: time.Since(started).Milliseconds(), From: c.Period.From, To: c.Period.To}}, DataQuality: map[string]any{"status": "unknown", "last_transaction_date": agentNullableString(latest), "query_from": c.Period.From, "query_to": c.Period.To, "reason": "Les opérations importées ne prouvent pas la couverture complète de la période."}}
	label := "dépenses enregistrées"
	if direction == "income" {
		label = "entrées d’argent enregistrées"
	}
	if len(parsed.Categories) == 0 {
		result.Status = "insufficient_data"
		result.Answer = "Je n’ai aucune opération correspondant aux " + label + " " + agentPeriodWording(c.Period) + " dans ce périmètre. Cela ne prouve pas un montant bancaire nul."
		if latest.Valid {
			result.Answer += " La dernière opération enregistrée date du " + agentFrenchDate(latest.String) + "."
		}
		return result, nil
	}
	lines := []string{}
	for i, category := range parsed.Categories {
		amount, err := parseAnalysisAmount(category.Amount.String())
		if err != nil || amount <= 0 || category.Count < 1 {
			return nil, fmt.Errorf("montant de catégorie invalide")
		}
		if i < 6 {
			lines = append(lines, fmt.Sprintf("• %s : %s € (%d opération(s))", category.Category, strings.ReplaceAll(amount.String(), ".", ","), category.Count))
		}
	}
	result.Answer = "Pour les données " + agentPeriodWording(c.Period) + ", voici la répartition des " + label + " par catégorie :\n" + strings.Join(lines, "\n")
	if len(parsed.Categories) > 6 {
		result.Answer += "\nLes six premières catégories sont résumées ici ; les autres sont consultables dans les données sources."
	}
	if parsed.Truncated {
		result.Answer += "\nLa source est limitée aux 200 premières catégories ; cette liste ne doit pas être additionnée comme un total complet."
	}
	result.Answer += "\nLes virements internes et l’épargne sont exclus. La couverture bancaire complète de la période n’est pas établie."
	if direction == "income" {
		result.Answer += " Ces crédits peuvent inclure des remboursements ; ils ne constituent pas un revenu fiscal calculé."
	}
	return result, nil
}
