package http

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/capybara-translation/goblog/internal/service"
)

// mockPostService は PostService のモック実装です
type mockPostService struct {
	getPublishedPostsFunc func(limit, offset int) ([]*domain.Post, error)
	getPostBySlugFunc     func(slug string) (*domain.Post, error)
	getAllPostsFunc       func(status *domain.PostStatus, limit, offset int) ([]*domain.Post, error)
}

func (m *mockPostService) GetPublishedPosts(limit, offset int) ([]*domain.Post, error) {
	if m.getPublishedPostsFunc != nil {
		return m.getPublishedPostsFunc(limit, offset)
	}
	return []*domain.Post{}, nil
}

func (m *mockPostService) GetPostBySlug(slug string) (*domain.Post, error) {
	if m.getPostBySlugFunc != nil {
		return m.getPostBySlugFunc(slug)
	}
	return nil, nil
}

func (m *mockPostService) GetAllPosts(status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
	if m.getAllPostsFunc != nil {
		return m.getAllPostsFunc(status, limit, offset)
	}
	return []*domain.Post{}, nil
}

func (m *mockPostService) GetPostByID(id int64) (*domain.Post, error) {
	return nil, nil
}

func (m *mockPostService) CreatePost(title, slug, content, tags string) (*domain.Post, error) {
	return nil, nil
}

func (m *mockPostService) UpdatePost(id int64, title, slug, content, tags string) (*domain.Post, error) {
	return nil, nil
}

func (m *mockPostService) PublishPost(id int64) (*domain.Post, error) {
	return nil, nil
}

func (m *mockPostService) UnpublishPost(id int64) (*domain.Post, error) {
	return nil, nil
}

func (m *mockPostService) DeletePost(id int64) error {
	return nil
}

var _ service.PostService = (*mockPostService)(nil)

func TestHandleHome(t *testing.T) {
	tests := []struct {
		name           string
		mockFunc       func(limit, offset int) ([]*domain.Post, error)
		expectedStatus int
		containsText   []string
		notContains    []string
	}{
		{
			name: "記事がない場合",
			mockFunc: func(limit, offset int) ([]*domain.Post, error) {
				return []*domain.Post{}, nil
			},
			expectedStatus: http.StatusOK,
			containsText: []string{
				"<!DOCTYPE html>",
				"ようこそgoblogへ",
				"まだ記事がありません",
			},
		},
		{
			name: "記事がある場合",
			mockFunc: func(limit, offset int) ([]*domain.Post, error) {
				publishedAt := time.Now()
				return []*domain.Post{
					{
						ID:          1,
						Title:       "テスト記事1",
						Slug:        "test-post-1",
						Content:     "これはテスト記事です",
						Status:      domain.PostStatusPublished,
						Tags:        "Go,テスト",
						PublishedAt: &publishedAt,
					},
					{
						ID:          2,
						Title:       "テスト記事2",
						Slug:        "test-post-2",
						Content:     "2つ目の記事",
						Status:      domain.PostStatusPublished,
						PublishedAt: &publishedAt,
					},
				}, nil
			},
			expectedStatus: http.StatusOK,
			containsText: []string{
				"<!DOCTYPE html>",
				"ようこそgoblogへ",
				"テスト記事1",
				"テスト記事2",
				"/posts/test-post-1",
				"/posts/test-post-2",
				"Go,テスト",
			},
			notContains: []string{
				"まだ記事がありません",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockPostService{
				getPublishedPostsFunc: tt.mockFunc,
			}
			router := NewRouterWithTemplates(mockService, nil, false, "goblog", "../view/templates/*.html")

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			contentType := w.Header().Get("Content-Type")
			expected := "text/html; charset=utf-8"
			if contentType != expected {
				t.Errorf("expected Content-Type %q, got %q", expected, contentType)
			}

			body := w.Body.String()
			for _, text := range tt.containsText {
				if !strings.Contains(body, text) {
					t.Errorf("expected body to contain %q", text)
				}
			}

			for _, text := range tt.notContains {
				if strings.Contains(body, text) {
					t.Errorf("expected body NOT to contain %q", text)
				}
			}
		})
	}
}

