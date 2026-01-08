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

const testTemplatePattern = "../view/templates/*.html"

// mockPostService は PostService のモック実装です
type mockPostService struct {
	getPublishedPostsFunc       func(limit, offset int) ([]*domain.Post, error)
	getPublishedPostsByTagFunc  func(tag string, limit, offset int) ([]*domain.Post, error)
	getPublishedTagsFunc        func() (map[string]int, error)
	getPostBySlugFunc           func(slug string) (*domain.Post, error)
	getAllPostsFunc             func(status *domain.PostStatus, limit, offset int) ([]*domain.Post, error)
}

func (m *mockPostService) GetPublishedPosts(limit, offset int) ([]*domain.Post, error) {
	if m.getPublishedPostsFunc != nil {
		return m.getPublishedPostsFunc(limit, offset)
	}
	return []*domain.Post{}, nil
}

func (m *mockPostService) GetPublishedPostsByTag(tag string, limit, offset int) ([]*domain.Post, error) {
	if m.getPublishedPostsByTagFunc != nil {
		return m.getPublishedPostsByTagFunc(tag, limit, offset)
	}
	return []*domain.Post{}, nil
}

func (m *mockPostService) GetAllPostsByTag(tag string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
	return []*domain.Post{}, nil
}

func (m *mockPostService) GetPublishedTags() (map[string]int, error) {
	if m.getPublishedTagsFunc != nil {
		return m.getPublishedTagsFunc()
	}
	return map[string]int{}, nil
}

