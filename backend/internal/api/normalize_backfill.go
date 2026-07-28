package api

import (
	"log"
	"strings"

	"themis/internal/normalize"
)

// normalizerVersion doit être incrémenté quand le dictionnaire ou le pipeline
// de normalisation évolue : toutes les transactions seront alors re-normalisées
// au prochain démarrage (idempotent, quelques millisecondes à l'échelle perso).
const normalizerVersion = "3"

// BackfillDisplayLabels normalise les transactions qui ne le sont pas encore :
// celles d'avant la migration 014, et toutes si le normaliseur a évolué.
func (s *Server) BackfillDisplayLabels() {
	var stored string
	_ = s.DB.QueryRow(`SELECT value #>> '{}' FROM settings WHERE key = 'normalizer_version'`).Scan(&stored)
	where := `display_label = ''`
	if stored != normalizerVersion {
		where = `TRUE`
	}
	rows, err := s.DB.Query(`SELECT id, label, amount FROM transactions WHERE ` + where)
	if err != nil {
		log.Printf("normalisation: lecture impossible: %v", err)
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
	// Surnoms utilisateur : chargés une fois, ils priment sur le nom auto.
	aliases := map[string]string{}
	if arows, aerr := s.DB.Query(`SELECT lower(match_key), display FROM merchant_aliases`); aerr == nil {
		for arows.Next() {
			var k, d string
			if arows.Scan(&k, &d) == nil {
				aliases[k] = d
			}
		}
		arows.Close()
	}
	for _, r := range todo {
		n := normalize.Label(r.label, r.amount)
		disp := n.Display
		// le surnom prime : clé = marchand normalisé, sinon libellé brut
		key := strings.ToLower(n.Merchant)
		if key == "" {
			key = strings.ToLower(r.label)
		}
		if alias, ok := aliases[key]; ok {
			disp = alias
		}
		_, _ = s.DB.Exec(`UPDATE transactions SET display_label = $1, merchant = $2, label_kind = $3 WHERE id = $4`, disp, n.Merchant, n.Kind, r.id)
	}
	if len(todo) > 0 {
		log.Printf("normalisation: %d libellé(s) d'affichage calculé(s)", len(todo))
	}
	_, _ = s.DB.Exec(`
		INSERT INTO settings (key, value) VALUES ('normalizer_version', to_jsonb($1::text))
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`, normalizerVersion)
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
