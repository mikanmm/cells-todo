package main

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	cookieName    = "cells_session"
	sessionTTL    = 14 * 24 * time.Hour
	bcryptCost    = 12
	csrfHeader    = "X-Requested-With"
	csrfHeaderVal = "cells"
)

type User struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
}

type App struct {
	db        *sql.DB
	limiter   *loginLimiter
	dummyHash []byte
	mail      mailConfig
}

func newApp(db *sql.DB) *App {
	// Hash palsu dipakai agar waktu respons login sama, baik username ada maupun tidak.
	dummy, _ := bcrypt.GenerateFromPassword([]byte("tidak-dipakai"), bcryptCost)
	return &App{db: db, limiter: newLoginLimiter(8, 10*time.Minute), dummyHash: dummy, mail: loadMailConfig()}
}

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /api/login", a.handleLogin)
	mux.HandleFunc("POST /api/logout", a.handleLogout)
	mux.HandleFunc("GET /api/me", a.auth(a.handleMe))
	mux.HandleFunc("GET /api/notify", a.auth(a.handleNotifySettings))
	mux.HandleFunc("PUT /api/notify", a.auth(a.handleSaveNotifySettings))
	mux.HandleFunc("POST /api/notify/test", a.auth(a.handleTestEmail))
	mux.HandleFunc("POST /api/notify/broadcast", a.auth(a.handleBroadcastEmail))
	mux.HandleFunc("GET /api/notify/schedule", a.auth(a.handleGetSchedule))
	mux.HandleFunc("PUT /api/notify/schedule", a.auth(a.handleSaveSchedule))
	mux.HandleFunc("POST /api/notify/preview", a.auth(a.handlePreviewNotify))
	mux.HandleFunc("GET /api/notify/discord", a.auth(a.handleGetDiscord))
	mux.HandleFunc("PUT /api/notify/discord", a.auth(a.handleSaveDiscord))
	mux.HandleFunc("POST /api/notify/discord/send", a.auth(a.handleSendDiscord))
	mux.HandleFunc("POST /api/notify/discord/preview", a.auth(a.handlePreviewDiscord))

	mux.HandleFunc("DELETE /api/categories/{name}", a.auth(a.handleDeleteCategory))
	mux.HandleFunc("GET /api/categories", a.auth(a.handleCategories))
	mux.HandleFunc("POST /api/categories", a.auth(a.handleAddCategory))
	mux.HandleFunc("GET /api/todos", a.auth(a.handleListTodos))
	mux.HandleFunc("POST /api/todos", a.auth(a.handleCreateTodo))
	mux.HandleFunc("PUT /api/todos/{id}", a.auth(a.handleUpdateTodo))
	mux.HandleFunc("PATCH /api/todos/{id}", a.auth(a.handleSetDone))
	mux.HandleFunc("DELETE /api/todos/{id}", a.auth(a.handleDeleteTodo))

	return a.csrf(mux)
}

