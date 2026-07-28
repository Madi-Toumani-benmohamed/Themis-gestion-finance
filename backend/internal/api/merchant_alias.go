package api

import (
	"themis/internal/normalize"
	"database/sql"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// applyAlias renvoie le surnom utilisateur s'il existe pour ce marchand/libellé.
// Priorité : le surnom prime sur le dictionnaire et sur toute suggestion IA.
func (s *Server) aliasFor(owner int64, merchant, rawLabel string) (string, bool) {
	key := merchant
	if key == "" {
		key = rawLabel
	}
	var disp string
	err := s.DB.QueryRow(`SELECT display FROM merchant_aliases WHERE owner_id = $1 AND lower(match_key) = lower($2)`, owner, key).Scan(&disp)
	if err == nil && disp != "" {
		return disp, true
	}
	return "", false
}

// listAliases renvoie les surnoms définis par l'utilisateur.
func (s *Server) listAliases(w http.ResponseWriter, r *http.Request) {
	owner := s.ownerID(r)
	rows, err := s.DB.Query(`SELECT id, match_key, display FROM merchant_aliases WHERE owner_id = $1 ORDER BY display`, owner)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id int64
		var k, d string
		if rows.Scan(&id, &k, &d) == nil {
			out = append(out, map[string]any{"id": id, "match_key": k, "display": d})
		}
	}
	writeJSON(w, http.StatusOK, out)
}

