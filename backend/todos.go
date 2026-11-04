package main

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/lib/pq"
)

type Todo struct {
	ID         int64         `json:"id"`
	Title      string        `json:"title"`
	Notes      string        `json:"notes"`
	Priority   string        `json:"priority"`
	Category   string        `json:"category"`
	DueDate    *string       `json:"dueDate"`
	RepeatType string        `json:"repeatType"`
	RepeatDays pq.Int64Array `json:"repeatDays"`
	Notify     bool          `json:"notify"`
	Done       bool          `json:"done"`
	DoneAt     *time.Time    `json:"doneAt"`
	CreatedAt  time.Time     `json:"createdAt"`
}

type todoInput struct {
	Title      string        `json:"title"`
	Notes      string        `json:"notes"`
	Priority   string        `json:"priority"`
	Category   string        `json:"category"`
	DueDate    string        `json:"dueDate"`
	RepeatType string        `json:"repeatType"`
	RepeatDays pq.Int64Array `json:"repeatDays"`
	Notify     bool          `json:"notify"`
}

const todoCols = `id, title, notes, priority, category, due_date, done, done_at, created_at, repeat_type, repeat_days, notify`

type rowScanner interface{ Scan(dest ...any) error }

func scanTodo(row rowScanner) (Todo, error) {
	var (
		t      Todo
		due    sql.NullTime
		doneAt sql.NullTime
	)
	if err := row.Scan(&t.ID, &t.Title, &t.Notes, &t.Priority, &t.Category, &due, &t.Done, &doneAt, &t.CreatedAt, &t.RepeatType, &t.RepeatDays, &t.Notify); err != nil {
		return t, err
	}
	if due.Valid {
		s := due.Time.Format("2006-01-02")
		t.DueDate = &s
	}
	if doneAt.Valid {
		d := doneAt.Time
		t.DoneAt = &d
	}
	return t, nil
}

// validate normalizes input and returns an error message, or an empty string when valid.
func (in *todoInput) validate() (due any, msg string) {
	in.Title = strings.TrimSpace(in.Title)
	in.Notes = strings.TrimSpace(in.Notes)
	in.Category = strings.TrimSpace(in.Category)
	in.DueDate = strings.TrimSpace(in.DueDate)

	switch {
	case in.Title == "":
		return nil, "A task title is required."
	case utf8.RuneCountInString(in.Title) > 200:
		return nil, "Task titles must not exceed 200 characters."
	case utf8.RuneCountInString(in.Notes) > 2000:
		return nil, "Notes must not exceed 2,000 characters."
	case utf8.RuneCountInString(in.Category) > 40:
		return nil, "Categories must not exceed 40 characters."
	}

	switch in.Priority {
	case "":
		in.Priority = "medium"
	case "low", "medium", "high":
	default:
		return nil, "Unknown priority."
	}

	if msg := in.validateRepeat(); msg != "" {
		return nil, msg
	}
	if in.DueDate == "" {
		return nil, ""
	}
	d, err := time.Parse("2006-01-02", in.DueDate)
	if err != nil {
		return nil, "Invalid date format."
	}
	return d, ""
}

func pathID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	return id, err == nil && id > 0
}

func serverError(w http.ResponseWriter, where string, err error) {
	log.Printf("%s: %v", where, err)
	writeError(w, http.StatusInternalServerError, "An internal server error occurred.")
}

func (a *App) handleListTodos(w http.ResponseWriter, r *http.Request, u User) {
	if err := a.refreshRepeatingTodos(); err != nil {
		serverError(w, "refresh repeating todos", err)
		return
	}
	rows, err := a.db.Query(`SELECT `+todoCols+` FROM todos WHERE user_id = $1 ORDER BY id DESC`, u.ID)
	if err != nil {
		serverError(w, "list todos", err)
		return
	}
	defer rows.Close()

	todos := []Todo{}
	for rows.Next() {
		t, err := scanTodo(rows)
		if err != nil {
			serverError(w, "scan todo", err)
			return
		}
		todos = append(todos, t)
	}
	if err := rows.Err(); err != nil {
		serverError(w, "list todos", err)
		return
	}
	writeJSON(w, http.StatusOK, todos)
}

