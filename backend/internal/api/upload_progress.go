package api

import (
	"net/http"
	"sync"
	"time"
)

// uploadProgressEntry est l'état publié d'un import en cours. Les étapes
// viennent du vrai pipeline (extraction, conversion, OCR page i/N,
// enregistrement, IA) — jamais d'animation inventée.
type uploadProgressEntry struct {
	Stage  string    `json:"stage"`
	Detail string    `json:"detail,omitempty"`
	Pct    int       `json:"pct"`
	Done   bool      `json:"done"`
	Owner  int64     `json:"-"`
	At     time.Time `json:"-"`
}

var (
	upMu    sync.Mutex
	upStore = map[string]uploadProgressEntry{}
)

// setUploadProgress publie l'étape courante sous le jeton fourni par le client.
func setUploadProgress(token string, owner int64, stage, detail string, pct int, done bool) {
	if token == "" {
		return
	}
	upMu.Lock()
	defer upMu.Unlock()
	upStore[token] = uploadProgressEntry{Stage: stage, Detail: detail, Pct: pct, Done: done, Owner: owner, At: time.Now()}
	// ménage : purge des entrées de plus de 10 minutes (imports abandonnés)
	for k, v := range upStore {
		if time.Since(v.At) > 10*time.Minute {
			delete(upStore, k)
		}
	}
}

// uploadProgress renvoie l'état de l'import identifié par ?token=. Scopé au
// propriétaire : on ne lit pas la progression d'un autre utilisateur.
func (s *Server) uploadProgress(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	owner := s.ownerID(r)
	upMu.Lock()
	e, ok := upStore[token]
	upMu.Unlock()
	if !ok || e.Owner != owner {
		writeJSON(w, http.StatusOK, map[string]any{"stage": "", "pct": 0, "done": false})
		return
	}
	writeJSON(w, http.StatusOK, e)
}
