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

var agentSpendingQuestion = regexp.MustCompile(`\b(?:combien|ai|je|j|d|que|total|montant|somme|depenses|depensees|sorties|tout|argent|sur|pendant|durant|dans)\b`)

var agentIncomeQuestion = regexp.MustCompile(`\b(?:combien|ai|je|j|d|que|total|montant|somme|revenus?|recu|recus|entrees?|encaisse|encaissements?|percu|percus|sur|pendant|durant|dans)\b`)

func agentSimpleIncome(text string) bool {
	normalized := agentTemporalText(agentText(text))
	if agentContains(normalized, "depens", "achat", "sortie", "top", "gros", "grand", "compar") {
		return false
	}
	return agentSimpleTop(agentIncomeQuestion.ReplaceAllString(normalized, " "))
}

// Only unqualified observation questions bypass the planner. Merchant,
// category, amount, comparison and planning qualifiers still use its tools.
func agentSimpleSpending(text string) bool {
	s := agentTemporalText(agentText(text))
	return agentSimpleTop(agentSpendingQuestion.ReplaceAllString(s, " "))
}
func (s *Server) agentObservedCashflow(ctx context.Context, owner int64, c agentContext, text string, direction string) (*chatResult, error) {
	if direction != "expense" && direction != "income" {
		return nil, fmt.Errorf("direction d’observation invalide")
	}
	noun, description, missing := "dépense", "tes dépenses enregistrées", "Cela ne signifie pas que tu n’as rien dépensé."
	if direction == "income" {
		noun, description, missing = "revenu", "tes entrées d’argent enregistrées", "Cela ne signifie pas que tu n’as aucun revenu."
	}
	started := time.Now()
	tr := agentFinancialSummary(ctx, s.DB, owner, map[string]any{"start_date": c.Period.From, "end_date": c.Period.To, "direction": direction}, agentScopedIDs(c, text))
	if tr.Error != "" {
		return nil, fmt.Errorf("synthèse financière indisponible")
	}
	var summary struct {
		Available bool        `json:"available"`
		Expenses  json.Number `json:"expenses"`
		Income    json.Number `json:"income"`
		Count     int         `json:"transaction_count"`
	}
	if json.Unmarshal([]byte(tr.Content), &summary) != nil {
		return nil, fmt.Errorf("synthèse illisible")
	}
	decoder := json.NewDecoder(strings.NewReader(tr.Content))
	decoder.UseNumber()
	var source map[string]any
	if decoder.Decode(&source) != nil {
		return nil, fmt.Errorf("source illisible")
	}
	// A directional query must not claim observations of the other direction
	// or a net balance of all cash flows.
	if direction == "income" {
		delete(source, "expenses")
	} else {
		delete(source, "income")
	}
	delete(source, "net")
	var latest sql.NullString
	if err := s.DB.QueryRowContext(ctx, `SELECT max(tx_date)::text FROM transactions WHERE owner_id=$1`, owner).Scan(&latest); err != nil {
		return nil, err
	}
	result := &chatResult{Status: "ok", Evidence: []any{map[string]any{"type": "tool", "tool": "get_financial_summary", "data": source}}, Steps: []agentStep{{Name: "get_financial_summary", Status: "success", Duration: time.Since(started).Milliseconds(), From: c.Period.From, To: c.Period.To}}}
	result.DataQuality = map[string]any{"status": "unknown", "last_transaction_date": agentNullableString(latest), "query_from": c.Period.From, "query_to": c.Period.To, "reason": "Les opérations enregistrées ne prouvent pas la couverture complète de la période."}
	if !summary.Available {
		result.Status = "insufficient_data"
		result.Answer = "Je n’ai aucune opération de " + noun + " enregistrée " + agentPeriodWording(c.Period) + " dans ce périmètre. " + missing
		if latest.Valid {
			result.Answer += " La dernière opération enregistrée date du " + agentFrenchDate(latest.String) + "."
		}
		return result, nil
	}
	raw := summary.Expenses
	if direction == "income" {
		raw = summary.Income
	}
	amount, err := parseAnalysisAmount(raw.String())
	if err != nil || amount <= 0 || summary.Count < 1 {
		return nil, fmt.Errorf("montants sources invalides")
	}
	operation := "opération de " + noun
	if summary.Count > 1 {
		operation = "opérations de " + noun
	}
	result.Answer = fmt.Sprintf("Pour les données %s, %s représentent %s €, sur %d %s. Les virements internes et l’épargne sont exclus. La couverture bancaire complète de cette période n’est pas établie.", agentPeriodWording(c.Period), description, strings.ReplaceAll(amount.String(), ".", ","), summary.Count, operation)
	if direction == "income" {
		result.Answer += " Ces crédits peuvent inclure des remboursements ; ils ne constituent pas un revenu fiscal calculé."
	}
	return result, nil
}
