package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

/* ==========================================================================
   Multi-comptes
   ========================================================================== */

type account struct {
	ID           int64    `json:"id"`
	Name         string   `json:"name"`
	Kind         string   `json:"kind"`
	InterestRate *float64 `json:"interest_rate"`
	DepositLimit *float64 `json:"deposit_limit"`
	Balance      *float64 `json:"balance"`      // dernier solde connu
	BalanceDate  *string  `json:"balance_date"` // date de ce solde
	Statements   int      `json:"statements"`
}

var accountKinds = map[string]bool{
	"courant": true, "livret_a": true, "ldds": true, "pel": true, "bourse": true, "autre": true,
}

// listAccounts retourne les comptes avec leur dernier solde connu :
// dernier relevé importé, ou dernier pointage manuel — le plus récent des deux.
func (s *Server) listAccounts(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.Query(`
		SELECT a.id, a.name, a.kind, a.interest_rate, a.deposit_limit,
		       (SELECT count(*) FROM statements st WHERE st.account_id = a.id) AS stmts,
		       lastpos.balance, lastpos.d
		FROM accounts a
		LEFT JOIN LATERAL (
			SELECT balance, d FROM (
				SELECT st.closing_balance AS balance, st.period_end::date AS d
				FROM statements st
				WHERE st.account_id = a.id AND st.closing_balance IS NOT NULL
				UNION ALL
				SELECT sn.balance, sn.snap_date FROM account_snapshots sn WHERE sn.account_id = a.id
			) x WHERE d IS NOT NULL ORDER BY d DESC LIMIT 1
		) lastpos ON true
		WHERE a.owner_id = $1
		ORDER BY a.id`, s.ownerID(r))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	out := []account{}
	for rows.Next() {
		var a account
		var d sql.NullString
		var bal sql.NullFloat64
		if err := rows.Scan(&a.ID, &a.Name, &a.Kind, &a.InterestRate, &a.DepositLimit, &a.Statements, &bal, &d); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		if bal.Valid {
			a.Balance = &bal.Float64
		}
		if d.Valid {
			s := d.String
			if len(s) > 10 {
				s = s[:10]
			}
			a.BalanceDate = &s
		}
		out = append(out, a)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) createAccount(w http.ResponseWriter, r *http.Request) {
	var in account
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || strings.TrimSpace(in.Name) == "" {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("nom requis"))
		return
	}
	if !accountKinds[in.Kind] {
		in.Kind = "autre"
	}
	err := s.DB.QueryRow(`INSERT INTO accounts (name, kind, interest_rate, deposit_limit, owner_id)
	                      VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		strings.TrimSpace(in.Name), in.Kind, in.InterestRate, in.DepositLimit, s.ownerID(r)).Scan(&in.ID)
	if err != nil {
		writeErr(w, http.StatusConflict, fmt.Errorf("un compte porte déjà ce nom"))
		return
	}
	writeJSON(w, http.StatusCreated, in)
}

func (s *Server) deleteAccount(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	owner := s.ownerID(r)
	var n int
	_ = s.DB.QueryRow(`SELECT count(*) FROM statements WHERE account_id = $1 AND owner_id = $2`, id, owner).Scan(&n)
	if n > 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("%d relevé(s) rattaché(s) : supprime-les d'abord", n))
		return
	}
	if _, err := s.DB.Exec(`DELETE FROM accounts WHERE id = $1 AND owner_id = $2`, id, owner); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "supprimé"})
}

// addSnapshot enregistre un pointage manuel de solde (livrets, bourse…).
func (s *Server) addSnapshot(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var in struct {
		Date    string  `json:"date"`
		Balance float64 `json:"balance"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("corps JSON invalide"))
		return
	}
	if _, err := time.Parse("2006-01-02", in.Date); err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("date au format YYYY-MM-DD requise"))
		return
	}
	// Vérifier que le compte appartient à l'utilisateur avant d'y ajouter un point.
	var ok bool
	_ = s.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM accounts WHERE id = $1 AND owner_id = $2)`, id, s.ownerID(r)).Scan(&ok)
	if !ok {
		writeErr(w, http.StatusNotFound, fmt.Errorf("compte introuvable"))
		return
	}
	_, err := s.DB.Exec(`INSERT INTO account_snapshots (account_id, snap_date, balance) VALUES ($1, $2, $3)
	                     ON CONFLICT (account_id, snap_date) DO UPDATE SET balance = EXCLUDED.balance`,
		id, in.Date, in.Balance)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

/* ==========================================================================
   Crédits & taxes : les échéances
   ========================================================================== */

// Les engagements sont définis dans engagement_*.go.

/* ==========================================================================
   Domaines (hiérarchiques) & tags
   ========================================================================== */

type domain struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	ParentID *int64 `json:"parent_id"`
	Color    string `json:"color"`
}

func (s *Server) listDomains(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.Query(`SELECT id, name, parent_id, color FROM domains WHERE owner_id = $1 ORDER BY COALESCE(parent_id, id), id`, s.ownerID(r))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	out := []domain{}
	for rows.Next() {
		var d domain
		if err := rows.Scan(&d.ID, &d.Name, &d.ParentID, &d.Color); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		out = append(out, d)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) createDomain(w http.ResponseWriter, r *http.Request) {
	var in domain
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || strings.TrimSpace(in.Name) == "" {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("nom requis"))
		return
	}
	if in.ParentID != nil { // un seul niveau de sous-domaines
		var pp *int64
		if err := s.DB.QueryRow(`SELECT parent_id FROM domains WHERE id = $1 AND owner_id = $2`, *in.ParentID, s.ownerID(r)).Scan(&pp); err != nil {
			writeErr(w, http.StatusBadRequest, fmt.Errorf("domaine parent introuvable"))
			return
		}
		if pp != nil {
			writeErr(w, http.StatusBadRequest, fmt.Errorf("un sous-domaine ne peut pas avoir de sous-domaine"))
			return
		}
	}
	if in.Color == "" {
		in.Color = "#66736C"
	}
	err := s.DB.QueryRow(`INSERT INTO domains (name, parent_id, color, owner_id) VALUES ($1, $2, $3, $4) RETURNING id`,
		strings.TrimSpace(in.Name), in.ParentID, in.Color, s.ownerID(r)).Scan(&in.ID)
	if err != nil {
		writeErr(w, http.StatusConflict, fmt.Errorf("ce domaine existe déjà à ce niveau"))
		return
	}
	writeJSON(w, http.StatusCreated, in)
}

// updateDomain modifie la couleur et/ou le nom d'une catégorie. Permet à
// l'utilisateur de choisir ses couleurs plutôt que de subir l'aléatoire.
func (s *Server) updateDomain(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	var in struct {
		Name  *string `json:"name"`
		Color *string `json:"color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	owner := s.ownerID(r)
	// Construire un UPDATE dynamique selon les champs fournis.
	if in.Color != nil && *in.Color != "" {
		if _, err := s.DB.Exec(`UPDATE domains SET color = $1 WHERE id = $2 AND owner_id = $3`, *in.Color, id, owner); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
	}
	if in.Name != nil && strings.TrimSpace(*in.Name) != "" {
		if _, err := s.DB.Exec(`UPDATE domains SET name = $1 WHERE id = $2 AND owner_id = $3`, strings.TrimSpace(*in.Name), id, owner); err != nil {
			writeErr(w, http.StatusConflict, fmt.Errorf("ce nom existe déjà"))
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) deleteDomain(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if _, err := s.DB.Exec(`DELETE FROM domains WHERE id = $1 AND owner_id = $2`, id, s.ownerID(r)); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "supprimé"})
}

func (s *Server) listTags(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.Query(`SELECT t.name, count(tt.transaction_id)
	                         FROM tags t LEFT JOIN transaction_tags tt ON tt.tag_id = t.id
	                         WHERE t.owner_id = $1
	                         GROUP BY t.id ORDER BY t.name`, s.ownerID(r))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	type tag struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	out := []tag{}
	for rows.Next() {
		var t tag
		if err := rows.Scan(&t.Name, &t.Count); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		out = append(out, t)
	}
	writeJSON(w, http.StatusOK, out)
}
