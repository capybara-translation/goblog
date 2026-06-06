package repo

import (
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func setupReactionTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db, err := sqlx.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	applyMigrations(t, db,
		"../../migrations/001_create_posts.sql",
		"../../migrations/008_create_reactions.sql",
	)
	db.MustExec("INSERT INTO posts (id, title, slug, content, status) VALUES (1, 'P1', 'p1', 'c', 'published')")
	// Deactivate one type to test the active filter (🤔, sort_order 50).
	db.MustExec("UPDATE reaction_types SET is_active = 0 WHERE emoji = '🤔'")
	return db
}

func TestReactionRepository_AddAndSummaries(t *testing.T) {
	db := setupReactionTestDB(t)
	defer db.Close()
	r := NewReactionRepository(db)

	const visitor = "visitor-hash-a"
	if err := r.Add(1, 1, visitor); err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	// Second identical Add must be a no-op (UNIQUE / INSERT OR IGNORE).
	if err := r.Add(1, 1, visitor); err != nil {
		t.Fatalf("second Add failed: %v", err)
	}

	summaries, err := r.FindSummariesByPostID(1, visitor)
	if err != nil {
		t.Fatalf("FindSummaries failed: %v", err)
	}
	// Only the 4 active types are returned, ordered by sort_order.
	if len(summaries) != 4 {
		t.Fatalf("expected 4 active summaries, got %d", len(summaries))
	}
	if summaries[0].Emoji != "👍" || summaries[0].Count != 1 || !summaries[0].Reacted {
		t.Fatalf("first active type (👍) should have count=1 reacted=true, got %+v", summaries[0])
	}
	if summaries[1].Count != 0 || summaries[1].Reacted {
		t.Fatalf("type 2 should have count=0 reacted=false, got %+v", summaries[1])
	}
}

func TestReactionRepository_ReactedIsPerVisitor(t *testing.T) {
	db := setupReactionTestDB(t)
	defer db.Close()
	r := NewReactionRepository(db)

	if err := r.Add(1, 1, "visitor-A"); err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	summaries, err := r.FindSummariesByPostID(1, "visitor-B")
	if err != nil {
		t.Fatalf("FindSummaries failed: %v", err)
	}
	if summaries[0].Count != 1 || summaries[0].Reacted {
		t.Fatalf("visitor-B should see count=1 reacted=false, got %+v", summaries[0])
	}
}

func TestReactionRepository_Remove(t *testing.T) {
	db := setupReactionTestDB(t)
	defer db.Close()
	r := NewReactionRepository(db)

	if err := r.Add(1, 1, "visitor-A"); err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if err := r.Remove(1, 1, "visitor-A"); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
	summaries, err := r.FindSummariesByPostID(1, "visitor-A")
	if err != nil {
		t.Fatalf("FindSummaries failed: %v", err)
	}
	if summaries[0].Count != 0 || summaries[0].Reacted {
		t.Fatalf("after remove expected count=0 reacted=false, got %+v", summaries[0])
	}
}

func TestReactionRepository_EmptyVisitorAndNoReactions(t *testing.T) {
	db := setupReactionTestDB(t)
	defer db.Close()
	r := NewReactionRepository(db)

	// Query with an empty visitorKey before any Add calls.
	summaries, err := r.FindSummariesByPostID(1, "")
	if err != nil {
		t.Fatalf("FindSummaries failed: %v", err)
	}
	// All 4 active types must be returned even when the post has no reactions.
	if len(summaries) != 4 {
		t.Fatalf("expected 4 active summaries, got %d", len(summaries))
	}
	for _, s := range summaries {
		if s.Count != 0 {
			t.Errorf("emoji %s: expected count=0, got %d", s.Emoji, s.Count)
		}
		if s.Reacted {
			t.Errorf("emoji %s: expected reacted=false for empty visitorKey, got true", s.Emoji)
		}
	}
}

func TestReactionRepository_IsActiveType(t *testing.T) {
	db := setupReactionTestDB(t)
	defer db.Close()
	r := NewReactionRepository(db)

	active, err := r.IsActiveType(1)
	if err != nil {
		t.Fatalf("IsActiveType failed: %v", err)
	}
	if !active {
		t.Fatal("type 1 should be active")
	}

	var inactiveID int64
	if err := db.Get(&inactiveID, "SELECT id FROM reaction_types WHERE emoji = '🤔'"); err != nil {
		t.Fatalf("lookup failed: %v", err)
	}
	active, err = r.IsActiveType(inactiveID)
	if err != nil {
		t.Fatalf("IsActiveType failed: %v", err)
	}
	if active {
		t.Fatal("deactivated type should not be active")
	}

	active, err = r.IsActiveType(99999)
	if err != nil {
		t.Fatalf("IsActiveType failed: %v", err)
	}
	if active {
		t.Fatal("nonexistent type should not be active")
	}
}
