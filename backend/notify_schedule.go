package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	htmltemplate "html/template"
	"net/http"
	"strings"
	"text/template"
	"text/template/parse"
	"time"
)

type notifySchedule struct {
	Enabled bool   `json:"enabled"`
	Hour    string `json:"hour"`
	Mode    string `json:"mode"`
	Format  string `json:"format"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

func defaultSchedule() notifySchedule {
	return notifySchedule{Hour: "08:00", Mode: "template", Format: "html", Subject: "Task reminder — {{.Total}} active tasks", Body: defaultNotifyTable}
}

const defaultNotifyTable = `<div style="font-family:Arial,sans-serif;color:#253746;font-size:14px;line-height:1.6">
<h2 style="font-size:20px;font-weight:600;margin:0 0 8px">Task reminder</h2>
<p style="margin:0 0 20px;color:#64748b">Hello {{.Name}}, here is your active task summary for {{.Date}}.</p>
<table border="1" cellpadding="12" cellspacing="0" style="border-collapse:collapse;width:100%;border:1px solid #dce4eb;border-color:#dce4eb;font-size:13px">
<thead><tr style="background:#eef4f7;text-align:left;color:#365164"><th>Task</th><th>Priority</th><th>Category</th><th>Due date</th><th>Status</th></tr></thead>
<tbody>{{range .Tasks}}
<tr style="vertical-align:top"><td{{if not .Title}} align="center"{{end}}>{{if .Title}}{{.Title}}{{else}}-{{end}}</td><td{{if not .Priority}} align="center"{{end}}>{{if eq .Priority "high"}}Urgent / high{{else if eq .Priority "medium"}}Medium{{else if eq .Priority "low"}}Low{{else}}-{{end}}</td><td{{if not .Category}} align="center"{{end}}>{{if .Category}}{{.Category}}{{else}}-{{end}}</td><td{{if not .DueDate}} align="center"{{end}}>{{if .DueDate}}{{.DueDate}}{{else}}-{{end}}</td><td>Incomplete</td></tr>
{{else}}<tr><td colspan="5" align="center" style="padding:24px;color:#94a3b8">-</td></tr>{{end}}</tbody>
<tfoot><tr style="background:#f7fafc"><td colspan="5"><strong>Total: {{.Total}} active tasks</strong> · Urgent: {{.Urgent}} · Medium: {{.Medium}} · Low: {{.Low}}</td></tr></tfoot>
</table>
<p style="font-size:12px;color:#94a3b8;margin-top:16px">Sorted by priority and due date. Up to 100 tasks are shown; totals include all active tasks.</p>
</div>`

func validateNotifyFormat(format string) error {
	if format != "" && format != "text" && format != "html" {
		return errors.New("Message format must be text or HTML.")
	}
	return nil
}
func jakarta() *time.Location {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return time.FixedZone("WIB", 7*3600)
	}
	return loc
}
func nextNotifyRun(hour string, now time.Time) time.Time {
	t, _ := time.Parse("15:04", hour)
	local := now.In(jakarta())
	next := time.Date(local.Year(), local.Month(), local.Day(), t.Hour(), 0, 0, 0, jakarta())
	if !next.After(now) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}
func (s notifySchedule) validate() error {
	if err := validateNotifyFormat(s.Format); err != nil {
		return err
	}
	t, err := time.Parse("15:04", s.Hour)
	if err != nil || t.Minute() != 0 || t.Format("15:04") != s.Hour {
		return errors.New("Select an hourly time from 00:00 to 23:00.")
	}
	if s.Mode != "text" && s.Mode != "template" {
		return errors.New("Select plain text or custom template mode.")
	}
	return validateEmailContent(s.Subject, s.Body)
}

type notifyTemplateData struct {
	Name        string
	Date        string
	Total       int
	Urgent      int
	High        int
	Medium      int
	Low         int
	Tasks       []Todo
	HighTasks   []Todo
	MediumTasks []Todo
	LowTasks    []Todo
}

// Bounded output prevents nested template loops from exhausting memory.
type templateOutput struct{ bytes.Buffer }

func (b *templateOutput) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 100000 {
		return 0, errors.New("Template output is too long (maximum 100,000 bytes).")
	}
	return b.Buffer.Write(p)
}
func renderNotifyTemplate(source string, data notifyTemplateData, formats ...string) (string, error) {
	if len(formats) > 0 && formats[0] == "html" {
		t, err := htmltemplate.New("notify").Option("missingkey=error").Parse(source)
		if err != nil {
			return "", fmt.Errorf("Invalid HTML template: %w", err)
		}
		if err = validateTemplateTree(t.Tree.Root, 0); err != nil {
			return "", err
		}
		var out templateOutput
		if err = t.Execute(&out, data); err != nil {
			return "", fmt.Errorf("HTML template rendering failed: %w", err)
		}
		return out.String(), nil
	}
	t, err := template.New("notify").Option("missingkey=error").Parse(source)
	if err != nil {
		return "", fmt.Errorf("Invalid template: %w", err)
	}
	if err = validateTemplateTree(t.Tree.Root, 0); err != nil {
		return "", err
	}
	var out templateOutput
	if err = t.Execute(&out, data); err != nil {
		return "", fmt.Errorf("Template rendering failed: %w", err)
	}
	return out.String(), nil
}
func validateTemplateTree(node parse.Node, depth int) error {
	if node == nil {
		return nil
	}
	switch n := node.(type) {
	case *parse.ListNode:
		if n == nil {
			return nil
		}
		for _, child := range n.Nodes {
			if err := validateTemplateTree(child, depth); err != nil {
				return err
			}
		}
	case *parse.RangeNode:
		if depth > 0 {
			return errors.New("Nested range loops are not supported. Use separate priority lists.")
		}
		if err := validateTemplateTree(n.List, depth+1); err != nil {
			return err
		}
		return validateTemplateTree(n.ElseList, depth+1)
	case *parse.IfNode:
		if err := validateTemplateTree(n.List, depth); err != nil {
			return err
		}
		return validateTemplateTree(n.ElseList, depth)
	case *parse.WithNode:
		if err := validateTemplateTree(n.List, depth); err != nil {
			return err
		}
		return validateTemplateTree(n.ElseList, depth)
	case *parse.TemplateNode:
		return errors.New("Recursive template calls are not supported.")
	}
	return nil
}
func (a *App) renderNotify(userID int64, subject, body string, formats ...string) (string, string, error) {
	data := notifyTemplateData{Date: time.Now().In(jakarta()).Format("02 January 2006")}
	if err := a.db.QueryRow(`SELECT display_name FROM users WHERE id=$1`, userID).Scan(&data.Name); err != nil {
		return "", "", errors.New("Unable to retrieve the task owner.")
	}
	rows, err := a.db.Query(`SELECT `+todoCols+` FROM todos WHERE user_id=$1 AND done=FALSE ORDER BY CASE priority WHEN 'high' THEN 0 WHEN 'medium' THEN 1 ELSE 2 END, due_date NULLS LAST, id`, userID)
	if err != nil {
		return "", "", errors.New("Unable to retrieve tasks.")
	}
	defer rows.Close()
	for rows.Next() {
		task, err := scanTodo(rows)
		if err != nil {
			return "", "", err
		}
		data.Total++
		switch task.Priority {
		case "high":
			data.High++
			if len(data.HighTasks) < 100 {
				data.HighTasks = append(data.HighTasks, task)
			}
		case "medium":
			data.Medium++
			if len(data.MediumTasks) < 100 {
				data.MediumTasks = append(data.MediumTasks, task)
			}
		case "low":
			data.Low++
			if len(data.LowTasks) < 100 {
				data.LowTasks = append(data.LowTasks, task)
			}
		}
		if len(data.Tasks) < 100 {
			data.Tasks = append(data.Tasks, task)
		}
	}
	if err := rows.Err(); err != nil {
		return "", "", err
	}
	data.Urgent = data.High
	renderedSubject, err := renderNotifyTemplate(subject, data)
	if err != nil {
		return "", "", err
	}
	renderedBody, err := renderNotifyTemplate(body, data, formats...)
	if err != nil {
		return "", "", err
	}
	err = validateEmailContent(renderedSubject, renderedBody, formats...)
	return renderedSubject, renderedBody, err
}
func (a *App) handleGetSchedule(w http.ResponseWriter, r *http.Request, u User) {
	s := defaultSchedule()
	var raw []byte
	var next, last sql.NullTime
	var status string
	err := a.db.QueryRow(`SELECT config,next_run,last_run,last_status FROM notify_schedule WHERE user_id=$1`, u.ID).Scan(&raw, &next, &last, &status)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		serverError(w, "schedule", err)
		return
	}
	if err == nil {
		s.Format = "text"
		if err = json.Unmarshal(raw, &s); err != nil {
			serverError(w, "schedule decode", err)
			return
		}
	}
	writeJSON(w, 200, map[string]any{"schedule": s, "defaultTemplate": defaultSchedule(), "timezone": "Asia/Jakarta", "nextRun": next, "lastRun": last, "lastStatus": status})
}
func (a *App) handleSaveSchedule(w http.ResponseWriter, r *http.Request, u User) {
	var s notifySchedule
	if !decode(w, r, &s) {
		return
	}
	s.Subject = strings.TrimSpace(s.Subject)
	if err := s.validate(); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	if s.Mode == "template" {
		if _, _, err := a.renderNotify(u.ID, s.Subject, s.Body, s.Format); err != nil {
			writeError(w, 400, err.Error())
			return
		}
	}
	var next any
	if s.Enabled {
		if err := a.mail.validate(); err != nil {
			writeError(w, 400, err.Error())
			return
		}
		recipients, err := a.notifyRecipients(u)
		if err != nil {
			serverError(w, "schedule recipients", err)
			return
		}
		if len(recipients) == 0 {
			writeError(w, 400, "Save recipients before enabling the schedule.")
			return
		}
		next = nextNotifyRun(s.Hour, time.Now())
	}
	raw, _ := json.Marshal(s)
	_, err := a.db.Exec(`INSERT INTO notify_schedule(user_id,config,next_run) VALUES($1,$2::jsonb,$3) ON CONFLICT(user_id) DO UPDATE SET config=EXCLUDED.config,next_run=EXCLUDED.next_run`, u.ID, string(raw), next)
	if err != nil {
		serverError(w, "schedule save", err)
		return
	}
	a.handleGetSchedule(w, r, u)
}
func (a *App) handlePreviewNotify(w http.ResponseWriter, r *http.Request, u User) {
	var s notifySchedule
	if !decode(w, r, &s) {
		return
	}
	subject, body := s.Subject, s.Body
	if err := validateNotifyFormat(s.Format); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	var err error
	if s.Mode == "template" {
		subject, body, err = a.renderNotify(u.ID, subject, body, s.Format)
	} else if s.Mode != "text" {
		err = errors.New("Invalid mode.")
	}
	if err == nil {
		err = validateEmailContent(subject, body, s.Format)
	}
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"subject": subject, "body": body, "format": s.Format})
}

func (a *App) runNotifyScheduler() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		a.processNotifySchedule()
		a.processDiscordSchedule()
	}
}
func (a *App) processNotifySchedule() {
	// Reserve a due job and mailbox slot together, before any SMTP side effect.
	// Advancing next_run before sending avoids duplicate mail after restarts.
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()
	var id int64
	var raw []byte
	err = tx.QueryRowContext(ctx, `SELECT user_id,config FROM notify_schedule WHERE next_run <= CURRENT_TIMESTAMP ORDER BY next_run,user_id LIMIT 1 FOR UPDATE SKIP LOCKED`).Scan(&id, &raw)
	if err != nil {
		return
	}
	result, err := tx.ExecContext(ctx, `UPDATE notify_mail_state SET last_attempt_at=CURRENT_TIMESTAMP WHERE id=1 AND (last_attempt_at IS NULL OR last_attempt_at <= CURRENT_TIMESTAMP - INTERVAL '1 minute')`)
	if err != nil {
		return
	}
	n, err := result.RowsAffected()
	if err != nil || n == 0 {
		return
	}
	var s notifySchedule
	if json.Unmarshal(raw, &s) != nil {
		return
	}
	next := nextNotifyRun(s.Hour, time.Now())
	_, err = tx.ExecContext(ctx, `UPDATE notify_schedule SET next_run=$2,last_run=CURRENT_TIMESTAMP,last_status='Sending; if the server stops, check the inbox before sending again.' WHERE user_id=$1`, id, next)
	if err != nil {
		return
	}
	if tx.Commit() != nil {
		return
	}
	status := "Accepted by the SMTP server for delivery."
	recipients, err := a.notifyRecipients(User{ID: id})
	subject, body := s.Subject, s.Body
	if err == nil && s.Mode == "template" {
		subject, body, err = a.renderNotify(id, subject, body, s.Format)
	}
	if err == nil {
		err = a.mail.send(ctx, recipients, subject, body, s.Format)
	}
	if err != nil {
		status = "Delivery failed. Check the SMTP configuration, recipients, and template."
		if errors.Is(err, errDeliveryUnknown) {
			status = "Delivery status is uncertain. Check the inbox and mail server logs before sending again."
		}
	}
	a.db.Exec(`UPDATE notify_schedule SET last_status=$2 WHERE user_id=$1`, id, status)
}
