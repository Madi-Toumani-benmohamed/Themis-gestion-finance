package api

import (
	"github.com/lib/pq"
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// coverage indique la période couverte par les données : jusqu'où vont les
// transactions, et quand a eu lieu le dernier import. Répond à « suis-je à
// jour ? » d'un coup d'œil.
func (s *Server) coverage(w http.ResponseWriter, r *http.Request) {
	owner := s.ownerID(r)
	var firstTx, lastTx, lastImport, lastPeriodEnd sql.NullString
	_ = s.DB.QueryRow(`SELECT min(tx_date)::text, max(tx_date)::text FROM transactions WHERE owner_id = $1`, owner).Scan(&firstTx, &lastTx)
	_ = s.DB.QueryRow(`SELECT max(uploaded_at)::date::text, max(period_end)::text FROM statements WHERE owner_id = $1`, owner).Scan(&lastImport, &lastPeriodEnd)

	// Fraîcheur : si la donnée la plus récente a plus de ~35 jours, un relevé
	// manque probablement.
	stale := false
	if lastTx.Valid {
		if d, err := time.Parse("2006-01-02", lastTx.String); err == nil {
			stale = time.Since(d) > 35*24*time.Hour
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"first_tx":        firstTx.String,
		"last_tx":         lastTx.String,
		"last_import":     lastImport.String,
		"last_period_end": lastPeriodEnd.String,
		"stale":           stale,
	})
}

// coherenceExample : une transaction concrète illustrant une incohérence,
// avec assez de contexte (catégorie ET tags) pour juger s'il y a vraiment
// problème sans quitter l'audit.
type coherenceExample struct {
	Display  string   `json:"display"`
	Category string   `json:"category"`
	Tags     []string `json:"tags"`
}

// coherenceFinding : une incohérence détectée entre une règle et l'état réel
// des transactions.
type coherenceFinding struct {
	Rule     string             `json:"rule"`
	Kind     string             `json:"kind"` // wrong_category | missing_tags
	Detail   string             `json:"detail"`
	Count    int                `json:"count"`
	Query    string             `json:"query"` // pour rebondir vers les transactions concernées
	Examples []coherenceExample `json:"examples"`
}

// coherence audite la cohérence entre les règles et les transactions :
//  1. déterministe (toujours) : transactions dont le libellé correspond à une
//     règle mais rangées dans une AUTRE catégorie, ou privées des tags de la
//     règle. Fiable, vérifiable, sans IA.
//  2. IA (si ?ai=1 et LLM configuré) : relecture d'ensemble par le modèle pour
//     les incohérences plus subtiles (transactions similaires taguées
//     différemment, catégories douteuses).
func (s *Server) coherence(w http.ResponseWriter, r *http.Request) {
	owner := s.ownerID(r)

	rows, err := s.DB.Query(`
		SELECT r.pattern, r.domain_id, d.name, r.tags
		FROM rules r JOIN domains d ON d.id = r.domain_id
		WHERE r.owner_id = $1 ORDER BY r.pattern`, owner)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	type rl struct {
		pattern, domName, tags string
		domID                  int64
	}
	var rules []rl
	for rows.Next() {
		var x rl
		if rows.Scan(&x.pattern, &x.domID, &x.domName, &x.tags) == nil {
			rules = append(rules, x)
		}
	}
	rows.Close()

	findings := []coherenceFinding{}
	for _, ru := range rules {
		if strings.HasPrefix(ru.pattern, "re:") {
			continue // les motifs regex ne s'expriment pas en ILIKE ; audités par l'IA
		}
		// 1a. Mauvaise catégorie : le libellé matche mais la catégorie diffère.
		var n int
		var ex []coherenceExample
		erows, e := s.DB.Query(`
			SELECT COALESCE(NULLIF(t.display_label,''), t.label), COALESCE(d.name,'(sans)'),
			       COALESCE((SELECT array_agg(tg.name ORDER BY tg.name) FROM transaction_tags tt JOIN tags tg ON tg.id = tt.tag_id WHERE tt.transaction_id = t.id), '{}')
			FROM transactions t
			LEFT JOIN domains d ON d.id = t.domain_id
			WHERE t.owner_id = $1 AND t.label ILIKE '%'||$2||'%'
			  AND t.domain_id IS NOT NULL AND t.domain_id <> $3
			LIMIT 4`, owner, ru.pattern, ru.domID)
		if e == nil {
			for erows.Next() {
				var disp, cat string
				var tags []string
				if erows.Scan(&disp, &cat, pq.Array(&tags)) == nil {
					ex = append(ex, coherenceExample{Display: disp, Category: cat, Tags: tags})
				}
			}
			erows.Close()
		}
		_ = s.DB.QueryRow(`
			SELECT count(*) FROM transactions t
			WHERE t.owner_id = $1 AND t.label ILIKE '%'||$2||'%'
			  AND t.domain_id IS NOT NULL AND t.domain_id <> $3`, owner, ru.pattern, ru.domID).Scan(&n)
		if n > 0 {
			findings = append(findings, coherenceFinding{
				Rule: ru.pattern, Kind: "wrong_category",
				Detail:   fmt.Sprintf("%d transaction(s) correspondant à « %s » sont rangées ailleurs que dans %s.", n, ru.pattern, ru.domName),
				Count:    n, Query: ru.pattern, Examples: ex,
			})
		}
		// 1b. Tags manquants : la règle porte des tags, des transactions
		// correspondantes ne les ont pas tous.
		if strings.TrimSpace(ru.tags) != "" {
			for _, tag := range strings.Split(ru.tags, ",") {
				tag = strings.TrimSpace(tag)
				if tag == "" {
					continue
				}
				var m int
				_ = s.DB.QueryRow(`
					SELECT count(*) FROM transactions t
					WHERE t.owner_id = $1 AND t.label ILIKE '%'||$2||'%'
					  AND NOT EXISTS (
						SELECT 1 FROM transaction_tags tt JOIN tags tg ON tg.id = tt.tag_id
						WHERE tt.transaction_id = t.id AND lower(tg.name) = lower($3))`,
					owner, ru.pattern, tag).Scan(&m)
				if m > 0 {
					// exemples : montrer les tags actuels, pour voir ce qui manque
					var tex []coherenceExample
					trows, te := s.DB.Query(`
						SELECT COALESCE(NULLIF(t.display_label,''), t.label), COALESCE(d.name,'(sans)'),
						       COALESCE((SELECT array_agg(tg.name ORDER BY tg.name) FROM transaction_tags tt JOIN tags tg ON tg.id = tt.tag_id WHERE tt.transaction_id = t.id), '{}')
						FROM transactions t LEFT JOIN domains d ON d.id = t.domain_id
						WHERE t.owner_id = $1 AND t.label ILIKE '%'||$2||'%'
						  AND NOT EXISTS (
							SELECT 1 FROM transaction_tags tt JOIN tags tg ON tg.id = tt.tag_id
							WHERE tt.transaction_id = t.id AND lower(tg.name) = lower($3))
						LIMIT 4`, owner, ru.pattern, tag)
					if te == nil {
						for trows.Next() {
							var disp, cat string
							var tags []string
							if trows.Scan(&disp, &cat, pq.Array(&tags)) == nil {
								tex = append(tex, coherenceExample{Display: disp, Category: cat, Tags: tags})
							}
						}
						trows.Close()
					}
					findings = append(findings, coherenceFinding{
						Rule: ru.pattern, Kind: "missing_tags",
						Detail: fmt.Sprintf("%d transaction(s) « %s » n'ont pas le tag « %s » prévu par la règle. « Appliquer les règles » corrigera.", m, ru.pattern, tag),
						Count:  m, Query: ru.pattern, Examples: tex,
					})
				}
			}
		}
	}

	out := map[string]any{"findings": findings, "rules_checked": len(rules)}

	// 2. Relecture IA optionnelle : règles + échantillon de transactions triées.
	if r.URL.Query().Get("ai") == "1" {
		client := s.getLLM()
		if client == nil {
			out["ai"] = ""
			out["ai_error"] = "IA non configurée (onglet Admin)."
		} else {
			var b strings.Builder
			b.WriteString("Tu audites le tri de transactions bancaires. Voici les règles de l'utilisateur (motif → catégorie [tags]) :\n")
			for _, ru := range rules {
				b.WriteString(fmt.Sprintf("- %s → %s [%s]\n", ru.pattern, ru.domName, ru.tags))
			}
			b.WriteString("\nÉchantillon de transactions triées (libellé | catégorie | tags) :\n")
			trows, e := s.DB.Query(`
				SELECT COALESCE(NULLIF(t.display_label,''), t.label), COALESCE(d.name,'(sans)'),
				       COALESCE((SELECT string_agg(tg.name, ',') FROM transaction_tags tt JOIN tags tg ON tg.id=tt.tag_id WHERE tt.transaction_id=t.id), '')
				FROM transactions t LEFT JOIN domains d ON d.id = t.domain_id
				WHERE t.owner_id = $1 AND t.domain_id IS NOT NULL
				ORDER BY t.tx_date DESC LIMIT 120`, owner)
			if e == nil {
				for trows.Next() {
					var l, c, tg string
					if trows.Scan(&l, &c, &tg) == nil {
						b.WriteString(fmt.Sprintf("%s | %s | %s\n", l, c, tg))
					}
				}
				trows.Close()
			}
			b.WriteString("\nLa catégorie « Transfert interne » est réservée aux virements entre comptes personnels (ni dépense ni revenu).\n")
			b.WriteString("Signale en français, en liste courte : (1) transactions qui contredisent une règle, (2) transactions similaires triées différemment, (3) tags incohérents ou manquants. Sois concret, cite les libellés. Si tout est cohérent, dis-le simplement.")
			ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
			defer cancel()
			txt, aerr := client.Complete(ctx,
				"Tu es un auditeur méticuleux de comptabilité personnelle. Réponds en français, de façon concrète et brève.",
				b.String(), 1200, false)
			if aerr != nil {
				out["ai_error"] = aerr.Error()
			} else {
				out["ai"] = txt
			}
		}
	}
	writeJSON(w, http.StatusOK, out)
}
