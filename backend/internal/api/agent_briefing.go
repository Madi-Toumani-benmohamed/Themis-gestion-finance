package api

import (
	"context"
	"database/sql"

	"github.com/lib/pq"
)

// Read-only observations from one PostgreSQL snapshot. Counts describe imported
// operations, never bank coverage or a recommendation inferred by a model.
type agentPeriodBriefing struct {
	Period            agentPeriod `json:"period"`
	Timezone          string      `json:"timezone"`
	Available         bool        `json:"available"`
	TransactionCount  int         `json:"transaction_count"`
	ExpenseCount      int         `json:"expense_count"`
	UnclassifiedCount int         `json:"unclassified_expense_count"`
	UnclassifiedIDs   []int64     `json:"unclassified_transaction_ids"`
	HasMore           bool        `json:"has_more"`
	Coverage          string      `json:"coverage"`
}

func agentReadPeriodBriefing(ctx context.Context, db *sql.DB, owner int64, period agentPeriod, zone string) (agentPeriodBriefing, error) {
	out := agentPeriodBriefing{Period: period, Timezone: zone, Coverage: "unknown", UnclassifiedIDs: []int64{}}
	err := db.QueryRowContext(ctx, `WITH scoped AS NOT MATERIALIZED (
 SELECT t.id,t.tx_date,t.domain_id,(t.amount<0 AND `+financialExpenseSQLT+`) AS expense
 FROM transactions t WHERE t.owner_id=$1 AND t.tx_date BETWEEN $2::date AND $3::date
 ) SELECT count(*),count(*) FILTER (WHERE expense),count(*) FILTER (WHERE expense AND domain_id IS NULL),
 ARRAY(SELECT id FROM scoped WHERE expense AND domain_id IS NULL ORDER BY tx_date DESC,id DESC LIMIT 101)
 FROM scoped`, owner, period.From, period.To).Scan(&out.TransactionCount, &out.ExpenseCount, &out.UnclassifiedCount, pq.Array(&out.UnclassifiedIDs))
	if err != nil {
		return out, err
	}
	out.Available = out.TransactionCount > 0
	out.HasMore = len(out.UnclassifiedIDs) > 100
	if out.HasMore {
		out.UnclassifiedIDs = out.UnclassifiedIDs[:100]
	}
	return out, nil
}
