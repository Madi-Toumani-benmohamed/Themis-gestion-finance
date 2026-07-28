package api

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// securityHeaders ajoute des en-têtes de sécurité à toutes les réponses.
// Défensif par défaut : empêche le sniffing de type MIME, l'affichage en iframe
// (clickjacking) et pose une CSP stricte adaptée à l'app (tout est servi depuis
// la même origine ; pas de CDN externe requis au runtime).
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "same-origin")
		// L'app est une SPA autonome : scripts/styles inline autorisés (Vite
		// injecte quelques styles), mais aucune source externe. 'self' couvre
		// l'API et les assets. connect-src 'self' bloque toute exfiltration
		// vers un tiers depuis le navigateur.
		h.Set("Content-Security-Policy",
			"default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; "+
				"script-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		next.ServeHTTP(w, r)
	})
}

// ── Rate limiting du login (anti brute-force) ──
// Limiteur en mémoire, sans dépendance : compte les échecs récents par IP et
// bloque temporairement au-delà d'un seuil. Volontairement simple — pour une
// app auto-hébergée, ça suffit à décourager un brute-force, sans Redis ni
// service externe. Les tentatives réussies ne comptent pas.

const (
	loginMaxFails  = 8               // échecs tolérés dans la fenêtre
	loginWindow    = 15 * time.Minute // fenêtre glissante
	loginBlockFor  = 15 * time.Minute // durée de blocage une fois le seuil atteint
)

type loginAttempts struct {
	fails      []time.Time
	blockUntil time.Time
}

var (
	loginMu    sync.Mutex
	loginByIP  = map[string]*loginAttempts{}
)

// clientIP extrait l'IP de la requête (en tenant compte d'un éventuel proxy de
// confiance via X-Forwarded-For — première valeur).
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := indexByte(xff, ','); i >= 0 {
			return trimSpace(xff[:i])
		}
		return trimSpace(xff)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// loginBlocked indique si l'IP est actuellement bloquée pour cause de trop
// d'échecs. Appelé avant de vérifier le mot de passe.
func loginBlocked(ip string) bool {
	loginMu.Lock()
	defer loginMu.Unlock()
	a := loginByIP[ip]
	if a == nil {
		return false
	}
	return time.Now().Before(a.blockUntil)
}

// recordLoginFail enregistre un échec et arme le blocage si le seuil est atteint.
func recordLoginFail(ip string) {
	loginMu.Lock()
	defer loginMu.Unlock()
	now := time.Now()
	a := loginByIP[ip]
	if a == nil {
		a = &loginAttempts{}
		loginByIP[ip] = a
	}
	// ne garder que les échecs dans la fenêtre
	cutoff := now.Add(-loginWindow)
	kept := a.fails[:0]
	for _, t := range a.fails {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	a.fails = append(kept, now)
	if len(a.fails) >= loginMaxFails {
		a.blockUntil = now.Add(loginBlockFor)
		a.fails = nil // on repart à zéro après blocage
	}
	// ménage grossier : borne la carte pour éviter une croissance illimitée
	if len(loginByIP) > 10000 {
		for k, v := range loginByIP {
			if now.After(v.blockUntil) && len(v.fails) == 0 {
				delete(loginByIP, k)
			}
		}
	}
}

// recordLoginSuccess efface l'historique d'échecs d'une IP après une connexion
// réussie.
func recordLoginSuccess(ip string) {
	loginMu.Lock()
	defer loginMu.Unlock()
	delete(loginByIP, ip)
}

// petits helpers sans importer strings (garder le fichier autonome)
func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
