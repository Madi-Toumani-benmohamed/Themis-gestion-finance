package models

import "time"

type Transaction struct {
	SourcePage  int       `json:"source_page,omitempty"`
	SourceLine  int       `json:"source_line,omitempty"`
	ValueDate   string    `json:"value_date,omitempty"`
	ID          int64     `json:"id"`
	StatementID *int64    `json:"statement_id,omitempty"`
	Date        string    `json:"date"` // YYYY-MM-DD
	Label       string    `json:"label"`
	Display     string    `json:"display_label"`
	Merchant    string    `json:"merchant"`
	Kind        string    `json:"kind,omitempty"`
	Amount      float64   `json:"amount"` // négatif = débit
	DomainID    *int64    `json:"domain_id"`
	Tags        []string  `json:"tags"`
	AltAmount   *float64  `json:"-"` // lecture OCR alternative non appliquée, nécessitant une vérification
	Hash        string    `json:"-"`
	_           time.Time `json:"-"`
}

type Statement struct {
	ID             int64    `json:"id"`
	Filename       string   `json:"filename"`
	PeriodStart    *string  `json:"period_start"`
	PeriodEnd      *string  `json:"period_end"`
	OpeningBalance *float64 `json:"opening_balance"`
	ClosingBalance *float64 `json:"closing_balance"`
	Parser         string   `json:"parser"`
	Validated      bool     `json:"validated"`
	UploadedAt     string   `json:"uploaded_at"`
	TxCount        int      `json:"tx_count"`
}

type Category struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Color      string `json:"color"`
	IsTransfer bool   `json:"is_transfer"`
}

type Rule struct {
	ID       int64  `json:"id"`
	Pattern  string `json:"pattern"`
	DomainID int64  `json:"domain_id"`
}

// ParseResult est la sortie normalisée d'un parseur (local ou IA).
type ParseResult struct {
	DocumentHash   string        `json:"-"`
	Report         *ImportReport `json:"import_report,omitempty"`
	Transactions   []Transaction `json:"transactions"`
	PeriodStart    string        `json:"period_start,omitempty"`
	PeriodEnd      string        `json:"period_end,omitempty"`
	OpeningBalance *float64      `json:"opening_balance,omitempty"`
	ClosingBalance *float64      `json:"closing_balance,omitempty"`
	Parser         string        `json:"parser"`
	Validated      bool          `json:"validated"`
}

// ImportReport stores controls and source references, never a second copy of
// bank labels. All numeric fields suffixed _cents are exact integer cents.
type ImportReport struct {
	RequiresReview      bool              `json:"requires_review"`
	Warnings            []string          `json:"warnings"`
	Extraction          string            `json:"extraction,omitempty"`
	Detected            int               `json:"detected"`
	Parsed              int               `json:"parsed"`
	Inserted            int               `json:"inserted"`
	Duplicates          int               `json:"duplicates"`
	Rejected            int               `json:"rejected"`
	ExpectedCount       *int              `json:"expected_count"`
	DeclaredDebits      *int64            `json:"declared_debits_cents"`
	DeclaredCredits     *int64            `json:"declared_credits_cents"`
	Debits              int64             `json:"debits_cents"`
	Credits             int64             `json:"credits_cents"`
	BalanceDifference   *int64            `json:"balance_difference_cents"`
	CountControl        string            `json:"count_control"`
	TraceabilityControl string            `json:"traceability_control,omitempty"`
	TotalsControl       string            `json:"totals_control"`
	BalanceControl      string            `json:"balance_control"`
	Completeness        string            `json:"completeness"`
	Issues              []string          `json:"issues"`
	Candidates          []ImportCandidate `json:"candidates"`
}
type ImportCandidate struct {
	Index  int    `json:"index"`
	Page   int    `json:"page"`
	Line   int    `json:"line"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}
