package api

import (
	"context"
	"fmt"
	"time"

	"github.com/lib/pq"
	"themis/internal/categorize"
)

func validCalendarMonth(month string) bool {
	date, err := time.Parse("2006-01", month)
	return err == nil && date.Year() > 0 && date.Format("2006-01") == month
}

func classificationForRule(domain *int64) string {
	if domain != nil {
		return "rule"
	}
	return "unknown"
}

func (s *Server) validateRule(ctx context.Context, owner, domain int64, pattern string) error {
	if _, err := categorize.CompilePattern(pattern); err != nil {
		return err
	}
	var valid bool
	if err := s.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM domains WHERE id=$1 AND owner_id=$2)`, domain, owner).Scan(&valid); err != nil {
		return err
	}
	if !valid {
		return fmt.Errorf("catégorie indisponible")
	}
	return nil
}

// Explicit application evaluates all owned rules; a selected rule may only
// apply where it is the winner. Existing tags are never removed.
func (s *Server) applyStoredRuleToTransactions(owner, id int64) (int64, error) {
	return s.applyClassificationRules(owner, 0, id)
}
func (s *Server) applyClassificationRules(owner, statement int64, selectedID int64) (int64, error) {
	ctx := context.Background()
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	eng, err := categorize.LoadFrom(ctx, tx, owner)
	if err != nil {
		return 0, err
	}
	// Applications from the public rule handlers always select a persisted rule.
	// No ad-hoc rule may bypass the owned, persisted ruleset.
	if selectedID > 0 {
		var valid bool
		if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM rules r JOIN domains d ON d.id=r.domain_id AND d.owner_id=r.owner_id WHERE r.id=$1 AND r.owner_id=$2)`, selectedID, owner).Scan(&valid); err != nil {
			return 0, err
		}
		if !valid {
			return 0, fmt.Errorf("règle indisponible")
		}
	}
	rows, err := tx.QueryContext(ctx, `SELECT t.id,t.label,t.domain_id,t.classification_source,
 COALESCE((SELECT array_agg(tg.name ORDER BY tg.name) FROM transaction_tags tt JOIN tags tg ON tg.id=tt.tag_id AND tg.owner_id=t.owner_id WHERE tt.transaction_id=t.id),'{}')
 FROM transactions t WHERE t.owner_id=$1 AND ($2::bigint=0 OR t.statement_id=$2) ORDER BY t.id FOR UPDATE OF t`, owner, statement)
	if err != nil {
		return 0, err
	}
	plans := []categorize.Plan{}
	for rows.Next() {
		var in categorize.Input
		if err = rows.Scan(&in.ID, &in.Label, &in.DomainID, &in.Source, pq.Array(&in.Tags)); err != nil {
			rows.Close()
			return 0, err
		}
		plan := eng.Plan(in)
		if plan.Winner != nil && (selectedID == 0 || plan.Winner.ID == selectedID) {
			plans = append(plans, plan)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return 0, err
	}
	var applied int64
	for _, plan := range plans {
		if plan.SetDomain {
			res, e := tx.ExecContext(ctx, `UPDATE transactions SET domain_id=$1,classification_source='rule' WHERE id=$2 AND owner_id=$3 AND domain_id IS NULL AND classification_source<>'manual'`, *plan.ProposedDomain, plan.TransactionID, owner)
			if e != nil {
				return 0, e
			}
			n, e := res.RowsAffected()
			if e != nil {
				return 0, e
			}
			applied += n
		}
		tags, e := validateTransactionTags(plan.AddTags)
		if e != nil {
			return 0, e
		}
		for _, name := range tags {
			var tag int64
			if e = tx.QueryRowContext(ctx, `INSERT INTO tags(name,owner_id) VALUES($1,$2) ON CONFLICT(owner_id,name) DO UPDATE SET name=EXCLUDED.name RETURNING id`, name, owner).Scan(&tag); e != nil {
				return 0, e
			}
			if _, e = tx.ExecContext(ctx, `INSERT INTO transaction_tags(transaction_id,tag_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, plan.TransactionID, tag); e != nil {
				return 0, e
			}
		}
	}
	return applied, tx.Commit()
}
