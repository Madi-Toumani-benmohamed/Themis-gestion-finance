package classificationaudit

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
)

// Run never opens the application server, migrator, normalizer or job worker.
// The connection is held in one read-only snapshot, always rolled back.
func Run(ctx context.Context, db *sql.DB, owner int64) (Report, error) {
	if owner <= 0 {
		return Report{}, fmt.Errorf("owner required")
	}
	tx, e := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if e != nil {
		return Report{}, fmt.Errorf("read-only snapshot unavailable")
	}
	defer tx.Rollback()
	d := Dataset{}
	queries := []struct {
		q    string
		scan func(*sql.Rows) error
	}{
		{`SELECT id,COALESCE(parent_id,0),name,is_transfer FROM domains WHERE owner_id=$1 ORDER BY id LIMIT 50001`, func(rows *sql.Rows) error {
			var v Domain
			e := rows.Scan(&v.ID, &v.Parent, &v.Name, &v.Transfer)
			d.Domains = append(d.Domains, v)
			return e
		}},
		{`SELECT id,name FROM tags WHERE owner_id=$1 ORDER BY id LIMIT 50001`, func(rows *sql.Rows) error {
			var v Tag
			e := rows.Scan(&v.ID, &v.Name)
			d.Tags = append(d.Tags, v)
			return e
		}},
		{`SELECT id,domain_id,pattern,tags FROM rules WHERE owner_id=$1 ORDER BY id LIMIT 50001`, func(rows *sql.Rows) error {
			var v Rule
			e := rows.Scan(&v.ID, &v.Domain, &v.Pattern, &v.Tags)
			d.Rules = append(d.Rules, v)
			return e
		}},
		{`SELECT t.id,COALESCE(t.domain_id,0),t.label,t.merchant,t.label_kind,t.classification_source,t.amount::text,
 ARRAY(SELECT tt.tag_id FROM transaction_tags tt WHERE tt.transaction_id=t.id ORDER BY tt.tag_id)
 FROM transactions t WHERE t.owner_id=$1 ORDER BY t.id LIMIT 50001`, func(rows *sql.Rows) error {
			var v Transaction
			e := rows.Scan(&v.ID, &v.Domain, &v.Label, &v.Merchant, &v.Kind, &v.Source, &v.Amount, pq.Array(&v.Tags))
			d.Transactions = append(d.Transactions, v)
			return e
		}},
	}
	for _, q := range queries {
		rows, e := tx.QueryContext(ctx, q.q, owner)
		if e != nil {
			return Report{}, fmt.Errorf("audit read unavailable")
		}
		for rows.Next() {
			if e = q.scan(rows); e != nil {
				rows.Close()
				return Report{}, fmt.Errorf("audit row invalid")
			}
			if len(d.Domains)+len(d.Tags)+len(d.Rules)+len(d.Transactions) > MaxRows {
				rows.Close()
				return Report{}, fmt.Errorf("audit size limit")
			}
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return Report{}, fmt.Errorf("audit read interrupted")
		}
	}
	if e = ctx.Err(); e != nil {
		return Report{}, fmt.Errorf("audit interrupted")
	}
	r, e := Analyze(d)
	if e != nil {
		return Report{}, e
	}
	if ctx.Err() != nil {
		return Report{}, fmt.Errorf("audit interrupted")
	}
	return r, nil
}
