package api

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

/* --------------------------------------------------------------------------
   PBKDF2-HMAC-SHA256 en bibliothèque standard uniquement (RFC 2898).
   Choix délibéré : pas de dépendance externe pour la crypto de mots de
   passe ; l'implémentation tient en 20 lignes auditables.
   -------------------------------------------------------------------------- */

const pbkdf2Iters = 210_000

func pbkdf2SHA256(password, salt []byte, iters, keyLen int) []byte {
	prf := hmac.New(sha256.New, password)
	hashLen := prf.Size()
	numBlocks := (keyLen + hashLen - 1) / hashLen
	out := make([]byte, 0, numBlocks*hashLen)
	var block [4]byte
	for i := 1; i <= numBlocks; i++ {
		prf.Reset()
		prf.Write(salt)
		binary.BigEndian.PutUint32(block[:], uint32(i))
		prf.Write(block[:])
		u := prf.Sum(nil)
		t := make([]byte, len(u))
		copy(t, u)
		for j := 1; j < iters; j++ {
			prf.Reset()
			prf.Write(u)
			u = prf.Sum(nil)
			for k := range t {
				t[k] ^= u[k]
			}
		}
		out = append(out, t...)
	}
	return out[:keyLen]
}

func hashPassword(pw string) string {
	salt := make([]byte, 16)
	_, _ = rand.Read(salt)
	dk := pbkdf2SHA256([]byte(pw), salt, pbkdf2Iters, 32)
	return fmt.Sprintf("pbkdf2$sha256$%d$%s$%s", pbkdf2Iters,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(dk))
}

func verifyPassword(pw, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 5 || parts[0] != "pbkdf2" || parts[1] != "sha256" {
		return false
	}
	iters, err := strconv.Atoi(parts[2])
	if err != nil || iters < 1000 {
		return false
	}
	salt, err1 := base64.RawStdEncoding.DecodeString(parts[3])
	want, err2 := base64.RawStdEncoding.DecodeString(parts[4])
	if err1 != nil || err2 != nil {
		return false
	}
	got := pbkdf2SHA256([]byte(pw), salt, iters, len(want))
	return subtle.ConstantTimeCompare(got, want) == 1
}

func validPassword(pw string) error {
	if len(pw) < 10 {
		return fmt.Errorf("mot de passe trop court : 10 caractères minimum")
	}
	var hasLetter, hasDigit bool
	for _, c := range pw {
		switch {
		case c >= '0' && c <= '9':
			hasDigit = true
		case (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z'):
			hasLetter = true
		}
	}
	if !hasLetter || !hasDigit {
		return fmt.Errorf("le mot de passe doit mêler lettres et chiffres")
	}
	return nil
}

/* --------------------------------------------------------------------------
   Limitation des tentatives (anti force-brute, en mémoire)
   -------------------------------------------------------------------------- */

var loginGuard = struct {
	sync.Mutex
	fails map[string]struct {
		n     int
		until time.Time
	}
}{fails: map[string]struct {
	n     int
	until time.Time
}{}}

func loginLocked(user string) bool {
	loginGuard.Lock()
	defer loginGuard.Unlock()
	f := loginGuard.fails[user]
	return time.Now().Before(f.until)
}

func loginFailed(user string) {
	loginGuard.Lock()
	defer loginGuard.Unlock()
	f := loginGuard.fails[user]
	f.n++
	if f.n >= 5 {
		f.until = time.Now().Add(30 * time.Second)
		f.n = 0
	}
	loginGuard.fails[user] = f
}

func loginSucceeded(user string) {
	loginGuard.Lock()
	defer loginGuard.Unlock()
	delete(loginGuard.fails, user)
}

/* --------------------------------------------------------------------------
   Sessions
   -------------------------------------------------------------------------- */

const sessionCookie = "themis_session"
const sessionTTL = 30 * 24 * time.Hour

type authUser struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

func newToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func (s *Server) userFromRequest(r *http.Request) *authUser {
	c, err := r.Cookie(sessionCookie)
	if err != nil || c.Value == "" {
		return nil
	}
	var u authUser
	err = s.DB.QueryRow(`
		SELECT u.id, u.username, u.role FROM sessions se
		JOIN users u ON u.id = se.user_id
		WHERE se.token = $1 AND se.expires_at > now()`, c.Value).
		Scan(&u.ID, &u.Username, &u.Role)
	if err != nil {
		return nil
	}
	return &u
}

func (s *Server) setSessionCookie(w http.ResponseWriter, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.Cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	})
}

