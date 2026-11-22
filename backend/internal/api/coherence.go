package api

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"net/http"
	"strings"
	"themis/internal/categorize"
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
	ID       int64    `json:"id,omitempty"`
	Display  string   `json:"display"`
	Category string   `json:"category"`
	Tags     []string `json:"tags"`
}

// coherenceFinding : une incohérence détectée entre une règle et l'état réel
// des transactions.
type coherenceFinding struct {
	TransactionIDs []int64            `json:"transaction_ids,omitempty"`
	Rule           string             `json:"rule"`
	Kind           string             `json:"kind"` // wrong_category | missing_tags
	Detail         string             `json:"detail"`
	Count          int                `json:"count"`
	Query          string             `json:"query"` // pour rebondir vers les transactions concernées
	Examples       []coherenceExample `json:"examples"`
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

	tx, err := s.DB.BeginTx(r.Context(), &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		writeErr(w, 500, fmt.Errorf("audit indisponible"))
		return
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(r.Context(), `SELECT r.id,r.pattern,r.domain_id,d.name,COALESCE(r.tags,'') FROM rules r JOIN domains d ON d.id=r.domain_id AND d.owner_id=r.owner_id WHERE r.owner_id=$1 ORDER BY length(r.pattern) DESC,r.id LIMIT 2001`, owner)
	if err != nil {
		writeErr(w, 500, fmt.Errorf("règles indisponibles"))
		return
	}
	type rl struct {
		id, domID              int64
		pattern, domName, tags string
	}
	rules := []rl{}
	specs := []categorize.Rule{}
	names := map[int64]string{}
	for rows.Next() {
		var ru rl
		if err = rows.Scan(&ru.id, &ru.pattern, &ru.domID, &ru.domName, &ru.tags); err != nil {
			rows.Close()
			writeErr(w, 500, fmt.Errorf("règle illisible"))
			return
		}
		rules = append(rules, ru)
		names[ru.domID] = ru.domName
		specs = append(specs, categorize.Rule{ID: ru.id, Pattern: ru.pattern, DomainID: ru.domID, Tags: strings.Split(ru.tags, ",")})
	}
	err = rows.Err()
	rows.Close()
	if err != nil || len(rules) > 2000 {
		writeErr(w, 422, fmt.Errorf("audit incomplet : règles indisponibles ou limite dépassée"))
		return
	}
	eng := categorize.New(specs)
	trows, err := tx.QueryContext(r.Context(), `SELECT t.id,t.label,COALESCE(NULLIF(t.display_label,''),t.label),t.domain_id,t.classification_source,COALESCE(d.name,'Sans catégorie'),COALESCE((SELECT array_agg(tg.name ORDER BY tg.name) FROM transaction_tags tt JOIN tags tg ON tg.id=tt.tag_id AND tg.owner_id=t.owner_id WHERE tt.transaction_id=t.id),'{}'),COALESCE(cs.domain_locked,false),COALESCE(cs.tags_locked,false),COALESCE((SELECT array_agg(g.name) FROM classification_tag_decisions cd JOIN tags g ON g.id=cd.tag_id AND g.owner_id=cd.owner_id WHERE cd.owner_id=t.owner_id AND cd.transaction_id=t.id AND cd.decision='removed'),'{}') FROM transactions t LEFT JOIN classification_state cs ON cs.owner_id=t.owner_id AND cs.transaction_id=t.id LEFT JOIN domains d ON d.id=t.domain_id AND d.owner_id=t.owner_id WHERE t.owner_id=$1 ORDER BY t.id LIMIT 50001`, owner)
	if err != nil {
		writeErr(w, 500, fmt.Errorf("opérations indisponibles"))
		return
	}
	findings := []coherenceFinding{}
	indices := map[string]int{}
	scanned := 0
	for trows.Next() {
		var in categorize.Input
		var display, category string
		if err = trows.Scan(&in.ID, &in.Label, &display, &in.DomainID, &in.Source, &category, pq.Array(&in.Tags), &in.DomainLocked, &in.TagsLocked, pq.Array(&in.RemovedTags)); err != nil {
			trows.Close()
			writeErr(w, 500, fmt.Errorf("opération illisible"))
			return
		}
		scanned++
		if scanned > 50000 || len(rules)*scanned > 5000000 {
			trows.Close()
			writeErr(w, 422, fmt.Errorf("audit incomplet : limite dépassée, aucune conclusion produite"))
			return
		}
		in.ProtectionsKnown = true
		plan := eng.Plan(in)
		if plan.Winner == nil {
			continue
		}
		record := func(kind, detail string) {
			key := fmt.Sprintf("%d:%s", plan.Winner.ID, kind)
			i, ok := indices[key]
			if !ok {
				i = len(findings)
				indices[key] = i
				findings = append(findings, coherenceFinding{Rule: plan.Winner.Pattern, Kind: kind, Detail: detail, Examples: []coherenceExample{}, TransactionIDs: []int64{}})
			}
			f := &findings[i]
			f.Count++
			if len(f.TransactionIDs) < 1000 {
				f.TransactionIDs = append(f.TransactionIDs, in.ID)
			}
			if len(f.Examples) < 4 {
				f.Examples = append(f.Examples, coherenceExample{ID: in.ID, Display: display, Category: category, Tags: in.Tags})
			}
		}
		if in.DomainID == nil || *in.DomainID != *plan.ProposedDomain {
			detail := fmt.Sprintf("Règle gagnante « %s » : proposition %s. Le domaine déjà attribué est conservé ; vérifier le choix avant toute modification.", plan.Winner.Pattern, names[*plan.ProposedDomain])
			kind := "wrong_category"
			if plan.ManualDomainProtected || in.DomainLocked {
				kind = "manual_choice"
				detail = "Choix manuel conservé, y compris Sans domaine ; la règle gagnante propose une autre classification. Aucune correction automatique du domaine."
			} else if in.DomainID == nil {
				detail = "Sans domaine : la règle gagnante propose une catégorie ; appliquer les règles pourra la compléter."
			}
			record(kind, detail)
		}
		if len(plan.AddTags) > 0 {
			record("missing_tags", "Tags proposés par la seule règle gagnante. Les tags existants, les verrous et les retraits tracés sont respectés.")
		}
	}
	err = trows.Err()
	trows.Close()
	if err != nil {
		writeErr(w, 500, fmt.Errorf("audit interrompu"))
		return
	}
	for _, invalid := range eng.Evaluate("").Invalid {
		for _, ru := range rules {
			if ru.id == invalid.ID {
				findings = append(findings, coherenceFinding{Rule: ru.pattern, Kind: "invalid_rule", Detail: "Motif invalide : règle ignorée par le moteur, à corriger explicitement.", Examples: []coherenceExample{}})
			}
		}
	}
	tx.Rollback() // Release the snapshot before optional network work.
	out := map[string]any{"findings": findings, "rules_checked": len(rules), "transactions_checked": scanned, "policy": "winning_rule_only"}

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
				       COALESCE((SELECT string_agg(tg.name, ',') FROM transaction_tags tt JOIN tags tg ON tg.id=tt.tag_id AND tg.owner_id=t.owner_id WHERE tt.transaction_id=t.id), '')
				FROM transactions t LEFT JOIN domains d ON d.id = t.domain_id AND d.owner_id=t.owner_id
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
