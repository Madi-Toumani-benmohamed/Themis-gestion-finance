package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// exportAll produit une archive JSON de toutes les données de l'utilisateur :
// comptes, relevés, transactions (avec catégorie, domaine, tags), catégories,
// règles, budgets, crédits, taxes, domaines. Portabilité et sauvegarde.
func (s *Server) exportAll(w http.ResponseWriter, r *http.Request) {
	dump := map[string]any{
		"exported_at": time.Now().Format(time.RFC3339),
		"app":         "themis",
		"version":     1,
	}

	owner := s.ownerID(r)
	tables := map[string]string{
		"credit_payments": `SELECT p.* FROM credit_payments p JOIN credits c ON c.id=p.credit_id AND c.owner_id=p.owner_id WHERE p.owner_id=$1 ORDER BY p.id`,
		"credit_history":  `SELECT * FROM credit_history WHERE owner_id=$1 ORDER BY id`,
		"tax_history":     `SELECT * FROM tax_history WHERE owner_id=$1 ORDER BY id`,
		"accounts":        `SELECT id, name, kind, interest_rate, deposit_limit FROM accounts WHERE owner_id = $1 ORDER BY id`,
		"domains":         `SELECT id, name, parent_id, color FROM domains WHERE owner_id = $1 ORDER BY id`,
		"rules":           `SELECT id, pattern, domain_id FROM rules WHERE owner_id = $1 ORDER BY id`,
		"budgets":         `SELECT id, domain_id, monthly_limit FROM budgets WHERE owner_id = $1 ORDER BY id`,
		"credits":         `SELECT id, name, monthly_amount, months_total, start_month, note, opening_paid_cents, closed_at, archived_at FROM credits WHERE owner_id = $1 ORDER BY id`,
		"taxes":           `SELECT id, name, amount, due_date::text, paid, paid_amount_cents, overdue_confirmed, archived_at FROM taxes WHERE owner_id = $1 ORDER BY id`,
		"statements":      `SELECT id, account_id, filename, period_start::text, period_end::text, opening_balance, closing_balance, parser, validated FROM statements WHERE owner_id = $1 ORDER BY id`,
		"transactions": `SELECT t.id, t.statement_id, t.tx_date::text, t.label, t.amount, t.domain_id,
		                        COALESCE(ARRAY_TO_STRING(ARRAY_AGG(tg.name) FILTER (WHERE tg.name IS NOT NULL), ','), '')
		                 FROM transactions t
		                 LEFT JOIN transaction_tags tt ON tt.transaction_id = t.id
		                 LEFT JOIN tags tg ON tg.id = tt.tag_id AND tg.owner_id = t.owner_id
		                 WHERE t.owner_id = $1
		                 GROUP BY t.id ORDER BY t.id`,
	}

	for name, q := range tables {
		rows, err := s.DB.Query(q, owner)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, fmt.Errorf("export indisponible"))
			return
		}
		cols, _ := rows.Columns()
		list := []map[string]any{}
		for rows.Next() {
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				rows.Close()
				writeErr(w, http.StatusInternalServerError, fmt.Errorf("export indisponible"))
				return
			}
			row := map[string]any{}
			for i, c := range cols {
				v := vals[i]
				if b, ok := v.([]byte); ok { // NUMERIC arrive en []byte
					v = string(b)
				}
				row[c] = v
			}
			list = append(list, row)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			writeErr(w, 500, fmt.Errorf("export indisponible"))
			return
		}
		rows.Close()
		dump[name] = list
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="themis-export-%s.json"`, time.Now().Format("2006-01-02")))
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(dump)
}