// setAlias crée ou met à jour un surnom, puis l'applique immédiatement à toutes
// les transactions concernées (le renommage doit être visible tout de suite).
func (s *Server) setAlias(w http.ResponseWriter, r *http.Request) {
	owner := s.ownerID(r)
	var in struct {
		MatchKey  string `json:"match_key"`
		Display   string `json:"display"`
		IsPattern bool   `json:"is_pattern"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	in.MatchKey = strings.TrimSpace(in.MatchKey)
	in.Display = strings.TrimSpace(in.Display)
	if in.MatchKey == "" || in.Display == "" {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("clé et nom requis"))
		return
	}
	if _, err := s.DB.Exec(`
		INSERT INTO merchant_aliases (owner_id, match_key, display, is_pattern) VALUES ($1, $2, $3, $4)
		ON CONFLICT (owner_id, lower(match_key)) DO UPDATE SET display = EXCLUDED.display, is_pattern = EXCLUDED.is_pattern`,
		owner, in.MatchKey, in.Display, in.IsPattern); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	// Application immédiate. En mode motif, on matche toute transaction dont le
	// libellé CONTIENT la clé (couvre les références variables). Sinon, matche
	// exact sur le marchand (ou le libellé brut à défaut).
	var res sql.Result
	if in.IsPattern {
		res, _ = s.DB.Exec(`
			UPDATE transactions SET display_label = $1
			WHERE owner_id = $2 AND (label ILIKE '%'||$3||'%' OR merchant ILIKE '%'||$3||'%')`,
			in.Display, owner, in.MatchKey)
	} else {
		res, _ = s.DB.Exec(`
			UPDATE transactions SET display_label = $1
			WHERE owner_id = $2 AND lower(COALESCE(NULLIF(merchant,''), label)) = lower($3)`,
			in.Display, owner, in.MatchKey)
	}
	n := int64(0)
	if res != nil {
		n, _ = res.RowsAffected()
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "updated": n})
}

// deleteAlias retire un surnom et re-normalise ses transactions (retour au nom
// automatique).
func (s *Server) deleteAlias(w http.ResponseWriter, r *http.Request) {
	owner := s.ownerID(r)
	id := r.PathValue("id")
	var key string
	_ = s.DB.QueryRow(`SELECT match_key FROM merchant_aliases WHERE id = $1 AND owner_id = $2`, id, owner).Scan(&key)
	if _, err := s.DB.Exec(`DELETE FROM merchant_aliases WHERE id = $1 AND owner_id = $2`, id, owner); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	// re-normaliser les transactions qui portaient ce surnom
	if key != "" {
		s.renormalizeMatching(owner, key)
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// unrecognizedMerchants liste les marchands/ libellés que le pipeline n'a pas
// su nommer proprement, groupés, avec le nombre de transactions et le total.
// C'est la file de travail du renommage : les plus fréquents d'abord.
func (s *Server) unrecognizedMerchants(w http.ResponseWriter, r *http.Request) {
	owner := s.ownerID(r)
	// « non reconnu » = pas d'entrée du dictionnaire (label_kind vide ET
	// merchant == display, i.e. le nom vient du simple nettoyage) et pas de
	// surnom utilisateur.
	rows, err := s.DB.Query(`
		SELECT COALESCE(NULLIF(t.merchant,''), t.label) AS key,
		       MAX(t.display_label) AS display,
		       count(*) AS n, COALESCE(SUM(-t.amount) FILTER (WHERE t.amount < 0),0) AS spent
		FROM transactions t
		WHERE t.owner_id = $1
		  AND t.label_kind = ''
		  AND NOT EXISTS (SELECT 1 FROM merchant_aliases a WHERE a.owner_id = $1 AND lower(a.match_key) = lower(COALESCE(NULLIF(t.merchant,''), t.label)))
		GROUP BY key
		HAVING count(*) >= 1
		ORDER BY n DESC, spent DESC
		LIMIT 40`, owner)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var key, disp string
		var n int
		var spent float64
		if rows.Scan(&key, &disp, &n, &spent) == nil {
			out = append(out, map[string]any{"key": key, "display": disp, "count": n, "spent": spent})
		}
	}
	writeJSON(w, http.StatusOK, out)
}

// suggestMerchantNames demande à l'IA de proposer un nom propre pour des
// marchands non reconnus. L'IA PROPOSE seulement : rien n'est appliqué sans un
// clic de l'utilisateur (qui créera alors le surnom). Sans LLM configuré,
// renvoie un message clair.
func (s *Server) suggestMerchantNames(w http.ResponseWriter, r *http.Request) {
	client := s.getLLM()
	if client == nil {
		writeJSON(w, http.StatusOK, map[string]any{"suggestions": []any{}, "ai_error": "IA non configurée (onglet Admin)."})
		return
	}
	var in struct {
		Keys []string `json:"keys"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || len(in.Keys) == 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("aucun marchand à nommer"))
		return
	}
	if len(in.Keys) > 20 {
		in.Keys = in.Keys[:20]
	}
	var b strings.Builder
	b.WriteString("Pour chaque libellé bancaire ci-dessous, donne un nom de commerçant court et lisible.\n")
	b.WriteString("Réponds UNIQUEMENT avec un tableau JSON, rien d'autre. Format exact :\n")
	b.WriteString(`[{"key":"LIBELLE EXACT","name":"Nom lisible"}]` + "\n\n")
	b.WriteString("Libellés :\n")
	for _, k := range in.Keys {
		fmt.Fprintf(&b, "- %s\n", k)
	}
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	txt, err := client.Complete(ctx,
		"Tu nommes des commerçants à partir de libellés bancaires. Réponds uniquement avec un tableau JSON valide, sans aucun texte avant ou après.",
		b.String(), 800, true)
	if err != nil {
		// Erreur de communication avec le modèle (timeout, indisponible…) :
		// message distinct du cas « réponse illisible », pour le diagnostic.
		writeJSON(w, http.StatusOK, map[string]any{
			"suggestions": []any{},
			"ai_error":    "L'IA n'a pas répondu : " + err.Error(),
		})
		return
	}
	sugg, perr := parseSuggestions(txt)
	if perr != nil {
		// Repli déterministe : plutôt que d'échouer sèchement, on propose le
		// libellé nettoyé par le normaliseur — souvent déjà lisible. L'IA reste
		// un bonus, pas un point de défaillance unique.
		fallback := []map[string]string{}
		for _, k := range in.Keys {
			res := normalize.Label(k, -1)
			if res.Display != "" && !strings.EqualFold(res.Display, k) {
				fallback = append(fallback, map[string]string{"key": k, "name": res.Display})
			}
		}
		// Extrait de la réponse brute (tronqué) pour comprendre ce que le modèle
		// a renvoyé — visible dans la réponse pour diagnostic.
		preview := strings.TrimSpace(txt)
		if len(preview) > 200 {
			preview = preview[:200] + "…"
		}
		msg := "L'IA n'a pas renvoyé de JSON exploitable."
		if len(fallback) > 0 {
			msg = "L'IA n'a pas renvoyé de JSON exploitable — voici des noms nettoyés automatiquement, à valider."
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"suggestions": fallback,
			"ai_error":    msg,
			"ai_raw":      preview, // aperçu de la réponse brute (diagnostic)
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"suggestions": sugg})
}

