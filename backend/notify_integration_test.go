package main

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestNotifyPostgres(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to a disposable PostgreSQL database")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var databaseName string
	if err := db.QueryRow(`SELECT current_database()`).Scan(&databaseName); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(databaseName, "_test") {
		t.Fatal("test database name must end in _test")
	}
	if err := migrate(db); err != nil {
		t.Fatal(err)
	}
	if err := migrate(db); err != nil {
		t.Fatalf("migration is not idempotent: %v", err)
	}
	token, err := newToken()
	if err != nil {
		t.Fatal(err)
	}
	u := User{Username: "notify-test-" + token[:12]}
	if err := db.QueryRow(`INSERT INTO users(username, display_name, password_hash) VALUES($1, 'Test', 'unused') RETURNING id`, u.Username).Scan(&u.ID); err != nil {
		t.Fatal(err)
	}
	defer db.Exec(`DELETE FROM users WHERE id=$1`, u.ID)
	if _, err := db.Exec(`INSERT INTO sessions(token_hash,user_id,expires_at) VALUES($1,$2,CURRENT_TIMESTAMP + INTERVAL '1 hour')`, hashToken(token), u.ID); err != nil {
		t.Fatal(err)
	}
	a := &App{db: db, mail: mailConfig{Host: "smtp.hostinger.com", Port: "465", Username: "sender@example.com", FromName: "Cells", DefaultRecipients: []string{"user@example.com"}}}
	request := func(method, path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.AddCookie(&http.Cookie{Name: cookieName, Value: token})
		r.Header.Set(csrfHeader, csrfHeaderVal)
		w := httptest.NewRecorder()
		a.routes().ServeHTTP(w, r)
		return w
	}
	w := request("GET", "/api/notify", "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), "user@example.com") || !strings.Contains(w.Body.String(), `"configured":false`) {
		t.Fatalf("defaults: %d %s", w.Code, w.Body.String())
	}
	w = request("PUT", "/api/notify", `{"recipients":["first@example.com","first@example.com","second@example.com"]}`)
	if w.Code != 200 {
		t.Fatalf("save: %d %s", w.Code, w.Body.String())
	}
	// Read through a fresh App to prove settings survive process state changes.
	fresh := &App{db: db, mail: a.mail}
	saved, err := fresh.notifyRecipients(u)
	if err != nil || len(saved) != 2 || saved[0] != "first@example.com" {
		t.Fatalf("persistence: %v, %v", saved, err)
	}
	other, err := fresh.notifyRecipients(User{ID: -1})
	if err != nil || len(other) != 1 || other[0] != "user@example.com" {
		t.Fatalf("user isolation: %v, %v", other, err)
	}
	w = request("PUT", "/api/notify", `{"recipients":["invalid"]}`)
	if w.Code != 400 {
		t.Fatalf("invalid recipient: %d", w.Code)
	}
	w = request("POST", "/api/notify/test", "")
	if w.Code != 503 {
		t.Fatalf("missing credentials: %d", w.Code)
	}
	w = request("POST", "/api/notify/broadcast", `{"subject":"bad\r\nBcc: bad@example.com","body":"test"}`)
	if w.Code != 400 {
		t.Fatalf("header injection: %d", w.Code)
	}
	a.mail.Password = "must-not-be-exposed"
	w = request("GET", "/api/notify", "")
	if strings.Contains(w.Body.String(), a.mail.Password) {
		t.Fatal("password leaked")
	}
	var status struct {
		Configured bool `json:"configured"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil || !status.Configured {
		t.Fatalf("configuration status: %s", w.Body.String())
	}
	if _, err := db.Exec(`UPDATE notify_mail_state SET last_attempt_at=NULL WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	// Concurrent requests across processes must only reserve one send slot.
	var claims atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := db.Exec(`UPDATE notify_mail_state SET last_attempt_at=CURRENT_TIMESTAMP WHERE id=1 AND (last_attempt_at IS NULL OR last_attempt_at <= CURRENT_TIMESTAMP - INTERVAL '1 minute')`)
			if err != nil {
				t.Error(err)
				return
			}
			n, err := result.RowsAffected()
			if err != nil {
				t.Error(err)
				return
			}
			claims.Add(int32(n))
		}()
	}
	wg.Wait()
	if claims.Load() != 1 {
		t.Fatalf("expected one send slot, got %d", claims.Load())
	}
	w = request("POST", "/api/notify/test", "")
	if w.Code != 429 || w.Header().Get("Retry-After") == "" {
		t.Fatalf("cooldown: %d %s", w.Code, w.Body.String())
	}
	w = request("PUT", "/api/notify", `{"recipients":[]}`)
	if w.Code != 200 {
		t.Fatal("cannot clear recipients")
	}
	w = request("POST", "/api/notify/test", "")
	if w.Code != 400 {
		t.Fatalf("empty recipient send: %d", w.Code)
	}

	w = request("PUT", "/api/notify", `{"recipients":["user@example.com"]}`)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	_, err = db.Exec(`INSERT INTO todos(user_id,title,notes,priority) VALUES($1,'Urgent task','','high')`, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	w = request("POST", "/api/notify/preview", `{"mode":"template","subject":"Reminder {{.Urgent}}","body":"{{range .HighTasks}}{{.Title}}{{end}}"}`)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Reminder 1") || !strings.Contains(w.Body.String(), "Urgent task") {
		t.Fatalf("preview: %d %s", w.Code, w.Body.String())
	}
	w = request("PUT", "/api/notify/schedule", `{"enabled":true,"hour":"08:00","mode":"template","subject":"Reminder {{.Urgent}}","body":"{{range .HighTasks}}{{.Title}}{{end}}"}`)
	if w.Code != 200 {
		t.Fatalf("schedule save: %d %s", w.Code, w.Body.String())
	}
	w = request("GET", "/api/notify/schedule", "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"enabled":true`) {
		t.Fatal("schedule did not persist")
	}
	// Force a due job but disable credentials to ensure no real email can be sent.
	a.mail.Password = ""
	db.Exec(`UPDATE notify_mail_state SET last_attempt_at=NULL WHERE id=1`)
	db.Exec(`UPDATE notify_schedule SET next_run=CURRENT_TIMESTAMP - INTERVAL '1 minute' WHERE user_id=$1`, u.ID)
	a.processNotifySchedule()
	var run, next sql.NullTime
	var lastStatus string
	if err := db.QueryRow(`SELECT last_run,next_run,last_status FROM notify_schedule WHERE user_id=$1`, u.ID).Scan(&run, &next, &lastStatus); err != nil {
		t.Fatal(err)
	}
	if !run.Valid || !next.Valid || !strings.Contains(lastStatus, "Gagal") {
		t.Fatalf("job was not processed: %v %v %s", run, next, lastStatus)
	}
	db.Exec(`UPDATE notify_mail_state SET last_attempt_at=NULL WHERE id=1`)
	a.processNotifySchedule()
	var secondRun sql.NullTime
	db.QueryRow(`SELECT last_run FROM notify_schedule WHERE user_id=$1`, u.ID).Scan(&secondRun)
	if !run.Time.Equal(secondRun.Time) {
		t.Fatal("duplicate job processed")
	}

	w = request("GET", "/api/notify/discord", "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"configured":false`) {
		t.Fatal("Discord defaults", w.Body.String())
	}
	w = request("PUT", "/api/notify/discord", `{"webhook":"https://discord.com/api/webhooks/123/secret-test-token","enabled":false,"hour":"08:00","mode":"template","subject":"Reminder {{.Urgent}}","body":"{{range .HighTasks}}{{.Title}}{{end}}"}`)
	if w.Code != 200 || strings.Contains(w.Body.String(), "secret-test-token") {
		t.Fatalf("Discord save/leak: %d %s", w.Code, w.Body.String())
	}
	w = request("PUT", "/api/notify/discord", `{"webhook":"","enabled":true,"hour":"09:00","mode":"text","subject":"Reminder","body":"Check tasks"}`)
	if w.Code != 200 {
		t.Fatalf("Discord preserve webhook: %d %s", w.Code, w.Body.String())
	}
	hook, ds, _, _, err := a.discordSettings(u.ID)
	if err != nil || !strings.Contains(hook, "secret-test-token") || ds.Hour != "09:00" {
		t.Fatal("Discord persistence failed")
	}
	w = request("POST", "/api/notify/discord/preview", `{"mode":"template","subject":"Reminder {{.Urgent}}","body":"{{range .HighTasks}}{{.Title}}{{end}}"}`)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Urgent task") {
		t.Fatal("Discord preview failed", w.Body.String())
	}
	originalClient := discordClient
	discordClient = &http.Client{Transport: discordTestTransport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`)), Header: http.Header{}}, nil
	})}
	defer func() { discordClient = originalClient }()
	w = request("POST", "/api/notify/discord/send", `{"test":true}`)
	if w.Code != 200 {
		t.Fatal("Discord send failed", w.Body.String())
	}
	w = request("POST", "/api/notify/discord/send", `{"test":true}`)
	if w.Code != 429 {
		t.Fatal("Discord cooldown failed")
	}
	db.Exec(`UPDATE notify_discord SET next_run=CURRENT_TIMESTAMP-INTERVAL '1 minute',last_attempt=NULL WHERE user_id=$1`, u.ID)
	a.processDiscordSchedule()
	_, _, dn, discordStatus, err := a.discordSettings(u.ID)
	if err != nil || !dn.Valid || !strings.Contains(discordStatus, "berhasil") {
		t.Fatal("Discord scheduled send failed", discordStatus, err)
	}
}