func (a *App) handleCreateTodo(w http.ResponseWriter, r *http.Request, u User) {
	var in todoInput
	if !decode(w, r, &in) {
		return
	}
	due, msg := in.validate()
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	if in.DueDate != "" && in.DueDate < time.Now().In(jakarta()).Format("2006-01-02") {
		writeError(w, http.StatusBadRequest, "Choose today or a future due date.")
		return
	}
	var id int64
	err := a.db.QueryRow(
		`INSERT INTO todos (user_id, title, notes, priority, category, due_date, repeat_type, repeat_days, notify) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`,
		u.ID, in.Title, in.Notes, in.Priority, in.Category, due, in.RepeatType, in.RepeatDays, in.Notify,
	).Scan(&id)
	if err != nil {
		serverError(w, "create todo", err)
		return
	}
	t, err := scanTodo(a.db.QueryRow(`SELECT `+todoCols+` FROM todos WHERE id = $1 AND user_id = $2`, id, u.ID))
	if err != nil {
		serverError(w, "read created todo", err)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (a *App) handleUpdateTodo(w http.ResponseWriter, r *http.Request, u User) {
	id, ok := pathID(r)
	if !ok {
		writeError(w, http.StatusNotFound, "Task not found.")
		return
	}
	var in todoInput
	if !decode(w, r, &in) {
		return
	}
	due, msg := in.validate()
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	if in.DueDate != "" && in.DueDate < time.Now().In(jakarta()).Format("2006-01-02") {
		var unchanged bool
		err := a.db.QueryRow(`SELECT coalesce(due_date = $3::date,false) FROM todos WHERE id=$1 AND user_id=$2`, id, u.ID, in.DueDate).Scan(&unchanged)
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, 404, "Task not found.")
			return
		}
		if err != nil {
			serverError(w, "check due date", err)
			return
		}
		if !unchanged {
			writeError(w, 400, "Choose today or a future due date.")
			return
		}
	}
	res, err := a.db.Exec(
		`UPDATE todos SET updated_at = CURRENT_TIMESTAMP, title = $1, notes = $2, priority = $3, category = $4, due_date = $5, repeat_type = $8, repeat_days = $9, notify = $10 WHERE id = $6 AND user_id = $7`,
		in.Title, in.Notes, in.Priority, in.Category, due, id, u.ID, in.RepeatType, in.RepeatDays, in.Notify,
	)
	if err != nil {
		serverError(w, "update todo", err)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeError(w, http.StatusNotFound, "Task not found.")
		return
	}
	t, err := scanTodo(a.db.QueryRow(`SELECT `+todoCols+` FROM todos WHERE id = $1 AND user_id = $2`, id, u.ID))
	if err != nil {
		serverError(w, "read updated todo", err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (a *App) handleSetDone(w http.ResponseWriter, r *http.Request, u User) {
	id, ok := pathID(r)
	if !ok {
		writeError(w, http.StatusNotFound, "Task not found.")
		return
	}
	var in struct {
		Done *bool `json:"done"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Done == nil {
		writeError(w, http.StatusBadRequest, "Completion status is required.")
		return
	}

	var doneAt any
	if *in.Done {
		doneAt = time.Now().UTC()
	}
	res, err := a.db.Exec(
		`UPDATE todos SET updated_at = CURRENT_TIMESTAMP, done = $1, done_at = $2 WHERE id = $3 AND user_id = $4`,
		*in.Done, doneAt, id, u.ID,
	)
	if err != nil {
		serverError(w, "set done", err)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeError(w, http.StatusNotFound, "Task not found.")
		return
	}
	t, err := scanTodo(a.db.QueryRow(`SELECT `+todoCols+` FROM todos WHERE id = $1 AND user_id = $2`, id, u.ID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "Task not found.")
			return
		}
		serverError(w, "read todo", err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (a *App) handleDeleteTodo(w http.ResponseWriter, r *http.Request, u User) {
	id, ok := pathID(r)
	if !ok {
		writeError(w, http.StatusNotFound, "Task not found.")
		return
	}
	res, err := a.db.Exec(`DELETE FROM todos WHERE id = $1 AND user_id = $2`, id, u.ID)
	if err != nil {
		serverError(w, "delete todo", err)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeError(w, http.StatusNotFound, "Task not found.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
