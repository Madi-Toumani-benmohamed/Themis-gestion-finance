package api

import (
	"context"
	"fmt"
	"strings"
	"time"

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

func (s *Server) applyRuleToTransactions(owner int64, pattern string, domain int64, tagsCSV string) (int64, error) {
	return s.applyRuleToTransactionsScoped(owner, pattern, domain, tagsCSV, 0)
}

// One deterministic matcher and one transaction for all changes from a rule.
// A fresh import applies tags only to its own statement, never to old entries.
func (s *Server) applyRuleToTransactionsScoped(owner int64, pattern string, domain int64, tagsCSV string, statement int64) (int64, error) {
	matches, err := categorize.CompilePattern(pattern)
	if err != nil {
		return 0, err
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var valid bool
	if err = tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM domains WHERE id=$1 AND owner_id=$2)`, domain, owner).Scan(&valid); err != nil {
		return 0, err
	}
	if !valid {
		return 0, fmt.Errorf("catégorie indisponible")
	}
	rows, err := tx.Query(`SELECT id,label FROM transactions WHERE owner_id=$1 AND ($2::bigint=0 OR statement_id=$2) ORDER BY id FOR UPDATE`, owner, statement)
	if err != nil {
		return 0, err
	}
	ids := []int64{}
	for rows.Next() {
		var id int64
		var label string
		if err = rows.Scan(&id, &label); err != nil {
			rows.Close()
			return 0, err
		}
		if matches(label) {
			ids = append(ids, id)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return 0, err
	}
	var applied int64
	for _, id := range ids {
		result, err := tx.Exec(`UPDATE transactions SET domain_id=$1,classification_source='rule' WHERE id=$2 AND owner_id=$3 AND domain_id IS NULL`, domain, id, owner)
		if err != nil {
			return 0, err
		}
		n, err := result.RowsAffected()
		if err != nil {
			return 0, err
		}
		applied += n
		for _, raw := range strings.Split(tagsCSV, ",") {
			name := strings.ToLower(strings.TrimSpace(raw))
			if name == "" {
				continue
			}
			if len(name) > 40 {
				return 0, fmt.Errorf("tag trop long")
			}
			var tag int64
			if err = tx.QueryRow(`INSERT INTO tags(name,owner_id) VALUES($1,$2) ON CONFLICT(owner_id,name) DO UPDATE SET name=EXCLUDED.name RETURNING id`, name, owner).Scan(&tag); err != nil {
				return 0, err
			}
			if _, err = tx.Exec(`INSERT INTO transaction_tags(transaction_id,tag_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, id, tag); err != nil {
				return 0, err
			}
		}
	}
	return applied, tx.Commit()
}