func TestHandlePosts(t *testing.T) {
	tests := []struct {
		name           string
		mockFunc       func(limit, offset int) ([]*domain.Post, error)
		expectedStatus int
		containsText   []string
		notContains    []string
	}{
		{
			name: "記事がない場合",
			mockFunc: func(limit, offset int) ([]*domain.Post, error) {
				return []*domain.Post{}, nil
			},
			expectedStatus: http.StatusOK,
			containsText: []string{
				"<!DOCTYPE html>",
				"記事一覧",
				"まだ記事がありません",
			},
		},
		{
			name: "記事がある場合",
			mockFunc: func(limit, offset int) ([]*domain.Post, error) {
				publishedAt := time.Now()
				return []*domain.Post{
					{
						ID:          1,
						Title:       "公開記事1",
						Slug:        "published-post-1",
						Content:     "これは公開されている記事です。長い内容でも最初の部分だけ表示されます。" + strings.Repeat("テスト", 100),
						Status:      domain.PostStatusPublished,
						Tags:        "タグ1,タグ2",
						PublishedAt: &publishedAt,
					},
					{
						ID:          2,
						Title:       "公開記事2",
						Slug:        "published-post-2",
						Content:     "2つ目の公開記事",
						Status:      domain.PostStatusPublished,
						PublishedAt: &publishedAt,
					},
				}, nil
			},
			expectedStatus: http.StatusOK,
			containsText: []string{
				"<!DOCTYPE html>",
				"記事一覧",
				"公開記事1",
				"公開記事2",
				"/posts/published-post-1",
				"/posts/published-post-2",
				"タグ1,タグ2",
				"続きを読む",
			},
			notContains: []string{
				"まだ記事がありません",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockPostService{
				getPublishedPostsFunc: tt.mockFunc,
			}
			router := NewRouterWithTemplates(mockService, nil, false, "goblog", "../view/templates/*.html")

			req := httptest.NewRequest(http.MethodGet, "/posts", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			contentType := w.Header().Get("Content-Type")
			expected := "text/html; charset=utf-8"
			if contentType != expected {
				t.Errorf("expected Content-Type %q, got %q", expected, contentType)
			}

			body := w.Body.String()
			for _, text := range tt.containsText {
				if !strings.Contains(body, text) {
					t.Errorf("expected body to contain %q", text)
				}
			}

			for _, text := range tt.notContains {
				if strings.Contains(body, text) {
					t.Errorf("expected body NOT to contain %q", text)
				}
			}
		})
	}
}

func TestHandlePosts_Pagination(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		mockFunc       func(limit, offset int) ([]*domain.Post, error)
		expectedStatus int
		containsText   []string
		notContains    []string
	}{
		{
			name: "1ページ目（次のページあり）",
			url:  "/posts?page=1",
			mockFunc: func(limit, offset int) ([]*domain.Post, error) {
				if limit != 21 || offset != 0 {
					t.Errorf("expected limit=21, offset=0, got limit=%d, offset=%d", limit, offset)
				}
				// 21件返して次のページがあることを示す
				posts := make([]*domain.Post, 21)
				publishedAt := time.Now()
				for i := 0; i < 21; i++ {
					posts[i] = &domain.Post{
						ID:          int64(i + 1),
						Title:       "記事" + strconv.Itoa(i+1),
						Slug:        "post-" + strconv.Itoa(i+1),
						Content:     "内容" + strconv.Itoa(i+1),
						Status:      domain.PostStatusPublished,
						PublishedAt: &publishedAt,
					}
				}
				return posts, nil
			},
			expectedStatus: http.StatusOK,
			containsText: []string{
				"ページ 1",
				"次のページ",
				"/posts?page=2",
			},
			notContains: []string{
				"/posts?page=0",
			},
		},
		{
			name: "2ページ目（前後のページあり）",
			url:  "/posts?page=2",
			mockFunc: func(limit, offset int) ([]*domain.Post, error) {
				if limit != 21 || offset != 20 {
					t.Errorf("expected limit=21, offset=20, got limit=%d, offset=%d", limit, offset)
				}
				// 21件返して次のページがあることを示す
				posts := make([]*domain.Post, 21)
				publishedAt := time.Now()
				for i := 0; i < 21; i++ {
					posts[i] = &domain.Post{
						ID:          int64(i + 21),
						Title:       "記事" + strconv.Itoa(i+21),
						Slug:        "post-" + strconv.Itoa(i+21),
						Content:     "内容" + strconv.Itoa(i+21),
						Status:      domain.PostStatusPublished,
						PublishedAt: &publishedAt,
					}
				}
				return posts, nil
			},
			expectedStatus: http.StatusOK,
			containsText: []string{
				"ページ 2",
				"前のページ",
				"/posts?page=1",
				"次のページ",
				"/posts?page=3",
			},
		},
		{
			name: "最後のページ（次のページなし）",
			url:  "/posts?page=3",
			mockFunc: func(limit, offset int) ([]*domain.Post, error) {
				if limit != 21 || offset != 40 {
					t.Errorf("expected limit=21, offset=40, got limit=%d, offset=%d", limit, offset)
				}
				// 10件だけ返して次のページがないことを示す
				posts := make([]*domain.Post, 10)
				publishedAt := time.Now()
				for i := 0; i < 10; i++ {
					posts[i] = &domain.Post{
						ID:          int64(i + 41),
						Title:       "記事" + strconv.Itoa(i+41),
						Slug:        "post-" + strconv.Itoa(i+41),
						Content:     "内容" + strconv.Itoa(i+41),
						Status:      domain.PostStatusPublished,
						PublishedAt: &publishedAt,
					}
				}
				return posts, nil
			},
			expectedStatus: http.StatusOK,
			containsText: []string{
				"ページ 3",
				"前のページ",
				"/posts?page=2",
			},
			notContains: []string{
				"/posts?page=4",
			},
		},
		{
			name: "不正なページパラメータ（デフォルトで1ページ目）",
			url:  "/posts?page=invalid",
			mockFunc: func(limit, offset int) ([]*domain.Post, error) {
				if limit != 21 || offset != 0 {
					t.Errorf("expected limit=21, offset=0, got limit=%d, offset=%d", limit, offset)
				}
				posts := make([]*domain.Post, 5)
				publishedAt := time.Now()
				for i := 0; i < 5; i++ {
					posts[i] = &domain.Post{
						ID:          int64(i + 1),
						Title:       "記事" + strconv.Itoa(i+1),
						Slug:        "post-" + strconv.Itoa(i+1),
						Content:     "内容" + strconv.Itoa(i+1),
						Status:      domain.PostStatusPublished,
						PublishedAt: &publishedAt,
					}
				}
				return posts, nil
			},
			expectedStatus: http.StatusOK,
			containsText: []string{
				"ページ 1",
			},
		},
		{
			name: "page=0（1ページ目として扱う）",
			url:  "/posts?page=0",
			mockFunc: func(limit, offset int) ([]*domain.Post, error) {
				if limit != 21 || offset != 0 {
					t.Errorf("expected limit=21, offset=0, got limit=%d, offset=%d", limit, offset)
				}
				posts := make([]*domain.Post, 5)
				publishedAt := time.Now()
				for i := 0; i < 5; i++ {
					posts[i] = &domain.Post{
						ID:          int64(i + 1),
						Title:       "記事" + strconv.Itoa(i+1),
						Slug:        "post-" + strconv.Itoa(i+1),
						Content:     "内容" + strconv.Itoa(i+1),
						Status:      domain.PostStatusPublished,
						PublishedAt: &publishedAt,
					}
				}
				return posts, nil
			},
			expectedStatus: http.StatusOK,
			containsText: []string{
				"ページ 1",
			},
		},
		{
			name: "page=-1（1ページ目として扱う）",
			url:  "/posts?page=-1",
			mockFunc: func(limit, offset int) ([]*domain.Post, error) {
				if limit != 21 || offset != 0 {
					t.Errorf("expected limit=21, offset=0, got limit=%d, offset=%d", limit, offset)
				}
				posts := make([]*domain.Post, 5)
				publishedAt := time.Now()
				for i := 0; i < 5; i++ {
					posts[i] = &domain.Post{
						ID:          int64(i + 1),
						Title:       "記事" + strconv.Itoa(i+1),
						Slug:        "post-" + strconv.Itoa(i+1),
						Content:     "内容" + strconv.Itoa(i+1),
						Status:      domain.PostStatusPublished,
						PublishedAt: &publishedAt,
					}
				}
				return posts, nil
			},
			expectedStatus: http.StatusOK,
			containsText: []string{
				"ページ 1",
			},
		},
		{
			name: "ページパラメータなし（デフォルトで1ページ目）",
			url:  "/posts",
			mockFunc: func(limit, offset int) ([]*domain.Post, error) {
				if limit != 21 || offset != 0 {
					t.Errorf("expected limit=21, offset=0, got limit=%d, offset=%d", limit, offset)
				}
				posts := make([]*domain.Post, 21)
				publishedAt := time.Now()
				for i := 0; i < 21; i++ {
					posts[i] = &domain.Post{
						ID:          int64(i + 1),
						Title:       "記事" + strconv.Itoa(i+1),
						Slug:        "post-" + strconv.Itoa(i+1),
						Content:     "内容" + strconv.Itoa(i+1),
						Status:      domain.PostStatusPublished,
						PublishedAt: &publishedAt,
					}
				}
				return posts, nil
			},
			expectedStatus: http.StatusOK,
			containsText: []string{
				"ページ 1",
				"次のページ",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockPostService{
				getPublishedPostsFunc: tt.mockFunc,
			}
			router := NewRouterWithTemplates(mockService, nil, false, "goblog", "../view/templates/*.html")

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			body := w.Body.String()
			for _, text := range tt.containsText {
				if !strings.Contains(body, text) {
					t.Errorf("expected body to contain %q", text)
				}
			}

			for _, text := range tt.notContains {
				if strings.Contains(body, text) {
					t.Errorf("expected body NOT to contain %q", text)
				}
			}
		})
	}
}

