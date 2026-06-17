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
	if err := SeedReactionTypes(db); err != nil {
		t.Fatalf("seed reaction types: %v", err)
	}
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

func TestReactionRepository_FindTotalsByPostIDs(t *testing.T) {
	db := setupReactionTestDB(t)
	defer db.Close()
	db.MustExec("INSERT INTO posts (id, title, slug, content, status) VALUES (2, 'P2', 'p2', 'c', 'published')")
	r := NewReactionRepository(db)

	// Resolve the deactivated 🤔 type id (setupReactionTestDB deactivates it).
	var thinkingID int64
	if err := db.Get(&thinkingID, "SELECT id FROM reaction_types WHERE emoji = '🤔'"); err != nil {
		t.Fatalf("lookup 🤔 id: %v", err)
	}
	// 👍 (type 1, active) and 🤔 (inactive) both reacted on post 1.
	if err := r.Add(1, 1, "v1"); err != nil {
		t.Fatalf("Add 👍: %v", err)
	}
	if err := r.Add(1, thinkingID, "v1"); err != nil {
		t.Fatalf("Add 🤔: %v", err)
	}

	m, err := r.FindTotalsByPostIDs([]int64{1, 2})
	if err != nil {
		t.Fatalf("FindTotalsByPostIDs: %v", err)
	}
	if len(m) != 2 {
		t.Fatalf("expected 2 posts, got %d", len(m))
	}

	// Post 1: all 5 types present, ordered by sort_order.
	p1 := m[1]
	if len(p1) != 5 {
		t.Fatalf("post 1: expected 5 types, got %d", len(p1))
	}
	if p1[0].Emoji != "👍" || p1[0].Count != 1 || !p1[0].IsActive {
		t.Errorf("post 1[0] expected 👍 count=1 active, got %+v", p1[0])
	}
	if p1[1].Emoji != "❤️" || p1[1].Count != 0 || !p1[1].IsActive {
		t.Errorf("post 1[1] expected ❤️ count=0 active, got %+v", p1[1])
	}
	// 🤔 deactivated but has a reaction -> last by sort_order, count=1, inactive.
	if p1[4].Emoji != "🤔" || p1[4].Count != 1 || p1[4].IsActive {
		t.Errorf("post 1[4] expected 🤔 count=1 inactive, got %+v", p1[4])
	}

	// Post 2: every type present at count 0 (incl. the inactive 🤔).
	p2 := m[2]
	if len(p2) != 5 {
		t.Fatalf("post 2: expected 5 types, got %d", len(p2))
	}
	for _, c := range p2 {
		if c.Count != 0 {
			t.Errorf("post 2 %s: expected count=0, got %d", c.Emoji, c.Count)
		}
	}
	if p2[4].Emoji != "🤔" || p2[4].IsActive {
		t.Errorf("post 2[4] expected 🤔 inactive, got %+v", p2[4])
	}
}

func TestReactionRepository_FindTotalsByPostIDs_Empty(t *testing.T) {
	db := setupReactionTestDB(t)
	defer db.Close()
	r := NewReactionRepository(db)
	m, err := r.FindTotalsByPostIDs(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m) != 0 {
		t.Fatalf("expected empty map, got %d", len(m))
	}
}

func TestReactionRepository_FindTotalsByPostIDs_SinglePost(t *testing.T) {
	db := setupReactionTestDB(t)
	defer db.Close()
	r := NewReactionRepository(db)
	m, err := r.FindTotalsByPostIDs([]int64{1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m[1]) != 5 {
		t.Fatalf("expected 5 types for single post, got %d", len(m[1]))
	}
}

func TestReactionRepository_FindTotalsByPostIDs_CountsMultipleVisitors(t *testing.T) {
	db := setupReactionTestDB(t)
	defer db.Close()
	r := NewReactionRepository(db)
	if err := r.Add(1, 1, "v1"); err != nil {
		t.Fatalf("Add v1: %v", err)
	}
	if err := r.Add(1, 1, "v2"); err != nil {
		t.Fatalf("Add v2: %v", err)
	}
	m, err := r.FindTotalsByPostIDs([]int64{1})
	if err != nil {
		t.Fatalf("FindTotalsByPostIDs: %v", err)
	}
	if m[1][0].Emoji != "👍" || m[1][0].Count != 2 {
		t.Errorf("expected 👍 count=2 from two visitors, got %+v", m[1][0])
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

func TestReactionRepository_FindTotalsByPostIDs_DeduplicatesInput(t *testing.T) {
	db := setupReactionTestDB(t)
	defer db.Close()
	r := NewReactionRepository(db)
	// One real reaction on post 1, 👍.
	if err := r.Add(1, 1, "v1"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	// A duplicated post id must not multiply the virtual-posts rows and inflate
	// COUNT after GROUP BY.
	m, err := r.FindTotalsByPostIDs([]int64{1, 1})
	if err != nil {
		t.Fatalf("FindTotalsByPostIDs: %v", err)
	}
	if len(m) != 1 {
		t.Fatalf("expected 1 post key for duplicated input, got %d", len(m))
	}
	if m[1][0].Emoji != "👍" || m[1][0].Count != 1 {
		t.Errorf("duplicate post id must not inflate count: expected 👍 count=1, got %+v", m[1][0])
	}
}
