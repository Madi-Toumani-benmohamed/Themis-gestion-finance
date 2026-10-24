package ai

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"themis/internal/llm"
)

const enrichSystem = `Tu es un assistant qui range des transactions bancaires françaises selon DEUX axes distincts.

1. DOMAINE (obligatoire, un seul) : le pan de vie concerné. Tu DOIS choisir
   EXACTEMENT l'un des domaines autorisés fournis, à l'identique. Si aucun ne
   convient vraiment, omets la transaction plutôt que de forcer.

2. TAGS (0 à 4, libres, en minuscules) : des étiquettes courtes et réutilisables
   décrivant le détail. Préfère RÉUTILISER un tag de la liste des tags déjà
   employés quand c'est pertinent, pour éviter les doublons ("elec" vs
   "électricité"). Crée un nouveau tag seulement s'il apporte une info utile.
   Exemples : un loyer -> domaine "Logement", tags ["loyer"] ; une facture EDF
   -> domaine "Logement", tags ["facture", "électricité"].

Réponds UNIQUEMENT avec un objet JSON valide, sans markdown ni commentaire :
{"assignments": [{"id": 123, "domain": "Nom exact d'un domaine autorisé", "tags": ["tag1", "tag2"]}]}
Règles :
- "domain" doit être EXACTEMENT l'un des noms autorisés.
- N'inclus pas les transactions dont tu n'es pas raisonnablement sûr.`

// EnrichResult describes unverified proposals. Legacy mutation counters remain zero.
type EnrichResult struct {
	Suggested  int `json:"suggested"`
	DomainsSet int `json:"domains_set"`
	TagsSet    int `json:"tags_set"`
	Processed  int `json:"processed"`
	Remaining  int `json:"remaining"`
}

