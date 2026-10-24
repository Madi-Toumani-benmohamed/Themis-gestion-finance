package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/lib/pq"
	"themis/internal/models"
)

const importPreviewTTL = 15 * time.Minute
const importPreviewMaxBytes = 32 << 20

type pendingImport struct {
	Owner         int64
	AccountID     int64
	Filename      string
	Result        *models.ParseResult
	ProgressToken string
	Expires       time.Time
	Size          int
}
type importPreviewStore struct {
	mu    sync.Mutex
	items map[string]pendingImport
}

func parsedImportSize(res *models.ParseResult) int {
	size := len(res.PeriodStart) + len(res.PeriodEnd) + len(res.Parser) + 256
	for _, t := range res.Transactions {
		size += len(t.Label) + len(t.Date) + len(t.Hash) + len(t.Display) + len(t.Merchant) + 256
		for _, tag := range t.Tags {
			size += len(tag)
		}
	}
	return size
}
func (s *importPreviewStore) put(p pendingImport, now time.Time) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.items == nil {
		s.items = map[string]pendingImport{}
	}
	total, owned := 0, 0
	for k, v := range s.items {
		if !v.Expires.After(now) {
			delete(s.items, k)
			continue
		}
		total += v.Size
		if v.Owner == p.Owner {
			owned++
		}
	}
	p.Size = parsedImportSize(p.Result)
	if owned >= 3 || len(s.items) >= 32 || total+p.Size > importPreviewMaxBytes {
		return "", fmt.Errorf("trop d’aperçus en attente ; annule un aperçu ou réessaie plus tard")
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	p.Expires = now.Add(importPreviewTTL)
	s.items[token] = p
	return token, nil
}

// take consomme atomiquement le jeton : une confirmation ne s'exécute qu'une fois.
func (s *importPreviewStore) take(token string, owner int64, now time.Time) (pendingImport, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.items[token]
	if !ok || p.Owner != owner {
		return pendingImport{}, false
	}
	delete(s.items, token)
	if !p.Expires.After(now) {
		return pendingImport{}, false
	}
	return p, true
}
func (s *importPreviewStore) discard(token string, owner int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p, ok := s.items[token]; ok && p.Owner == owner {
		delete(s.items, token)
	}
}

type importPreviewTransaction struct {
	Date      string  `json:"date"`
	Label     string  `json:"label"`
	Amount    float64 `json:"amount"`
	Duplicate bool    `json:"duplicate"`
}
type importPreviewResponse struct {
	Token             string                     `json:"token"`
	Expires           time.Time                  `json:"expires_at"`
	Filename          string                     `json:"filename"`
	AccountID         int64                      `json:"account_id"`
	AccountName       string                     `json:"account_name"`
	Parser            string                     `json:"parser"`
	Validated         bool                       `json:"validated"`
	PeriodStart       string                     `json:"period_start"`
	PeriodEnd         string                     `json:"period_end"`
	OpeningBalance    *float64                   `json:"opening_balance"`
	ClosingBalance    *float64                   `json:"closing_balance"`
	BalanceDifference *float64                   `json:"balance_difference"`
	NewCount          int                        `json:"new_count"`
	Duplicates        int                        `json:"duplicates"`
	Transactions      []importPreviewTransaction `json:"transactions"`
}

// Les empreintes et la portée (owner, hash) sont exactement celles de persist.
// L'aperçu ne modifie pas les transactions et ne renvoie pas leurs empreintes.
func describeImport(res *models.ParseResult, existing map[string]bool) importPreviewResponse {
	out := importPreviewResponse{Parser: res.Parser, Validated: res.Validated, PeriodStart: res.PeriodStart, PeriodEnd: res.PeriodEnd, OpeningBalance: res.OpeningBalance, ClosingBalance: res.ClosingBalance, Transactions: []importPreviewTransaction{}}
	seen := map[string]bool{}
	sum := 0.0
	for _, t := range res.Transactions {
		duplicate := existing[t.Hash] || seen[t.Hash]
		seen[t.Hash] = true
		if duplicate {
			out.Duplicates++
		} else {
			out.NewCount++
		}
		sum += t.Amount
		out.Transactions = append(out.Transactions, importPreviewTransaction{t.Date, t.Label, t.Amount, duplicate})
	}
	if res.OpeningBalance != nil && res.ClosingBalance != nil {
		diff := *res.OpeningBalance + sum - *res.ClosingBalance
		out.BalanceDifference = &diff
	}
	return out
}

