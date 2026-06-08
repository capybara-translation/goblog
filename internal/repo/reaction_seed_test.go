package repo

import (
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func TestSeedReactionTypes_SeedsAndIsIdempotent(t *testing.T) {
	db, err := sqlx.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	applyMigrations(t, db, "../../migrations/008_create_reactions.sql")

	if err := SeedReactionTypes(db); err != nil {
		t.Fatalf("SeedReactionTypes: %v", err)
	}
	var count int
	if err := db.Get(&count, "SELECT COUNT(*) FROM reaction_types"); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != len(DefaultReactionTypes) {
		t.Fatalf("expected %d rows, got %d", len(DefaultReactionTypes), count)
	}

	if err := SeedReactionTypes(db); err != nil {
		t.Fatalf("second SeedReactionTypes: %v", err)
	}
	if err := db.Get(&count, "SELECT COUNT(*) FROM reaction_types"); err != nil {
		t.Fatalf("re-count: %v", err)
	}
	if count != len(DefaultReactionTypes) {
		t.Fatalf("seed not idempotent: got %d rows", count)
	}
}

func TestIsSeedEmoji(t *testing.T) {
	for _, rt := range DefaultReactionTypes {
		if !IsSeedEmoji(rt.Emoji) {
			t.Errorf("IsSeedEmoji(%q) = false, want true", rt.Emoji)
		}
	}
	if IsSeedEmoji("🚀") {
		t.Error("IsSeedEmoji(🚀) = true, want false")
	}
}
