package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func TestSessionCookieFlags(t *testing.T) {
	for _, secure := range []bool{false, true} {
		r := httptest.NewRequest(http.MethodGet, "/api/me", nil)
		if secure {
			r.Header.Set("X-Forwarded-Proto", "https")
		}
		w := httptest.NewRecorder()
		setSessionCookie(w, r, "test-token", time.Now().Add(6*time.Hour))
		c := w.Result().Cookies()[0]
		if c.MaxAge != 21600 || !c.HttpOnly || c.Secure != secure || c.SameSite != http.SameSiteLaxMode || c.Path != "/" || c.Domain != "" {
			t.Fatalf("incorrect session cookie attributes: %+v", c)
		}
		w = httptest.NewRecorder()
		clearCookie(w, r)
		c = w.Result().Cookies()[0]
		if c.MaxAge != -1 || !c.Expires.Before(time.Now()) || c.Value != "" || c.Secure != secure || !c.HttpOnly {
			t.Fatalf("cookie was not cleared securely: %+v", c)
		}
	}
}

func TestSessionPostgres(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to a disposable PostgreSQL database")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var name string
	if err := db.QueryRow(`SELECT current_database()`).Scan(&name); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(name, "_test") {
		t.Fatal("test database name must end in _test")
	}
	if err := migrate(db); err != nil {
		t.Fatal(err)
	}
	suffix, err := newToken()
	if err != nil {
		t.Fatal(err)
	}
	username := "session-test-" + suffix[:12]
	password := "test-only-password"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	var userID int64
	if err := db.QueryRow(`INSERT INTO users(username, display_name, password_hash) VALUES($1, 'Session Test', $2) RETURNING id`, username, string(hash)).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	defer db.Exec(`DELETE FROM users WHERE id=$1`, userID)
	a := newApp(db)
	handler := a.routes()
	request := func(method, path, body, token string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set(csrfHeader, csrfHeaderVal)
		r.Header.Set("X-Forwarded-Proto", "https")
		if token != "" {
			r.AddCookie(&http.Cookie{Name: cookieName, Value: token})
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	w := request(http.MethodPost, "/api/login", string(body), "")
	if w.Code != http.StatusOK {
		t.Fatalf("login failed: %d %s", w.Code, w.Body.String())
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || cookies[0].MaxAge != 21600 || !cookies[0].HttpOnly || !cookies[0].Secure {
		t.Fatal("login must issue a secure six-hour cookie")
	}
	token := cookies[0].Value
	var loginUser User
	if err := json.Unmarshal(w.Body.Bytes(), &loginUser); err != nil {
		t.Fatal(err)
	}
	var created, expires time.Time
	if err := db.QueryRow(`SELECT created_at, expires_at FROM sessions WHERE token_hash=$1`, hashToken(token)).Scan(&created, &expires); err != nil {
		t.Fatal(err)
	}
	if expires.Sub(created) != 6*time.Hour || loginUser.SessionExpiresAt.Sub(expires).Abs() > time.Millisecond {
		t.Fatal("database and login response must share the six-hour deadline")
	}

	for _, tc := range []struct {
		name       string
		age        time.Duration
		validFor   time.Duration
		authorized bool
	}{
		{"active", time.Hour, 5 * time.Hour, true},
		{"legacy active", 3 * time.Hour, 14 * 24 * time.Hour, true},
		{"six hour boundary", 6 * time.Hour, 14 * 24 * time.Hour, false},
		{"legacy expired", 7 * time.Hour, 14 * 24 * time.Hour, false},
		{"explicit expiry", time.Hour, -time.Minute, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Now().UTC().Truncate(time.Microsecond)
			created, expires := now.Add(-tc.age), now.Add(tc.validFor)
			if _, err := db.Exec(`UPDATE sessions SET created_at=$1, expires_at=$2 WHERE token_hash=$3`, created, expires, hashToken(token)); err != nil {
				t.Fatal(err)
			}
			for range 2 {
				w := request(http.MethodGet, "/api/me", "", token)
				if !tc.authorized {
					cleared := w.Result().Cookies()
					if w.Code != http.StatusUnauthorized || len(cleared) != 1 || cleared[0].MaxAge != -1 {
						t.Fatalf("expired session accepted or cookie not cleared: %d", w.Code)
					}
					continue
				}
				if w.Code != http.StatusOK {
					t.Fatalf("valid session rejected: %d", w.Code)
				}
				var u User
				if err := json.Unmarshal(w.Body.Bytes(), &u); err != nil {
					t.Fatal(err)
				}
				deadline := created.Add(6 * time.Hour)
				if expires.Before(deadline) {
					deadline = expires
				}
				if !u.SessionExpiresAt.Equal(deadline) {
					t.Fatal("activity must not extend the session deadline")
				}
				if tc.name == "legacy active" {
					updated := w.Result().Cookies()
					if len(updated) != 1 || updated[0].MaxAge > 10800 || updated[0].MaxAge < 10790 {
						t.Fatal("legacy cookie must be shortened to the remaining three hours")
					}
				}
			}
		})
	}

	w = request(http.MethodPost, "/api/logout", "", token)
	if w.Code != http.StatusNoContent || w.Result().Cookies()[0].MaxAge != -1 {
		t.Fatal("logout did not clear the cookie")
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE token_hash=$1`, hashToken(token)).Scan(&count); err != nil || count != 0 {
		t.Fatalf("logout did not revoke the server session: %v", err)
	}
	if w := request(http.MethodGet, "/api/me", "", token); w.Code != http.StatusUnauthorized {
		t.Fatal("logged-out token was accepted")
	}
}
