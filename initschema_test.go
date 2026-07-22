package goblog

import (
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/capybara-translation/goblog/internal/repo"
)

func TestInitSchema_MigratesAndSeeds(t *testing.T) {
	db, err := sqlx.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("InitSchema: %v", err)
	}

	var count int
	if err := db.Get(&count, "SELECT COUNT(*) FROM reaction_types"); err != nil {
		t.Fatalf("count reaction_types: %v", err)
	}
	if count != len(repo.DefaultReactionTypes) {
		t.Fatalf("expected %d seeded types, got %d", len(repo.DefaultReactionTypes), count)
	}

	// Idempotent across restarts.
	if err := InitSchema(db); err != nil {
		t.Fatalf("second InitSchema: %v", err)
	}
	if err := db.Get(&count, "SELECT COUNT(*) FROM reaction_types"); err != nil {
		t.Fatalf("re-count: %v", err)
	}
	if count != len(repo.DefaultReactionTypes) {
		t.Fatalf("InitSchema not idempotent: %d rows", count)
	}
}

func TestInitSchema_CreatesHealthTables(t *testing.T) {
	db, err := sqlx.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("InitSchema: %v", err)
	}

	for _, table := range []string{"health_records", "healthplanet_tokens"} {
		var name string
		if err := db.Get(&name, `SELECT name FROM sqlite_master WHERE type='table' AND name = ?`, table); err != nil {
			t.Errorf("table %s not created: %v", table, err)
		}
	}
}
