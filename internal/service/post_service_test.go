package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/capybara-translation/goblog/internal/domain"
)

// mockPostRepository はPostRepositoryのモック実装です
type mockPostRepository struct {
	findAllFunc            func(status *domain.PostStatus, limit, offset int) ([]*domain.Post, error)
	findAllByTagFunc       func(tag string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error)
	getAllTagsFunc         func(status *domain.PostStatus) (map[string]int, error)
	findBySlugFunc         func(slug string) (*domain.Post, error)
	findByIDFunc           func(id int64) (*domain.Post, error)
	createFunc             func(post *domain.Post) error
	updateFunc             func(post *domain.Post) error
	deleteFunc             func(id int64) error
	countFunc              func(status *domain.PostStatus) (int, error)
	countByTagFunc         func(tag string, status *domain.PostStatus) (int, error)
	searchFunc             func(query string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error)
	countSearchFunc        func(query string, status *domain.PostStatus) (int, error)
	findPinnedPublishedFunc func() ([]*domain.Post, error)
}

func (m *mockPostRepository) FindAll(status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
	if m.findAllFunc != nil {
		return m.findAllFunc(status, limit, offset)
	}
	return nil, nil
}

func (m *mockPostRepository) FindAllByTag(tag string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
	if m.findAllByTagFunc != nil {
		return m.findAllByTagFunc(tag, status, limit, offset)
	}
	return nil, nil
}

func (m *mockPostRepository) GetAllTags(status *domain.PostStatus) (map[string]int, error) {
	if m.getAllTagsFunc != nil {
		return m.getAllTagsFunc(status)
	}
	return nil, nil
}

func (m *mockPostRepository) FindBySlug(slug string) (*domain.Post, error) {
	if m.findBySlugFunc != nil {
		return m.findBySlugFunc(slug)
	}
	return nil, nil
}

func (m *mockPostRepository) FindByID(id int64) (*domain.Post, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(id)
	}
	return nil, nil
}

func (m *mockPostRepository) Create(post *domain.Post) error {
	if m.createFunc != nil {
		return m.createFunc(post)
	}
	return nil
}

func (m *mockPostRepository) Update(post *domain.Post) error {
	if m.updateFunc != nil {
		return m.updateFunc(post)
	}
	return nil
}

func (m *mockPostRepository) Delete(id int64) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(id)
	}
	return nil
}

func (m *mockPostRepository) Count(status *domain.PostStatus) (int, error) {
	if m.countFunc != nil {
		return m.countFunc(status)
	}
	return 0, nil
}

func (m *mockPostRepository) CountByTag(tag string, status *domain.PostStatus) (int, error) {
	if m.countByTagFunc != nil {
		return m.countByTagFunc(tag, status)
	}
	return 0, nil
}

func (m *mockPostRepository) Search(query string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
	if m.searchFunc != nil {
		return m.searchFunc(query, status, limit, offset)
	}
	return nil, nil
}

func (m *mockPostRepository) CountSearch(query string, status *domain.PostStatus) (int, error) {
	if m.countSearchFunc != nil {
		return m.countSearchFunc(query, status)
	}
	return 0, nil
}

func (m *mockPostRepository) FindPinnedPublished() ([]*domain.Post, error) {
	if m.findPinnedPublishedFunc != nil {
		return m.findPinnedPublishedFunc()
	}
	return nil, nil
}

func TestPostService_GetPublishedPosts(t *testing.T) {
	now := time.Now()
	publishedPosts := []*domain.Post{
		{
			ID:          1,
			Title:       "Published Post 1",
			Slug:        "published-1",
			Status:      domain.PostStatusPublished,
			PublishedAt: &now,
		},
		{
			ID:          2,
			Title:       "Published Post 2",
			Slug:        "published-2",
			Status:      domain.PostStatusPublished,
			PublishedAt: &now,
		},
	}

	mockRepo := &mockPostRepository{
		findAllFunc: func(status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
			// ステータスが公開済みであることを確認
			if status == nil || *status != domain.PostStatusPublished {
				t.Errorf("expected status to be published, got: %v", status)
			}
			return publishedPosts, nil
		},
	}

	service := NewPostService(mockRepo)
	posts, err := service.GetPublishedPosts(10, 0)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(posts) != 2 {
		t.Errorf("expected 2 posts, got %d", len(posts))
	}
}

