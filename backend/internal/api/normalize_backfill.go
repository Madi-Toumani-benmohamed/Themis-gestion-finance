package api

import (
	"database/sql"
	"log"
	"strings"

	"themis/internal/normalize"
)

// normalizerVersion doit être incrémenté quand le dictionnaire ou le pipeline
// de normalisation évolue : toutes les transactions seront alors re-normalisées
// au prochain démarrage (idempotent, quelques millisecondes à l'échelle perso).
// Version 4 also repairs display aliases previously mixed across owners. Raw
// bank labels, dates, categories and monetary amounts are never changed.
const normalizerVersion = "4"

func (s *Server) BackfillDisplayLabels() {
	if err := s.backfillDisplayLabels(); err != nil {
		// SQL errors can contain user values; keep banking data out of logs.
		log.Print("normalisation: actualisation interrompue, aucun changement validé")
	}
}

func (s *Server) backfillDisplayLabels() error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var stored string
	err = tx.QueryRow(`SELECT value #>> '{}' FROM settings WHERE key='normalizer_version' AND owner_id IS NULL`).Scan(&stored)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	where := `display_label = ''`
	if stored != normalizerVersion {
		where = `TRUE`
	}
	rows, err := tx.Query(`SELECT id,owner_id,label,amount FROM transactions WHERE ` + where)
	if err != nil {
		return err
	}
	type row struct {
		id     int64
		owner  sql.NullInt64
		label  string
		amount float64
	}
	var todo []row
	for rows.Next() {
		var r row
		if err = rows.Scan(&r.id, &r.owner, &r.label, &r.amount); err != nil {
			rows.Close()
			return err
		}
		todo = append(todo, r)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	type aliasKey struct {
		owner    int64
		merchant string
	}
	aliases := map[aliasKey]string{}
	type patternAlias struct{ key, display string }
	patterns := map[int64][]patternAlias{}
	arows, err := tx.Query(`SELECT owner_id,lower(match_key),display,is_pattern FROM merchant_aliases ORDER BY length(match_key) DESC,match_key`)
	if err != nil {
		return err
	}
	for arows.Next() {
		var owner int64
		var key, display string
		var pattern bool
		if err = arows.Scan(&owner, &key, &display, &pattern); err != nil {
			arows.Close()
			return err
		}
		if pattern {
			patterns[owner] = append(patterns[owner], patternAlias{key, display})
		} else {
			aliases[aliasKey{owner, key}] = display
		}
	}
	err = arows.Err()
	arows.Close()
	if err != nil {
		return err
	}
	for _, r := range todo {
		n := normalize.Label(r.label, r.amount)
		display := n.Display
		key := strings.ToLower(n.Merchant)
		if key == "" {
			key = strings.ToLower(r.label)
		}
		if r.owner.Valid {
			if alias, ok := aliases[aliasKey{r.owner.Int64, key}]; ok {
				display = alias
			} else {
				for _, alias := range patterns[r.owner.Int64] {
					if strings.Contains(key, alias.key) || strings.Contains(strings.ToLower(r.label), alias.key) {
						display = alias.display
						break
					}
				}
			}
		}
		if _, err = tx.Exec(`UPDATE transactions SET display_label=$1,merchant=$2,label_kind=$3 WHERE id=$4 AND owner_id IS NOT DISTINCT FROM $5`, display, n.Merchant, n.Kind, r.id, r.owner); err != nil {
			return err
		}
	}
	_, err = tx.Exec(`INSERT INTO settings(key,value,owner_id) VALUES('normalizer_version',to_jsonb($1::text),NULL)
 ON CONFLICT (COALESCE(owner_id,0),key) DO UPDATE SET value=EXCLUDED.value,updated_at=now()`, normalizerVersion)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// renormalizeMatching recalcule le nom auto des transactions d'une clé donnée
// (utilisé quand un surnom est supprimé).
func (s *Server) renormalizeMatching(owner int64, key string) {
	rows, err := s.DB.Query(`
		SELECT id, label, amount FROM transactions
		WHERE owner_id = $1 AND lower(COALESCE(NULLIF(merchant,''), label)) = lower($2)`, owner, key)
	if err != nil {
		return
	}
	type row struct {
		id     int64
		label  string
		amount float64
	}
	var todo []row
	for rows.Next() {
		var r row
		if rows.Scan(&r.id, &r.label, &r.amount) == nil {
			todo = append(todo, r)
		}
	}
	rows.Close()
	for _, r := range todo {
		n := normalize.Label(r.label, r.amount)
		_, _ = s.DB.Exec(`UPDATE transactions SET display_label = $1 WHERE id = $2`, n.Display, r.id)
	}
}
