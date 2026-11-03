package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf16"
)

var discordPath = regexp.MustCompile(`^/api(?:/v10)?/webhooks/[0-9]+/[A-Za-z0-9_-]+$`)

func validateDiscordURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || (u.Host != "discord.com" && u.Host != "discordapp.com") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || !discordPath.MatchString(u.Path) || u.RawPath != "" {
		return errors.New("Gunakan URL HTTPS webhook Discord: https://discord.com/api/webhooks/ID/TOKEN (channel teks biasa).")
	}
	return nil
}
func defaultDiscordSchedule() notifySchedule {
	return notifySchedule{Hour: "08:00", Mode: "template", Format: "text", Subject: "Pengingat tugas", Body: "Halo {{.Name}},\nUrgent: {{.Urgent}} · Sedang: {{.Medium}} · Rendah: {{.Low}}\n{{range .HighTasks}}- {{.Title}}\n{{end}}"}
}

func (a *App) discordSettings(id int64) (string, notifySchedule, sql.NullTime, string, error) {
	s := defaultDiscordSchedule()
	var webhook, status string
	var raw []byte
	var next sql.NullTime
	err := a.db.QueryRow(`SELECT webhook,config,next_run,last_status FROM notify_discord WHERE user_id=$1`, id).Scan(&webhook, &raw, &next, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return "", s, next, "", nil
	}
	if err != nil {
		return "", s, next, "", err
	}
	err = json.Unmarshal(raw, &s)
	return webhook, s, next, status, err
}
func (a *App) handleGetDiscord(w http.ResponseWriter, r *http.Request, u User) {
	hook, s, next, status, err := a.discordSettings(u.ID)
	if err != nil {
		serverError(w, "discord settings", err)
		return
	}
	writeJSON(w, 200, map[string]any{"configured": hook != "", "schedule": s, "nextRun": next, "lastStatus": status})
}
func (a *App) discordContent(id int64, s notifySchedule) (string, error) {
	if s.Format != "" && s.Format != "text" {
		return "", errors.New("Discord mendukung teks/Markdown, gunakan format teks.")
	}
	if err := validateEmailContent(strings.TrimSpace(s.Subject), s.Body); err != nil {
		return "", err
	}
	subject, body := s.Subject, s.Body
	var err error
	if s.Mode == "template" {
		subject, body, err = a.renderNotify(id, subject, body)
	} else if s.Mode != "text" {
		return "", errors.New("Mode pesan tidak valid.")
	}
	if err != nil {
		return "", err
	}
	content := subject + "\n\n" + body
	if len(utf16.Encode([]rune(content))) > 2000 {
		return "", errors.New("Hasil judul dan pesan Discord maksimal 2.000 karakter. Ringkas template atau daftar tugas.")
	}
	return content, nil
}
func (a *App) handleSaveDiscord(w http.ResponseWriter, r *http.Request, u User) {
	var in struct {
		notifySchedule
		Webhook string `json:"webhook"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Format = "text"
	in.Subject = strings.TrimSpace(in.Subject)
	if err := in.notifySchedule.validate(); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	if _, err := a.discordContent(u.ID, in.notifySchedule); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	hook := strings.TrimSpace(in.Webhook)
	if hook != "" {
		if err := validateDiscordURL(hook); err != nil {
			writeError(w, 400, err.Error())
			return
		}
	}
	var next any
	if in.Enabled {
		next = nextNotifyRun(in.Hour, time.Now())
	}
	raw, _ := json.Marshal(in.notifySchedule)
	// Empty URL retains the saved secret without sending it back to the browser.
	res, err := a.db.Exec(`INSERT INTO notify_discord(user_id,webhook,config,next_run) SELECT $1,$2,$3::jsonb,$4 WHERE $2<>'' OR EXISTS(SELECT 1 FROM notify_discord WHERE user_id=$1) ON CONFLICT(user_id) DO UPDATE SET webhook=CASE WHEN EXCLUDED.webhook='' THEN notify_discord.webhook ELSE EXCLUDED.webhook END,config=EXCLUDED.config,next_run=EXCLUDED.next_run`, u.ID, hook, string(raw), next)
	if err != nil {
		serverError(w, "discord save", err)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeError(w, 400, "Isi URL webhook Discord terlebih dahulu.")
		return
	}
	a.handleGetDiscord(w, r, u)
}
func (a *App) handlePreviewDiscord(w http.ResponseWriter, r *http.Request, u User) {
	var s notifySchedule
	if !decode(w, r, &s) {
		return
	}
	content, err := a.discordContent(u.ID, s)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"content": content})
}

var discordClient = &http.Client{Timeout: 12 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}

func sendDiscord(ctx context.Context, hook, content string) error {
	if err := validateDiscordURL(hook); err != nil {
		return err
	}
	raw, _ := json.Marshal(map[string]any{"content": content, "allowed_mentions": map[string]any{"parse": []string{}}, "flags": 4})
	req, err := http.NewRequestWithContext(ctx, "POST", hook+"?wait=true", bytes.NewReader(raw))
	if err != nil {
		return errors.New("Permintaan Discord tidak valid.")
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := discordClient.Do(req)
	if err != nil {
		return errors.New("Status pengiriman belum pasti. Periksa channel Discord sebelum mengirim ulang.")
	}
	defer response.Body.Close()
	io.Copy(io.Discard, io.LimitReader(response.Body, 8192))
	if response.StatusCode == 429 {
		return errors.New("Discord membatasi pengiriman. Tunggu sebelum mencoba kembali.")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return errors.New("Discord menolak pesan. Periksa webhook, akses channel, dan panjang pesan.")
	}
	return nil
}
func (a *App) handleSendDiscord(w http.ResponseWriter, r *http.Request, u User) {
	var in struct {
		notifySchedule
		Test bool `json:"test"`
	}
	if !decode(w, r, &in) {
		return
	}
	hook, _, _, _, err := a.discordSettings(u.ID)
	if err != nil {
		serverError(w, "discord send settings", err)
		return
	}
	if hook == "" {
		writeError(w, 400, "Simpan webhook terlebih dahulu.")
		return
	}
	content := "Tes notifikasi Cells\nKoneksi Discord Webhook berhasil."
	if !in.Test {
		content, err = a.discordContent(u.ID, in.notifySchedule)
		if err != nil {
			writeError(w, 400, err.Error())
			return
		}
	}
	res, err := a.db.Exec(`UPDATE notify_discord SET last_attempt=CURRENT_TIMESTAMP WHERE user_id=$1 AND (last_attempt IS NULL OR last_attempt <= CURRENT_TIMESTAMP - INTERVAL '1 minute')`, u.ID)
	if err != nil {
		serverError(w, "discord cooldown", err)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeError(w, 429, "Tunggu satu menit sebelum mengirim lagi.")
		return
	}
	err = sendDiscord(r.Context(), hook, content)
	status := "Pesan berhasil dikirim ke Discord."
	if err != nil {
		status = err.Error()
	}
	a.db.Exec(`UPDATE notify_discord SET last_status=$2 WHERE user_id=$1`, u.ID, status)
	if err != nil {
		writeError(w, 502, status)
		return
	}
	writeJSON(w, 200, map[string]string{"message": status})
}
func (a *App) processDiscordSchedule() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()
	var id int64
	var hook string
	var raw []byte
	err = tx.QueryRowContext(ctx, `SELECT user_id,webhook,config FROM notify_discord WHERE next_run <= CURRENT_TIMESTAMP AND (last_attempt IS NULL OR last_attempt <= CURRENT_TIMESTAMP - INTERVAL '1 minute') ORDER BY next_run,user_id LIMIT 1 FOR UPDATE SKIP LOCKED`).Scan(&id, &hook, &raw)
	if err != nil {
		return
	}
	var s notifySchedule
	if json.Unmarshal(raw, &s) != nil {
		return
	}
	_, err = tx.ExecContext(ctx, `UPDATE notify_discord SET next_run=$2,last_attempt=CURRENT_TIMESTAMP,last_status='Sedang dikirim; jika server terhenti, periksa channel sebelum mengirim ulang.' WHERE user_id=$1`, id, nextNotifyRun(s.Hour, time.Now()))
	if err != nil {
		return
	}
	if tx.Commit() != nil {
		return
	}
	content, err := a.discordContent(id, s)
	if err == nil {
		err = sendDiscord(ctx, hook, content)
	}
	status := "Pesan terjadwal berhasil dikirim ke Discord."
	if err != nil {
		status = err.Error()
	}
	a.db.Exec(`UPDATE notify_discord SET last_status=$2 WHERE user_id=$1`, id, status)
}