func TestPostService_GetPostBySlug(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		slug           string
		repoPost       *domain.Post
		expectedResult *domain.Post
		expectNil      bool
	}{
		{
			name: "公開済み記事を取得",
			slug: "published-post",
			repoPost: &domain.Post{
				ID:          1,
				Title:       "Published Post",
				Slug:        "published-post",
				Status:      domain.PostStatusPublished,
				PublishedAt: &now,
			},
			expectedResult: &domain.Post{
				ID:          1,
				Title:       "Published Post",
				Slug:        "published-post",
				Status:      domain.PostStatusPublished,
				PublishedAt: &now,
			},
			expectNil: false,
		},
		{
			name: "下書き記事は取得できない",
			slug: "draft-post",
			repoPost: &domain.Post{
				ID:     2,
				Title:  "Draft Post",
				Slug:   "draft-post",
				Status: domain.PostStatusDraft,
			},
			expectedResult: nil,
			expectNil:      true,
		},
		{
			name:           "存在しない記事",
			slug:           "non-existent",
			repoPost:       nil,
			expectedResult: nil,
			expectNil:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockPostRepository{
				findBySlugFunc: func(slug string) (*domain.Post, error) {
					if slug != tt.slug {
						t.Errorf("expected slug %q, got %q", tt.slug, slug)
					}
					return tt.repoPost, nil
				},
			}

			service := NewPostService(mockRepo)
			post, err := service.GetPostBySlug(tt.slug)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expectNil {
				if post != nil {
					t.Errorf("expected nil, got: %v", post)
				}
			} else {
				if post == nil {
					t.Fatal("expected post, got nil")
				}
				if post.ID != tt.expectedResult.ID {
					t.Errorf("expected ID %d, got %d", tt.expectedResult.ID, post.ID)
				}
			}
		})
	}
}

func TestPostService_CreatePost(t *testing.T) {
	t.Run("正常に記事を作成", func(t *testing.T) {
		mockRepo := &mockPostRepository{
			findBySlugFunc: func(slug string) (*domain.Post, error) {
				// スラッグが存在しない
				return nil, nil
			},
			createFunc: func(post *domain.Post) error {
				// IDを設定してCreateをシミュレート
				post.ID = 1
				return nil
			},
		}

		service := NewPostService(mockRepo)
		post, err := service.CreatePost("Test Post", "test-post", "Test content", "go,test", false)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if post.ID == 0 {
			t.Error("expected ID to be set")
		}
		if post.Title != "Test Post" {
			t.Errorf("expected title %q, got %q", "Test Post", post.Title)
		}
		if post.Status != domain.PostStatusDraft {
			t.Errorf("expected status %q, got %q", domain.PostStatusDraft, post.Status)
		}
	})

	t.Run("スラッグが重複している場合はエラー", func(t *testing.T) {
		mockRepo := &mockPostRepository{
			findBySlugFunc: func(slug string) (*domain.Post, error) {
				// スラッグが既に存在
				return &domain.Post{
					ID:   99,
					Slug: slug,
				}, nil
			},
		}

		service := NewPostService(mockRepo)
		_, err := service.CreatePost("Test Post", "existing-slug", "Test content", "go", false)

		if err == nil {
			t.Fatal("expected error for duplicate slug, got nil")
		}
	})
}

