package main

import (
	"bufio"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

var usernameRe = regexp.MustCompile(`^[A-Za-z0-9_.-]{3,64}$`)

func die(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", a...)
	os.Exit(1)
}

// readPassword reads the password argument, or stdin when the argument is "-".
func readPassword(arg string) string {
	if arg != "-" {
		return arg
	}
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		die("no password provided on stdin")
	}
	return strings.TrimRight(line, "\r\n")
}

func checkPassword(p string) {
	if utf8.RuneCountInString(p) < 8 {
		die("passwords must contain at least 8 characters")
	}
	if len(p) > 72 {
		die("passwords must not exceed 72 bytes (bcrypt limit)")
	}
}

func openMigrated() *sql.DB {
	db, err := openDB()
	if err != nil {
		die("%v", err)
	}
	if err := migrate(db); err != nil {
		die("migration: %v", err)
	}
	return db
}

func runHash(args []string) {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, usageText)
		os.Exit(2)
	}
	p := readPassword(args[0])
	checkPassword(p)
	h, err := bcrypt.GenerateFromPassword([]byte(p), bcryptCost)
	if err != nil {
		die("%v", err)
	}
	fmt.Println(string(h))
}

func runAddUser(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, usageText)
		os.Exit(2)
	}
	username := strings.TrimSpace(args[0])
	if !usernameRe.MatchString(username) {
		die("usernames must contain 3–64 characters: letters, digits, periods, underscores, or hyphens")
	}
	password := readPassword(args[1])
	checkPassword(password)

	display := username
	if len(args) > 2 {
		display = strings.TrimSpace(strings.Join(args[2:], " "))
	}
	if display == "" || utf8.RuneCountInString(display) > 100 {
		die("display names must contain 1–100 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		die("%v", err)
	}

	db := openMigrated()
	defer db.Close()

	_, err = db.Exec(
		`INSERT INTO users (username, display_name, password_hash) VALUES ($1, $2, $3)`,
		username, display, string(hash),
	)
	var me *pq.Error
	if errors.As(err, &me) && me.Code == "23505" {
		die("username %q is already in use", username)
	}
	if err != nil {
		die("%v", err)
	}
	fmt.Printf("Account %q created successfully.\n", username)
}

func runPasswd(args []string) {
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, usageText)
		os.Exit(2)
	}
	username := strings.TrimSpace(args[0])
	password := readPassword(args[1])
	checkPassword(password)

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		die("%v", err)
	}

	db := openMigrated()
	defer db.Close()

	res, err := db.Exec(`UPDATE users SET password_hash = $1 WHERE username = $2`, string(hash), username)
	if err != nil {
		die("%v", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		die("account %q not found", username)
	}
	// Revoke existing sessions so other devices must sign in again.
	_, _ = db.Exec(`DELETE FROM sessions WHERE user_id = (SELECT id FROM users WHERE username = $1)`, username)
	fmt.Printf("Password for account %q changed. All previous sessions have been revoked.\n", username)
}
