package repo

import (
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// setupPostViewTestDB sets up an in-memory SQLite database with posts and post_views tables
func setupPostViewTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	migrations := []string{
		"../../migrations/001_create_posts.sql",
		"../../migrations/003_add_is_pinned.sql",
		"../../migrations/006_add_post_views.sql",
	}

	for _, migrationPath := range migrations {
		schema, err := os.ReadFile(migrationPath)
		if err != nil {
			t.Fatalf("failed to read migration file %s: %v", migrationPath, err)
		}
		if _, err := db.Exec(string(schema)); err != nil {
			t.Fatalf("failed to execute migration %s: %v", migrationPath, err)
		}
	}

	// Insert test posts
	db.Exec("INSERT INTO posts (id, title, slug, content, status) VALUES (1, 'Post 1', 'post-1', 'content', 'published')")
	db.Exec("INSERT INTO posts (id, title, slug, content, status) VALUES (2, 'Post 2', 'post-2', 'content', 'published')")
	db.Exec("INSERT INTO posts (id, title, slug, content, status) VALUES (3, 'Post 3', 'post-3', 'content', 'draft')")

	return db
}

func TestPostViewRepository_Record(t *testing.T) {
	db := setupPostViewTestDB(t)
	defer db.Close()

	repo := NewPostViewRepository(db)

	t.Run("records a view", func(t *testing.T) {
		recorded, err := repo.Record(1, "192.168.1.1", "Mozilla/5.0", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !recorded {
			t.Error("expected recorded to be true")
		}

		count, _ := repo.CountByPostID(1)
		if count != 1 {
			t.Errorf("expected count 1, got %d", count)
		}
	})

	t.Run("records multiple views for same post without dedup", func(t *testing.T) {
		recorded1, _ := repo.Record(2, "10.0.0.1", "Chrome", 0)
		recorded2, _ := repo.Record(2, "10.0.0.1", "Chrome", 0)
		recorded3, _ := repo.Record(2, "10.0.0.1", "Chrome", 0)

		if !recorded1 || !recorded2 || !recorded3 {
			t.Error("expected all records to succeed with dedup=0")
		}

		count, _ := repo.CountByPostID(2)
		if count != 3 {
			t.Errorf("expected count 3, got %d", count)
		}
	})

	t.Run("deduplicates same IP+UA within window", func(t *testing.T) {
		recorded1, err := repo.Record(3, "10.0.0.1", "Firefox", 30*time.Minute)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !recorded1 {
			t.Error("first record should succeed")
		}

		// Same IP+UA within window should be deduplicated
		recorded2, err := repo.Record(3, "10.0.0.1", "Firefox", 30*time.Minute)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if recorded2 {
			t.Error("duplicate within window should not be recorded")
		}

		count, _ := repo.CountByPostID(3)
		if count != 1 {
			t.Errorf("expected count 1 after dedup, got %d", count)
		}
	})

	t.Run("allows different UA from same IP within window", func(t *testing.T) {
		// Use a fresh post to avoid interference
		db.Exec("INSERT INTO posts (id, title, slug, content, status) VALUES (10, 'Post 10', 'post-10', 'content', 'published')")

		repo.Record(10, "10.0.0.1", "Chrome", 30*time.Minute)
		recorded, _ := repo.Record(10, "10.0.0.1", "Firefox", 30*time.Minute)

		if !recorded {
			t.Error("different UA should be recorded even from same IP")
		}

		count, _ := repo.CountByPostID(10)
		if count != 2 {
			t.Errorf("expected count 2, got %d", count)
		}
	})

	t.Run("allows same IP+UA for different posts within window", func(t *testing.T) {
		db.Exec("INSERT INTO posts (id, title, slug, content, status) VALUES (11, 'Post 11', 'post-11', 'content', 'published')")
		db.Exec("INSERT INTO posts (id, title, slug, content, status) VALUES (12, 'Post 12', 'post-12', 'content', 'published')")

		recorded1, _ := repo.Record(11, "10.0.0.5", "Chrome", 30*time.Minute)
		recorded2, _ := repo.Record(12, "10.0.0.5", "Chrome", 30*time.Minute)

		if !recorded1 || !recorded2 {
			t.Error("same IP+UA should be allowed for different posts")
		}
	})

	t.Run("stores ip address and user agent", func(t *testing.T) {
		db.Exec("INSERT INTO posts (id, title, slug, content, status) VALUES (20, 'Post 20', 'post-20', 'content', 'published')")

		repo.Record(20, "172.16.0.1", "TestAgent/1.0", 0)

		var ip, ua string
		err := db.QueryRow("SELECT ip_address, user_agent FROM post_views WHERE post_id = 20").Scan(&ip, &ua)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ip != "172.16.0.1" {
			t.Errorf("expected IP 172.16.0.1, got %s", ip)
		}
		if ua != "TestAgent/1.0" {
			t.Errorf("expected UA TestAgent/1.0, got %s", ua)
		}
	})
}

func TestPostViewRepository_CountByPostID(t *testing.T) {
	db := setupPostViewTestDB(t)
	defer db.Close()

	repo := NewPostViewRepository(db)

	t.Run("returns zero for post with no views", func(t *testing.T) {
		count, err := repo.CountByPostID(1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0, got %d", count)
		}
	})

	t.Run("returns correct count after recording views", func(t *testing.T) {
		repo.Record(1, "10.0.0.1", "Chrome", 0)
		repo.Record(1, "10.0.0.2", "Firefox", 0)

		count, err := repo.CountByPostID(1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 2 {
			t.Errorf("expected 2, got %d", count)
		}
	})
}

func TestPostViewRepository_CountByPostIDs(t *testing.T) {
	db := setupPostViewTestDB(t)
	defer db.Close()

	repo := NewPostViewRepository(db)

	// Record views: post 1 = 3 views, post 2 = 1 view, post 3 = 0 views
	repo.Record(1, "10.0.0.1", "Chrome", 0)
	repo.Record(1, "10.0.0.2", "Firefox", 0)
	repo.Record(1, "10.0.0.3", "Safari", 0)
	repo.Record(2, "10.0.0.1", "Chrome", 0)

	t.Run("returns counts for multiple posts", func(t *testing.T) {
		counts, err := repo.CountByPostIDs([]int64{1, 2, 3})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if counts[1] != 3 {
			t.Errorf("expected post 1 count 3, got %d", counts[1])
		}
		if counts[2] != 1 {
			t.Errorf("expected post 2 count 1, got %d", counts[2])
		}
		if counts[3] != 0 {
			t.Errorf("expected post 3 count 0, got %d", counts[3])
		}
	})

	t.Run("returns empty map for empty input", func(t *testing.T) {
		counts, err := repo.CountByPostIDs([]int64{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(counts) != 0 {
			t.Errorf("expected empty map, got %v", counts)
		}
	})

	t.Run("returns counts only for requested posts", func(t *testing.T) {
		counts, err := repo.CountByPostIDs([]int64{2})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if counts[2] != 1 {
			t.Errorf("expected post 2 count 1, got %d", counts[2])
		}
		if _, exists := counts[1]; exists {
			t.Error("expected post 1 not to be in result")
		}
	})
}

func TestPostViewRepository_ScrubDeviceInfoOlderThan(t *testing.T) {
	db := setupPostViewTestDB(t)
	r := NewPostViewRepository(db)

	// One row older than the retention window, one recent.
	if _, err := db.Exec(`INSERT INTO post_views (post_id, viewed_at, ip_address, user_agent) VALUES (1, datetime('now','-2 hours'), '203.0.113.1', 'UA-old')`); err != nil {
		t.Fatalf("insert old: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO post_views (post_id, viewed_at, ip_address, user_agent) VALUES (1, datetime('now'), '203.0.113.2', 'UA-new')`); err != nil {
		t.Fatalf("insert new: %v", err)
	}

	cutoff := time.Now().UTC().Add(-1 * time.Hour)
	n, err := r.ScrubDeviceInfoOlderThan(cutoff)
	if err != nil {
		t.Fatalf("ScrubDeviceInfoOlderThan: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 row scrubbed, got %d", n)
	}

	// Cumulative PV is preserved (rows kept).
	if cnt, _ := r.CountByPostID(1); cnt != 2 {
		t.Fatalf("PV count must be preserved, got %d", cnt)
	}

	// Old row blanked; recent row keeps its device info.
	var blanked, kept int
	db.Get(&blanked, `SELECT COUNT(*) FROM post_views WHERE post_id=1 AND ip_address='' AND user_agent=''`)
	db.Get(&kept, `SELECT COUNT(*) FROM post_views WHERE post_id=1 AND ip_address='203.0.113.2' AND user_agent='UA-new'`)
	if blanked != 1 || kept != 1 {
		t.Fatalf("expected old row blanked and recent row kept; blanked=%d kept=%d", blanked, kept)
	}

	// Idempotent: a second run touches nothing (already blank / still recent).
	if n2, _ := r.ScrubDeviceInfoOlderThan(cutoff); n2 != 0 {
		t.Fatalf("expected 0 rows on second scrub, got %d", n2)
	}
}

// Pins the strict `<` boundary: a row whose viewed_at equals the cutoff is NOT
// scrubbed, while a row just before the cutoff is. Binds the same time.Time for
// insert and cutoff so the comparison is exact (avoids datetime() format skew).
func TestPostViewRepository_ScrubDeviceInfoOlderThan_CutoffBoundary(t *testing.T) {
	db := setupPostViewTestDB(t)
	r := NewPostViewRepository(db)

	cutoff := time.Now().UTC().Add(-90 * time.Minute)
	// Exactly at cutoff -> kept (not "older than").
	if _, err := db.Exec(`INSERT INTO post_views (post_id, viewed_at, ip_address, user_agent) VALUES (1, ?, '203.0.113.9', 'UA-edge')`, cutoff); err != nil {
		t.Fatalf("insert edge: %v", err)
	}
	// One second before cutoff -> scrubbed.
	if _, err := db.Exec(`INSERT INTO post_views (post_id, viewed_at, ip_address, user_agent) VALUES (1, ?, '203.0.113.8', 'UA-old')`, cutoff.Add(-time.Second)); err != nil {
		t.Fatalf("insert old: %v", err)
	}

	n, err := r.ScrubDeviceInfoOlderThan(cutoff)
	if err != nil {
		t.Fatalf("scrub: %v", err)
	}
	if n != 1 {
		t.Fatalf("only the row before cutoff should be scrubbed (strict <), got %d", n)
	}

	var keptIP string
	if err := db.Get(&keptIP, `SELECT ip_address FROM post_views WHERE ip_address != ''`); err != nil {
		t.Fatalf("select kept: %v", err)
	}
	if keptIP != "203.0.113.9" {
		t.Fatalf("the exact-cutoff row must survive, surviving ip=%q", keptIP)
	}
	var blanked int
	db.Get(&blanked, `SELECT COUNT(*) FROM post_views WHERE ip_address='' AND user_agent=''`)
	if blanked != 1 {
		t.Fatalf("expected exactly 1 blanked row, got %d", blanked)
	}
}