// StartSessionCleanup purge les sessions expirées toutes les heures.
func (s *Server) StartSessionCleanup() {
	go func() {
		t := time.NewTicker(time.Hour)
		defer t.Stop()
		for range t.C {
			if _, err := s.DB.Exec(`DELETE FROM sessions WHERE expires_at < now()`); err != nil {
				log.Printf("nettoyage sessions: %v", err)
			}
		}
	}()
}

// hasUsers indique si au moins un compte existe (sinon : premier lancement).
func (s *Server) hasUsers() bool {
	var n int
	_ = s.DB.QueryRow(`SELECT count(*) FROM users`).Scan(&n)
	return n > 0
}

/* --------------------------------------------------------------------------
   Middleware : toute l'API exige une session, hormis la santé (probes k8s)
   et les routes d'authentification. /api/admin/* exige le rôle admin.
   Tant qu'aucun utilisateur n'existe, l'API reste ouverte pour permettre la
   création du premier compte (l'UI l'exige immédiatement).
   -------------------------------------------------------------------------- */

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if !strings.HasPrefix(p, "/api/") ||
			p == "/api/health" || strings.HasPrefix(p, "/api/auth/") {
			next.ServeHTTP(w, r)
			return
		}
		if !s.hasUsers() { // premier lancement : configuration initiale
			next.ServeHTTP(w, r)
			return
		}
		u := s.userFromRequest(r)
		if u == nil {
			writeErr(w, http.StatusUnauthorized, fmt.Errorf("authentification requise"))
			return
		}
		if strings.HasPrefix(p, "/api/admin/") && u.Role != "admin" {
			writeErr(w, http.StatusForbidden, fmt.Errorf("réservé aux administrateurs"))
			return
		}
		// Rendre l'utilisateur courant disponible aux handlers pour filtrer
		// les données par propriétaire (isolation).
		ctx := context.WithValue(r.Context(), ctxUserKey, u)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// clé de contexte pour l'utilisateur courant (type privé = pas de collision).
type ctxKey string

const ctxUserKey ctxKey = "themis.user"

// ownerID renvoie l'id de l'utilisateur courant, ou 0 si absent (ne devrait pas
// arriver derrière authMiddleware). Les handlers filtrent leurs requêtes dessus.
func (s *Server) ownerID(r *http.Request) int64 {
	if u, ok := r.Context().Value(ctxUserKey).(*authUser); ok && u != nil {
		return u.ID
	}
	return 0
}

/* --------------------------------------------------------------------------
   Handlers /api/auth
   -------------------------------------------------------------------------- */

// authStatus : l'UI sait s'il faut afficher la création du premier compte.
func (s *Server) authStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"setup_required": !s.hasUsers()})
}

// authSetup crée le PREMIER compte (admin). Refusé dès qu'un compte existe.
func (s *Server) authSetup(w http.ResponseWriter, r *http.Request) {
	if s.hasUsers() {
		writeErr(w, http.StatusForbidden, fmt.Errorf("l'installation est déjà configurée"))
		return
	}
	var in struct{ Username, Password string }
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("corps JSON invalide"))
		return
	}
	in.Username = strings.TrimSpace(in.Username)
	if in.Username == "" {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("nom d'utilisateur requis"))
		return
	}
	if err := validPassword(in.Password); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	var uid int64
	err := s.DB.QueryRow(`INSERT INTO users (username, password_hash, role)
	                      VALUES ($1, $2, 'admin') RETURNING id`,
		in.Username, hashPassword(in.Password)).Scan(&uid)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	s.seedUserData(uid)
	s.openSession(w, uid)
	log.Printf("AUTH — compte administrateur initial créé : %s", in.Username)
	writeJSON(w, http.StatusCreated, authUser{ID: uid, Username: in.Username, Role: "admin"})
}

func (s *Server) openSession(w http.ResponseWriter, uid int64) {
	token := newToken()
	_, _ = s.DB.Exec(`INSERT INTO sessions (token, user_id, expires_at) VALUES ($1, $2, $3)`,
		token, uid, time.Now().Add(sessionTTL))
	_, _ = s.DB.Exec(`DELETE FROM sessions WHERE expires_at < now()`)
	s.setSessionCookie(w, token, int(sessionTTL.Seconds()))
}

