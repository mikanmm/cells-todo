package main

import (
	"net/http"
	"strings"
	"unicode/utf8"
)

func (a *App) handleCategories(w http.ResponseWriter, r *http.Request, u User) {
	rows, err := a.db.Query(`SELECT name FROM categories WHERE user_id=$1 ORDER BY name`, u.ID)
	if err != nil {
		serverError(w, "categories", err)
		return
	}
	defer rows.Close()
	names := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			serverError(w, "categories", err)
			return
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		serverError(w, "categories", err)
		return
	}
	writeJSON(w, http.StatusOK, names)
}
func (a *App) handleAddCategory(w http.ResponseWriter, r *http.Request, u User) {
	var in struct {
		Name string `json:"name"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || utf8.RuneCountInString(in.Name) > 40 {
		writeError(w, 400, "Category names must contain 1–40 characters.")
		return
	}
	res, err := a.db.Exec(`INSERT INTO categories(user_id,name) VALUES($1,$2) ON CONFLICT DO NOTHING`, u.ID, in.Name)
	if err != nil {
		serverError(w, "category", err)
		return
	}
	count, err := res.RowsAffected()
	if err != nil {
		serverError(w, "category", err)
		return
	}
	if count == 0 {
		writeError(w, http.StatusConflict, "This category already exists.")
		return
	}
	writeJSON(w, http.StatusCreated, in)
}

func (a *App) handleDeleteCategory(w http.ResponseWriter, r *http.Request, u User) {
	tx, err := a.db.Begin()
	if err != nil {
		serverError(w, "category", err)
		return
	}
	defer tx.Rollback()
	name := r.PathValue("name")
	res, err := tx.Exec(`DELETE FROM categories WHERE user_id=$1 AND name=$2`, u.ID, name)
	if err != nil {
		serverError(w, "category", err)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeError(w, 404, "Category not found.")
		return
	}
	_, err = tx.Exec(`UPDATE todos SET category='', updated_at=CURRENT_TIMESTAMP WHERE user_id=$1 AND category=$2`, u.ID, name)
	if err != nil {
		serverError(w, "category", err)
		return
	}
	if err = tx.Commit(); err != nil {
		serverError(w, "category", err)
		return
	}
	w.WriteHeader(204)
}
