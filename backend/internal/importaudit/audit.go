// Package importaudit performs a read-only historical audit. It neither repairs
// records nor assumes that a balanced statement has a complete extraction.
package importaudit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"themis/internal/models"
)

type Result struct {
	StatementID     int64  `json:"statement_id"`
	Count           int    `json:"recorded_count"`
	Debits          int64  `json:"debits_cents"`
	Credits         int64  `json:"credits_cents"`
	Opening         *int64 `json:"opening_cents"`
	Closing         *int64 `json:"closing_cents"`
	Difference      *int64 `json:"balance_difference_cents"`
	BalanceControl  string `json:"balance_control"`
	TotalsControl   string `json:"totals_control"`
	CountControl    string `json:"count_control"`
	OutsidePeriod   int    `json:"outside_period_count"`
	LegacyValidated bool   `json:"legacy_validated"`
	SourceTraceable bool   `json:"source_traceable"`
	NeedsReview     bool   `json:"needs_review"`
	Completeness    string `json:"completeness"`
}

func Run(ctx context.Context, d *sql.DB, owner int64) ([]Result, error) {
	if owner <= 0 {
		return nil, fmt.Errorf("propriétaire explicite requis")
	}
	tx, err := d.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT st.id,st.validated,(st.opening_balance*100)::bigint,(st.closing_balance*100)::bigint,
 x.n,x.debits,x.credits,x.outside,(to_jsonb(st)->'import_report')::text,COALESCE(to_jsonb(st)->>'source_sha256','')<>''
 FROM statements st CROSS JOIN LATERAL(SELECT count(*) n,
 COALESCE(-SUM(t.amount) FILTER(WHERE t.amount<0)*100,0)::bigint debits,
 COALESCE(SUM(t.amount) FILTER(WHERE t.amount>0)*100,0)::bigint credits,
 count(*) FILTER(WHERE t.tx_date<st.period_start OR t.tx_date>st.period_end) outside
 FROM transactions t WHERE t.statement_id=st.id AND t.owner_id=st.owner_id) x
 WHERE st.owner_id=$1 ORDER BY st.id`, owner)
	if err != nil {
		return nil, fmt.Errorf("audit des relevés indisponible")
	}
	out := []Result{}
	for rows.Next() {
		var r Result
		var opening, closing sql.NullInt64
		var raw sql.NullString
		if err = rows.Scan(&r.StatementID, &r.LegacyValidated, &opening, &closing, &r.Count, &r.Debits, &r.Credits, &r.OutsidePeriod, &raw, &r.SourceTraceable); err != nil {
			rows.Close()
			return nil, err
		}
		r.BalanceControl = "unverifiable"
		r.TotalsControl = "unverifiable"
		r.CountControl = "unverifiable"
		r.Completeness = "unknown"
		if opening.Valid {
			v := opening.Int64
			r.Opening = &v
		}
		if closing.Valid {
			v := closing.Int64
			r.Closing = &v
		}
		if opening.Valid && closing.Valid {
			diff := closing.Int64 - opening.Int64 - r.Credits + r.Debits
			r.Difference = &diff
			r.BalanceControl = "passed"
			if diff != 0 {
				r.BalanceControl = "failed"
				r.NeedsReview = true
			}
		}
		if raw.Valid {
			var report *models.ImportReport
			if err = json.Unmarshal([]byte(raw.String), &report); err != nil {
				rows.Close()
				return nil, fmt.Errorf("rapport d'import invalide")
			}
			if report != nil {
				if report.DeclaredDebits != nil && report.DeclaredCredits != nil {
					r.TotalsControl = "passed"
					if *report.DeclaredDebits != r.Debits || *report.DeclaredCredits != r.Credits {
						r.TotalsControl = "failed"
						r.NeedsReview = true
					}
				}
				if report.ExpectedCount != nil {
					r.CountControl = "passed"
					if *report.ExpectedCount != r.Count {
						r.CountControl = "failed"
						r.NeedsReview = true
					}
				}
				if report.Rejected > 0 || len(report.Issues) > 0 || report.Parsed != r.Count {
					r.NeedsReview = true
				}
			}
		}
		if r.OutsidePeriod > 0 {
			r.NeedsReview = true
		}
		if r.NeedsReview {
			r.Completeness = "failed"
		} else if r.CountControl == "passed" && r.TotalsControl == "passed" && r.BalanceControl == "passed" {
			r.Completeness = "verified"
		}
		out = append(out, r)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return out, nil
}
