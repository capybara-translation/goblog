package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/capybara-translation/goblog/internal/domain"
)

// mockPostRepository はPostRepositoryのモック実装です
type mockPostRepository struct {
	findAllFunc    func(status *domain.PostStatus, limit, offset int) ([]*domain.Post, error)
	findBySlugFunc func(slug string) (*domain.Post, error)
	findByIDFunc   func(id int64) (*domain.Post, error)
	createFunc     func(post *domain.Post) error
	updateFunc     func(post *domain.Post) error
	deleteFunc     func(id int64) error
}

func (m *mockPostRepository) FindAll(status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
	if m.findAllFunc != nil {
		return m.findAllFunc(status, limit, offset)
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
		post, err := service.CreatePost("Test Post", "test-post", "Test content", "go,test")

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
		_, err := service.CreatePost("Test Post", "existing-slug", "Test content", "go")

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
		updated, err := service.UpdatePost(1, "Updated Title", "updated-slug", "Updated content", "go")

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
		_, err := service.UpdatePost(1, "Updated Title", "same-slug", "Updated content", "go")

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
		_, err := service.UpdatePost(1, "Title", "existing-slug", "Content", "go")

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
		_, err := service.UpdatePost(999, "Title", "slug", "Content", "go")

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
