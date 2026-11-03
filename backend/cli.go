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
	fmt.Fprintf(os.Stderr, "Gagal: "+format+"\n", a...)
	os.Exit(1)
}

// readPassword mengembalikan password dari argumen, atau dari stdin jika argumennya "-".
func readPassword(arg string) string {
	if arg != "-" {
		return arg
	}
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		die("tidak ada password di stdin")
	}
	return strings.TrimRight(line, "\r\n")
}

func checkPassword(p string) {
	if utf8.RuneCountInString(p) < 8 {
		die("password minimal 8 karakter")
	}
	if len(p) > 72 {
		die("password maksimal 72 byte (batas bcrypt)")
	}
}

func openMigrated() *sql.DB {
	db, err := openDB()
	if err != nil {
		die("%v", err)
	}
	if err := migrate(db); err != nil {
		die("migrasi: %v", err)
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
		die("username harus 3-64 karakter: huruf, angka, titik, garis bawah, atau strip")
	}
	password := readPassword(args[1])
	checkPassword(password)

	display := username
	if len(args) > 2 {
		display = strings.TrimSpace(strings.Join(args[2:], " "))
	}
	if display == "" || utf8.RuneCountInString(display) > 100 {
		die("nama tampilan harus 1-100 karakter")
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
		die("username %q sudah dipakai", username)
	}
	if err != nil {
		die("%v", err)
	}
	fmt.Printf("Akun %q berhasil dibuat.\n", username)
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
		die("akun %q tidak ditemukan", username)
	}
	// Semua sesi lama dicabut agar perangkat lain harus masuk ulang.
	_, _ = db.Exec(`DELETE FROM sessions WHERE user_id = (SELECT id FROM users WHERE username = $1)`, username)
	fmt.Printf("Password akun %q diganti. Semua sesi lama dicabut.\n", username)
}
