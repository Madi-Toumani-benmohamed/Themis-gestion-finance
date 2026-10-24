package api

import "themis/internal/financialsql"

// Shared with AI analysis: transfer descendants and normalized savings are
// excluded from ordinary financial flows; other credits remain credits.
const (
	transferDomainIDsSQL = financialsql.TransferDomainIDs
	financialExpenseSQLT = financialsql.ExpenseT
	notTransfer          = ` AND ` + financialsql.Expense + ` `
	notTransferT         = ` AND ` + financialsql.ExpenseT + ` `
)