// csrf menolak request yang mengubah data jika tidak membawa header khusus.
// Browser tidak bisa mengirim header ini dari situs lain tanpa izin CORS.
func (a *App) csrf(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
		default:
			if r.Header.Get(csrfHeader) != csrfHeaderVal {
				writeError(w, http.StatusForbidden, "Permintaan ditolak.")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) auth(next func(http.ResponseWriter, *http.Request, User)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(cookieName)
		if err != nil || c.Value == "" {
			writeError(w, http.StatusUnauthorized, "Silakan masuk terlebih dahulu.")
			return
		}
		var u User
		err = a.db.QueryRow(
			`SELECT u.id, u.username, u.display_name
			   FROM sessions s JOIN users u ON u.id = s.user_id
			  WHERE s.token_hash = $1 AND s.expires_at > $2`,
			hashToken(c.Value), time.Now().UTC(),
		).Scan(&u.ID, &u.Username, &u.DisplayName)
		if errors.Is(err, sql.ErrNoRows) {
			clearCookie(w, r)
			writeError(w, http.StatusUnauthorized, "Sesi berakhir. Silakan masuk lagi.")
			return
		}
		if err != nil {
			log.Printf("auth: %v", err)
			writeError(w, http.StatusInternalServerError, "Terjadi kesalahan pada server.")
			return
		}
		next(w, r, u)
	}
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Username = strings.TrimSpace(in.Username)
	if in.Username == "" || in.Password == "" {
		writeError(w, http.StatusBadRequest, "Isi username dan password.")
		return
	}

	ip := clientIP(r)
	if !a.limiter.allow(ip) {
		writeError(w, http.StatusTooManyRequests, "Terlalu banyak percobaan masuk. Coba lagi dalam 10 menit.")
		return
	}

	var u User
	var hash string
	err := a.db.QueryRow(
		`SELECT id, username, display_name, password_hash FROM users WHERE username = $1`,
		in.Username,
	).Scan(&u.ID, &u.Username, &u.DisplayName, &hash)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		_ = bcrypt.CompareHashAndPassword(a.dummyHash, []byte(in.Password))
		a.limiter.fail(ip)
		writeError(w, http.StatusUnauthorized, "Username atau password salah.")
		return
	case err != nil:
		log.Printf("login: %v", err)
		writeError(w, http.StatusInternalServerError, "Terjadi kesalahan pada server.")
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		a.limiter.fail(ip)
		writeError(w, http.StatusUnauthorized, "Username atau password salah.")
		return
	}

	token, err := newToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Terjadi kesalahan pada server.")
		return
	}
	expires := time.Now().UTC().Add(sessionTTL)
	if _, err := a.db.Exec(
		`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		hashToken(token), u.ID, expires,
	); err != nil {
		log.Printf("login session: %v", err)
		writeError(w, http.StatusInternalServerError, "Terjadi kesalahan pada server.")
		return
	}

	a.limiter.reset(ip)
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		MaxAge:   int(sessionTTL.Seconds()),
		HttpOnly: true,
		Secure:   isHTTPS(r),
		SameSite: http.SameSiteLaxMode,
	})
	writeJSON(w, http.StatusOK, u)
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cookieName); err == nil {
		_, _ = a.db.Exec(`DELETE FROM sessions WHERE token_hash = $1`, hashToken(c.Value))
	}
	clearCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) handleMe(w http.ResponseWriter, r *http.Request, u User) {
	writeJSON(w, http.StatusOK, u)
}

func (a *App) cleanupSessions() {
	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for range t.C {
		if _, err := a.db.Exec(`DELETE FROM sessions WHERE expires_at < $1`, time.Now().UTC()); err != nil {
			log.Printf("cleanup: %v", err)
		}
	}
}

func clearCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isHTTPS(r),
		SameSite: http.SameSiteLaxMode,
	})
}

func isHTTPS(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func clientIP(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// loginLimiter membatasi percobaan login gagal per alamat IP.
type loginLimiter struct {
	mu     sync.Mutex
	max    int
	window time.Duration
	fails  map[string][]time.Time
}

func newLoginLimiter(max int, window time.Duration) *loginLimiter {
	return &loginLimiter{max: max, window: window, fails: map[string][]time.Time{}}
}

func (l *loginLimiter) prune(ip string, now time.Time) []time.Time {
	kept := l.fails[ip][:0]
	for _, t := range l.fails[ip] {
		if now.Sub(t) < l.window {
			kept = append(kept, t)
		}
	}
	if len(kept) == 0 {
		delete(l.fails, ip)
		return nil
	}
	l.fails[ip] = kept
	return kept
}

func (l *loginLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.prune(ip, time.Now())) < l.max
}

func (l *loginLimiter) fail(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	l.prune(ip, now)
	l.fails[ip] = append(l.fails[ip], now)
}

func (l *loginLimiter) reset(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.fails, ip)
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<16)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "Data yang dikirim tidak valid.")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
