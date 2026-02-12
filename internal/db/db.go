package db

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// Open opens a connection to SQLite database
func Open(dbPath string) (*sqlx.DB, error) {
	// Ensure the database file directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open the database
	db, err := sqlx.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// RunMigrations executes the embedded migrations
func RunMigrations(db *sqlx.DB, migrationsFS embed.FS, migrationPaths ...string) error {
	for _, migrationPath := range migrationPaths {
		// Read the embedded migration file
		content, err := migrationsFS.ReadFile(migrationPath)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", migrationPath, err)
		}

		// Execute the migration
		if _, err := db.Exec(string(content)); err != nil {
			// SQLite does not support ALTER TABLE ADD COLUMN IF NOT EXISTS,
			// so we ignore duplicate column errors (when migration is already applied)
			if isDuplicateColumnError(err) {
				continue
			}
			return fmt.Errorf("failed to execute migration %s: %w", migrationPath, err)
		}
	}

	return nil
}

// isDuplicateColumnError checks if the error is a duplicate column error
func isDuplicateColumnError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "duplicate column name")
}