// AutoEnrich generates suggestions using the existing provider. It never changes
// assigned categories or tags. A later explicit PATCH is required to apply one.
func AutoEnrich(ctx context.Context, db *sql.DB, client llm.Client, ownerID, statementID int64, month string) (EnrichResult, error) {
	var res EnrichResult
	if client == nil {
		return res, fmt.Errorf("aucun fournisseur d'IA configuré")
	}

	// Domaines autorisés (racines uniquement : la liste fermée simple)
	domByName := map[string]int64{}
	domNames := []string{}
	drows, err := db.Query(`SELECT id, name FROM domains WHERE parent_id IS NULL AND owner_id = $1 ORDER BY name`, ownerID)
	if err != nil {
		return res, err
	}
	for drows.Next() {
		var id int64
		var name string
		if err := drows.Scan(&id, &name); err != nil {
			drows.Close()
			return res, err
		}
		domByName[strings.ToUpper(strings.TrimSpace(name))] = id
		domNames = append(domNames, name)
	}
	if err := drows.Err(); err != nil {
		drows.Close()
		return res, err
	}
	drows.Close()
	if len(domNames) == 0 {
		return res, fmt.Errorf("aucun domaine défini")
	}

	// Tags déjà utilisés (pour encourager la réutilisation)
	existingTags := []string{}
	if trows, err := db.Query(`SELECT name FROM tags WHERE owner_id = $1 ORDER BY name LIMIT 200`, ownerID); err == nil {
		for trows.Next() {
			var n string
			if trows.Scan(&n) == nil {
				existingTags = append(existingTags, n)
			}
		}
		trows.Close()
	}

	// Transactions sans domaine
	type tx struct {
		ID     int64
		Label  string
		Amount string
	}
	q := `SELECT id, label, amount::text FROM transactions WHERE domain_id IS NULL AND owner_id = $1`
	args := []any{ownerID}
	if statementID > 0 {
		args = append(args, statementID)
		q += fmt.Sprintf(" AND statement_id = $%d", len(args))
	}
	if month != "" {
		args = append(args, month)
		q += fmt.Sprintf(" AND to_char(tx_date, 'YYYY-MM') = $%d", len(args))
	}
	q += " ORDER BY id LIMIT 500"
	trows, err := db.Query(q, args...)
	if err != nil {
		return res, err
	}
	var txs []tx
	for trows.Next() {
		var t tx
		if err := trows.Scan(&t.ID, &t.Label, &t.Amount); err != nil {
			trows.Close()
			return res, err
		}
		txs = append(txs, t)
	}
	if err := trows.Err(); err != nil {
		trows.Close()
		return res, err
	}
	trows.Close()
	if len(txs) == 0 {
		return res, nil
	}

	for start := 0; start < len(txs); start += chunkSize {
		end := start + chunkSize
		if end > len(txs) {
			end = len(txs)
		}
		chunk := txs[start:end]

		var b strings.Builder
		fmt.Fprintf(&b, "Domaines autorisés : %s\n", strings.Join(domNames, " | "))
		if len(existingTags) > 0 {
			fmt.Fprintf(&b, "Tags déjà utilisés (à réutiliser si pertinent) : %s\n", strings.Join(existingTags, ", "))
		}
		b.WriteString("Si une catégorie « Transfert interne » existe, elle est réservée aux virements entre comptes personnels (vers un livret, une épargne…) : utilise-la pour ces mouvements, jamais pour de vraies dépenses.\n")
		b.WriteString("\nTransactions :\n")
		for _, t := range chunk {
			fmt.Fprintf(&b, "id=%d | %s € | %s\n", t.ID, t.Amount, t.Label)
		}

		callCtx, cancel := context.WithTimeout(ctx, llm.CategorizeTimeout())
		raw, err := client.Complete(callCtx, enrichSystem, b.String(), 4000, true)
		cancel()
		if err != nil {
			return res, err
		}

		var out struct {
			Assignments []struct {
				ID     int64    `json:"id"`
				Domain string   `json:"domain"`
				Tags   []string `json:"tags"`
			} `json:"assignments"`
		}
		if err := json.Unmarshal([]byte(extractJSONObject(raw)), &out); err != nil {
			return res, fmt.Errorf("réponse IA non parsable: %w", err)
		}

		validIDs := map[int64]bool{}
		for _, t := range chunk {
			validIDs[t.ID] = true
		}
		seen := map[int64]bool{}
		for _, a := range out.Assignments {
			if !validIDs[a.ID] || seen[a.ID] {
				continue
			}
			seen[a.ID] = true
			res.Processed++
			domID, ok := domByName[strings.ToUpper(strings.TrimSpace(a.Domain))]
			if !ok {
				continue
			}
			tags := []string{}
			used := map[string]bool{}
			for _, raw := range a.Tags {
				name := strings.ToLower(strings.TrimSpace(raw))
				if name == "" || len(name) > 40 || used[name] {
					continue
				}
				used[name] = true
				tags = append(tags, name)
				if len(tags) == 4 {
					break
				}
			}
			// Re-check ownership and absence of assignment after the LLM call:
			// a concurrent manual edit always wins over this proposal.
			result, err := db.ExecContext(ctx, `INSERT INTO transaction_suggestions(transaction_id,owner_id,domain_id,tags)
			 SELECT t.id,t.owner_id,d.id,$4::text[] FROM transactions t JOIN domains d ON d.id=$3 AND d.owner_id=t.owner_id
			 WHERE t.id=$1 AND t.owner_id=$2 AND t.domain_id IS NULL
			 ON CONFLICT(transaction_id) DO UPDATE SET domain_id=EXCLUDED.domain_id,tags=EXCLUDED.tags,created_at=now()
			 WHERE transaction_suggestions.owner_id=EXCLUDED.owner_id`, a.ID, ownerID, domID, pq.Array(tags))
			if err != nil {
				return res, err
			}
			if n, _ := result.RowsAffected(); n == 1 {
				res.Suggested++
			}
		}
	}
	res.Remaining = len(txs)
	return res, nil
}

const chunkSize = 40

// extractJSONObject isole le premier objet JSON d'une réponse LLM, en tolérant
// un éventuel enrobage markdown (```json … ```) ou du texte autour.
func extractJSONObject(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}
