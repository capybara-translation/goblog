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
	db, err := sqlx.Open("sqlite3", ":memory:")
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
	if count != 5 {
		t.Fatalf("expected 5 active reaction types, got %d", count)
	}

	// Re-running the migration must stay idempotent (INSERT OR IGNORE).
	applyMigrations(t, db, "../../migrations/008_create_reactions.sql")
	if err := db.Get(&count, "SELECT COUNT(*) FROM reaction_types"); err != nil {
		t.Fatalf("failed to re-count: %v", err)
	}
	if count != 5 {
		t.Fatalf("expected migration to be idempotent (5 rows), got %d", count)
	}
}