func TestHandlePosts_Error(t *testing.T) {
	mockService := &mockPostService{
		getPublishedPostsFunc: func(limit, offset int) ([]*domain.Post, error) {
			return nil, fmt.Errorf("database connection error")
		},
	}
	router := NewRouterWithTemplates(mockService, nil, false, "goblog", "../view/templates/*.html")

	req := httptest.NewRequest(http.MethodGet, "/posts", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Internal Server Error") {
		t.Errorf("expected body to contain 'Internal Server Error', got %q", body)
	}
}

func TestHandlePostDetail(t *testing.T) {
	tests := []struct {
		name           string
		slug           string
		mockFunc       func(slug string) (*domain.Post, error)
		expectedStatus int
		containsText   []string
	}{
		{
			name: "記事が見つかる場合",
			slug: "test-post",
			mockFunc: func(slug string) (*domain.Post, error) {
				if slug == "test-post" {
					publishedAt := time.Now()
					return &domain.Post{
						ID:          1,
						Title:       "テスト記事のタイトル",
						Slug:        "test-post",
						Content:     "これはテスト記事の本文です。\n改行も含まれます。",
						Status:      domain.PostStatusPublished,
						Tags:        "Go,テスト,ブログ",
						PublishedAt: &publishedAt,
					}, nil
				}
				return nil, nil
			},
			expectedStatus: http.StatusOK,
			containsText: []string{
				"<!DOCTYPE html>",
				"テスト記事のタイトル",
				"これはテスト記事の本文です",
				"Go,テスト,ブログ",
				"記事一覧に戻る",
			},
		},
		{
			name: "記事が見つからない場合",
			slug: "non-existent",
			mockFunc: func(slug string) (*domain.Post, error) {
				return nil, nil
			},
			expectedStatus: http.StatusNotFound,
			containsText:   []string{},
		},
		{
			name: "英語のスラッグ",
			slug: "hello-world",
			mockFunc: func(slug string) (*domain.Post, error) {
				if slug == "hello-world" {
					publishedAt := time.Now()
					return &domain.Post{
						ID:          2,
						Title:       "Hello World",
						Slug:        "hello-world",
						Content:     "First post in English",
						Status:      domain.PostStatusPublished,
						PublishedAt: &publishedAt,
					}, nil
				}
				return nil, nil
			},
			expectedStatus: http.StatusOK,
			containsText: []string{
				"Hello World",
				"First post in English",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockPostService{
				getPostBySlugFunc: tt.mockFunc,
			}
			router := NewRouterWithTemplates(mockService, nil, false, "goblog", "../view/templates/*.html")

			req := httptest.NewRequest(http.MethodGet, "/posts/"+tt.slug, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				body := w.Body.String()
				for _, text := range tt.containsText {
					if !strings.Contains(body, text) {
						t.Errorf("expected body to contain %q", text)
					}
				}
			}
		})
	}
}

func TestHandleAdmin(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{
			name: "/admin",
			path: "/admin",
		},
		{
			name: "/admin/posts",
			path: "/admin/posts",
		},
		{
			name: "/admin/settings",
			path: "/admin/settings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockPostService{}
			router := NewRouterWithTemplates(mockService, nil, false, "goblog", "../view/templates/*.html")

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
			}

			contentType := w.Header().Get("Content-Type")
			expected := "text/html; charset=utf-8"
			if contentType != expected {
				t.Errorf("expected Content-Type %q, got %q", expected, contentType)
			}

			body := w.Body.String()
			if !strings.Contains(body, "管理画面") {
				t.Error("expected body to contain '管理画面'")
			}
			if !strings.Contains(body, "React SPA") {
				t.Error("expected body to contain 'React SPA'")
			}
		})
	}
}
