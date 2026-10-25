package api

import (
	"database/sql"
	"testing"
)

// Existing analytical fixtures explicitly assert documented sources. Add their
// control balances to the synthetic fixture, never to a real bank database.
func seedDocumentedStatementBalances(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`ALTER TABLE statements ADD COLUMN opening_balance numeric(12,2) DEFAULT 0;
 ALTER TABLE statements ADD COLUMN closing_balance numeric(12,2);
 WITH totals AS (SELECT st.id,st.account_id,st.owner_id,st.period_start,
 COALESCE((SELECT SUM(t.amount) FROM transactions t WHERE t.statement_id=st.id AND t.owner_id=st.owner_id),0) total FROM statements st),
 positions AS (SELECT id,total,COALESCE(SUM(total) OVER(PARTITION BY account_id,owner_id ORDER BY period_start NULLS FIRST,id ROWS BETWEEN UNBOUNDED PRECEDING AND 1 PRECEDING),0) opening FROM totals)
 UPDATE statements st SET opening_balance=p.opening,closing_balance=p.opening+p.total FROM positions p WHERE st.id=p.id`)
	if err != nil {
		t.Fatal(err)
	}
}