// parseSuggestions extrait une liste {key,name} d'une réponse LLM tolérante :
// texte autour, bloc ```json, ou objet englobant {"suggestions":[...]}.
func parseSuggestions(txt string) ([]map[string]string, error) {
	txt = strings.TrimSpace(txt)
	// isoler le premier tableau JSON [ ... ] présent dans la réponse
	start := strings.Index(txt, "[")
	end := strings.LastIndex(txt, "]")
	var out []map[string]string
	if start >= 0 && end > start {
		if err := json.Unmarshal([]byte(txt[start:end+1]), &out); err == nil {
			return out, nil
		}
	}
	// sinon, tenter un objet englobant {"suggestions": [...]} ou {"key":...}
	objStart := strings.Index(txt, "{")
	objEnd := strings.LastIndex(txt, "}")
	if objStart >= 0 && objEnd > objStart {
		blob := txt[objStart : objEnd+1]
		var wrap struct {
			Suggestions []map[string]string `json:"suggestions"`
		}
		if err := json.Unmarshal([]byte(blob), &wrap); err == nil && len(wrap.Suggestions) > 0 {
			return wrap.Suggestions, nil
		}
		// Format fréquent des modèles locaux : un objet plat clé→nom, du type
		// {"CB ACHEEL FR": "Acheel", "CB CARREFOUR": "Carrefour"}. On le
		// convertit en la structure attendue.
		var flat map[string]string
		if err := json.Unmarshal([]byte(blob), &flat); err == nil && len(flat) > 0 {
			out = out[:0]
			for k, v := range flat {
				if strings.TrimSpace(v) != "" {
					out = append(out, map[string]string{"key": k, "name": v})
				}
			}
			if len(out) > 0 {
				return out, nil
			}
		}
	}
	// Dernier recours : certains modèles ignorent la consigne JSON et répondent
	// en liste markdown « - LIBELLE : Nom » ou « LIBELLE => Nom ». On l'extrait.
	if lines := parseMarkdownPairs(txt); len(lines) > 0 {
		return lines, nil
	}
	return nil, fmt.Errorf("aucun tableau JSON exploitable")
}

// parseMarkdownPairs récupère les paires « clé : valeur » d'une réponse en
// texte libre (puces markdown, flèches). Tolérant aux préfixes « - », « * »,
// « • » et aux séparateurs « : », « => », « → », « - ».
func parseMarkdownPairs(txt string) []map[string]string {
	var out []map[string]string
	for _, line := range strings.Split(txt, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimLeft(line, "-*•\t ")
		if line == "" {
			continue
		}
		var sep string
		for _, s := range []string{" => ", "=>", " → ", "→", " : ", ": ", " - "} {
			if strings.Contains(line, s) {
				sep = s
				break
			}
		}
		if sep == "" {
			continue
		}
		parts := strings.SplitN(line, sep, 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.Trim(strings.TrimSpace(parts[0]), `"'«»`)
		name := strings.Trim(strings.TrimSpace(parts[1]), `"'«».`)
		if key != "" && name != "" && len(name) < 60 {
			out = append(out, map[string]string{"key": key, "name": name})
		}
	}
	return out
}
