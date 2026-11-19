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

// Only unqualified observation questions bypass the planner. Merchant,
// category, amount, comparison and planning qualifiers still use its tools.
func agentSimpleSpending(text string) bool {
	s := agentTemporalText(agentText(text))
	return agentSimpleTop(agentSpendingQuestion.ReplaceAllString(s, " "))
}
func (s *Server) agentObservedSpending(ctx context.Context, owner int64, c agentContext, text string) (*chatResult, error) {
	started := time.Now()
	tr := agentFinancialSummary(ctx, s.DB, owner, map[string]any{"start_date": c.Period.From, "end_date": c.Period.To, "direction": "expense"}, agentScopedIDs(c, text))
	if tr.Error != "" {
		return nil, fmt.Errorf("synthèse des dépenses indisponible")
	}
	var summary struct {
		Available bool        `json:"available"`
		Expenses  json.Number `json:"expenses"`
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
	// The query intentionally excludes income. Do not present its zero or
	// negative net as an observation of all cash flows in the period.
	delete(source, "income")
	delete(source, "net")
	var latest sql.NullString
	if err := s.DB.QueryRowContext(ctx, `SELECT max(tx_date)::text FROM transactions WHERE owner_id=$1`, owner).Scan(&latest); err != nil {
		return nil, err
	}
	result := &chatResult{Status: "ok", Evidence: []any{map[string]any{"type": "tool", "tool": "get_financial_summary", "data": source}}, Steps: []agentStep{{Name: "get_financial_summary", Status: "success", Duration: time.Since(started).Milliseconds(), From: c.Period.From, To: c.Period.To}}}
	result.DataQuality = map[string]any{"status": "unknown", "last_transaction_date": agentNullableString(latest), "query_from": c.Period.From, "query_to": c.Period.To, "reason": "Les dépenses enregistrées ne prouvent pas la couverture complète de la période."}
	if !summary.Available {
		result.Status = "insufficient_data"
		result.Answer = "Je n’ai aucune opération de dépense enregistrée " + agentPeriodWording(c.Period) + " dans ce périmètre. Cela ne signifie pas que tu n’as rien dépensé."
		if latest.Valid {
			result.Answer += " La dernière opération enregistrée date du " + agentFrenchDate(latest.String) + "."
		}
		return result, nil
	}
	amount, err := parseAnalysisAmount(summary.Expenses.String())
	if err != nil || amount <= 0 || summary.Count < 1 {
		return nil, fmt.Errorf("montants sources invalides")
	}
	operation := "opération de dépense"
	if summary.Count > 1 {
		operation = "opérations de dépense"
	}
	result.Answer = fmt.Sprintf("Pour les données %s, tes dépenses enregistrées représentent %s €, sur %d %s. Les virements internes et l’épargne sont exclus. La couverture bancaire complète de cette période n’est pas établie.", agentPeriodWording(c.Period), strings.ReplaceAll(amount.String(), ".", ","), summary.Count, operation)
	return result, nil
}