func (s *Server) createImportPreview(w http.ResponseWriter, r *http.Request, filename string, res *models.ParseResult, accountID int64, progressToken string) {
	w.Header().Set("Cache-Control", "no-store")
	owner := s.ownerID(r)
	if len(res.Transactions) > 20000 {
		writeErr(w, http.StatusRequestEntityTooLarge, fmt.Errorf("plus de 20 000 opérations détectées ; utilise un export plus court"))
		return
	}
	var name string
	if err := s.DB.QueryRowContext(r.Context(), `SELECT name FROM accounts WHERE id=$1 AND owner_id=$2`, accountID, owner).Scan(&name); err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("compte cible indisponible"))
		return
	}
	hashes := make([]string, 0, len(res.Transactions))
	for _, t := range res.Transactions {
		hashes = append(hashes, t.Hash)
	}
	rows, err := s.DB.QueryContext(r.Context(), `SELECT hash FROM transactions WHERE owner_id=$1 AND hash=ANY($2)`, owner, pq.Array(hashes))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Errorf("vérification des doublons impossible"))
		return
	}
	existing := map[string]bool{}
	for rows.Next() {
		var hash string
		if err = rows.Scan(&hash); err != nil {
			break
		}
		existing[hash] = true
	}
	rowErr := rows.Err()
	rows.Close()
	if err != nil || rowErr != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Errorf("vérification des doublons impossible"))
		return
	}
	out := describeImport(res, existing)
	now := time.Now()
	token, err := s.importPreviews.put(pendingImport{Owner: owner, AccountID: accountID, Filename: filename, Result: res, ProgressToken: progressToken}, now)
	if err != nil {
		writeErr(w, http.StatusTooManyRequests, err)
		return
	}
	// Le timer ne capture pas les données bancaires. Aucune donnée d'aperçu
	// n'est persistée ; annulation, expiration ou redémarrage les suppriment.
	time.AfterFunc(importPreviewTTL, func() { s.importPreviews.discard(token, owner) })
	out.Token = token
	out.Expires = now.Add(importPreviewTTL)
	out.Filename = filename
	out.AccountID = accountID
	out.AccountName = name
	setUploadProgress(progressToken, owner, "Aperçu prêt — confirmation requise", "", 100, true)
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) confirmImportPreview(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	owner := s.ownerID(r)
	p, ok := s.importPreviews.take(r.PathValue("token"), owner, time.Now())
	if !ok {
		writeErr(w, http.StatusGone, fmt.Errorf("aperçu expiré, annulé ou déjà confirmé ; sélectionne à nouveau le fichier"))
		return
	}
	var allowed bool
	err := s.DB.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM accounts WHERE id=$1 AND owner_id=$2)`, p.AccountID, owner).Scan(&allowed)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Errorf("vérification du compte impossible ; recrée un aperçu"))
		return
	}
	if !allowed {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("compte cible indisponible ; recrée un aperçu"))
		return
	}
	// Déduplication revérifiée par la contrainte SQL au moment de l'insertion.
	s.finishStatementImport(w, r, p.Filename, p.Result, p.AccountID, owner, p.ProgressToken)
}
func (s *Server) cancelImportPreview(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	s.importPreviews.discard(r.PathValue("token"), s.ownerID(r))
	w.WriteHeader(http.StatusNoContent)
}
