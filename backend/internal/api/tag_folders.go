package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ── Vue « dossier » par tag ──
// Un tag peut représenter un projet/période (« Vacances 2026 ») ou une simple
// étiquette récurrente (« midi »). Seuls les premiers méritent un « dossier ».
// La détection est automatique (montant + concentration temporelle) et
// l'utilisateur peut épingler/masquer un tag pour outrepasser l'heuristique.

const (
	folderMinTotal   = 150.0 // € : en-dessous, un tag n'est pas un « projet »
	folderConcentr   = 0.70  // ≥70 % des tx sur une fenêtre courte
	folderWindowMon  = 2     // fenêtre « courte » en mois
)

type tagFolder struct {
	Name        string  `json:"name"`
	Total       float64 `json:"total"`
	Count       int     `json:"count"`
	FirstDate   string  `json:"first_date"`
	LastDate    string  `json:"last_date"`
	Days        int     `json:"days"`
	IsProject   bool    `json:"is_project"` // détecté comme projet (avant préférence)
	Pinned      bool    `json:"pinned"`
	Hidden      bool    `json:"hidden"`
	ShowAsFolder bool   `json:"show_as_folder"` // décision finale (auto + préférence)
}

// tagFolders liste tous les tags avec leur statut « dossier », en séparant les
// projets (mis en avant) du reste (étiquettes récurrentes).
func (s *Server) tagFolders(w http.ResponseWriter, r *http.Request) {
	owner := s.ownerID(r)

	prefs := map[string]string{}
	if prows, err := s.DB.Query(`SELECT tag_name, state FROM tag_folder_prefs WHERE owner_id = $1`, owner); err == nil {
		for prows.Next() {
			var n, st string
			if prows.Scan(&n, &st) == nil {
				prefs[strings.ToLower(n)] = st
			}
		}
		prows.Close()
	}

	rows, err := s.DB.Query(`
		WITH tag_tx AS (
			SELECT tg.name, t.tx_date, -t.amount AS amt
			FROM tags tg
			JOIN transaction_tags tt ON tt.tag_id = tg.id
			JOIN transactions t ON t.id = tt.transaction_id
			WHERE tg.owner_id = $1 AND t.amount < 0
			  AND (t.domain_id IS NULL OR t.domain_id NOT IN (SELECT id FROM domains WHERE is_transfer))
		)
		SELECT name,
		       COALESCE(SUM(amt),0) AS total,
		       COUNT(*) AS n,
		       MIN(tx_date)::text, MAX(tx_date)::text,
		       -- concentration : nb max de tx dans une fenêtre glissante de 2 mois,
		       -- rapporté au total. Calcul par auto-jointure sur la fenêtre.
		       (SELECT MAX(cnt) FROM (
		            SELECT COUNT(*) AS cnt
		            FROM tag_tx a
		            JOIN tag_tx b ON b.name = a.name
		                 AND b.tx_date >= a.tx_date
		                 AND b.tx_date < a.tx_date + INTERVAL '2 months'
		            WHERE a.name = tag_tx.name
		            GROUP BY a.tx_date
		        ) w)::float / GREATEST(COUNT(*),1) AS concentration
		FROM tag_tx
		GROUP BY name
		ORDER BY total DESC`, owner)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	folders := []tagFolder{}
	others := []tagFolder{}
	for rows.Next() {
		var f tagFolder
		var concentration float64
		if err := rows.Scan(&f.Name, &f.Total, &f.Count, &f.FirstDate, &f.LastDate, &concentration); err != nil {
			continue
		}
		f.Days = daysBetween(f.FirstDate, f.LastDate)
		f.IsProject = f.Total >= folderMinTotal && f.Count >= 2 && concentration >= folderConcentr

		st := prefs[strings.ToLower(f.Name)]
		f.Pinned = st == "pinned"
		f.Hidden = st == "hidden"
		f.ShowAsFolder = (f.IsProject || f.Pinned) && !f.Hidden

		if f.ShowAsFolder {
			folders = append(folders, f)
		} else {
			others = append(others, f)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"folders": folders, "others": others})
}


