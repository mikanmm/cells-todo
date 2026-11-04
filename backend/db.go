package main

import (
	"database/sql"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/url"
	"os"
	"time"

	_ "github.com/lib/pq"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// openDB connects to PostgreSQL and waits until the database is ready
// (the database container may take a few seconds to initialize).
func openDB() (*sql.DB, error) {
	u := &url.URL{Scheme: "postgres", Host: env("DB_HOST", "db") + ":" + env("DB_PORT", "5432"), Path: env("DB_NAME", "cells"), User: url.UserPassword(env("DB_USER", "cells"), os.Getenv("DB_PASSWORD"))}
	q := u.Query()
	q.Set("sslmode", env("DB_SSLMODE", "disable"))
	u.RawQuery = q.Encode()
	db, err := sql.Open("postgres", u.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	var lastErr error
	for i := 1; i <= 45; i++ {
		if lastErr = db.Ping(); lastErr == nil {
			return db, nil
		}
		log.Printf("waiting for the database (%d/45): %v", i, lastErr)
		time.Sleep(2 * time.Second)
	}
	return nil, fmt.Errorf("unable to connect to the database: %w", lastErr)
}

var schema = []string{
	`CREATE TABLE IF NOT EXISTS registration_codes (email TEXT PRIMARY KEY, digest TEXT NOT NULL, issued TIMESTAMPTZ NOT NULL DEFAULT now(), attempts INTEGER NOT NULL DEFAULT 0)`,

	`CREATE TABLE IF NOT EXISTS users (
		id            BIGSERIAL PRIMARY KEY,
		username      VARCHAR(64)  NOT NULL UNIQUE,
		display_name  VARCHAR(100) NOT NULL,
		password_hash VARCHAR(100) NOT NULL,
		created_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,

	`CREATE UNIQUE INDEX IF NOT EXISTS users_username_lower_key ON users (lower(username))`,
	`ALTER TABLE users ADD COLUMN IF NOT EXISTS verified_email TEXT UNIQUE`,
	`CREATE OR REPLACE FUNCTION registration_mailbox(address text) RETURNS text
     LANGUAGE sql IMMUTABLE STRICT AS $$
       SELECT CASE WHEN lower(address) ~ '@(gmail|googlemail)[.]com$'
         THEN replace(split_part(split_part(lower(address), '@', 1), '+', 1), '.', '') || '@gmail.com'
         ELSE lower(address) END
     $$`,
	`CREATE UNIQUE INDEX IF NOT EXISTS users_registration_mailbox_key
     ON users (registration_mailbox(verified_email))`,
	`CREATE TABLE IF NOT EXISTS sessions (
		token_hash CHAR(64) NOT NULL PRIMARY KEY,
		user_id    BIGINT NOT NULL,
		expires_at TIMESTAMPTZ NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		CONSTRAINT fk_sessions_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	)`,

	`CREATE TABLE IF NOT EXISTS todos (
		id         BIGSERIAL PRIMARY KEY,
		user_id    BIGINT NOT NULL,
		title      VARCHAR(200) NOT NULL,
		notes      TEXT NOT NULL,
		priority   VARCHAR(6) CHECK (priority IN ('low','medium','high')) NOT NULL DEFAULT 'medium',
		category   VARCHAR(40) NOT NULL DEFAULT '',
		due_date   DATE NULL,
		done       BOOLEAN NOT NULL DEFAULT FALSE,
		done_at    TIMESTAMPTZ NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		CONSTRAINT fk_todos_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	)`,
	`ALTER TABLE todos ADD COLUMN IF NOT EXISTS repeat_type TEXT NOT NULL DEFAULT 'none' CHECK (repeat_type IN ('none', 'weekdays', 'daily', 'custom'))`,
	`ALTER TABLE todos ADD COLUMN IF NOT EXISTS repeat_days INTEGER[] NOT NULL DEFAULT '{}'`,
	`ALTER TABLE todos ADD COLUMN IF NOT EXISTS notify BOOLEAN NOT NULL DEFAULT FALSE`,
	`CREATE TABLE IF NOT EXISTS categories (user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE, name VARCHAR(40) NOT NULL, PRIMARY KEY(user_id, name))`,
	`INSERT INTO categories SELECT DISTINCT user_id, category FROM todos WHERE category <> '' ON CONFLICT DO NOTHING`,
	`CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at)`,
	`CREATE INDEX IF NOT EXISTS idx_todos_user ON todos(user_id, done)`,
	`CREATE TABLE IF NOT EXISTS notify_settings (
		user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
		recipients JSONB NOT NULL DEFAULT '[]'::jsonb
	)`,
	`CREATE TABLE IF NOT EXISTS notify_mail_state (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		last_attempt_at TIMESTAMPTZ
	)`,
	`INSERT INTO notify_mail_state(id) VALUES (1) ON CONFLICT DO NOTHING`,
	`CREATE TABLE IF NOT EXISTS notify_schedule (user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE, config JSONB NOT NULL, next_run TIMESTAMPTZ, last_run TIMESTAMPTZ, last_status TEXT NOT NULL DEFAULT '')`,
	`CREATE TABLE IF NOT EXISTS notify_discord (user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE, webhook TEXT NOT NULL, config JSONB NOT NULL, next_run TIMESTAMPTZ, last_attempt TIMESTAMPTZ, last_status TEXT NOT NULL DEFAULT '')`,
}

func migrate(db *sql.DB) error {
	for _, stmt := range schema {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("admin"), bcryptCost)
	if err != nil {
		return err
	}
	_, err = db.Exec(`INSERT INTO users (username, display_name, password_hash) VALUES ('admin', 'Admin', $1) ON CONFLICT DO NOTHING`, string(hash))
	return err
}
