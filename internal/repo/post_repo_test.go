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
	schema, err := os.ReadFile("../../migrations/001_create_posts.sql")
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

func TestPostRepository_FindAllByTag(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostRepository(db)

	// テストデータを作成
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

	t.Run("タグ'Go'で検索（全ステータス）", func(t *testing.T) {
		results, err := repo.FindAllByTag("Go", nil, 10, 0)
		if err != nil {
			t.Fatalf("failed to find posts by tag: %v", err)
		}

		// "Go"タグを含む記事は4件（"Golang"は含まない）
		if len(results) != 4 {
			t.Errorf("expected 4 posts, got %d", len(results))
		}

		// 新しい順にソートされていることを確認
		for i := 1; i < len(results); i++ {
			if results[i-1].CreatedAt.Before(results[i].CreatedAt) {
				t.Errorf("posts not sorted in descending order")
			}
		}
	})

	t.Run("タグ'Go'で検索（公開のみ）", func(t *testing.T) {
		publishedStatus := domain.PostStatusPublished
		results, err := repo.FindAllByTag("Go", &publishedStatus, 10, 0)
		if err != nil {
			t.Fatalf("failed to find posts by tag: %v", err)
		}

		// 公開済みの"Go"タグを含む記事は3件
		if len(results) != 3 {
			t.Errorf("expected 3 published posts, got %d", len(results))
		}

		// すべて公開ステータスであることを確認
		for _, post := range results {
			if post.Status != domain.PostStatusPublished {
				t.Errorf("expected published status, got %s", post.Status)
			}
		}
	})

	t.Run("タグ'Go'で検索（下書きのみ）", func(t *testing.T) {
		draftStatus := domain.PostStatusDraft
		results, err := repo.FindAllByTag("Go", &draftStatus, 10, 0)
		if err != nil {
			t.Fatalf("failed to find posts by tag: %v", err)
		}

		// 下書きの"Go"タグを含む記事は1件
		if len(results) != 1 {
			t.Errorf("expected 1 draft post, got %d", len(results))
		}

		if results[0].Status != domain.PostStatusDraft {
			t.Errorf("expected draft status, got %s", results[0].Status)
		}
	})

	t.Run("タグ'React'で検索", func(t *testing.T) {
		results, err := repo.FindAllByTag("React", nil, 10, 0)
		if err != nil {
			t.Fatalf("failed to find posts by tag: %v", err)
		}

		// "React"タグを含む記事は4件
		if len(results) != 4 {
			t.Errorf("expected 4 posts, got %d", len(results))
		}
	})

	t.Run("存在しないタグで検索", func(t *testing.T) {
		results, err := repo.FindAllByTag("NonExistent", nil, 10, 0)
		if err != nil {
			t.Fatalf("failed to find posts by tag: %v", err)
		}

		if len(results) != 0 {
			t.Errorf("expected 0 posts for non-existent tag, got %d", len(results))
		}
	})

	t.Run("部分一致しないことを確認", func(t *testing.T) {
		// "Go"で検索しても"Golang"は含まれない
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

	t.Run("LIMIT/OFFSETのテスト", func(t *testing.T) {
		// 最初の2件を取得
		results, err := repo.FindAllByTag("Go", nil, 2, 0)
		if err != nil {
			t.Fatalf("failed to find posts with limit: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("expected 2 posts with limit, got %d", len(results))
		}

		// 次の2件を取得
		nextResults, err := repo.FindAllByTag("Go", nil, 2, 2)
		if err != nil {
			t.Fatalf("failed to find posts with offset: %v", err)
		}

		if len(nextResults) != 2 {
			t.Errorf("expected 2 posts with offset, got %d", len(nextResults))
		}

		// 重複していないことを確認
		if results[0].ID == nextResults[0].ID {
			t.Errorf("offset results should be different from first page")
		}
	})
}

func TestPostRepository_GetAllTags(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostRepository(db)

	// テストデータを作成
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

	t.Run("全記事のタグをカウント", func(t *testing.T) {
		tagCounts, err := repo.GetAllTags(nil)
		if err != nil {
			t.Fatalf("failed to get all tags: %v", err)
		}

		// 期待されるタグ数
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

	t.Run("公開記事のみのタグをカウント", func(t *testing.T) {
		publishedStatus := domain.PostStatusPublished
		tagCounts, err := repo.GetAllTags(&publishedStatus)
		if err != nil {
			t.Fatalf("failed to get published tags: %v", err)
		}

		// 公開記事のみの期待されるタグ数
		expectedCounts := map[string]int{
			"Go":         2, // Post 1, 2
			"React":      2, // Post 1, 4
			"Docker":     1, // Post 1
			"JavaScript": 1, // Post 2
			"TypeScript": 1, // Post 4
			// "Python"は下書きのPost 3にのみあるので含まれない
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

		// "Python"が含まれていないことを確認
		if _, exists := tagCounts["Python"]; exists {
			t.Errorf("expected 'Python' tag to not exist in published posts")
		}
	})

	t.Run("下書き記事のみのタグをカウント", func(t *testing.T) {
		draftStatus := domain.PostStatusDraft
		tagCounts, err := repo.GetAllTags(&draftStatus)
		if err != nil {
			t.Fatalf("failed to get draft tags: %v", err)
		}

		// 下書き記事のみの期待されるタグ数
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

	t.Run("タグがない場合", func(t *testing.T) {
		// すべての記事を削除
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
