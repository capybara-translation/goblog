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

func TestReactionRepository_FindSummariesByPostIDs(t *testing.T) {
	db := setupReactionTestDB(t)
	defer db.Close()
	// Add a second post for the batch test.
	db.MustExec("INSERT INTO posts (id, title, slug, content, status) VALUES (2, 'P2', 'p2', 'c', 'published')")
	r := NewReactionRepository(db)

	// Visitor-A reacts 👍 (type 1) on post 1 only.
	if err := r.Add(1, 1, "visitor-A"); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	t.Run("visitor-A sees reacted=true for post1 thumbsup, false elsewhere", func(t *testing.T) {
		m, err := r.FindSummariesByPostIDs([]int64{1, 2}, "visitor-A")
		if err != nil {
			t.Fatalf("FindSummariesByPostIDs failed: %v", err)
		}
		if len(m) != 2 {
			t.Fatalf("expected entries for 2 posts, got %d", len(m))
		}

		// Post 1: 4 active types (🤔 is deactivated in setupReactionTestDB)
		s1 := m[1]
		if len(s1) != 4 {
			t.Fatalf("post 1: expected 4 active summaries, got %d", len(s1))
		}
		// First active type ordered by sort_order is 👍
		if s1[0].Emoji != "👍" {
			t.Errorf("post 1 first type: expected 👍, got %s", s1[0].Emoji)
		}
		if s1[0].Count != 1 || !s1[0].Reacted {
			t.Errorf("post 1 👍: expected count=1 reacted=true, got %+v", s1[0])
		}
		for _, s := range s1[1:] {
			if s.Count != 0 || s.Reacted {
				t.Errorf("post 1 %s: expected count=0 reacted=false, got %+v", s.Emoji, s)
			}
		}

		// Post 2: all types should have count=0 reacted=false
		s2 := m[2]
		if len(s2) != 4 {
			t.Fatalf("post 2: expected 4 active summaries, got %d", len(s2))
		}
		for _, s := range s2 {
			if s.Count != 0 || s.Reacted {
				t.Errorf("post 2 %s: expected count=0 reacted=false, got %+v", s.Emoji, s)
			}
		}
	})

	t.Run("empty visitorKey yields reacted=false everywhere", func(t *testing.T) {
		m, err := r.FindSummariesByPostIDs([]int64{1, 2}, "")
		if err != nil {
			t.Fatalf("FindSummariesByPostIDs failed: %v", err)
		}
		for postID, sums := range m {
			for _, s := range sums {
				if s.Reacted {
					t.Errorf("post %d %s: expected reacted=false for empty visitorKey, got true", postID, s.Emoji)
				}
			}
		}
		// post 1 should still have count=1 for 👍
		if m[1][0].Count != 1 {
			t.Errorf("post 1 👍: expected count=1, got %d", m[1][0].Count)
		}
	})

	t.Run("empty input returns empty map", func(t *testing.T) {
		m, err := r.FindSummariesByPostIDs([]int64{}, "visitor-A")
		if err != nil {
			t.Fatalf("FindSummariesByPostIDs failed: %v", err)
		}
		if len(m) != 0 {
			t.Fatalf("expected empty map for empty input, got %d entries", len(m))
		}
	})
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
