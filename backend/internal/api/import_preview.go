package api

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"themis/internal/parser"
	"time"

	"github.com/lib/pq"
	"themis/internal/models"
	"themis/internal/money"
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
	if res.Report != nil {
		raw, _ := json.Marshal(res.Report)
		size += len(raw)
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
	Report              *models.ImportReport       `json:"import_report,omitempty"`
	BlockedReason       string                     `json:"blocked_reason,omitempty"`
	PotentialDuplicates int                        `json:"potential_duplicates"`
	Token               string                     `json:"token"`
	Expires             time.Time                  `json:"expires_at"`
	Filename            string                     `json:"filename"`
	AccountID           int64                      `json:"account_id"`
	AccountName         string                     `json:"account_name"`
	Parser              string                     `json:"parser"`
	Validated           bool                       `json:"validated"`
	PeriodStart         string                     `json:"period_start"`
	PeriodEnd           string                     `json:"period_end"`
	OpeningBalance      *float64                   `json:"opening_balance"`
	ClosingBalance      *float64                   `json:"closing_balance"`
	BalanceDifference   *float64                   `json:"balance_difference"`
	NewCount            int                        `json:"new_count"`
	Duplicates          int                        `json:"duplicates"`
	Transactions        []importPreviewTransaction `json:"transactions"`
}

// Les empreintes et la portée (owner, hash) sont exactement celles de persist.
// L'aperçu ne modifie pas les transactions et ne renvoie pas leurs empreintes.
func describeImport(res *models.ParseResult, existing map[string]bool) importPreviewResponse {
	out := importPreviewResponse{Report: res.Report, Parser: res.Parser, Validated: res.Validated, PeriodStart: res.PeriodStart, PeriodEnd: res.PeriodEnd, OpeningBalance: res.OpeningBalance, ClosingBalance: res.ClosingBalance, Transactions: []importPreviewTransaction{}}
	seen := map[string]bool{}
	sum := money.Cents(0)
	amountsValid := true
	for _, t := range res.Transactions {
		duplicate := existing[t.Hash] || seen[t.Hash]
		seen[t.Hash] = true
		if duplicate {
			out.Duplicates++
		} else {
			out.NewCount++
		}
		amount, e := money.FromLegacy(t.Amount)
		if e != nil {
			amountsValid = false
		} else {
			sum += amount
		}
		out.Transactions = append(out.Transactions, importPreviewTransaction{t.Date, t.Label, t.Amount, duplicate})
	}
	if res.OpeningBalance != nil && res.ClosingBalance != nil {
		opening, e1 := money.FromLegacy(*res.OpeningBalance)
		closing, e2 := money.FromLegacy(*res.ClosingBalance)
		if amountsValid && e1 == nil && e2 == nil {
			diff := float64(closing-opening-sum) / 100
			out.BalanceDifference = &diff
		}
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
	if res.DocumentHash != "" {
		out = describeImport(res, map[string]bool{})
		// Equal entries within one source are separate occurrences, not duplicates.
		out.NewCount = len(res.Transactions)
		out.Duplicates = 0
		for i := range out.Transactions {
			out.Transactions[i].Duplicate = false
		}
		var fingerprint string
		var count int
		err = s.DB.QueryRowContext(r.Context(), `SELECT st.source_parse_hash,(SELECT count(*) FROM transactions t WHERE t.statement_id=st.id AND t.owner_id=st.owner_id) FROM statements st WHERE st.owner_id=$1 AND st.account_id=$2 AND st.source_sha256=$3`, owner, accountID, res.DocumentHash).Scan(&fingerprint, &count)
		if err == nil {
			if fingerprint == parseFingerprint(res) && count == len(res.Transactions) {
				out.Duplicates = count
				out.NewCount = 0
				for i := range out.Transactions {
					out.Transactions[i].Duplicate = true
				}
			} else {
				out.BlockedReason = "Ce document est déjà enregistré avec un résultat différent. Un réimport contrôlé doit préserver les corrections manuelles."
			}
		} else if err != sql.ErrNoRows {
			writeErr(w, 500, fmt.Errorf("vérification du document impossible"))
			return
		} else {
			hashes = []string{}
			for _, t := range res.Transactions {
				hashes = append(hashes, parser.TxHash(t))
			}
			if err = s.DB.QueryRowContext(r.Context(), `SELECT count(*) FROM transactions t JOIN statements st ON st.id=t.statement_id AND st.owner_id=t.owner_id WHERE t.owner_id=$1 AND st.account_id=$2 AND (t.hash=ANY($3) OR t.source_business_hash=ANY($3))`, owner, accountID, pq.Array(hashes)).Scan(&out.PotentialDuplicates); err != nil {
				writeErr(w, 500, fmt.Errorf("vérification des similitudes impossible"))
				return
			}
			if res.Parser != "csv" && res.PeriodStart != "" && res.PeriodEnd != "" {
				var overlap bool
				if err = s.DB.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM statements WHERE owner_id=$1 AND account_id=$2 AND period_start<$4::date AND period_end>$3::date)`, owner, accountID, res.PeriodStart, res.PeriodEnd).Scan(&overlap); err != nil {
					writeErr(w, 500, fmt.Errorf("vérification des périodes impossible"))
					return
				}
				if overlap {
					out.BlockedReason = "Période recouvrant un relevé existant. Un réimport contrôlé est nécessaire avant tout remplacement."
				}
			}
			if out.PotentialDuplicates > 0 {
				out.BlockedReason = "Des opérations ressemblent à un autre import. Aucun doublon n'est prouvé : comparer les sources avant un réimport contrôlé."
			}
		}
	}
	if res.Report != nil && res.Report.Completeness == "failed" {
		out.BlockedReason = "Les contrôles d'extraction ou de rapprochement ont échoué. Aucune opération ne sera enregistrée."
	}
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