func (m *mockPostService) GetAllTags(status *domain.PostStatus) (map[string]int, error) {
	return map[string]int{}, nil
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
				">Go</a>",
				">テスト</a>",
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
			router := NewRouterWithTemplates(mockService, nil, false, "goblog", testTemplatePattern)

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

func TestHandleHome_BlogTitle(t *testing.T) {
	tests := []struct {
		name         string
		blogTitle    string
		mockFunc     func(limit, offset int) ([]*domain.Post, error)
		containsText []string
	}{
		{
			name:      "デフォルトのブログタイトル",
			blogTitle: "goblog",
			mockFunc: func(limit, offset int) ([]*domain.Post, error) {
				return []*domain.Post{}, nil
			},
			containsText: []string{
				"<title>goblog - ホーム</title>",
				"<a href=\"/\"",
				">goblog</a>",
				"ようこそgoblogへ",
				"&copy; 2025 goblog. All rights reserved.",
			},
		},
		{
			name:      "カスタムブログタイトル",
			blogTitle: "My Awesome Blog",
			mockFunc: func(limit, offset int) ([]*domain.Post, error) {
				return []*domain.Post{}, nil
			},
			containsText: []string{
				"<title>My Awesome Blog - ホーム</title>",
				"<a href=\"/\"",
				">My Awesome Blog</a>",
				"ようこそMy Awesome Blogへ",
				"&copy; 2025 My Awesome Blog. All rights reserved.",
			},
		},
		{
			name:      "日本語のブログタイトル",
			blogTitle: "テストブログ",
			mockFunc: func(limit, offset int) ([]*domain.Post, error) {
				publishedAt := time.Now()
				return []*domain.Post{
					{
						ID:          1,
						Title:       "記事1",
						Slug:        "post-1",
						Content:     "内容1",
						Status:      domain.PostStatusPublished,
						PublishedAt: &publishedAt,
					},
				}, nil
			},
			containsText: []string{
				"<title>テストブログ - ホーム</title>",
				"<a href=\"/\"",
				">テストブログ</a>",
				"ようこそテストブログへ",
				"&copy; 2025 テストブログ. All rights reserved.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockPostService{
				getPublishedPostsFunc: tt.mockFunc,
			}
			router := NewRouterWithTemplates(mockService, nil, false, tt.blogTitle, testTemplatePattern)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
			}

			body := w.Body.String()
			for _, text := range tt.containsText {
				if !strings.Contains(body, text) {
					t.Errorf("expected body to contain %q", text)
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
				">タグ1</a>",
				">タグ2</a>",
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
			router := NewRouterWithTemplates(mockService, nil, false, "goblog", testTemplatePattern)

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
				"ページ",
				">1</span>",
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
				"ページ",
				">2</span>",
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
				"ページ",
				">3</span>",
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
				"ページ",
				">1</span>",
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
				"ページ",
				">1</span>",
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
				"ページ",
				">1</span>",
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
				"ページ",
				">1</span>",
				"次のページ",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockPostService{
				getPublishedPostsFunc: tt.mockFunc,
			}
			router := NewRouterWithTemplates(mockService, nil, false, "goblog", testTemplatePattern)

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
	router := NewRouterWithTemplates(mockService, nil, false, "goblog", testTemplatePattern)

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

func TestHandlePosts_BlogTitle(t *testing.T) {
	tests := []struct {
		name         string
		blogTitle    string
		mockFunc     func(limit, offset int) ([]*domain.Post, error)
		containsText []string
	}{
		{
			name:      "デフォルトのブログタイトル",
			blogTitle: "goblog",
			mockFunc: func(limit, offset int) ([]*domain.Post, error) {
				return []*domain.Post{}, nil
			},
			containsText: []string{
				"<title>記事一覧 - goblog</title>",
				"<a href=\"/\"",
				">goblog</a>",
				"&copy; 2025 goblog. All rights reserved.",
			},
		},
		{
			name:      "カスタムブログタイトル",
			blogTitle: "My Tech Blog",
			mockFunc: func(limit, offset int) ([]*domain.Post, error) {
				publishedAt := time.Now()
				return []*domain.Post{
					{
						ID:          1,
						Title:       "Test Post",
						Slug:        "test-post",
						Content:     "Test content",
						Status:      domain.PostStatusPublished,
						PublishedAt: &publishedAt,
					},
				}, nil
			},
			containsText: []string{
				"<title>記事一覧 - My Tech Blog</title>",
				"<a href=\"/\"",
				">My Tech Blog</a>",
				"&copy; 2025 My Tech Blog. All rights reserved.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockPostService{
				getPublishedPostsFunc: tt.mockFunc,
			}
			router := NewRouterWithTemplates(mockService, nil, false, tt.blogTitle, testTemplatePattern)

			req := httptest.NewRequest(http.MethodGet, "/posts", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
			}

			body := w.Body.String()
			for _, text := range tt.containsText {
				if !strings.Contains(body, text) {
					t.Errorf("expected body to contain %q", text)
				}
			}
		})
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
				`href="/tags/Go"`,
				`>Go</a>`,
				`>テスト</a>`,
				`>ブログ</a>`,
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
			router := NewRouterWithTemplates(mockService, nil, false, "goblog", testTemplatePattern)

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

func TestHandlePostDetail_BlogTitle(t *testing.T) {
	tests := []struct {
		name         string
		blogTitle    string
		slug         string
		mockFunc     func(slug string) (*domain.Post, error)
		containsText []string
	}{
		{
			name:      "デフォルトのブログタイトル",
			blogTitle: "goblog",
			slug:      "test-article",
			mockFunc: func(slug string) (*domain.Post, error) {
				if slug == "test-article" {
					publishedAt := time.Now()
					return &domain.Post{
						ID:          1,
						Title:       "テスト記事",
						Slug:        "test-article",
						Content:     "記事の本文です",
						Status:      domain.PostStatusPublished,
						PublishedAt: &publishedAt,
					}, nil
				}
				return nil, nil
			},
			containsText: []string{
				"<title>テスト記事 - goblog</title>",
				"<a href=\"/\"",
				">goblog</a>",
				"&copy; 2025 goblog. All rights reserved.",
			},
		},
		{
			name:      "カスタムブログタイトル",
			blogTitle: "開発ブログ",
			slug:      "golang-tips",
			mockFunc: func(slug string) (*domain.Post, error) {
				if slug == "golang-tips" {
					publishedAt := time.Now()
					return &domain.Post{
						ID:          2,
						Title:       "Go言語のTips",
						Slug:        "golang-tips",
						Content:     "Goの便利な機能について",
						Status:      domain.PostStatusPublished,
						Tags:        "Go,プログラミング",
						PublishedAt: &publishedAt,
					}, nil
				}
				return nil, nil
			},
			containsText: []string{
				"<title>Go言語のTips - 開発ブログ</title>",
				"<a href=\"/\"",
				">開発ブログ</a>",
				"&copy; 2025 開発ブログ. All rights reserved.",
			},
		},
		{
			name:      "英語のブログタイトル",
			blogTitle: "Tech Insights",
			slug:      "first-post",
			mockFunc: func(slug string) (*domain.Post, error) {
				if slug == "first-post" {
					publishedAt := time.Now()
					return &domain.Post{
						ID:          3,
						Title:       "My First Post",
						Slug:        "first-post",
						Content:     "Welcome to my blog!",
						Status:      domain.PostStatusPublished,
						PublishedAt: &publishedAt,
					}, nil
				}
				return nil, nil
			},
			containsText: []string{
				"<title>My First Post - Tech Insights</title>",
				"<a href=\"/\"",
				">Tech Insights</a>",
				"&copy; 2025 Tech Insights. All rights reserved.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockPostService{
				getPostBySlugFunc: tt.mockFunc,
			}
			router := NewRouterWithTemplates(mockService, nil, false, tt.blogTitle, testTemplatePattern)

			req := httptest.NewRequest(http.MethodGet, "/posts/"+tt.slug, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
			}

			body := w.Body.String()
			for _, text := range tt.containsText {
				if !strings.Contains(body, text) {
					t.Errorf("expected body to contain %q", text)
				}
			}
		})
	}
}


func TestTruncateRunes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxRunes int
		expected string
	}{
		{
			name:     "English text - no truncation needed",
			input:    "Hello World",
			maxRunes: 20,
			expected: "Hello World",
		},
		{
			name:     "English text - truncation needed",
			input:    "Hello World, this is a long text",
			maxRunes: 11,
			expected: "Hello World",
		},
		{
			name:     "Japanese text - no truncation needed",
			input:    "こんにちは世界",
			maxRunes: 10,
			expected: "こんにちは世界",
		},
		{
			name:     "Japanese text - truncation needed",
			input:    "Goを使ってシンプルなブログシステムを構築しました。このブログシステムは以下の機能を持っています：",
			maxRunes: 30,
			expected: "Goを使ってシンプルなブログシステムを構築しました。このブロ",
		},
		{
			name:     "Japanese text - exactly 200 runes",
			input:    "Goを使ってシンプルなブログシステムを構築しました。このブログシステムは以下の機能を持っています：記事の作成・編集・削除、公開・非公開の切り替え、記事一覧表示。また、セキュリティ機能として、パスワードポリシーの適用、bcryptによるパスワードハッシュ化、セッション管理、CSRF保護、ブルートフォース対策なども実装されています。シンプルで使いやすいインターフェースが特徴です。",
			maxRunes: 200,
			expected: "Goを使ってシンプルなブログシステムを構築しました。このブログシステムは以下の機能を持っています：記事の作成・編集・削除、公開・非公開の切り替え、記事一覧表示。また、セキュリティ機能として、パスワードポリシーの適用、bcryptによるパスワードハッシュ化、セッション管理、CSRF保護、ブルートフォース対策なども実装されています。シンプルで使いやすいインターフェースが特徴です。",
		},
		{
			name:     "Mixed Japanese and English",
			input:    "Hello世界、これはTest文章です",
			maxRunes: 10,
			expected: "Hello世界、これ",
		},
		{
			name:     "Emoji handling",
			input:    "こんにちは🎉世界😊",
			maxRunes: 6,
			expected: "こんにちは🎉",
		},
		{
			name:     "Empty string",
			input:    "",
			maxRunes: 10,
			expected: "",
		},
		{
			name:     "Zero maxRunes",
			input:    "Hello",
			maxRunes: 0,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateRunes(tt.input, tt.maxRunes)
			if result != tt.expected {
				t.Errorf("truncateRunes() = %q, want %q", result, tt.expected)
			}

			// 重要: 切り詰めた結果に不正なUTF-8シーケンス（�）が含まれていないことを確認
			if strings.Contains(result, "�") {
				t.Errorf("truncateRunes() produced invalid UTF-8 sequence (�) for input %q", tt.input)
			}
		})
	}
}

func TestHandleTags(t *testing.T) {
	tests := []struct {
		name           string
		mockFunc       func() (map[string]int, error)
		expectedStatus int
		containsText   []string
		notContains    []string
	}{
		{
			name: "タグがある場合",
			mockFunc: func() (map[string]int, error) {
				return map[string]int{
					"Go":     10,
					"React":  8,
					"Docker": 5,
				}, nil
			},
			expectedStatus: http.StatusOK,
			containsText: []string{
				"<!DOCTYPE html>",
				"タグ一覧",
				`href="/tags/Go"`,
				">Go</h3>",
				"10件",
				">React</h3>",
				"8件",
				">Docker</h3>",
				"5件",
			},
		},
		{
			name: "タグがない場合",
			mockFunc: func() (map[string]int, error) {
				return map[string]int{}, nil
			},
			expectedStatus: http.StatusOK,
			containsText: []string{
				"タグ一覧",
				"まだタグがありません",
			},
		},
		{
			name: "エラーが発生した場合",
			mockFunc: func() (map[string]int, error) {
				return nil, fmt.Errorf("database error")
			},
			expectedStatus: http.StatusInternalServerError,
			containsText: []string{
				"Internal Server Error",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockPostService{
				getPublishedTagsFunc: tt.mockFunc,
			}
			router := NewRouterWithTemplates(mockService, nil, false, "goblog", testTemplatePattern)

			req := httptest.NewRequest(http.MethodGet, "/tags", nil)
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

func TestHandleTagPosts(t *testing.T) {
	tests := []struct {
		name           string
		tag            string
		mockFunc       func(tag string, limit, offset int) ([]*domain.Post, error)
		expectedStatus int
		containsText   []string
		notContains    []string
	}{
		{
			name: "タグに記事がある場合",
			tag:  "Go",
			mockFunc: func(tag string, limit, offset int) ([]*domain.Post, error) {
				if tag == "Go" {
					publishedAt := time.Now()
					return []*domain.Post{
						{
							ID:          1,
							Title:       "Go入門",
							Slug:        "go-introduction",
							Content:     "Goの基礎を学びましょう",
							Status:      domain.PostStatusPublished,
							Tags:        "Go,プログラミング",
							PublishedAt: &publishedAt,
						},
						{
							ID:          2,
							Title:       "Goの並行処理",
							Slug:        "go-concurrency",
							Content:     "ゴルーチンとチャネルについて",
							Status:      domain.PostStatusPublished,
							Tags:        "Go,並行処理",
							PublishedAt: &publishedAt,
						},
					}, nil
				}
				return []*domain.Post{}, nil
			},
			expectedStatus: http.StatusOK,
			containsText: []string{
				"<!DOCTYPE html>",
				"inline-flex items-center",
				"Go入門",
				"Goの並行処理",
				"/posts/go-introduction",
				"/posts/go-concurrency",
			},
		},
		{
			name: "タグに記事がない場合",
			tag:  "Python",
			mockFunc: func(tag string, limit, offset int) ([]*domain.Post, error) {
				return []*domain.Post{}, nil
			},
			expectedStatus: http.StatusOK,
			containsText: []string{
				"inline-flex items-center",
				"このタグの記事はまだありません",
			},
		},
		{
			name: "エラーが発生した場合",
			tag:  "Go",
			mockFunc: func(tag string, limit, offset int) ([]*domain.Post, error) {
				return nil, fmt.Errorf("database error")
			},
			expectedStatus: http.StatusInternalServerError,
			containsText: []string{
				"Internal Server Error",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockPostService{
				getPublishedPostsByTagFunc: tt.mockFunc,
			}
			router := NewRouterWithTemplates(mockService, nil, false, "goblog", testTemplatePattern)

			req := httptest.NewRequest(http.MethodGet, "/tags/"+tt.tag, nil)
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

func TestHandleTagPosts_Pagination(t *testing.T) {
	tests := []struct {
		name           string
		tag            string
		url            string
		mockFunc       func(tag string, limit, offset int) ([]*domain.Post, error)
		expectedStatus int
		containsText   []string
		notContains    []string
	}{
		{
			name: "1ページ目（次のページあり）",
			tag:  "Go",
			url:  "/tags/Go?page=1",
			mockFunc: func(tag string, limit, offset int) ([]*domain.Post, error) {
				if limit != 21 || offset != 0 {
					t.Errorf("expected limit=21, offset=0, got limit=%d, offset=%d", limit, offset)
				}
				posts := make([]*domain.Post, 21)
				publishedAt := time.Now()
				for i := 0; i < 21; i++ {
					posts[i] = &domain.Post{
						ID:          int64(i + 1),
						Title:       "Go記事" + strconv.Itoa(i+1),
						Slug:        "go-post-" + strconv.Itoa(i+1),
						Content:     "内容" + strconv.Itoa(i+1),
						Status:      domain.PostStatusPublished,
						Tags:        "Go",
						PublishedAt: &publishedAt,
					}
				}
				return posts, nil
			},
			expectedStatus: http.StatusOK,
			containsText: []string{
				"inline-flex items-center",
				"ページ",
				">1</span>",
				"次のページ",
				"/tags/Go?page=2",
			},
			notContains: []string{
				"/tags/Go?page=0",
			},
		},
		{
			name: "2ページ目",
			tag:  "React",
			url:  "/tags/React?page=2",
			mockFunc: func(tag string, limit, offset int) ([]*domain.Post, error) {
				if limit != 21 || offset != 20 {
					t.Errorf("expected limit=21, offset=20, got limit=%d, offset=%d", limit, offset)
				}
				posts := make([]*domain.Post, 10)
				publishedAt := time.Now()
				for i := 0; i < 10; i++ {
					posts[i] = &domain.Post{
						ID:          int64(i + 21),
						Title:       "React記事" + strconv.Itoa(i+21),
						Slug:        "react-post-" + strconv.Itoa(i+21),
						Content:     "内容" + strconv.Itoa(i+21),
						Status:      domain.PostStatusPublished,
						Tags:        "React",
						PublishedAt: &publishedAt,
					}
				}
				return posts, nil
			},
			expectedStatus: http.StatusOK,
			containsText: []string{
				"inline-flex items-center",
				"ページ",
				">2</span>",
				"前のページ",
				"/tags/React?page=1",
			},
			notContains: []string{
				"/tags/React?page=3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockPostService{
				getPublishedPostsByTagFunc: tt.mockFunc,
			}
			router := NewRouterWithTemplates(mockService, nil, false, "goblog", testTemplatePattern)

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

func TestFormatDateWithTZ(t *testing.T) {
	tests := []struct {
		name           string
		timeZone       string
		inputTime      time.Time
		expectedOutput string
	}{
		{
			name:           "Asia/Tokyo timezone",
			timeZone:       "Asia/Tokyo",
			inputTime:      time.Date(2024, 12, 26, 0, 30, 0, 0, time.UTC),
			expectedOutput: "2024-12-26 (JST)",
		},
		{
			name:           "UTC timezone",
			timeZone:       "UTC",
			inputTime:      time.Date(2024, 12, 26, 15, 30, 0, 0, time.UTC),
			expectedOutput: "2024-12-26 (UTC)",
		},
		{
			name:           "America/New_York timezone",
			timeZone:       "America/New_York",
			inputTime:      time.Date(2024, 12, 26, 5, 0, 0, 0, time.UTC),
			expectedOutput: "2024-12-26 (EST)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TZ", tt.timeZone)
			result := formatDateWithTZ(tt.inputTime)
			if result != tt.expectedOutput {
				t.Errorf("formatDateWithTZ() = %q, expected %q", result, tt.expectedOutput)
			}
		})
	}
}

func TestFormatDateDetailWithTZ(t *testing.T) {
	tests := []struct {
		name           string
		timeZone       string
		inputTime      time.Time
		expectedOutput string
	}{
		{
			name:           "Asia/Tokyo timezone",
			timeZone:       "Asia/Tokyo",
			inputTime:      time.Date(2024, 12, 26, 0, 30, 0, 0, time.UTC),
			expectedOutput: "2024-12-26 09:30 (JST)",
		},
		{
			name:           "UTC timezone",
			timeZone:       "UTC",
			inputTime:      time.Date(2024, 12, 26, 15, 30, 0, 0, time.UTC),
			expectedOutput: "2024-12-26 15:30 (UTC)",
		},
		{
			name:           "America/New_York timezone",
			timeZone:       "America/New_York",
			inputTime:      time.Date(2024, 12, 26, 5, 0, 0, 0, time.UTC),
			expectedOutput: "2024-12-26 00:00 (EST)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TZ", tt.timeZone)
			result := formatDateDetailWithTZ(tt.inputTime)
			if result != tt.expectedOutput {
				t.Errorf("formatDateDetailWithTZ() = %q, expected %q", result, tt.expectedOutput)
			}
		})
	}
}

func TestFormatDateWithTZ_Integration(t *testing.T) {
	// テンプレートとの統合テスト：実際のHTMLレスポンスに正しい日付フォーマットが含まれているか確認
	publishedAt := time.Date(2024, 12, 25, 15, 30, 0, 0, time.UTC)

	t.Run("JST timezone in rendered HTML", func(t *testing.T) {
		t.Setenv("TZ", "Asia/Tokyo")

		mockService := &mockPostService{
			getPublishedPostsFunc: func(limit, offset int) ([]*domain.Post, error) {
				return []*domain.Post{
					{
						ID:          1,
						Title:       "Test Post",
						Slug:        "test-post",
						Content:     "Test content",
						Status:      domain.PostStatusPublished,
						PublishedAt: &publishedAt,
					},
				}, nil
			},
		}

		router := NewRouterWithTemplates(mockService, nil, false, "Test Blog", testTemplatePattern)
		req := httptest.NewRequest(http.MethodGet, "/posts", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		body := w.Body.String()

		// 表示: 2024-12-26 (JST)
		expectedDate := ">2024-12-26 (JST)</time>"
		if !strings.Contains(body, expectedDate) {
			t.Errorf("expected body to contain %q, but it didn't. Body: %s", expectedDate, body)
		}
		// title属性に時刻+タイムゾーン略称が含まれる
		expectedTitle := `title="2024-12-26 00:30 (JST)"`
		if !strings.Contains(body, expectedTitle) {
			t.Errorf("expected body to contain %q, but it didn't. Body: %s", expectedTitle, body)
		}
	})

	t.Run("UTC timezone in rendered HTML", func(t *testing.T) {
		t.Setenv("TZ", "UTC")

		mockService := &mockPostService{
			getPublishedPostsFunc: func(limit, offset int) ([]*domain.Post, error) {
				return []*domain.Post{
					{
						ID:          1,
						Title:       "Test Post",
						Slug:        "test-post",
						Content:     "Test content",
						Status:      domain.PostStatusPublished,
						PublishedAt: &publishedAt,
					},
				}, nil
			},
		}

		router := NewRouterWithTemplates(mockService, nil, false, "Test Blog", testTemplatePattern)
		req := httptest.NewRequest(http.MethodGet, "/posts", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		body := w.Body.String()

		// 表示: 2024-12-25 (UTC)
		expectedDate := ">2024-12-25 (UTC)</time>"
		if !strings.Contains(body, expectedDate) {
			t.Errorf("expected body to contain %q, but it didn't. Body: %s", expectedDate, body)
		}
		// title属性に時刻+タイムゾーン略称が含まれる
		expectedTitle := `title="2024-12-25 15:30 (UTC)"`
		if !strings.Contains(body, expectedTitle) {
			t.Errorf("expected body to contain %q, but it didn't. Body: %s", expectedTitle, body)
		}
	})
}