func (s *Server) authLogin(w http.ResponseWriter, r *http.Request) {
	var in struct{ Username, Password string }
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("corps JSON invalide"))
		return
	}
	in.Username = strings.TrimSpace(in.Username)
	ip := clientIP(r)
	// Double protection anti brute-force : par IP (balayage massif) et par
	// utilisateur (compte ciblé). Les deux sont complémentaires.
	if loginBlocked(ip) {
		writeErr(w, http.StatusTooManyRequests, fmt.Errorf("trop de tentatives depuis cette adresse : réessaie plus tard"))
		return
	}
	if loginLocked(in.Username) {
		writeErr(w, http.StatusTooManyRequests, fmt.Errorf("trop de tentatives : réessaie dans 30 secondes"))
		return
	}
	var uid int64
	var hash, role string
	err := s.DB.QueryRow(`SELECT id, password_hash, role FROM users WHERE username = $1`, in.Username).
		Scan(&uid, &hash, &role)
	if err != nil || !verifyPassword(in.Password, hash) {
		loginFailed(in.Username)
		recordLoginFail(ip)
		writeErr(w, http.StatusUnauthorized, fmt.Errorf("identifiants incorrects"))
		return
	}
	loginSucceeded(in.Username)
	recordLoginSuccess(ip)
	s.openSession(w, uid)
	writeJSON(w, http.StatusOK, authUser{ID: uid, Username: in.Username, Role: role})
}

func (s *Server) authLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		_, _ = s.DB.Exec(`DELETE FROM sessions WHERE token = $1`, c.Value)
	}
	s.setSessionCookie(w, "", -1)
	writeJSON(w, http.StatusOK, map[string]string{"status": "déconnecté"})
}

func (s *Server) authMe(w http.ResponseWriter, r *http.Request) {
	u := s.userFromRequest(r)
	if u == nil {
		writeErr(w, http.StatusUnauthorized, fmt.Errorf("non connecté"))
		return
	}
	writeJSON(w, http.StatusOK, u)
}

/* --------------------------------------------------------------------------
   Gestion des utilisateurs (admin)
   -------------------------------------------------------------------------- */

func (s *Server) adminListUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.Query(`SELECT id, username, role, created_at::text FROM users ORDER BY id`)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	type u struct {
		ID        int64  `json:"id"`
		Username  string `json:"username"`
		Role      string `json:"role"`
		CreatedAt string `json:"created_at"`
	}
	out := []u{}
	for rows.Next() {
		var x u
		if err := rows.Scan(&x.ID, &x.Username, &x.Role, &x.CreatedAt); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		out = append(out, x)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) adminCreateUser(w http.ResponseWriter, r *http.Request) {
	var in struct{ Username, Password, Role string }
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("corps JSON invalide"))
		return
	}
	in.Username = strings.TrimSpace(in.Username)
	if in.Role != "admin" {
		in.Role = "user"
	}
	if in.Username == "" {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("nom d'utilisateur requis"))
		return
	}
	if err := validPassword(in.Password); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	var id int64
	err := s.DB.QueryRow(`INSERT INTO users (username, password_hash, role) VALUES ($1, $2, $3)
	                      ON CONFLICT (username) DO NOTHING RETURNING id`,
		in.Username, hashPassword(in.Password), in.Role).Scan(&id)
	if err != nil {
		writeErr(w, http.StatusConflict, fmt.Errorf("ce nom d'utilisateur existe déjà"))
		return
	}
	s.seedUserData(id)
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "username": in.Username, "role": in.Role})
}

func (s *Server) adminDeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	me := s.userFromRequest(r)
	if me != nil && me.ID == id {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("impossible de supprimer son propre compte"))
		return
	}
	var admins int
	_ = s.DB.QueryRow(`SELECT count(*) FROM users WHERE role = 'admin' AND id <> $1`, id).Scan(&admins)
	var role string
	_ = s.DB.QueryRow(`SELECT role FROM users WHERE id = $1`, id).Scan(&role)
	if role == "admin" && admins == 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("impossible de supprimer le dernier administrateur"))
		return
	}
	if _, err := s.DB.Exec(`DELETE FROM users WHERE id = $1`, id); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "supprimé"})
}

// authChangePassword : l'utilisateur connecté change son propre mot de passe.
func (s *Server) authChangePassword(w http.ResponseWriter, r *http.Request) {
	me := s.userFromRequest(r)
	if me == nil {
		writeErr(w, http.StatusUnauthorized, fmt.Errorf("non connecté"))
		return
	}
	var in struct{ Current, New string }
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("corps JSON invalide"))
		return
	}
	var hash string
	if err := s.DB.QueryRow(`SELECT password_hash FROM users WHERE id = $1`, me.ID).Scan(&hash); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if !verifyPassword(in.Current, hash) {
		writeErr(w, http.StatusForbidden, fmt.Errorf("mot de passe actuel incorrect"))
		return
	}
	if err := validPassword(in.New); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if _, err := s.DB.Exec(`UPDATE users SET password_hash = $1 WHERE id = $2`, hashPassword(in.New), me.ID); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	// invalide les autres sessions de ce compte (sauf la courante)
	if c, err := r.Cookie(sessionCookie); err == nil {
		_, _ = s.DB.Exec(`DELETE FROM sessions WHERE user_id = $1 AND token <> $2`, me.ID, c.Value)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "mot de passe modifié"})
}
