package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	args := os.Args[1:]
	cmd := "serve"
	if len(args) > 0 {
		cmd = args[0]
		args = args[1:]
	}

	switch cmd {
	case "serve":
		runServe()
	case "adduser":
		runAddUser(args)
	case "passwd":
		runPasswd(args)
	case "hash":
		runHash(args)
	default:
		fmt.Fprintln(os.Stderr, usageText)
		os.Exit(2)
	}
}

const usageText = `Pemakaian:
  server serve                                   jalankan API (default)
  server adduser <username> <password> [nama]    buat akun baru
  server passwd  <username> <password-baru>      ganti password akun
  server hash    <password>                      cetak hash bcrypt (untuk INSERT manual lewat SQL)

Gunakan "-" sebagai password untuk membaca password dari stdin.`

func runServe() {
	db, err := openDB()
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	if err := migrate(db); err != nil {
		log.Fatalf("migrasi: %v", err)
	}

	app := newApp(db)
	go app.cleanupSessions()
	go app.runNotifyScheduler()

	srv := &http.Server{
		Addr:              ":" + env("PORT", "8081"),
		Handler:           app.routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("API berjalan di %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
