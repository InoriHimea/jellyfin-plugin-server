package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func Open(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create db dir: %w", err)
	}

	db, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=5000")
	if err != nil {
		return fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(0)

	if err := db.Ping(); err != nil {
		return fmt.Errorf("ping sqlite: %w", err)
	}

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}

	// Migrations for existing databases — errors are ignored when column already exists.
	db.Exec(`ALTER TABLE plugins ADD COLUMN image_url TEXT NOT NULL DEFAULT ''`)

	DB = db
	return nil
}

func Close() {
	if DB != nil {
		DB.Close()
	}
}

func Now() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func WriteLog(level, message, detail string) {
	if DB == nil {
		return
	}
	DB.Exec(
		`INSERT INTO logs (level, message, detail, created_at) VALUES (?, ?, ?, ?)`,
		level, message, detail, Now(),
	)
}