// tagFolderDetail : le contenu complet d'un dossier — total, durée, répartition
// par catégorie, marchands, et les transactions.
func (s *Server) tagFolderDetail(w http.ResponseWriter, r *http.Request) {
	owner := s.ownerID(r)
	name := strings.TrimSpace(r.PathValue("name"))
	if name == "" {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("nom de tag requis"))
		return
	}

	var total float64
	var count int
	var firstD, lastD string
	_ = s.DB.QueryRow(`
		SELECT COALESCE(SUM(-t.amount) FILTER (WHERE t.amount<0),0), COUNT(*),
		       COALESCE(MIN(t.tx_date)::text,''), COALESCE(MAX(t.tx_date)::text,'')
		FROM tags tg JOIN transaction_tags tt ON tt.tag_id=tg.id JOIN transactions t ON t.id=tt.transaction_id
		WHERE tg.owner_id=$1 AND tg.name=$2
		  AND (t.domain_id IS NULL OR t.domain_id NOT IN (SELECT id FROM domains WHERE is_transfer))`,
		owner, name).Scan(&total, &count, &firstD, &lastD)

	// répartition par catégorie
	byCat := []map[string]any{}
	if crows, err := s.DB.Query(`
		SELECT COALESCE(d.name,'Sans catégorie'), SUM(-t.amount)
		FROM tags tg JOIN transaction_tags tt ON tt.tag_id=tg.id JOIN transactions t ON t.id=tt.transaction_id
		LEFT JOIN domains d ON d.id=t.domain_id
		WHERE tg.owner_id=$1 AND tg.name=$2 AND t.amount<0
		  AND (t.domain_id IS NULL OR t.domain_id NOT IN (SELECT id FROM domains WHERE is_transfer))
		GROUP BY d.name ORDER BY 2 DESC`, owner, name); err == nil {
		for crows.Next() {
			var n string
			var v float64
			if crows.Scan(&n, &v) == nil {
				byCat = append(byCat, map[string]any{"category": n, "total": v})
			}
		}
		crows.Close()
	}

	// principaux marchands
	byMerchant := []map[string]any{}
	if mrows, err := s.DB.Query(`
		SELECT COALESCE(NULLIF(t.merchant,''), t.display_label, t.label), SUM(-t.amount), COUNT(*)
		FROM tags tg JOIN transaction_tags tt ON tt.tag_id=tg.id JOIN transactions t ON t.id=tt.transaction_id
		WHERE tg.owner_id=$1 AND tg.name=$2 AND t.amount<0
		  AND (t.domain_id IS NULL OR t.domain_id NOT IN (SELECT id FROM domains WHERE is_transfer))
		GROUP BY 1 ORDER BY 2 DESC LIMIT 8`, owner, name); err == nil {
		for mrows.Next() {
			var n string
			var v float64
			var c int
			if mrows.Scan(&n, &v, &c) == nil {
				byMerchant = append(byMerchant, map[string]any{"name": n, "total": v, "count": c})
			}
		}
		mrows.Close()
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"name": name, "total": total, "count": count,
		"first_date": firstD, "last_date": lastD, "days": daysBetween(firstD, lastD),
		"by_category": byCat, "by_merchant": byMerchant,
	})
}

// setTagFolderPref épingle ('pinned'), masque ('hidden') ou réinitialise
// ('auto') le statut dossier d'un tag.
func (s *Server) setTagFolderPref(w http.ResponseWriter, r *http.Request) {
	owner := s.ownerID(r)
	name := strings.TrimSpace(r.PathValue("name"))
	var in struct {
		State string `json:"state"` // pinned | hidden | auto
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	switch in.State {
	case "auto":
		_, _ = s.DB.Exec(`DELETE FROM tag_folder_prefs WHERE owner_id=$1 AND tag_name=$2`, owner, name)
	case "pinned", "hidden":
		_, err := s.DB.Exec(`
			INSERT INTO tag_folder_prefs (owner_id, tag_name, state) VALUES ($1,$2,$3)
			ON CONFLICT (owner_id, tag_name) DO UPDATE SET state=EXCLUDED.state`, owner, name, in.State)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
	default:
		writeErr(w, http.StatusBadRequest, fmt.Errorf("state doit être pinned, hidden ou auto"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func daysBetween(a, b string) int {
	ta, e1 := time.Parse("2006-01-02", a)
	tb, e2 := time.Parse("2006-01-02", b)
	if e1 != nil || e2 != nil {
		return 0
	}
	return int(tb.Sub(ta).Hours()/24) + 1
}
