package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"themis/internal/llm"
	"themis/internal/models"
)

const parseSystem = `Tu es un extracteur de relevés bancaires français. On te donne le texte brut d'un relevé PDF.
Réponds UNIQUEMENT avec un objet JSON valide, sans markdown, sans backticks, sans commentaire, au format :
{
  "period_start": "YYYY-MM-DD" ou null,
  "period_end": "YYYY-MM-DD" ou null,
  "opening_balance": nombre ou null,
  "closing_balance": nombre ou null,
  "transactions": [
    {"date": "YYYY-MM-DD", "label": "libellé nettoyé", "amount": nombre}
  ]
}
Règles :
- amount est NÉGATIF pour un débit (dépense), POSITIF pour un crédit (revenu).
- Ignore les lignes de solde, totaux, frais annoncés à titre indicatif, publicités.
- Fusionne les libellés sur plusieurs lignes en un seul.
- Ne déduis jamais une transaction absente du texte.`

var reJSONFence = regexp.MustCompile("```(?:json)?")

// extractJSON isole le premier objet JSON de la réponse : les petits modèles
// locaux ajoutent parfois du texte autour malgré les consignes.
func extractJSON(s string) string {
	s = reJSONFence.ReplaceAllString(s, "")
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}

// ParseWithLLM parse le texte d'un relevé via le fournisseur d'IA configuré
// (API Claude ou Ollama local). Utilisé en fallback du parseur positionnel.
func ParseWithLLM(ctx context.Context, client llm.Client, text string) (*models.ParseResult, error) {
	if client == nil {
		return nil, fmt.Errorf("aucun fournisseur d'IA configuré : fallback indisponible")
	}
	raw, err := client.Complete(ctx, parseSystem, text, 8000, true)
	if err != nil {
		return nil, err
	}
	raw = extractJSON(raw)

	var out struct {
		PeriodStart    *string  `json:"period_start"`
		PeriodEnd      *string  `json:"period_end"`
		OpeningBalance *float64 `json:"opening_balance"`
		ClosingBalance *float64 `json:"closing_balance"`
		Transactions   []struct {
			Date   string  `json:"date"`
			Label  string  `json:"label"`
			Amount float64 `json:"amount"`
		} `json:"transactions"`
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("réponse IA non parsable: %w", err)
	}

	res := &models.ParseResult{
		Parser:         "ia",
		OpeningBalance: out.OpeningBalance,
		ClosingBalance: out.ClosingBalance,
	}
	if out.PeriodStart != nil {
		res.PeriodStart = *out.PeriodStart
	}
	if out.PeriodEnd != nil {
		res.PeriodEnd = *out.PeriodEnd
	}
	for _, t := range out.Transactions {
		tx := models.Transaction{Date: t.Date, Label: cleanLabel(t.Label), Amount: t.Amount}
		tx.Hash = TxHash(tx)
		res.Transactions = append(res.Transactions, tx)
	}
	if len(res.Transactions) == 0 {
		return nil, fmt.Errorf("le fallback IA n'a extrait aucune transaction")
	}
	res.Validated = Validate(res)
	return res, nil
}
