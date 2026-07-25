package repo

import (
	"os"
	"testing"
	"time"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// setupTestDB sets up an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	// Open in-memory SQLite
	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	// Read and execute migration files
	migrations := []string{
		"../../migrations/001_create_posts.sql",
		"../../migrations/003_add_is_pinned.sql",
		"../../migrations/012_add_post_health_date.sql",
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

	return db
}

func TestPostRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostRepository(db)

	now := time.Now()
	post := &domain.Post{
		Title:     "Test Post",
		Slug:      "test-post",
		Content:   "This is a test post.",
		Status:    domain.PostStatusDraft,
		Tags:      "go,test",
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := repo.Create(post)
	if err != nil {
		t.Fatalf("failed to create post: %v", err)
	}

	// Verify ID is set
	if post.ID == 0 {
		t.Error("expected post ID to be set, got 0")
	}

	// Retrieve from database and verify
	retrieved, err := repo.FindByID(post.ID)
	if err != nil {
		t.Fatalf("failed to retrieve post: %v", err)
	}

	if retrieved.Title != post.Title {
		t.Errorf("expected title %q, got %q", post.Title, retrieved.Title)
	}
	if retrieved.Slug != post.Slug {
		t.Errorf("expected slug %q, got %q", post.Slug, retrieved.Slug)
	}
	if retrieved.Content != post.Content {
		t.Errorf("expected content %q, got %q", post.Content, retrieved.Content)
	}
	if retrieved.Status != post.Status {
		t.Errorf("expected status %q, got %q", post.Status, retrieved.Status)
	}
}

func TestPostRepository_FindBySlug(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostRepository(db)

	// Create test data
	now := time.Now()
	post := &domain.Post{
		Title:     "Find By Slug Test",
		Slug:      "find-by-slug",
		Content:   "Test content",
		Status:    domain.PostStatusPublished,
		Tags:      "test",
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := repo.Create(post)
	if err != nil {
		t.Fatalf("failed to create post: %v", err)
	}

	// Retrieve by slug
	found, err := repo.FindBySlug("find-by-slug")
	if err != nil {
		t.Fatalf("failed to find post by slug: %v", err)
	}

	if found == nil {
		t.Fatal("expected to find post, got nil")
	}

	if found.ID != post.ID {
		t.Errorf("expected ID %d, got %d", post.ID, found.ID)
	}
	if found.Title != post.Title {
		t.Errorf("expected title %q, got %q", post.Title, found.Title)
	}

	// Retrieve with non-existent slug
	notFound, err := repo.FindBySlug("non-existent")
	if err != nil {
		t.Fatalf("expected no error for non-existent slug, got: %v", err)
	}
	if notFound != nil {
		t.Errorf("expected nil for non-existent slug, got: %v", notFound)
	}
}

func TestPostRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostRepository(db)

	// Create test data
	now := time.Now()
	post := &domain.Post{
		Title:     "Find By ID Test",
		Slug:      "find-by-id",
		Content:   "Test content",
		Status:    domain.PostStatusDraft,
		Tags:      "test",
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := repo.Create(post)
	if err != nil {
		t.Fatalf("failed to create post: %v", err)
	}

	// Retrieve by ID
	found, err := repo.FindByID(post.ID)
	if err != nil {
		t.Fatalf("failed to find post by ID: %v", err)
	}

	if found == nil {
		t.Fatal("expected to find post, got nil")
	}

	if found.Title != post.Title {
		t.Errorf("expected title %q, got %q", post.Title, found.Title)
	}

	// Retrieve with non-existent ID
	notFound, err := repo.FindByID(99999)
	if err != nil {
		t.Fatalf("expected no error for non-existent ID, got: %v", err)
	}
	if notFound != nil {
		t.Errorf("expected nil for non-existent ID, got: %v", notFound)
	}
}

func TestPostRepository_FindAll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostRepository(db)

	// Create multiple test data entries
	now := time.Now()
	posts := []*domain.Post{
		{
			Title:     "Published Post 1",
			Slug:      "published-1",
			Content:   "Content 1",
			Status:    domain.PostStatusPublished,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			Title:     "Published Post 2",
			Slug:      "published-2",
			Content:   "Content 2",
			Status:    domain.PostStatusPublished,
			CreatedAt: now.Add(1 * time.Hour),
			UpdatedAt: now.Add(1 * time.Hour),
		},
		{
			Title:     "Draft Post",
			Slug:      "draft-1",
			Content:   "Draft content",
			Status:    domain.PostStatusDraft,
			CreatedAt: now.Add(2 * time.Hour),
			UpdatedAt: now.Add(2 * time.Hour),
		},
	}

	for _, p := range posts {
		if err := repo.Create(p); err != nil {
			t.Fatalf("failed to create post: %v", err)
		}
	}

	// Retrieve all posts
	allPosts, err := repo.FindAll(nil, 10, 0)
	if err != nil {
		t.Fatalf("failed to find all posts: %v", err)
	}

	if len(allPosts) != 3 {
		t.Errorf("expected 3 posts, got %d", len(allPosts))
	}

	// Retrieve only published posts
	publishedStatus := domain.PostStatusPublished
	publishedPosts, err := repo.FindAll(&publishedStatus, 10, 0)
	if err != nil {
		t.Fatalf("failed to find published posts: %v", err)
	}

	if len(publishedPosts) != 2 {
		t.Errorf("expected 2 published posts, got %d", len(publishedPosts))
	}

	// Retrieve only draft posts
	draftStatus := domain.PostStatusDraft
	draftPosts, err := repo.FindAll(&draftStatus, 10, 0)
	if err != nil {
		t.Fatalf("failed to find draft posts: %v", err)
	}

	if len(draftPosts) != 1 {
		t.Errorf("expected 1 draft post, got %d", len(draftPosts))
	}

	// Test LIMIT/OFFSET
	limitedPosts, err := repo.FindAll(nil, 2, 0)
	if err != nil {
		t.Fatalf("failed to find limited posts: %v", err)
	}

	if len(limitedPosts) != 2 {
		t.Errorf("expected 2 posts with limit, got %d", len(limitedPosts))
	}
}

func TestPostRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostRepository(db)

	// Create test data
	now := time.Now()
	post := &domain.Post{
		Title:     "Original Title",
		Slug:      "original-slug",
		Content:   "Original content",
		Status:    domain.PostStatusDraft,
		Tags:      "original",
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := repo.Create(post)
	if err != nil {
		t.Fatalf("failed to create post: %v", err)
	}

	// Update post
	post.Title = "Updated Title"
	post.Content = "Updated content"
	post.Status = domain.PostStatusPublished
	publishedAt := time.Now()
	post.PublishedAt = &publishedAt
	post.UpdatedAt = time.Now()

	err = repo.Update(post)
	if err != nil {
		t.Fatalf("failed to update post: %v", err)
	}

	// Verify updated content
	updated, err := repo.FindByID(post.ID)
	if err != nil {
		t.Fatalf("failed to retrieve updated post: %v", err)
	}

	if updated.Title != "Updated Title" {
		t.Errorf("expected title %q, got %q", "Updated Title", updated.Title)
	}
	if updated.Content != "Updated content" {
		t.Errorf("expected content %q, got %q", "Updated content", updated.Content)
	}
	if updated.Status != domain.PostStatusPublished {
		t.Errorf("expected status %q, got %q", domain.PostStatusPublished, updated.Status)
	}
	if updated.PublishedAt == nil {
		t.Error("expected published_at to be set, got nil")
	}
}

func TestPostRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostRepository(db)

	// Create test data
	now := time.Now()
	post := &domain.Post{
		Title:     "To Be Deleted",
		Slug:      "to-be-deleted",
		Content:   "This will be deleted",
		Status:    domain.PostStatusDraft,
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := repo.Create(post)
	if err != nil {
		t.Fatalf("failed to create post: %v", err)
	}

	// 削除
	err = repo.Delete(post.ID)
	if err != nil {
		t.Fatalf("failed to delete post: %v", err)
	}

	// Verify deletion
	deleted, err := repo.FindByID(post.ID)
	if err != nil {
		t.Fatalf("unexpected error when finding deleted post: %v", err)
	}
	if deleted != nil {
		t.Errorf("expected post to be deleted, but found: %v", deleted)
	}
}

func TestPostRepository_FindAllByTag(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostRepository(db)

	// Create test data
	now := time.Now()
	posts := []*domain.Post{
		{
			Title:     "Go Only",
			Slug:      "go-only",
			Content:   "Content 1",
			Status:    domain.PostStatusPublished,
			Tags:      "Go",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			Title:     "Go at Start",
			Slug:      "go-at-start",
			Content:   "Content 2",
			Status:    domain.PostStatusPublished,
			Tags:      "Go,React,Docker",
			CreatedAt: now.Add(1 * time.Hour),
			UpdatedAt: now.Add(1 * time.Hour),
		},
		{
			Title:     "Go in Middle",
			Slug:      "go-in-middle",
			Content:   "Content 3",
			Status:    domain.PostStatusPublished,
			Tags:      "React,Go,Docker",
			CreatedAt: now.Add(2 * time.Hour),
			UpdatedAt: now.Add(2 * time.Hour),
		},
		{
			Title:     "Go at End",
			Slug:      "go-at-end",
			Content:   "Content 4",
			Status:    domain.PostStatusDraft,
			Tags:      "React,Docker,Go",
			CreatedAt: now.Add(3 * time.Hour),
			UpdatedAt: now.Add(3 * time.Hour),
		},
		{
			Title:     "Golang (should not match)",
			Slug:      "golang",
			Content:   "Content 5",
			Status:    domain.PostStatusPublished,
			Tags:      "Golang,Programming",
			CreatedAt: now.Add(4 * time.Hour),
			UpdatedAt: now.Add(4 * time.Hour),
		},
		{
			Title:     "React Only",
			Slug:      "react-only",
			Content:   "Content 6",
			Status:    domain.PostStatusPublished,
			Tags:      "React,JavaScript",
			CreatedAt: now.Add(5 * time.Hour),
			UpdatedAt: now.Add(5 * time.Hour),
		},
	}

	for _, p := range posts {
		if err := repo.Create(p); err != nil {
			t.Fatalf("failed to create post: %v", err)
		}
	}

	t.Run("Search by tag 'Go' (all statuses)", func(t *testing.T) {
		results, err := repo.FindAllByTag("Go", nil, 10, 0)
		if err != nil {
			t.Fatalf("failed to find posts by tag: %v", err)
		}

		// 4 posts contain "Go" tag (excluding "Golang")
		if len(results) != 4 {
			t.Errorf("expected 4 posts, got %d", len(results))
		}

		// Verify sorted in descending order (newest first)
		for i := 1; i < len(results); i++ {
			if results[i-1].CreatedAt.Before(results[i].CreatedAt) {
				t.Errorf("posts not sorted in descending order")
			}
		}
	})

	t.Run("Search by tag 'Go' (published only)", func(t *testing.T) {
		publishedStatus := domain.PostStatusPublished
		results, err := repo.FindAllByTag("Go", &publishedStatus, 10, 0)
		if err != nil {
			t.Fatalf("failed to find posts by tag: %v", err)
		}

		// 3 published posts contain "Go" tag
		if len(results) != 3 {
			t.Errorf("expected 3 published posts, got %d", len(results))
		}

		// Verify all have published status
		for _, post := range results {
			if post.Status != domain.PostStatusPublished {
				t.Errorf("expected published status, got %s", post.Status)
			}
		}
	})

	t.Run("Search by tag 'Go' (drafts only)", func(t *testing.T) {
		draftStatus := domain.PostStatusDraft
		results, err := repo.FindAllByTag("Go", &draftStatus, 10, 0)
		if err != nil {
			t.Fatalf("failed to find posts by tag: %v", err)
		}

		// 1 draft post contains "Go" tag
		if len(results) != 1 {
			t.Errorf("expected 1 draft post, got %d", len(results))
		}

		if results[0].Status != domain.PostStatusDraft {
			t.Errorf("expected draft status, got %s", results[0].Status)
		}
	})

	t.Run("Search by tag 'React'", func(t *testing.T) {
		results, err := repo.FindAllByTag("React", nil, 10, 0)
		if err != nil {
			t.Fatalf("failed to find posts by tag: %v", err)
		}

		// 4 posts contain "React" tag
		if len(results) != 4 {
			t.Errorf("expected 4 posts, got %d", len(results))
		}
	})

	t.Run("Search by non-existent tag", func(t *testing.T) {
		results, err := repo.FindAllByTag("NonExistent", nil, 10, 0)
		if err != nil {
			t.Fatalf("failed to find posts by tag: %v", err)
		}

		if len(results) != 0 {
			t.Errorf("expected 0 posts for non-existent tag, got %d", len(results))
		}
	})

	t.Run("Verify no partial matching", func(t *testing.T) {
		// Searching for "Go" should not include "Golang"
		results, err := repo.FindAllByTag("Go", nil, 10, 0)
		if err != nil {
			t.Fatalf("failed to find posts by tag: %v", err)
		}

		for _, post := range results {
			if post.Slug == "golang" {
				t.Errorf("'Golang' should not match 'Go' tag search")
			}
		}
	})

	t.Run("Test LIMIT/OFFSET", func(t *testing.T) {
		// Retrieve first 2 posts
		results, err := repo.FindAllByTag("Go", nil, 2, 0)
		if err != nil {
			t.Fatalf("failed to find posts with limit: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("expected 2 posts with limit, got %d", len(results))
		}

		// Retrieve next 2 posts
		nextResults, err := repo.FindAllByTag("Go", nil, 2, 2)
		if err != nil {
			t.Fatalf("failed to find posts with offset: %v", err)
		}

		if len(nextResults) != 2 {
			t.Errorf("expected 2 posts with offset, got %d", len(nextResults))
		}

		// Verify no duplicates
		if results[0].ID == nextResults[0].ID {
			t.Errorf("offset results should be different from first page")
		}
	})
}

func TestPostRepository_GetAllTags(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostRepository(db)

	// Create test data
	now := time.Now()
	posts := []*domain.Post{
		{
			Title:     "Post 1",
			Slug:      "post-1",
			Content:   "Content 1",
			Status:    domain.PostStatusPublished,
			Tags:      "Go,React,Docker",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			Title:     "Post 2",
			Slug:      "post-2",
			Content:   "Content 2",
			Status:    domain.PostStatusPublished,
			Tags:      "Go,JavaScript",
			CreatedAt: now.Add(1 * time.Hour),
			UpdatedAt: now.Add(1 * time.Hour),
		},
		{
			Title:     "Post 3",
			Slug:      "post-3",
			Content:   "Content 3",
			Status:    domain.PostStatusDraft,
			Tags:      "Go,Python",
			CreatedAt: now.Add(2 * time.Hour),
			UpdatedAt: now.Add(2 * time.Hour),
		},
		{
			Title:     "Post 4",
			Slug:      "post-4",
			Content:   "Content 4",
			Status:    domain.PostStatusPublished,
			Tags:      "React,TypeScript",
			CreatedAt: now.Add(3 * time.Hour),
			UpdatedAt: now.Add(3 * time.Hour),
		},
		{
			Title:     "Post 5 (No Tags)",
			Slug:      "post-5",
			Content:   "Content 5",
			Status:    domain.PostStatusPublished,
			Tags:      "",
			CreatedAt: now.Add(4 * time.Hour),
			UpdatedAt: now.Add(4 * time.Hour),
		},
	}

	for _, p := range posts {
		if err := repo.Create(p); err != nil {
			t.Fatalf("failed to create post: %v", err)
		}
	}

	t.Run("Count tags from all posts", func(t *testing.T) {
		tagCounts, err := repo.GetAllTags(nil)
		if err != nil {
			t.Fatalf("failed to get all tags: %v", err)
		}

		// Expected tag counts
		expectedCounts := map[string]int{
			"Go":         3, // Post 1, 2, 3
			"React":      2, // Post 1, 4
			"Docker":     1, // Post 1
			"JavaScript": 1, // Post 2
			"Python":     1, // Post 3
			"TypeScript": 1, // Post 4
		}

		if len(tagCounts) != len(expectedCounts) {
			t.Errorf("expected %d tags, got %d", len(expectedCounts), len(tagCounts))
		}

		for tag, expectedCount := range expectedCounts {
			if count, exists := tagCounts[tag]; !exists {
				t.Errorf("expected tag %q to exist", tag)
			} else if count != expectedCount {
				t.Errorf("expected tag %q to have count %d, got %d", tag, expectedCount, count)
			}
		}
	})

	t.Run("Count tags from published posts only", func(t *testing.T) {
		publishedStatus := domain.PostStatusPublished
		tagCounts, err := repo.GetAllTags(&publishedStatus)
		if err != nil {
			t.Fatalf("failed to get published tags: %v", err)
		}

		// Expected tag counts from published posts only
		expectedCounts := map[string]int{
			"Go":         2, // Post 1, 2
			"React":      2, // Post 1, 4
			"Docker":     1, // Post 1
			"JavaScript": 1, // Post 2
			"TypeScript": 1, // Post 4
			// "Python" is only in draft Post 3, so not included
		}

		if len(tagCounts) != len(expectedCounts) {
			t.Errorf("expected %d tags, got %d", len(expectedCounts), len(tagCounts))
		}

		for tag, expectedCount := range expectedCounts {
			if count, exists := tagCounts[tag]; !exists {
				t.Errorf("expected tag %q to exist", tag)
			} else if count != expectedCount {
				t.Errorf("expected tag %q to have count %d, got %d", tag, expectedCount, count)
			}
		}

		// Verify "Python" is not included
		if _, exists := tagCounts["Python"]; exists {
			t.Errorf("expected 'Python' tag to not exist in published posts")
		}
	})

	t.Run("Count tags from draft posts only", func(t *testing.T) {
		draftStatus := domain.PostStatusDraft
		tagCounts, err := repo.GetAllTags(&draftStatus)
		if err != nil {
			t.Fatalf("failed to get draft tags: %v", err)
		}

		// Expected tag counts from draft posts only
		expectedCounts := map[string]int{
			"Go":     1, // Post 3
			"Python": 1, // Post 3
		}

		if len(tagCounts) != len(expectedCounts) {
			t.Errorf("expected %d tags, got %d", len(expectedCounts), len(tagCounts))
		}

		for tag, expectedCount := range expectedCounts {
			if count, exists := tagCounts[tag]; !exists {
				t.Errorf("expected tag %q to exist", tag)
			} else if count != expectedCount {
				t.Errorf("expected tag %q to have count %d, got %d", tag, expectedCount, count)
			}
		}
	})

	t.Run("When there are no tags", func(t *testing.T) {
		// Delete all posts
		allPosts, _ := repo.FindAll(nil, 100, 0)
		for _, p := range allPosts {
			repo.Delete(p.ID)
		}

		tagCounts, err := repo.GetAllTags(nil)
		if err != nil {
			t.Fatalf("failed to get tags: %v", err)
		}

		if len(tagCounts) != 0 {
			t.Errorf("expected 0 tags for empty database, got %d", len(tagCounts))
		}
	})
}

func TestPostRepository_Search(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostRepository(db)
	now := time.Now()

	// Create test data
	posts := []*domain.Post{
		{
			Title:       "Go言語入門",
			Slug:        "go-intro",
			Content:     "Goプログラミングの基礎を学びます。",
			Status:      domain.PostStatusPublished,
			Tags:        "Go,プログラミング",
			CreatedAt:   now,
			UpdatedAt:   now,
			PublishedAt: &now,
		},
		{
			Title:       "React入門",
			Slug:        "react-intro",
			Content:     "Reactでフロントエンド開発を始めよう。",
			Status:      domain.PostStatusPublished,
			Tags:        "React,JavaScript",
			CreatedAt:   now.Add(-1 * time.Hour),
			UpdatedAt:   now.Add(-1 * time.Hour),
			PublishedAt: &now,
		},
		{
			Title:     "データベース設計",
			Slug:      "db-design",
			Content:   "SQLiteとGoを使ったデータベース設計。",
			Status:    domain.PostStatusDraft,
			Tags:      "Go,Database",
			CreatedAt: now.Add(-2 * time.Hour),
			UpdatedAt: now.Add(-2 * time.Hour),
		},
	}

	for _, p := range posts {
		if err := repo.Create(p); err != nil {
			t.Fatalf("failed to create test post: %v", err)
		}
	}

	t.Run("Search by title", func(t *testing.T) {
		results, err := repo.Search("入門", nil, 10, 0)
		if err != nil {
			t.Fatalf("failed to search: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("expected 2 posts, got %d", len(results))
		}
	})

	t.Run("Search by content", func(t *testing.T) {
		results, err := repo.Search("フロントエンド", nil, 10, 0)
		if err != nil {
			t.Fatalf("failed to search: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("expected 1 post, got %d", len(results))
		}
		if len(results) > 0 && results[0].Title != "React入門" {
			t.Errorf("expected 'React入門', got %q", results[0].Title)
		}
	})

	t.Run("Case insensitive search", func(t *testing.T) {
		results, err := repo.Search("GO", nil, 10, 0)
		if err != nil {
			t.Fatalf("failed to search: %v", err)
		}

		// "Go Introduction" and "Database Design" (contains Go in content)
		if len(results) != 2 {
			t.Errorf("expected 2 posts, got %d", len(results))
		}
	})

	t.Run("Filter by status", func(t *testing.T) {
		publishedStatus := domain.PostStatusPublished
		results, err := repo.Search("入門", &publishedStatus, 10, 0)
		if err != nil {
			t.Fatalf("failed to search: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("expected 2 published posts, got %d", len(results))
		}
	})

	t.Run("Search drafts only", func(t *testing.T) {
		draftStatus := domain.PostStatusDraft
		results, err := repo.Search("Go", &draftStatus, 10, 0)
		if err != nil {
			t.Fatalf("failed to search: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("expected 1 draft post, got %d", len(results))
		}
	})

	t.Run("Zero search results", func(t *testing.T) {
		results, err := repo.Search("存在しない", nil, 10, 0)
		if err != nil {
			t.Fatalf("failed to search: %v", err)
		}

		if len(results) != 0 {
			t.Errorf("expected 0 posts, got %d", len(results))
		}
	})

	t.Run("Pagination", func(t *testing.T) {
		results, err := repo.Search("入門", nil, 1, 0)
		if err != nil {
			t.Fatalf("failed to search: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("expected 1 post with limit=1, got %d", len(results))
		}

		results2, err := repo.Search("入門", nil, 1, 1)
		if err != nil {
			t.Fatalf("failed to search: %v", err)
		}

		if len(results2) != 1 {
			t.Errorf("expected 1 post with offset=1, got %d", len(results2))
		}
	})
}

func TestPostRepository_CountSearch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostRepository(db)
	now := time.Now()

	// Create test data
	posts := []*domain.Post{
		{
			Title:       "Go言語入門",
			Slug:        "go-intro",
			Content:     "Goプログラミングの基礎。",
			Status:      domain.PostStatusPublished,
			CreatedAt:   now,
			UpdatedAt:   now,
			PublishedAt: &now,
		},
		{
			Title:       "React入門",
			Slug:        "react-intro",
			Content:     "Reactでフロントエンド開発。",
			Status:      domain.PostStatusPublished,
			CreatedAt:   now,
			UpdatedAt:   now,
			PublishedAt: &now,
		},
		{
			Title:     "Go応用",
			Slug:      "go-advanced",
			Content:   "Goの高度なテクニック。",
			Status:    domain.PostStatusDraft,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	for _, p := range posts {
		if err := repo.Create(p); err != nil {
			t.Fatalf("failed to create test post: %v", err)
		}
	}

	t.Run("Count all results", func(t *testing.T) {
		count, err := repo.CountSearch("入門", nil)
		if err != nil {
			t.Fatalf("failed to count: %v", err)
		}

		if count != 2 {
			t.Errorf("expected count 2, got %d", count)
		}
	})

	t.Run("Filter by status", func(t *testing.T) {
		publishedStatus := domain.PostStatusPublished
		count, err := repo.CountSearch("Go", &publishedStatus)
		if err != nil {
			t.Fatalf("failed to count: %v", err)
		}

		if count != 1 {
			t.Errorf("expected count 1 (published Go posts), got %d", count)
		}
	})

	t.Run("Zero search results", func(t *testing.T) {
		count, err := repo.CountSearch("存在しない", nil)
		if err != nil {
			t.Fatalf("failed to count: %v", err)
		}

		if count != 0 {
			t.Errorf("expected count 0, got %d", count)
		}
	})
}

func TestPostRepository_FindPinnedPublished(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostRepository(db)
	now := time.Now()

	// Create test data
	posts := []*domain.Post{
		{
			Title:       "ピン留め＆公開",
			Slug:        "pinned-published",
			Content:     "ピン留めされた公開記事",
			Status:      domain.PostStatusPublished,
			IsPinned:    true,
			CreatedAt:   now,
			UpdatedAt:   now,
			PublishedAt: &now,
		},
		{
			Title:     "ピン留め＆下書き",
			Slug:      "pinned-draft",
			Content:   "ピン留めされた下書き記事",
			Status:    domain.PostStatusDraft,
			IsPinned:  true,
			CreatedAt: now.Add(1 * time.Hour),
			UpdatedAt: now.Add(1 * time.Hour),
		},
		{
			Title:       "通常の公開記事",
			Slug:        "normal-published",
			Content:     "普通の公開記事",
			Status:      domain.PostStatusPublished,
			IsPinned:    false,
			CreatedAt:   now.Add(2 * time.Hour),
			UpdatedAt:   now.Add(2 * time.Hour),
			PublishedAt: &now,
		},
		{
			Title:       "もう一つのピン留め＆公開",
			Slug:        "pinned-published-2",
			Content:     "2つ目のピン留め公開記事",
			Status:      domain.PostStatusPublished,
			IsPinned:    true,
			CreatedAt:   now.Add(3 * time.Hour),
			UpdatedAt:   now.Add(3 * time.Hour),
			PublishedAt: &now,
		},
	}

	for _, p := range posts {
		if err := repo.Create(p); err != nil {
			t.Fatalf("failed to create post: %v", err)
		}
	}

	t.Run("Retrieve only pinned published posts", func(t *testing.T) {
		results, err := repo.FindPinnedPublished()
		if err != nil {
			t.Fatalf("failed to find pinned published posts: %v", err)
		}

		// 2 posts are pinned and published
		if len(results) != 2 {
			t.Errorf("expected 2 pinned published posts, got %d", len(results))
		}

		// Verify all are pinned and published
		for _, post := range results {
			if !post.IsPinned {
				t.Errorf("expected post to be pinned, got is_pinned=false for %q", post.Title)
			}
			if post.Status != domain.PostStatusPublished {
				t.Errorf("expected published status, got %s for %q", post.Status, post.Title)
			}
		}
	})

	t.Run("Sorted by title", func(t *testing.T) {
		results, err := repo.FindPinnedPublished()
		if err != nil {
			t.Fatalf("failed to find pinned published posts: %v", err)
		}

		// Verify sorted in ascending order by title
		for i := 1; i < len(results); i++ {
			if results[i-1].Title > results[i].Title {
				t.Errorf("posts not sorted by title: %q > %q", results[i-1].Title, results[i].Title)
			}
		}
	})

	t.Run("When there are no pinned posts", func(t *testing.T) {
		// Delete all posts
		allPosts, _ := repo.FindAll(nil, 100, 0)
		for _, p := range allPosts {
			repo.Delete(p.ID)
		}

		results, err := repo.FindPinnedPublished()
		if err != nil {
			t.Fatalf("failed to find pinned published posts: %v", err)
		}

		if len(results) != 0 {
			t.Errorf("expected 0 pinned posts for empty database, got %d", len(results))
		}
	})
}

func TestPostRepository_FindPublishedBySlugs(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostRepository(db)
	now := time.Now()

	pub1 := &domain.Post{
		Title:       "Published One",
		Slug:        "pub-one",
		Content:     "c",
		Status:      domain.PostStatusPublished,
		CreatedAt:   now,
		UpdatedAt:   now,
		PublishedAt: &now,
	}
	pub2 := &domain.Post{
		Title:       "Published Two",
		Slug:        "pub-two",
		Content:     "c",
		Status:      domain.PostStatusPublished,
		CreatedAt:   now.Add(time.Hour),
		UpdatedAt:   now.Add(time.Hour),
		PublishedAt: &now,
	}
	draft := &domain.Post{
		Title:     "Draft Post",
		Slug:      "draft-one",
		Content:   "c",
		Status:    domain.PostStatusDraft,
		CreatedAt: now,
		UpdatedAt: now,
	}
	for _, p := range []*domain.Post{pub1, pub2, draft} {
		if err := repo.Create(p); err != nil {
			t.Fatalf("failed to create post: %v", err)
		}
	}

	t.Run("returns matching published posts", func(t *testing.T) {
		results, err := repo.FindPublishedBySlugs([]string{"pub-one", "pub-two"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 posts, got %d", len(results))
		}
		slugs := map[string]bool{}
		for _, p := range results {
			slugs[p.Slug] = true
		}
		if !slugs["pub-one"] || !slugs["pub-two"] {
			t.Errorf("missing expected slugs, got %v", slugs)
		}
	})

	t.Run("excludes draft slug", func(t *testing.T) {
		results, err := repo.FindPublishedBySlugs([]string{"pub-one", "draft-one"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result (draft excluded), got %d", len(results))
		}
		if results[0].Slug != "pub-one" {
			t.Errorf("expected slug pub-one, got %s", results[0].Slug)
		}
	})

	t.Run("ignores nonexistent slug", func(t *testing.T) {
		results, err := repo.FindPublishedBySlugs([]string{"pub-one", "does-not-exist"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
	})

	t.Run("empty input returns empty slice", func(t *testing.T) {
		results, err := repo.FindPublishedBySlugs([]string{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 0 {
			t.Fatalf("expected empty slice, got %d", len(results))
		}
	})
}

func TestPostRepository_HealthDate_RoundTrip(t *testing.T) {
	db := setupTestDB(t)
	r := NewPostRepository(db)

	hd := "2026-07-20"
	post := &domain.Post{Title: "t", Slug: "hd-roundtrip", Content: "c", Status: domain.PostStatusDraft, HealthDate: &hd}
	if err := r.Create(post); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := r.FindByID(post.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.HealthDate == nil || *got.HealthDate != "2026-07-20" {
		t.Errorf("HealthDate = %v, want 2026-07-20", got.HealthDate)
	}

	// Update で解除（nil に戻す）
	got.HealthDate = nil
	if err := r.Update(got); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got2, err := r.FindByID(post.ID)
	if err != nil {
		t.Fatalf("FindByID after update: %v", err)
	}
	if got2.HealthDate != nil {
		t.Errorf("HealthDate = %v, want nil after clearing", got2.HealthDate)
	}
}
