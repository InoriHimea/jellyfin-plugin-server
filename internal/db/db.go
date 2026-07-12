package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func Open(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create db dir: %w", err)
	}

	// WAL mode: concurrent readers don't block each other or writers.
	// cache=shared lets multiple connections share the page cache.
	db, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=10000&cache=shared")
	if err != nil {
		return fmt.Errorf("open sqlite: %w", err)
	}

	// Allow up to 8 concurrent read connections; writes still serialise internally.
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(0)

	if err := db.Ping(); err != nil {
		return fmt.Errorf("ping sqlite: %w", err)
	}

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}

	// Migrations for existing databases — errors are ignored when column already exists.
	db.Exec(`ALTER TABLE plugins ADD COLUMN image_url TEXT NOT NULL DEFAULT ''`)
	db.Exec(`ALTER TABLE plugin_versions ADD COLUMN fail_reason TEXT NOT NULL DEFAULT ''`)

	if err := migrateVersionAbiUnique(db); err != nil {
		return fmt.Errorf("migrate plugin_versions unique key: %w", err)
	}

	DB = db
	return nil
}

// migrateVersionAbiUnique upgrades plugin_versions' unique key from
// (plugin_id, version) to (plugin_id, version, target_abi). The old key
// let a repo's multiple ABI-targeted builds published under one version
// number (e.g. separate 10.10/10.11 compat builds) collide: only the last
// one processed survived, keeping the first row's target_abi paired with
// the last row's checksum/source_url. SQLite can't alter a UNIQUE
// constraint in place, so this rebuilds the table when the old constraint
// is detected; a no-op on fresh databases that already have the new schema.
func migrateVersionAbiUnique(db *sql.DB) error {
	var ddl string
	err := db.QueryRow(
		`SELECT sql FROM sqlite_master WHERE type='table' AND name='plugin_versions'`,
	).Scan(&ddl)
	if err != nil {
		return nil
	}
	if !strings.Contains(ddl, "UNIQUE(plugin_id, version)") {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmts := []string{
		`CREATE TABLE plugin_versions_new (
			id             TEXT PRIMARY KEY,
			plugin_id      TEXT NOT NULL REFERENCES plugins(id) ON DELETE CASCADE,
			version        TEXT NOT NULL,
			changelog      TEXT,
			target_abi     TEXT,
			source_url     TEXT NOT NULL,
			checksum       TEXT NOT NULL,
			timestamp      TEXT,
			local_path     TEXT,
			download_status TEXT NOT NULL DEFAULT 'pending',
			downloaded_at  TEXT,
			fail_reason    TEXT NOT NULL DEFAULT '',
			UNIQUE(plugin_id, version, target_abi)
		)`,
		`INSERT INTO plugin_versions_new SELECT * FROM plugin_versions`,
		`DROP TABLE plugin_versions`,
		`ALTER TABLE plugin_versions_new RENAME TO plugin_versions`,
		`CREATE INDEX IF NOT EXISTS idx_versions_plugin       ON plugin_versions(plugin_id)`,
		`CREATE INDEX IF NOT EXISTS idx_versions_status       ON plugin_versions(download_status)`,
		`CREATE INDEX IF NOT EXISTS idx_versions_checksum     ON plugin_versions(checksum)`,
	}
	for _, s := range stmts {
		if _, err := tx.Exec(s); err != nil {
			return fmt.Errorf("%s: %w", s, err)
		}
	}
	return tx.Commit()
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