func TestPostService_UpdatePost(t *testing.T) {
	t.Run("正常に記事を更新", func(t *testing.T) {
		existingPost := &domain.Post{
			ID:      1,
			Title:   "Original Title",
			Slug:    "original-slug",
			Content: "Original content",
			Status:  domain.PostStatusDraft,
		}

		mockRepo := &mockPostRepository{
			findByIDFunc: func(id int64) (*domain.Post, error) {
				if id == 1 {
					return existingPost, nil
				}
				return nil, nil
			},
			findBySlugFunc: func(slug string) (*domain.Post, error) {
				// 新しいスラッグは使用されていない
				return nil, nil
			},
			updateFunc: func(post *domain.Post) error {
				return nil
			},
		}

		service := NewPostService(mockRepo)
		updated, err := service.UpdatePost(1, "Updated Title", "updated-slug", "Updated content", "go", false)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if updated.Title != "Updated Title" {
			t.Errorf("expected title %q, got %q", "Updated Title", updated.Title)
		}
		if updated.Slug != "updated-slug" {
			t.Errorf("expected slug %q, got %q", "updated-slug", updated.Slug)
		}
	})

	t.Run("同じスラッグのまま更新（自分自身なのでOK）", func(t *testing.T) {
		existingPost := &domain.Post{
			ID:      1,
			Title:   "Original Title",
			Slug:    "same-slug",
			Content: "Original content",
			Status:  domain.PostStatusDraft,
		}

		mockRepo := &mockPostRepository{
			findByIDFunc: func(id int64) (*domain.Post, error) {
				return existingPost, nil
			},
			findBySlugFunc: func(slug string) (*domain.Post, error) {
				// スラッグは変わっていないのでチェックされない
				return nil, nil
			},
			updateFunc: func(post *domain.Post) error {
				return nil
			},
		}

		service := NewPostService(mockRepo)
		_, err := service.UpdatePost(1, "Updated Title", "same-slug", "Updated content", "go", false)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("新しいスラッグが既に使用されている場合はエラー", func(t *testing.T) {
		existingPost := &domain.Post{
			ID:   1,
			Slug: "original-slug",
		}

		mockRepo := &mockPostRepository{
			findByIDFunc: func(id int64) (*domain.Post, error) {
				return existingPost, nil
			},
			findBySlugFunc: func(slug string) (*domain.Post, error) {
				// 新しいスラッグが既に別の記事で使用されている
				return &domain.Post{
					ID:   2,
					Slug: slug,
				}, nil
			},
		}

		service := NewPostService(mockRepo)
		_, err := service.UpdatePost(1, "Title", "existing-slug", "Content", "go", false)

		if err == nil {
			t.Fatal("expected error for duplicate slug, got nil")
		}
	})

	t.Run("存在しない記事を更新しようとするとエラー", func(t *testing.T) {
		mockRepo := &mockPostRepository{
			findByIDFunc: func(id int64) (*domain.Post, error) {
				return nil, nil
			},
		}

		service := NewPostService(mockRepo)
		_, err := service.UpdatePost(999, "Title", "slug", "Content", "go", false)

		if err == nil {
			t.Fatal("expected error for non-existent post, got nil")
		}
	})
}

func TestPostService_PublishPost(t *testing.T) {
	t.Run("記事を公開する", func(t *testing.T) {
		draftPost := &domain.Post{
			ID:      1,
			Title:   "Draft Post",
			Slug:    "draft-post",
			Status:  domain.PostStatusDraft,
			Content: "Content",
		}

		var updatedPost *domain.Post

		mockRepo := &mockPostRepository{
			findByIDFunc: func(id int64) (*domain.Post, error) {
				return draftPost, nil
			},
			updateFunc: func(post *domain.Post) error {
				updatedPost = post
				return nil
			},
		}

		service := NewPostService(mockRepo)
		published, err := service.PublishPost(1)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if published.Status != domain.PostStatusPublished {
			t.Errorf("expected status %q, got %q", domain.PostStatusPublished, published.Status)
		}

		if published.PublishedAt == nil {
			t.Error("expected published_at to be set, got nil")
		}

		if updatedPost == nil {
			t.Fatal("expected update to be called")
		}
	})

	t.Run("存在しない記事を公開しようとするとエラー", func(t *testing.T) {
		mockRepo := &mockPostRepository{
			findByIDFunc: func(id int64) (*domain.Post, error) {
				return nil, nil
			},
		}

		service := NewPostService(mockRepo)
		_, err := service.PublishPost(999)

		if err == nil {
			t.Fatal("expected error for non-existent post, got nil")
		}
	})
}

func TestPostService_UnpublishPost(t *testing.T) {
	t.Run("記事を下書きに戻す", func(t *testing.T) {
		now := time.Now()
		publishedPost := &domain.Post{
			ID:          1,
			Title:       "Published Post",
			Slug:        "published-post",
			Status:      domain.PostStatusPublished,
			PublishedAt: &now,
		}

		var updatedPost *domain.Post

		mockRepo := &mockPostRepository{
			findByIDFunc: func(id int64) (*domain.Post, error) {
				return publishedPost, nil
			},
			updateFunc: func(post *domain.Post) error {
				updatedPost = post
				return nil
			},
		}

		service := NewPostService(mockRepo)
		unpublished, err := service.UnpublishPost(1)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if unpublished.Status != domain.PostStatusDraft {
			t.Errorf("expected status %q, got %q", domain.PostStatusDraft, unpublished.Status)
		}

		if updatedPost == nil {
			t.Fatal("expected update to be called")
		}
	})
}

func TestPostService_DeletePost(t *testing.T) {
	t.Run("記事を削除する", func(t *testing.T) {
		var deletedID int64

		mockRepo := &mockPostRepository{
			deleteFunc: func(id int64) error {
				deletedID = id
				return nil
			},
		}

		service := NewPostService(mockRepo)
		err := service.DeletePost(123)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if deletedID != 123 {
			t.Errorf("expected to delete ID 123, got %d", deletedID)
		}
	})

	t.Run("削除でエラーが発生した場合", func(t *testing.T) {
		mockRepo := &mockPostRepository{
			deleteFunc: func(id int64) error {
				return fmt.Errorf("database error")
			},
		}

		service := NewPostService(mockRepo)
		err := service.DeletePost(123)

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestPostService_GetPublishedPostsByTag(t *testing.T) {
	now := time.Now()

	t.Run("公開済み記事をタグで取得", func(t *testing.T) {
		expectedPosts := []*domain.Post{
			{
				ID:          1,
				Title:       "Go Post",
				Slug:        "go-post",
				Tags:        "Go,Programming",
				Status:      domain.PostStatusPublished,
				PublishedAt: &now,
			},
		}

		mockRepo := &mockPostRepository{
			findAllByTagFunc: func(tag string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
				if tag != "Go" {
					t.Errorf("expected tag to be 'Go', got: %s", tag)
				}
				if status == nil || *status != domain.PostStatusPublished {
					t.Errorf("expected status to be published, got: %v", status)
				}
				return expectedPosts, nil
			},
		}

		service := NewPostService(mockRepo)
		posts, err := service.GetPublishedPostsByTag("Go", 10, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(posts) != 1 {
			t.Errorf("expected 1 post, got %d", len(posts))
		}

		if posts[0].Title != "Go Post" {
			t.Errorf("expected title 'Go Post', got %s", posts[0].Title)
		}
	})

	t.Run("空のタグでエラー", func(t *testing.T) {
		mockRepo := &mockPostRepository{}
		service := NewPostService(mockRepo)

		_, err := service.GetPublishedPostsByTag("", 10, 0)

		if err == nil {
			t.Fatal("expected error for empty tag, got nil")
		}

		if err.Error() != "tag cannot be empty" {
			t.Errorf("expected 'tag cannot be empty' error, got: %v", err)
		}
	})

	t.Run("リポジトリエラー", func(t *testing.T) {
		mockRepo := &mockPostRepository{
			findAllByTagFunc: func(tag string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
				return nil, fmt.Errorf("database error")
			},
		}

		service := NewPostService(mockRepo)
		_, err := service.GetPublishedPostsByTag("Go", 10, 0)

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestPostService_GetAllPostsByTag(t *testing.T) {
	now := time.Now()

	t.Run("全ステータスの記事をタグで取得", func(t *testing.T) {
		expectedPosts := []*domain.Post{
			{
				ID:          1,
				Title:       "Published Go Post",
				Tags:        "Go",
				Status:      domain.PostStatusPublished,
				PublishedAt: &now,
			},
			{
				ID:     2,
				Title:  "Draft Go Post",
				Tags:   "Go",
				Status: domain.PostStatusDraft,
			},
		}

		mockRepo := &mockPostRepository{
			findAllByTagFunc: func(tag string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
				if tag != "Go" {
					t.Errorf("expected tag to be 'Go', got: %s", tag)
				}
				if status != nil {
					t.Errorf("expected status to be nil, got: %v", status)
				}
				return expectedPosts, nil
			},
		}

		service := NewPostService(mockRepo)
		posts, err := service.GetAllPostsByTag("Go", nil, 10, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(posts) != 2 {
			t.Errorf("expected 2 posts, got %d", len(posts))
		}
	})

	t.Run("特定ステータスの記事をタグで取得", func(t *testing.T) {
		draftStatus := domain.PostStatusDraft
		expectedPosts := []*domain.Post{
			{
				ID:     1,
				Title:  "Draft Go Post",
				Tags:   "Go",
				Status: domain.PostStatusDraft,
			},
		}

		mockRepo := &mockPostRepository{
			findAllByTagFunc: func(tag string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
				if status == nil || *status != domain.PostStatusDraft {
					t.Errorf("expected status to be draft, got: %v", status)
				}
				return expectedPosts, nil
			},
		}

		service := NewPostService(mockRepo)
		posts, err := service.GetAllPostsByTag("Go", &draftStatus, 10, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(posts) != 1 {
			t.Errorf("expected 1 post, got %d", len(posts))
		}
	})

	t.Run("空のタグでエラー", func(t *testing.T) {
		mockRepo := &mockPostRepository{}
		service := NewPostService(mockRepo)

		_, err := service.GetAllPostsByTag("", nil, 10, 0)

		if err == nil {
			t.Fatal("expected error for empty tag, got nil")
		}
	})
}

func TestPostService_GetPublishedTags(t *testing.T) {
	t.Run("公開記事のタグを取得", func(t *testing.T) {
		expectedTags := map[string]int{
			"Go":     5,
			"React":  3,
			"Docker": 2,
		}

		mockRepo := &mockPostRepository{
			getAllTagsFunc: func(status *domain.PostStatus) (map[string]int, error) {
				if status == nil || *status != domain.PostStatusPublished {
					t.Errorf("expected status to be published, got: %v", status)
				}
				return expectedTags, nil
			},
		}

		service := NewPostService(mockRepo)
		tags, err := service.GetPublishedTags()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(tags) != 3 {
			t.Errorf("expected 3 tags, got %d", len(tags))
		}

		if tags["Go"] != 5 {
			t.Errorf("expected Go count to be 5, got %d", tags["Go"])
		}

		if tags["React"] != 3 {
			t.Errorf("expected React count to be 3, got %d", tags["React"])
		}

		if tags["Docker"] != 2 {
			t.Errorf("expected Docker count to be 2, got %d", tags["Docker"])
		}
	})

	t.Run("リポジトリエラー", func(t *testing.T) {
		mockRepo := &mockPostRepository{
			getAllTagsFunc: func(status *domain.PostStatus) (map[string]int, error) {
				return nil, fmt.Errorf("database error")
			},
		}

		service := NewPostService(mockRepo)
		_, err := service.GetPublishedTags()

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("タグが存在しない場合", func(t *testing.T) {
		mockRepo := &mockPostRepository{
			getAllTagsFunc: func(status *domain.PostStatus) (map[string]int, error) {
				return map[string]int{}, nil
			},
		}

		service := NewPostService(mockRepo)
		tags, err := service.GetPublishedTags()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(tags) != 0 {
			t.Errorf("expected 0 tags, got %d", len(tags))
		}
	})
}

func TestPostService_GetAllTags(t *testing.T) {
	t.Run("全記事のタグを取得（ステータス指定なし）", func(t *testing.T) {
		expectedTags := map[string]int{
			"Go":     8,
			"React":  5,
			"Python": 3,
		}

		mockRepo := &mockPostRepository{
			getAllTagsFunc: func(status *domain.PostStatus) (map[string]int, error) {
				if status != nil {
					t.Errorf("expected status to be nil, got: %v", status)
				}
				return expectedTags, nil
			},
		}

		service := NewPostService(mockRepo)
		tags, err := service.GetAllTags(nil)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(tags) != 3 {
			t.Errorf("expected 3 tags, got %d", len(tags))
		}

		if tags["Go"] != 8 {
			t.Errorf("expected Go count to be 8, got %d", tags["Go"])
		}
	})

	t.Run("特定ステータスのタグを取得", func(t *testing.T) {
		draftStatus := domain.PostStatusDraft
		expectedTags := map[string]int{
			"Go":     3,
			"Python": 2,
		}

		mockRepo := &mockPostRepository{
			getAllTagsFunc: func(status *domain.PostStatus) (map[string]int, error) {
				if status == nil || *status != domain.PostStatusDraft {
					t.Errorf("expected status to be draft, got: %v", status)
				}
				return expectedTags, nil
			},
		}

		service := NewPostService(mockRepo)
		tags, err := service.GetAllTags(&draftStatus)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(tags) != 2 {
			t.Errorf("expected 2 tags, got %d", len(tags))
		}
	})

	t.Run("リポジトリエラー", func(t *testing.T) {
		mockRepo := &mockPostRepository{
			getAllTagsFunc: func(status *domain.PostStatus) (map[string]int, error) {
				return nil, fmt.Errorf("database error")
			},
		}

		service := NewPostService(mockRepo)
		_, err := service.GetAllTags(nil)

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestPostService_SearchPosts(t *testing.T) {
	now := time.Now()

	t.Run("検索クエリで記事を取得", func(t *testing.T) {
		expectedPosts := []*domain.Post{
			{
				ID:      1,
				Title:   "Go入門",
				Content: "Goの基本的な使い方",
				Status:  domain.PostStatusPublished,
			},
		}

		mockRepo := &mockPostRepository{
			searchFunc: func(query string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
				if query != "Go" {
					t.Errorf("expected query 'Go', got %s", query)
				}
				if limit != 10 || offset != 0 {
					t.Errorf("expected limit=10, offset=0, got limit=%d, offset=%d", limit, offset)
				}
				return expectedPosts, nil
			},
		}

		service := NewPostService(mockRepo)
		posts, err := service.SearchPosts("Go", nil, 10, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(posts) != 1 {
			t.Errorf("expected 1 post, got %d", len(posts))
		}

		if posts[0].Title != "Go入門" {
			t.Errorf("expected title 'Go入門', got %s", posts[0].Title)
		}
	})

	t.Run("ステータス指定付きで検索", func(t *testing.T) {
		draftStatus := domain.PostStatusDraft
		expectedPosts := []*domain.Post{
			{
				ID:     1,
				Title:  "Draft Go Post",
				Status: domain.PostStatusDraft,
			},
		}

		mockRepo := &mockPostRepository{
			searchFunc: func(query string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
				if status == nil || *status != domain.PostStatusDraft {
					t.Errorf("expected status draft, got %v", status)
				}
				return expectedPosts, nil
			},
		}

		service := NewPostService(mockRepo)
		posts, err := service.SearchPosts("Go", &draftStatus, 10, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(posts) != 1 {
			t.Errorf("expected 1 post, got %d", len(posts))
		}
	})

	t.Run("空のクエリは全件取得にフォールバック", func(t *testing.T) {
		expectedPosts := []*domain.Post{
			{
				ID:          1,
				Title:       "Post 1",
				Status:      domain.PostStatusPublished,
				PublishedAt: &now,
			},
			{
				ID:     2,
				Title:  "Post 2",
				Status: domain.PostStatusDraft,
			},
		}

		mockRepo := &mockPostRepository{
			findAllFunc: func(status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
				return expectedPosts, nil
			},
			searchFunc: func(query string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
				t.Error("search should not be called for empty query")
				return nil, nil
			},
		}

		service := NewPostService(mockRepo)
		posts, err := service.SearchPosts("", nil, 10, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(posts) != 2 {
			t.Errorf("expected 2 posts, got %d", len(posts))
		}
	})

	t.Run("リポジトリエラー", func(t *testing.T) {
		mockRepo := &mockPostRepository{
			searchFunc: func(query string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
				return nil, fmt.Errorf("database error")
			},
		}

		service := NewPostService(mockRepo)
		_, err := service.SearchPosts("Go", nil, 10, 0)

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestPostService_CountSearchPosts(t *testing.T) {
	t.Run("検索結果の件数を取得", func(t *testing.T) {
		mockRepo := &mockPostRepository{
			countSearchFunc: func(query string, status *domain.PostStatus) (int, error) {
				if query != "Go" {
					t.Errorf("expected query 'Go', got %s", query)
				}
				return 5, nil
			},
		}

		service := NewPostService(mockRepo)
		count, err := service.CountSearchPosts("Go", nil)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if count != 5 {
			t.Errorf("expected count 5, got %d", count)
		}
	})

	t.Run("ステータス指定付きで件数取得", func(t *testing.T) {
		draftStatus := domain.PostStatusDraft
		mockRepo := &mockPostRepository{
			countSearchFunc: func(query string, status *domain.PostStatus) (int, error) {
				if status == nil || *status != domain.PostStatusDraft {
					t.Errorf("expected status draft, got %v", status)
				}
				return 3, nil
			},
		}

		service := NewPostService(mockRepo)
		count, err := service.CountSearchPosts("Go", &draftStatus)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if count != 3 {
			t.Errorf("expected count 3, got %d", count)
		}
	})

	t.Run("空のクエリは全件カウントにフォールバック", func(t *testing.T) {
		mockRepo := &mockPostRepository{
			countFunc: func(status *domain.PostStatus) (int, error) {
				return 10, nil
			},
			countSearchFunc: func(query string, status *domain.PostStatus) (int, error) {
				t.Error("countSearch should not be called for empty query")
				return 0, nil
			},
		}

		service := NewPostService(mockRepo)
		count, err := service.CountSearchPosts("", nil)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if count != 10 {
			t.Errorf("expected count 10, got %d", count)
		}
	})

	t.Run("リポジトリエラー", func(t *testing.T) {
		mockRepo := &mockPostRepository{
			countSearchFunc: func(query string, status *domain.PostStatus) (int, error) {
				return 0, fmt.Errorf("database error")
			},
		}

		service := NewPostService(mockRepo)
		_, err := service.CountSearchPosts("Go", nil)

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestPostService_SearchPublishedPosts(t *testing.T) {
	now := time.Now()

	t.Run("公開済み記事を検索", func(t *testing.T) {
		expectedPosts := []*domain.Post{
			{
				ID:          1,
				Title:       "Go入門ガイド",
				Status:      domain.PostStatusPublished,
				PublishedAt: &now,
			},
		}

		mockRepo := &mockPostRepository{
			searchFunc: func(query string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
				if query != "Go" {
					t.Errorf("expected query 'Go', got %s", query)
				}
				if status == nil || *status != domain.PostStatusPublished {
					t.Errorf("expected status published, got %v", status)
				}
				return expectedPosts, nil
			},
		}

		service := NewPostService(mockRepo)
		posts, err := service.SearchPublishedPosts("Go", 10, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(posts) != 1 {
			t.Errorf("expected 1 post, got %d", len(posts))
		}
	})

	t.Run("空のクエリは公開済み全件取得にフォールバック", func(t *testing.T) {
		expectedPosts := []*domain.Post{
			{
				ID:          1,
				Title:       "Published Post",
				Status:      domain.PostStatusPublished,
				PublishedAt: &now,
			},
		}

		mockRepo := &mockPostRepository{
			findAllFunc: func(status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
				if status == nil || *status != domain.PostStatusPublished {
					t.Errorf("expected status published, got %v", status)
				}
				return expectedPosts, nil
			},
			searchFunc: func(query string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
				t.Error("search should not be called for empty query")
				return nil, nil
			},
		}

		service := NewPostService(mockRepo)
		posts, err := service.SearchPublishedPosts("", 10, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(posts) != 1 {
			t.Errorf("expected 1 post, got %d", len(posts))
		}
	})

	t.Run("リポジトリエラー", func(t *testing.T) {
		mockRepo := &mockPostRepository{
			searchFunc: func(query string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
				return nil, fmt.Errorf("database error")
			},
		}

		service := NewPostService(mockRepo)
		_, err := service.SearchPublishedPosts("Go", 10, 0)

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestNormalizeTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "スペースなしのタグ",
			input:    "Go,React,TypeScript",
			expected: "Go,React,TypeScript",
		},
		{
			name:     "スペース付きのタグ（カンマ後）",
			input:    "Go, React, TypeScript",
			expected: "Go,React,TypeScript",
		},
		{
			name:     "スペース付きのタグ（カンマ前後）",
			input:    "Go , React , TypeScript",
			expected: "Go,React,TypeScript",
		},
		{
			name:     "複数スペース",
			input:    "Go,   React,    TypeScript",
			expected: "Go,React,TypeScript",
		},
		{
			name:     "空のタグを除外",
			input:    "Go,,React,  ,TypeScript",
			expected: "Go,React,TypeScript",
		},
		{
			name:     "単一タグ",
			input:    "Go",
			expected: "Go",
		},
		{
			name:     "単一タグ（前後スペース）",
			input:    "  Go  ",
			expected: "Go",
		},
		{
			name:     "空文字列",
			input:    "",
			expected: "",
		},
		{
			name:     "スペースのみ",
			input:    "   ",
			expected: "",
		},
		{
			name:     "カンマのみ",
			input:    ",,,",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeTags(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeTags(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestPostService_CreatePost_NormalizesTags(t *testing.T) {
	t.Run("タグが正規化されて保存される", func(t *testing.T) {
		var savedPost *domain.Post

		mockRepo := &mockPostRepository{
			findBySlugFunc: func(slug string) (*domain.Post, error) {
				return nil, nil
			},
			createFunc: func(post *domain.Post) error {
				savedPost = post
				post.ID = 1
				return nil
			},
		}

		service := NewPostService(mockRepo)
		post, err := service.CreatePost("Test Post", "test-post", "Content", "Go, React, TypeScript", false)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// 返却されたpostのタグが正規化されていることを確認
		if post.Tags != "Go,React,TypeScript" {
			t.Errorf("expected tags %q, got %q", "Go,React,TypeScript", post.Tags)
		}

		// リポジトリに保存されたタグも正規化されていることを確認
		if savedPost.Tags != "Go,React,TypeScript" {
			t.Errorf("expected saved tags %q, got %q", "Go,React,TypeScript", savedPost.Tags)
		}
	})
}

func TestPostService_UpdatePost_NormalizesTags(t *testing.T) {
	t.Run("タグが正規化されて更新される", func(t *testing.T) {
		existingPost := &domain.Post{
			ID:      1,
			Title:   "Original Title",
			Slug:    "original-slug",
			Content: "Original content",
			Tags:    "OldTag",
			Status:  domain.PostStatusDraft,
		}

		var updatedPost *domain.Post

		mockRepo := &mockPostRepository{
			findByIDFunc: func(id int64) (*domain.Post, error) {
				return existingPost, nil
			},
			findBySlugFunc: func(slug string) (*domain.Post, error) {
				return nil, nil
			},
			updateFunc: func(post *domain.Post) error {
				updatedPost = post
				return nil
			},
		}

		service := NewPostService(mockRepo)
		post, err := service.UpdatePost(1, "Updated Title", "updated-slug", "Content", "Go, React, TypeScript", false)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// 返却されたpostのタグが正規化されていることを確認
		if post.Tags != "Go,React,TypeScript" {
			t.Errorf("expected tags %q, got %q", "Go,React,TypeScript", post.Tags)
		}

		// リポジトリに保存されたタグも正規化されていることを確認
		if updatedPost.Tags != "Go,React,TypeScript" {
			t.Errorf("expected saved tags %q, got %q", "Go,React,TypeScript", updatedPost.Tags)
		}
	})
}

func TestPostService_CountSearchPublishedPosts(t *testing.T) {
	t.Run("公開済み記事の検索件数を取得", func(t *testing.T) {
		mockRepo := &mockPostRepository{
			countSearchFunc: func(query string, status *domain.PostStatus) (int, error) {
				if query != "Go" {
					t.Errorf("expected query 'Go', got %s", query)
				}
				if status == nil || *status != domain.PostStatusPublished {
					t.Errorf("expected status published, got %v", status)
				}
				return 3, nil
			},
		}

		service := NewPostService(mockRepo)
		count, err := service.CountSearchPublishedPosts("Go")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if count != 3 {
			t.Errorf("expected count 3, got %d", count)
		}
	})

	t.Run("空のクエリは公開済み全件カウントにフォールバック", func(t *testing.T) {
		mockRepo := &mockPostRepository{
			countFunc: func(status *domain.PostStatus) (int, error) {
				if status == nil || *status != domain.PostStatusPublished {
					t.Errorf("expected status published, got %v", status)
				}
				return 8, nil
			},
			countSearchFunc: func(query string, status *domain.PostStatus) (int, error) {
				t.Error("countSearch should not be called for empty query")
				return 0, nil
			},
		}

		service := NewPostService(mockRepo)
		count, err := service.CountSearchPublishedPosts("")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if count != 8 {
			t.Errorf("expected count 8, got %d", count)
		}
	})

	t.Run("リポジトリエラー", func(t *testing.T) {
		mockRepo := &mockPostRepository{
			countSearchFunc: func(query string, status *domain.PostStatus) (int, error) {
				return 0, fmt.Errorf("database error")
			},
		}

		service := NewPostService(mockRepo)
		_, err := service.CountSearchPublishedPosts("Go")

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestPostService_GetPinnedPosts(t *testing.T) {
	now := time.Now()

	t.Run("ピン留めされた公開記事を取得", func(t *testing.T) {
		expectedPosts := []*domain.Post{
			{
				ID:          1,
				Title:       "Pinned Post 1",
				Slug:        "pinned-1",
				Status:      domain.PostStatusPublished,
				IsPinned:    true,
				PublishedAt: &now,
			},
			{
				ID:          2,
				Title:       "Pinned Post 2",
				Slug:        "pinned-2",
				Status:      domain.PostStatusPublished,
				IsPinned:    true,
				PublishedAt: &now,
			},
		}

		mockRepo := &mockPostRepository{
			findPinnedPublishedFunc: func() ([]*domain.Post, error) {
				return expectedPosts, nil
			},
		}

		service := NewPostService(mockRepo)
		posts, err := service.GetPinnedPosts()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(posts) != 2 {
			t.Errorf("expected 2 posts, got %d", len(posts))
		}

		for _, post := range posts {
			if !post.IsPinned {
				t.Errorf("expected post to be pinned, got is_pinned=false")
			}
		}
	})

	t.Run("ピン留め記事がない場合", func(t *testing.T) {
		mockRepo := &mockPostRepository{
			findPinnedPublishedFunc: func() ([]*domain.Post, error) {
				return []*domain.Post{}, nil
			},
		}

		service := NewPostService(mockRepo)
		posts, err := service.GetPinnedPosts()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(posts) != 0 {
			t.Errorf("expected 0 posts, got %d", len(posts))
		}
	})

	t.Run("リポジトリエラー", func(t *testing.T) {
		mockRepo := &mockPostRepository{
			findPinnedPublishedFunc: func() ([]*domain.Post, error) {
				return nil, fmt.Errorf("database error")
			},
		}

		service := NewPostService(mockRepo)
		_, err := service.GetPinnedPosts()

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestPostService_SetPinned(t *testing.T) {
	t.Run("記事をピン留めする", func(t *testing.T) {
		existingPost := &domain.Post{
			ID:       1,
			Title:    "Test Post",
			Slug:     "test-post",
			Status:   domain.PostStatusPublished,
			IsPinned: false,
		}

		var updatedPost *domain.Post

		mockRepo := &mockPostRepository{
			findByIDFunc: func(id int64) (*domain.Post, error) {
				if id == 1 {
					return existingPost, nil
				}
				return nil, nil
			},
			updateFunc: func(post *domain.Post) error {
				updatedPost = post
				return nil
			},
		}

		service := NewPostService(mockRepo)
		post, err := service.SetPinned(1, true)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !post.IsPinned {
			t.Errorf("expected post to be pinned, got is_pinned=false")
		}

		if updatedPost == nil {
			t.Fatal("expected update to be called")
		}

		if !updatedPost.IsPinned {
			t.Errorf("expected updated post to be pinned")
		}
	})

	t.Run("記事のピン留めを解除する", func(t *testing.T) {
		existingPost := &domain.Post{
			ID:       1,
			Title:    "Test Post",
			Slug:     "test-post",
			Status:   domain.PostStatusPublished,
			IsPinned: true,
		}

		var updatedPost *domain.Post

		mockRepo := &mockPostRepository{
			findByIDFunc: func(id int64) (*domain.Post, error) {
				return existingPost, nil
			},
			updateFunc: func(post *domain.Post) error {
				updatedPost = post
				return nil
			},
		}

		service := NewPostService(mockRepo)
		post, err := service.SetPinned(1, false)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if post.IsPinned {
			t.Errorf("expected post to be unpinned, got is_pinned=true")
		}

		if updatedPost == nil {
			t.Fatal("expected update to be called")
		}

		if updatedPost.IsPinned {
			t.Errorf("expected updated post to be unpinned")
		}
	})

	t.Run("存在しない記事をピン留めしようとするとエラー", func(t *testing.T) {
		mockRepo := &mockPostRepository{
			findByIDFunc: func(id int64) (*domain.Post, error) {
				return nil, nil
			},
		}

		service := NewPostService(mockRepo)
		_, err := service.SetPinned(999, true)

		if err == nil {
			t.Fatal("expected error for non-existent post, got nil")
		}
	})

	t.Run("リポジトリのFindByIDでエラー", func(t *testing.T) {
		mockRepo := &mockPostRepository{
			findByIDFunc: func(id int64) (*domain.Post, error) {
				return nil, fmt.Errorf("database error")
			},
		}

		service := NewPostService(mockRepo)
		_, err := service.SetPinned(1, true)

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("リポジトリのUpdateでエラー", func(t *testing.T) {
		existingPost := &domain.Post{
			ID:       1,
			Title:    "Test Post",
			IsPinned: false,
		}

		mockRepo := &mockPostRepository{
			findByIDFunc: func(id int64) (*domain.Post, error) {
				return existingPost, nil
			},
			updateFunc: func(post *domain.Post) error {
				return fmt.Errorf("database error")
			},
		}

		service := NewPostService(mockRepo)
		_, err := service.SetPinned(1, true)

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
