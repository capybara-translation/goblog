package repo

import (
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// applyMigrations loads and executes the given migration files into db.
func applyMigrations(t *testing.T, db *sqlx.DB, paths ...string) {
	t.Helper()
	for _, p := range paths {
		schema, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("failed to read migration %s: %v", p, err)
		}
		if _, err := db.Exec(string(schema)); err != nil {
			t.Fatalf("failed to execute migration %s: %v", p, err)
		}
	}
}

func TestReactionsMigration_SeedsActiveTypes(t *testing.T) {
	const wantSeedCount = 5 // matches the INSERT OR IGNORE seed in 008_create_reactions.sql

	db, err := sqlx.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	applyMigrations(t, db,
		"../../migrations/001_create_posts.sql",
		"../../migrations/008_create_reactions.sql",
	)

	var count int
	if err := db.Get(&count, "SELECT COUNT(*) FROM reaction_types WHERE is_active = 1"); err != nil {
		t.Fatalf("failed to count reaction_types: %v", err)
	}
	if count != wantSeedCount {
		t.Fatalf("expected %d active reaction types, got %d", wantSeedCount, count)
	}

	// Re-running the migration must stay idempotent (INSERT OR IGNORE).
	applyMigrations(t, db, "../../migrations/008_create_reactions.sql")
	if err := db.Get(&count, "SELECT COUNT(*) FROM reaction_types"); err != nil {
		t.Fatalf("failed to re-count: %v", err)
	}
	if count != wantSeedCount {
		t.Fatalf("expected migration to be idempotent (%d rows), got %d", wantSeedCount, count)
	}
}
