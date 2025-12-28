package repo

import (
	"os"
	"testing"
	"time"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// setupTestDB はテスト用のインメモリSQLiteデータベースをセットアップします
func setupTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	// インメモリSQLiteを開く
	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	// マイグレーションファイルを読み込んで実行
	schema, err := os.ReadFile("../../migrations/001_init.sql")
	if err != nil {
		t.Fatalf("failed to read migration file: %v", err)
	}

	if _, err := db.Exec(string(schema)); err != nil {
		t.Fatalf("failed to create schema: %v", err)
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

	// IDが設定されているか確認
	if post.ID == 0 {
		t.Error("expected post ID to be set, got 0")
	}

	// データベースから取得して確認
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

	// テストデータを作成
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

	// スラッグで取得
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

	// 存在しないスラッグで取得
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

	// テストデータを作成
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

	// IDで取得
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

	// 存在しないIDで取得
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

	// 複数のテストデータを作成
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

	// すべての記事を取得
	allPosts, err := repo.FindAll(nil, 10, 0)
	if err != nil {
		t.Fatalf("failed to find all posts: %v", err)
	}

	if len(allPosts) != 3 {
		t.Errorf("expected 3 posts, got %d", len(allPosts))
	}

	// 公開済み記事のみを取得
	publishedStatus := domain.PostStatusPublished
	publishedPosts, err := repo.FindAll(&publishedStatus, 10, 0)
	if err != nil {
		t.Fatalf("failed to find published posts: %v", err)
	}

	if len(publishedPosts) != 2 {
		t.Errorf("expected 2 published posts, got %d", len(publishedPosts))
	}

	// 下書き記事のみを取得
	draftStatus := domain.PostStatusDraft
	draftPosts, err := repo.FindAll(&draftStatus, 10, 0)
	if err != nil {
		t.Fatalf("failed to find draft posts: %v", err)
	}

	if len(draftPosts) != 1 {
		t.Errorf("expected 1 draft post, got %d", len(draftPosts))
	}

	// LIMIT/OFFSETのテスト
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

	// テストデータを作成
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

	// 記事を更新
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

	// 更新された内容を確認
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

	// テストデータを作成
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

	// 削除されたことを確認
	deleted, err := repo.FindByID(post.ID)
	if err != nil {
		t.Fatalf("unexpected error when finding deleted post: %v", err)
	}
	if deleted != nil {
		t.Errorf("expected post to be deleted, but found: %v", deleted)
	}
}
