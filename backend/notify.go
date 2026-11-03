package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

func (a *App) notifyRecipients(u User) ([]string, error) {
	var data []byte
	err := a.db.QueryRow(`SELECT recipients FROM notify_settings WHERE user_id=$1`, u.ID).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return normalizeRecipients(a.mail.DefaultRecipients)
	}
	if err != nil {
		return nil, err
	}
	var recipients []string
	if err := json.Unmarshal(data, &recipients); err != nil {
		return nil, err
	}
	return normalizeRecipients(recipients)
}

func (a *App) handleNotifySettings(w http.ResponseWriter, r *http.Request, u User) {
	recipients, err := a.notifyRecipients(u)
	if err != nil {
		serverError(w, "notify settings", err)
		return
	}
	problem := ""
	if err := a.mail.validate(); err != nil {
		problem = err.Error()
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"recipients": recipients, "sender": a.mail.Username,
		"configured": problem == "", "configurationMessage": problem,
		"maxRecipients": maxRecipients,
	})
}

func (a *App) handleSaveNotifySettings(w http.ResponseWriter, r *http.Request, u User) {
	var in struct {
		Recipients []string `json:"recipients"`
	}
	if !decode(w, r, &in) {
		return
	}
	recipients, err := normalizeRecipients(in.Recipients)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	data, err := json.Marshal(recipients)
	if err != nil {
		serverError(w, "notify encode", err)
		return
	}
	_, err = a.db.Exec(`INSERT INTO notify_settings(user_id, recipients) VALUES($1, $2::jsonb)
		ON CONFLICT(user_id) DO UPDATE SET recipients=EXCLUDED.recipients`, u.ID, string(data))
	if err != nil {
		serverError(w, "notify save", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"recipients": recipients})
}

func (a *App) handleTestEmail(w http.ResponseWriter, r *http.Request, u User) {
	a.sendNotification(w, r, u, "Cells notification test", "Hello!\n\nThis is a test email from Cells sent through your configured SMTP server.\nReceiving this email confirms that notification delivery is working.")
}

func (a *App) handleBroadcastEmail(w http.ResponseWriter, r *http.Request, u User) {
	var in struct {
		Subject string `json:"subject"`
		Body    string `json:"body"`
		Mode    string `json:"mode"`
		Format  string `json:"format"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Subject = strings.TrimSpace(in.Subject)
	if err := validateNotifyFormat(in.Format); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	if in.Mode == "template" {
		var err error
		in.Subject, in.Body, err = a.renderNotify(u.ID, in.Subject, in.Body, in.Format)
		if err != nil {
			writeError(w, 400, err.Error())
			return
		}
	} else if in.Mode != "" && in.Mode != "text" {
		writeError(w, 400, "Invalid message mode.")
		return
	}
	if err := validateEmailContent(in.Subject, in.Body, in.Format); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	a.sendNotification(w, r, u, in.Subject, in.Body, in.Format)
}

func (a *App) sendNotification(w http.ResponseWriter, r *http.Request, u User, subject, body string, formats ...string) {
	if err := a.mail.validate(); err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	recipients, err := a.notifyRecipients(u)
	if err != nil {
		serverError(w, "notify recipients", err)
		return
	}
	if len(recipients) == 0 {
		writeError(w, http.StatusBadRequest, "Save at least one recipient email address first.")
		return
	}
	// Atomic, shared across users/replicas and retained across server restarts.
	result, err := a.db.ExecContext(r.Context(), `UPDATE notify_mail_state SET last_attempt_at=CURRENT_TIMESTAMP
		WHERE id=1 AND (last_attempt_at IS NULL OR last_attempt_at <= CURRENT_TIMESTAMP - INTERVAL '1 minute')`)
	if err != nil {
		serverError(w, "notify rate limit", err)
		return
	}
	n, err := result.RowsAffected()
	if err != nil {
		serverError(w, "notify rate limit result", err)
		return
	}
	if n == 0 {
		w.Header().Set("Retry-After", "60")
		writeError(w, http.StatusTooManyRequests, "Wait at least one minute after the previous delivery before sending again.")
		return
	}
	if err := a.mail.send(r.Context(), recipients, subject, body, formats...); err != nil {
		if errors.Is(err, errDeliveryUnknown) {
			writeError(w, http.StatusBadGateway, "Delivery status is uncertain. Check the inbox and mail server logs before trying again to avoid duplicate emails.")
		} else {
			// SMTP errors may contain mailbox or authentication details. Keep them
			// out of responses and logs; configuration errors above are actionable.
			writeError(w, http.StatusBadGateway, "The email has not been sent. Check SMTP credentials, recipient addresses, provider delivery limits, and the server connection to the SMTP port.")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"accepted": len(recipients), "message": "The SMTP server has accepted the email for delivery. Check the recipient inbox or spam folder."})
}
