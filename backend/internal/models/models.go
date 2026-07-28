package models

import "time"

type Transaction struct {
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
	AltAmount   *float64  `json:"-"` // lecture OCR alternative (réconciliation interne)
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
	Transactions   []Transaction `json:"transactions"`
	PeriodStart    string        `json:"period_start,omitempty"`
	PeriodEnd      string        `json:"period_end,omitempty"`
	OpeningBalance *float64      `json:"opening_balance,omitempty"`
	ClosingBalance *float64      `json:"closing_balance,omitempty"`
	Parser         string        `json:"parser"`
	Validated      bool          `json:"validated"`
}
