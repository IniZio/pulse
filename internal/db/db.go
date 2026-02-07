// Package db provides SQLite database operations for Pulse.
package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB represents the SQLite database connection.
type DB struct {
	*sql.DB
	path string
}

// New creates a new database connection.
func New(dataDir string) (*DB, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "pulse.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set pragmas for performance
	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA synchronous=NORMAL")
	db.Exec("PRAGMA busy_timeout=30000")

	return &DB{DB: db, path: dbPath}, nil
}

// Migrate runs database migrations.
func (db *DB) Migrate() error {
	// Create workspaces table
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS workspaces (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			settings TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("failed to create workspaces table: %w", err)
	}

	// Create issues table
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS issues (
			id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL,
			title TEXT NOT NULL,
			description TEXT,
			status TEXT DEFAULT 'backlog',
			priority INTEGER DEFAULT 0,
			assignee_id TEXT,
			estimate INTEGER,
			cycle_id TEXT,
			labels TEXT,
			parent_id TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			completed_at DATETIME,
			FOREIGN KEY (workspace_id) REFERENCES workspaces(id)
		)
	`); err != nil {
		return fmt.Errorf("failed to create issues table: %w", err)
	}

	// Create cycles table
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS cycles (
			id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL,
			name TEXT NOT NULL,
			start_date DATETIME,
			end_date DATETIME,
			status TEXT DEFAULT 'upcoming',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (workspace_id) REFERENCES workspaces(id)
		)
	`); err != nil {
		return fmt.Errorf("failed to create cycles table: %w", err)
	}

	// Create users table (for future auth)
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			name TEXT,
			avatar_url TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	// Create indexes
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_issues_workspace ON issues(workspace_id)`,
		`CREATE INDEX IF NOT EXISTS idx_issues_status ON issues(status)`,
		`CREATE INDEX IF NOT EXISTS idx_issues_assignee ON issues(assignee_id)`,
		`CREATE INDEX IF NOT EXISTS idx_issues_cycle ON issues(cycle_id)`,
		`CREATE INDEX IF NOT EXISTS idx_cycles_workspace ON cycles(workspace_id)`,
		`CREATE INDEX IF NOT EXISTS idx_cycles_status ON cycles(status)`,
	}

	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	// Create default workspace if none exists
	count := 0
	db.QueryRow("SELECT COUNT(*) FROM workspaces").Scan(&count)
	if count == 0 {
		now := time.Now().Format(time.RFC3339)
		_, err := db.Exec(`
			INSERT INTO workspaces (id, name, description, settings, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, "default", "Main Workspace", "Default workspace for tracking", "{}", now, now)
		if err != nil {
			return fmt.Errorf("failed to create default workspace: %w", err)
		}
	}

	return nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.DB.Close()
}

// Path returns the database file path.
func (db *DB) Path() string {
	return db.path
}
