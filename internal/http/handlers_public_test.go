package http

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/capybara-translation/goblog/internal/service"
)

const (
	testTemplatePattern = "../view/templates/*.html"
	testUploadDir       = ""                        // Upload directory is not used in tests
	testMaxUploadSize   = 5 * 1024 * 1024           // 5MB
	testSecureCookie    = false                     // Secure Cookie is disabled in tests
	testBlogTitle       = "goblog"                  // Blog title for testing
	testBaseURL         = "http://localhost:8080"   // Base URL for testing
	testPostsPerPage    = 20                        // Posts per page for testing
)

// mockPostService is a mock implementation of PostService
type mockPostService struct {
	getPublishedPostsFunc        func(limit, offset int) ([]*domain.Post, error)
	getPublishedPostsByTagFunc   func(tag string, limit, offset int) ([]*domain.Post, error)
	getPublishedTagsFunc         func() (map[string]int, error)
	getPostBySlugFunc            func(slug string) (*domain.Post, error)
	getAllPostsFunc              func(status *domain.PostStatus, limit, offset int) ([]*domain.Post, error)
	countPostsFunc               func(status *domain.PostStatus) (int, error)
	countPostsByTagFunc          func(tag string, status *domain.PostStatus) (int, error)
	searchPostsFunc              func(query string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error)
	countSearchPostsFunc         func(query string, status *domain.PostStatus) (int, error)
	searchPublishedPostsFunc     func(query string, limit, offset int) ([]*domain.Post, error)
	countSearchPublishedFunc     func(query string) (int, error)
	getPinnedPostsFunc           func() ([]*domain.Post, error)
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

func (m *mockPostService) CreatePost(title, slug, content, tags string, isPinned bool) (*domain.Post, error) {
	return nil, nil
}

func (m *mockPostService) UpdatePost(id int64, title, slug, content, tags string, isPinned bool) (*domain.Post, error) {
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

func (m *mockPostService) CountPosts(status *domain.PostStatus) (int, error) {
	if m.countPostsFunc != nil {
		return m.countPostsFunc(status)
	}
	return 0, nil
}

func (m *mockPostService) CountPostsByTag(tag string, status *domain.PostStatus) (int, error) {
	if m.countPostsByTagFunc != nil {
		return m.countPostsByTagFunc(tag, status)
	}
	return 0, nil
}

func (m *mockPostService) SearchPosts(query string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
	if m.searchPostsFunc != nil {
		return m.searchPostsFunc(query, status, limit, offset)
	}
	return []*domain.Post{}, nil
}

func (m *mockPostService) CountSearchPosts(query string, status *domain.PostStatus) (int, error) {
	if m.countSearchPostsFunc != nil {
		return m.countSearchPostsFunc(query, status)
	}
	return 0, nil
}

func (m *mockPostService) SearchPublishedPosts(query string, limit, offset int) ([]*domain.Post, error) {
	if m.searchPublishedPostsFunc != nil {
		return m.searchPublishedPostsFunc(query, limit, offset)
	}
	return []*domain.Post{}, nil
}

func (m *mockPostService) CountSearchPublishedPosts(query string) (int, error) {
	if m.countSearchPublishedFunc != nil {
		return m.countSearchPublishedFunc(query)
	}
	return 0, nil
}

func (m *mockPostService) GetPinnedPosts() ([]*domain.Post, error) {
	if m.getPinnedPostsFunc != nil {
		return m.getPinnedPostsFunc()
	}
	return []*domain.Post{}, nil
}

func (m *mockPostService) SetPinned(id int64, pinned bool) (*domain.Post, error) {
	return nil, nil
}

var _ service.PostService = (*mockPostService)(nil)

func TestHandleHome(t *testing.T) {
	tests := []struct {
		name                     string
		url                      string
		getPublishedPostsFunc    func(limit, offset int) ([]*domain.Post, error)
		searchPublishedPostsFunc func(query string, limit, offset int) ([]*domain.Post, error)
		expectedStatus           int
		containsText             []string
		notContains              []string
	}{
		{
			name: "default page shows up to 20 published posts",
			url:  "/",
			getPublishedPostsFunc: func(limit, offset int) ([]*domain.Post, error) {
				if limit != 21 || offset != 0 {
					t.Errorf("expected limit=21 offset=0, got limit=%d offset=%d", limit, offset)
				}
				publishedAt := time.Now()
				return []*domain.Post{
					{ID: 1, Title: "テスト記事1", Slug: "test-post-1", Status: domain.PostStatusPublished, PublishedAt: &publishedAt, Tags: "Go"},
					{ID: 2, Title: "テスト記事2", Slug: "test-post-2", Status: domain.PostStatusPublished, PublishedAt: &publishedAt},
				}, nil
			},
			expectedStatus: http.StatusOK,
			containsText: []string{
				"テスト記事1",
				"/posts/test-post-1",
				"テスト記事2",
				"#Go",
			},
			notContains: []string{"View all posts", "No posts yet."},
		},
		{
			name: "page=2 requests offset 20",
			url:  "/?page=2",
			getPublishedPostsFunc: func(limit, offset int) ([]*domain.Post, error) {
				if limit != 21 || offset != 20 {
					t.Errorf("expected limit=21 offset=20, got limit=%d offset=%d", limit, offset)
				}
				return []*domain.Post{}, nil
			},
			expectedStatus: http.StatusOK,
			containsText:   []string{"No posts yet."},
		},
		{
			name: "search routes to SearchPublishedPosts",
			url:  "/?q=Go",
			searchPublishedPostsFunc: func(query string, limit, offset int) ([]*domain.Post, error) {
				if query != "Go" || limit != 21 || offset != 0 {
					t.Errorf("unexpected args: query=%q limit=%d offset=%d", query, limit, offset)
				}
				publishedAt := time.Now()
				return []*domain.Post{
					{ID: 10, Title: "Go intro", Slug: "go-intro", Status: domain.PostStatusPublished, PublishedAt: &publishedAt},
				}, nil
			},
			expectedStatus: http.StatusOK,
			containsText:   []string{"/posts/go-intro", "Search results for"},
		},
		{
			name: "search with no matches shows search-aware empty message",
			url:  "/?q=zzznomatch",
			searchPublishedPostsFunc: func(query string, limit, offset int) ([]*domain.Post, error) {
				return []*domain.Post{}, nil
			},
			expectedStatus: http.StatusOK,
			containsText:   []string{`No posts found matching "zzznomatch"`},
			notContains:    []string{"No posts yet."},
		},
		{
			name: "no posts at all shows empty message",
			url:  "/",
			getPublishedPostsFunc: func(limit, offset int) ([]*domain.Post, error) {
				return []*domain.Post{}, nil
			},
			expectedStatus: http.StatusOK,
			containsText:   []string{"No posts yet."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockPostService{
				getPublishedPostsFunc:    tt.getPublishedPostsFunc,
				searchPublishedPostsFunc: tt.searchPublishedPostsFunc,
			}
			router := NewRouterWithTemplates(mockService, nil, nil, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize, testPostsPerPage)

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
			body := w.Body.String()
			for _, text := range tt.containsText {
				if !strings.Contains(body, text) {
					t.Errorf("expected body to contain %q\nbody: %s", text, body)
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
			name:      "Default blog title",
			blogTitle: "goblog",
			mockFunc: func(limit, offset int) ([]*domain.Post, error) {
				return []*domain.Post{}, nil
			},
			containsText: []string{
				"<title>goblog</title>",
				"<a href=\"/\"",
				">goblog</a>",
				"&copy; 2025 goblog</span>",
			},
		},
		{
			name:      "Custom blog title",
			blogTitle: "My Awesome Blog",
			mockFunc: func(limit, offset int) ([]*domain.Post, error) {
				return []*domain.Post{}, nil
			},
			containsText: []string{
				"<title>My Awesome Blog</title>",
				"<a href=\"/\"",
				">My Awesome Blog</a>",
				"&copy; 2025 My Awesome Blog</span>",
			},
		},
		{
			name:      "Japanese blog title",
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
				"<title>テストブログ</title>",
				"<a href=\"/\"",
				">テストブログ</a>",
				"&copy; 2025 テストブログ</span>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockPostService{
				getPublishedPostsFunc: tt.mockFunc,
			}
			router := NewRouterWithTemplates(mockService, nil, nil, false, nil, tt.blogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize, testPostsPerPage)

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

func TestHandleHome_Pagination(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		mockFunc       func(limit, offset int) ([]*domain.Post, error)
		expectedStatus int
		containsText   []string
		notContains    []string
	}{
		{
			name: "Page 1 (has next page)",
			url:  "/?page=1",
			mockFunc: func(limit, offset int) ([]*domain.Post, error) {
				if limit != 21 || offset != 0 {
					t.Errorf("expected limit=21, offset=0, got limit=%d, offset=%d", limit, offset)
				}
				// Return 21 items to indicate there's a next page
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
				"Page",
				"Page 1",
				"/?page=2",
			},
			notContains: []string{
				"/?page=0",
			},
		},
		{
			name: "Page 2 (has previous and next pages)",
			url:  "/?page=2",
			mockFunc: func(limit, offset int) ([]*domain.Post, error) {
				if limit != 21 || offset != 20 {
					t.Errorf("expected limit=21, offset=20, got limit=%d, offset=%d", limit, offset)
				}
				// Return 21 items to indicate there's a next page
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
				"Page",
				"Page 2",
				"/?page=1",
				"/?page=3",
			},
		},
		{
			name: "Last page (no next page)",
			url:  "/?page=3",
			mockFunc: func(limit, offset int) ([]*domain.Post, error) {
				if limit != 21 || offset != 40 {
					t.Errorf("expected limit=21, offset=40, got limit=%d, offset=%d", limit, offset)
				}
				// Return only 10 items to indicate there's no next page
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
				"Page",
				"Page 3",
				"/?page=2",
			},
			notContains: []string{
				"/?page=4",
			},
		},
		{
			name: "Invalid page parameter (defaults to page 1)",
			url:  "/?page=invalid",
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
				"Page",
				"Page 1",
			},
		},
		{
			name: "page=0 (treated as page 1)",
			url:  "/?page=0",
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
				"Page",
				"Page 1",
			},
		},
		{
			name: "page=-1 (treated as page 1)",
			url:  "/?page=-1",
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
				"Page",
				"Page 1",
			},
		},
		{
			name: "No page parameter (defaults to page 1)",
			url:  "/",
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
				"Page",
				"Page 1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockPostService{
				getPublishedPostsFunc: tt.mockFunc,
			}
			router := NewRouterWithTemplates(mockService, nil, nil, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize, testPostsPerPage)

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

func TestHandleHome_Error(t *testing.T) {
	mockService := &mockPostService{
		getPublishedPostsFunc: func(limit, offset int) ([]*domain.Post, error) {
			return nil, fmt.Errorf("database connection error")
		},
	}
	router := NewRouterWithTemplates(mockService, nil, nil, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize, testPostsPerPage)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
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
			name: "When post is found",
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
				"#Go",
				"#テスト",
				"#ブログ",
				"Back to posts",
			},
		},
		{
			name: "When post is not found",
			slug: "non-existent",
			mockFunc: func(slug string) (*domain.Post, error) {
				return nil, nil
			},
			expectedStatus: http.StatusNotFound,
			containsText: []string{
				"<!DOCTYPE html>",
				"404",
				"Page Not Found",
				"The page you're looking for doesn't exist or has been moved.",
				"Go Home",
				"View Posts",
			},
		},
		{
			name: "English slug",
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
			router := NewRouterWithTemplates(mockService, nil, nil, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize, testPostsPerPage)

			req := httptest.NewRequest(http.MethodGet, "/posts/"+tt.slug, nil)
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
			name:      "Default blog title",
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
				"&copy; 2025 goblog</span>",
			},
		},
		{
			name:      "Custom blog title",
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
				"&copy; 2025 開発ブログ</span>",
			},
		},
		{
			name:      "English blog title",
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
				"&copy; 2025 Tech Insights</span>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockPostService{
				getPostBySlugFunc: tt.mockFunc,
			}
			router := NewRouterWithTemplates(mockService, nil, nil, false, nil, tt.blogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize, testPostsPerPage)

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

			// Important: Verify that the truncated result does not contain invalid UTF-8 sequences
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
			name: "When tags exist",
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
				"Tags",
				`href="/tags/Go"`,
				`>#</span>Go`,
				"(10)",
				`>#</span>React`,
				"(8)",
				`>#</span>Docker`,
				"(5)",
			},
		},
		{
			name: "When no tags exist",
			mockFunc: func() (map[string]int, error) {
				return map[string]int{}, nil
			},
			expectedStatus: http.StatusOK,
			containsText: []string{
				"Tags",
				"No tags yet.",
			},
		},
		{
			name: "When an error occurs",
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
			router := NewRouterWithTemplates(mockService, nil, nil, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize, testPostsPerPage)

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
			name: "When tag has posts",
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
				`>#</span>Go`,
				"Go入門",
				"Goの並行処理",
				"/posts/go-introduction",
				"/posts/go-concurrency",
			},
		},
		{
			name: "When tag has no posts",
			tag:  "Python",
			mockFunc: func(tag string, limit, offset int) ([]*domain.Post, error) {
				return []*domain.Post{}, nil
			},
			expectedStatus: http.StatusOK,
			containsText: []string{
				`>#</span>Python`,
				"No posts with this tag yet.",
			},
		},
		{
			name: "When an error occurs",
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
			router := NewRouterWithTemplates(mockService, nil, nil, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize, testPostsPerPage)

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
			name: "Page 1 (has next page)",
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
				"Page",
				"Page 1",
				"/tags/Go?page=2",
			},
			notContains: []string{
				"/tags/Go?page=0",
			},
		},
		{
			name: "Page 2",
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
				"Page",
				"Page 2",
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
			router := NewRouterWithTemplates(mockService, nil, nil, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize, testPostsPerPage)

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
			expectedOutput: "2024-12-26 (JST, UTC+9)",
		},
		{
			name:           "UTC timezone",
			timeZone:       "UTC",
			inputTime:      time.Date(2024, 12, 26, 15, 30, 0, 0, time.UTC),
			expectedOutput: "2024-12-26 (UTC, UTC+0)",
		},
		{
			name:           "America/New_York timezone",
			timeZone:       "America/New_York",
			inputTime:      time.Date(2024, 12, 26, 5, 0, 0, 0, time.UTC),
			expectedOutput: "2024-12-26 (EST, UTC-5)",
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
			expectedOutput: "2024-12-26 09:30 (JST, UTC+9)",
		},
		{
			name:           "UTC timezone",
			timeZone:       "UTC",
			inputTime:      time.Date(2024, 12, 26, 15, 30, 0, 0, time.UTC),
			expectedOutput: "2024-12-26 15:30 (UTC, UTC+0)",
		},
		{
			name:           "America/New_York timezone",
			timeZone:       "America/New_York",
			inputTime:      time.Date(2024, 12, 26, 5, 0, 0, 0, time.UTC),
			expectedOutput: "2024-12-26 00:00 (EST, UTC-5)",
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

func TestHandleHome_Search(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		mockSearch     func(query string, limit, offset int) ([]*domain.Post, error)
		mockCount      func(query string) (int, error)
		expectedStatus int
		containsText   []string
		notContains    []string
	}{
		{
			name: "Get posts with search query",
			url:  "/?q=Go",
			mockSearch: func(query string, limit, offset int) ([]*domain.Post, error) {
				if query != "Go" {
					t.Errorf("expected query 'Go', got %s", query)
				}
				publishedAt := time.Now()
				return []*domain.Post{
					{
						ID:          1,
						Title:       "Go入門",
						Slug:        "go-introduction",
						Content:     "Goの基本的な使い方を解説します",
						Status:      domain.PostStatusPublished,
						Tags:        "Go,プログラミング",
						PublishedAt: &publishedAt,
					},
					{
						ID:          2,
						Title:       "Go応用テクニック",
						Slug:        "go-advanced",
						Content:     "Goの応用的なテクニック",
						Status:      domain.PostStatusPublished,
						PublishedAt: &publishedAt,
					},
				}, nil
			},
			mockCount: func(query string) (int, error) {
				return 2, nil
			},
			expectedStatus: http.StatusOK,
			containsText: []string{
				"Search results for \"<span class=\"font-medium\">Go</span>\"",
				"<mark>Go</mark>入門",           // Highlighted title
				"<mark>Go</mark>応用テクニック", // Highlighted title
				"/posts/go-introduction",
				"/posts/go-advanced",
				`value="Go"`, // Value remains in search box
			},
			notContains: []string{
				"No posts yet.",
			},
		},
		{
			name: "Japanese search query",
			url:  "/?q=%E3%83%97%E3%83%AD%E3%82%B0%E3%83%A9%E3%83%9F%E3%83%B3%E3%82%B0", // "プログラミング"
			mockSearch: func(query string, limit, offset int) ([]*domain.Post, error) {
				if query != "プログラミング" {
					t.Errorf("expected query 'プログラミング', got %s", query)
				}
				publishedAt := time.Now()
				return []*domain.Post{
					{
						ID:          1,
						Title:       "プログラミング入門",
						Slug:        "programming-intro",
						Content:     "プログラミングの基礎",
						Status:      domain.PostStatusPublished,
						PublishedAt: &publishedAt,
					},
				}, nil
			},
			mockCount: func(query string) (int, error) {
				return 1, nil
			},
			expectedStatus: http.StatusOK,
			containsText: []string{
				"Search results for \"<span class=\"font-medium\">プログラミング</span>\"",
				"<mark>プログラミング</mark>入門", // Highlighted title
			},
		},
		{
			name: "Search results are 0",
			url:  "/?q=notfound",
			mockSearch: func(query string, limit, offset int) ([]*domain.Post, error) {
				return []*domain.Post{}, nil
			},
			mockCount: func(query string) (int, error) {
				return 0, nil
			},
			expectedStatus: http.StatusOK,
			containsText: []string{
				"No posts found matching \"notfound\"",
				`value="notfound"`, // Value remains in search box
			},
			notContains: []string{
				"No posts yet.", // Not the normal empty message
			},
		},
		{
			name: "Search + pagination (page 1)",
			url:  "/?q=Go&page=1",
			mockSearch: func(query string, limit, offset int) ([]*domain.Post, error) {
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
			mockCount: func(query string) (int, error) {
				return 50, nil
			},
			expectedStatus: http.StatusOK,
			containsText: []string{
				"/?page=2&q=Go", // Search query is included in pagination
			},
		},
		{
			name: "Search + pagination (page 2)",
			url:  "/?q=Go&page=2",
			mockSearch: func(query string, limit, offset int) ([]*domain.Post, error) {
				if limit != 21 || offset != 20 {
					t.Errorf("expected limit=21, offset=20, got limit=%d, offset=%d", limit, offset)
				}
				posts := make([]*domain.Post, 10)
				publishedAt := time.Now()
				for i := 0; i < 10; i++ {
					posts[i] = &domain.Post{
						ID:          int64(i + 21),
						Title:       "Go記事" + strconv.Itoa(i+21),
						Slug:        "go-post-" + strconv.Itoa(i+21),
						Content:     "内容" + strconv.Itoa(i+21),
						Status:      domain.PostStatusPublished,
						Tags:        "Go",
						PublishedAt: &publishedAt,
					}
				}
				return posts, nil
			},
			mockCount: func(query string) (int, error) {
				return 30, nil
			},
			expectedStatus: http.StatusOK,
			containsText: []string{
				"/?page=1&q=Go",
			},
			notContains: []string{
				"/?page=3", // No next page
			},
		},
		{
			name: "Search error",
			url:  "/?q=error",
			mockSearch: func(query string, limit, offset int) ([]*domain.Post, error) {
				return nil, fmt.Errorf("database error")
			},
			mockCount: func(query string) (int, error) {
				return 0, nil
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
				searchPublishedPostsFunc: tt.mockSearch,
				countSearchPublishedFunc: tt.mockCount,
			}
			router := NewRouterWithTemplates(mockService, nil, nil, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize, testPostsPerPage)

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

func TestHandleHome_SearchQueryLimit(t *testing.T) {
	t.Run("Error when search query is too long", func(t *testing.T) {
		mockService := &mockPostService{}
		router := NewRouterWithTemplates(mockService, nil, nil, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize, testPostsPerPage)

		// 201-character query (limit is 200 characters)
		longQuery := strings.Repeat("a", 201)
		req := httptest.NewRequest(http.MethodGet, "/?q="+longQuery, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("Search query of exactly 200 characters is allowed", func(t *testing.T) {
		mockService := &mockPostService{
			searchPublishedPostsFunc: func(query string, limit, offset int) ([]*domain.Post, error) {
				return []*domain.Post{}, nil
			},
			countSearchPublishedFunc: func(query string) (int, error) {
				return 0, nil
			},
		}
		router := NewRouterWithTemplates(mockService, nil, nil, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize, testPostsPerPage)

		// 200-character query (exactly at the limit)
		exactQuery := strings.Repeat("a", 200)
		req := httptest.NewRequest(http.MethodGet, "/?q="+exactQuery, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}
	})
}

func TestFormatDateWithTZ_Integration(t *testing.T) {
	// Integration test with templates: Verify correct date format is included in actual HTML response
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

		router := NewRouterWithTemplates(mockService, nil, nil, false, nil, "Test Blog", testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize, testPostsPerPage)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		body := w.Body.String()

		// Verify timezone abbreviation and UTC offset are included
		// Note: Due to sync.Once caching, timezone may differ depending on test execution order
		// The + sign is escaped to HTML entity (&#43;)
		// The - sign is output as-is
		datePattern := regexp.MustCompile(`>\d{4}-\d{2}-\d{2} \([A-Z]+, UTC(&#43;|-)\d+\)</time>`)
		if !datePattern.MatchString(body) {
			t.Errorf("expected body to contain date with timezone and UTC offset pattern, but it didn't. Body: %s", body)
		}

		// The title attribute also contains a similar pattern
		titlePattern := regexp.MustCompile(`title="\d{4}-\d{2}-\d{2} \d{2}:\d{2} \([A-Z]+, UTC(&#43;|-)\d+\)"`)
		if !titlePattern.MatchString(body) {
			t.Errorf("expected body to contain title with timezone and UTC offset pattern, but it didn't. Body: %s", body)
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

		router := NewRouterWithTemplates(mockService, nil, nil, false, nil, "Test Blog", testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize, testPostsPerPage)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		body := w.Body.String()

		// Verify timezone abbreviation and UTC offset are included
		// The + sign is escaped to HTML entity (&#43;)
		datePattern := regexp.MustCompile(`>\d{4}-\d{2}-\d{2} \([A-Z]+, UTC(&#43;|-)\d+\)</time>`)
		if !datePattern.MatchString(body) {
			t.Errorf("expected body to contain date with timezone and UTC offset pattern, but it didn't. Body: %s", body)
		}

		// The title attribute also contains a similar pattern
		titlePattern := regexp.MustCompile(`title="\d{4}-\d{2}-\d{2} \d{2}:\d{2} \([A-Z]+, UTC(&#43;|-)\d+\)"`)
		if !titlePattern.MatchString(body) {
			t.Errorf("expected body to contain title with timezone and UTC offset pattern, but it didn't. Body: %s", body)
		}
	})
}

func TestHighlightQuery(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		query    string
		expected string
	}{
		{
			name:     "Normal highlight",
			text:     "Go言語入門",
			query:    "Go",
			expected: "<mark>Go</mark>言語入門",
		},
		{
			name:     "Case insensitive",
			text:     "Go言語入門",
			query:    "go",
			expected: "<mark>Go</mark>言語入門",
		},
		{
			name:     "Multiple matches",
			text:     "GoでGoを学ぶ",
			query:    "Go",
			expected: "<mark>Go</mark>で<mark>Go</mark>を学ぶ",
		},
		{
			name:     "Empty query",
			text:     "テスト文字列",
			query:    "",
			expected: "テスト文字列",
		},
		{
			name:     "HTML escape",
			text:     "<script>alert('xss')</script>",
			query:    "script",
			expected: "&lt;<mark>script</mark>&gt;alert(&#39;xss&#39;)&lt;/<mark>script</mark>&gt;",
		},
		{
			name:     "Escape regex special characters",
			text:     "test (括弧) test",
			query:    "(括弧)",
			expected: "test <mark>(括弧)</mark> test",
		},
		{
			name:     "Japanese search",
			text:     "プログラミング入門ガイド",
			query:    "プログラミング",
			expected: "<mark>プログラミング</mark>入門ガイド",
		},
		{
			name:     "No match",
			text:     "Hello World",
			query:    "Python",
			expected: "Hello World",
		},
		{
			name:     "Mixed text",
			text:     "GoとPythonで開発",
			query:    "Python",
			expected: "Goと<mark>Python</mark>で開発",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := highlightQuery(tt.text, tt.query)
			if string(result) != tt.expected {
				t.Errorf("highlightQuery() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestHandleHome_SearchHighlight(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		mockSearch     func(query string, limit, offset int) ([]*domain.Post, error)
		containsText   []string
		notContains    []string
	}{
		{
			name: "Highlight is applied to title and body",
			url:  "/?q=Go",
			mockSearch: func(query string, limit, offset int) ([]*domain.Post, error) {
				publishedAt := time.Now()
				return []*domain.Post{
					{
						ID:          1,
						Title:       "Go言語入門",
						Slug:        "go-intro",
						Content:     "Goの基本的な使い方を解説します",
						Status:      domain.PostStatusPublished,
						PublishedAt: &publishedAt,
					},
				}, nil
			},
			containsText: []string{
				"<mark>Go</mark>言語入門",            // Highlight in title
				"<mark>Go</mark>の基本的な使い方を解説します", // Highlight in body too
			},
		},
		{
			name: "Highlight with Japanese query",
			url:  "/?q=%E3%83%97%E3%83%AD%E3%82%B0%E3%83%A9%E3%83%9F%E3%83%B3%E3%82%B0", // プログラミング
			mockSearch: func(query string, limit, offset int) ([]*domain.Post, error) {
				publishedAt := time.Now()
				return []*domain.Post{
					{
						ID:          1,
						Title:       "プログラミング入門",
						Slug:        "programming-intro",
						Content:     "プログラミングを始めよう",
						Status:      domain.PostStatusPublished,
						PublishedAt: &publishedAt,
					},
				}, nil
			},
			containsText: []string{
				"<mark>プログラミング</mark>入門",   // Highlight in title
				"<mark>プログラミング</mark>を始めよう", // Highlight in body too
			},
		},
		{
			name: "No highlight when no query",
			url:  "/",
			mockSearch: func(query string, limit, offset int) ([]*domain.Post, error) {
				publishedAt := time.Now()
				return []*domain.Post{
					{
						ID:          1,
						Title:       "Go言語入門",
						Slug:        "go-intro",
						Content:     "Goの基本的な使い方を解説します",
						Status:      domain.PostStatusPublished,
						PublishedAt: &publishedAt,
					},
				}, nil
			},
			containsText: []string{
				">Go言語入門</a>", // Title as-is (no mark tag)
			},
			notContains: []string{
				"<mark>", // No mark tag
			},
		},
		{
			name: "Highlight in full text display for long body",
			url:  "/?q=Go",
			mockSearch: func(query string, limit, offset int) ([]*domain.Post, error) {
				publishedAt := time.Now()
				// Content exceeding 200 characters
				longContent := "Goは効率的なプログラミング言語です。" + strings.Repeat("この文章は長いテストコンテンツです。", 20)
				return []*domain.Post{
					{
						ID:          1,
						Title:       "Go言語の紹介",
						Slug:        "go-intro",
						Content:     longContent,
						Status:      domain.PostStatusPublished,
						PublishedAt: &publishedAt,
					},
				}, nil
			},
			containsText: []string{
				"<mark>Go</mark>言語の紹介",        // Highlight in title
				"<mark>Go</mark>は効率的なプログラミング言語です", // Highlight in body too
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockPostService{
				searchPublishedPostsFunc: tt.mockSearch,
				getPublishedPostsFunc: func(limit, offset int) ([]*domain.Post, error) {
					// This is called when there's no query
					return tt.mockSearch("", limit, offset)
				},
			}
			router := NewRouterWithTemplates(mockService, nil, nil, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize, testPostsPerPage)

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
			}

			body := w.Body.String()
			for _, text := range tt.containsText {
				if !strings.Contains(body, text) {
					t.Errorf("expected body to contain %q, got:\n%s", text, body)
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

func TestCustom404Page(t *testing.T) {
	tests := []struct {
		name         string
		blogTitle    string
		url          string
		setup        func(*mockPostService)
		containsText []string
	}{
		{
			name:      "Access to non-existent URL",
			blogTitle: "goblog",
			url:       "/not_exists",
			setup:     func(m *mockPostService) {},
			containsText: []string{
				"<!DOCTYPE html>",
				"<title>Page Not Found - goblog</title>",
				"404",
				"Page Not Found",
				"The page you're looking for doesn't exist or has been moved.",
				`href="/"`,
				"Go Home",
				`href="/posts"`,
				"View Posts",
			},
		},
		{
			name:      "Non-existent URL with deep path",
			blogTitle: "goblog",
			url:       "/foo/bar/baz",
			setup:     func(m *mockPostService) {},
			containsText: []string{
				"404",
				"Page Not Found",
			},
		},
		{
			name:      "404 page when post is not found",
			blogTitle: "goblog",
			url:       "/posts/non-existent-post",
			setup: func(m *mockPostService) {
				m.getPostBySlugFunc = func(slug string) (*domain.Post, error) {
					return nil, nil
				}
			},
			containsText: []string{
				"<!DOCTYPE html>",
				"<title>Page Not Found - goblog</title>",
				"404",
				"Page Not Found",
				"The page you're looking for doesn't exist or has been moved.",
				`href="/"`,
				"Go Home",
				`href="/posts"`,
				"View Posts",
				"&copy; 2025 goblog</span>",
			},
		},
		{
			name:      "404 page with custom blog title",
			blogTitle: "My Tech Blog",
			url:       "/posts/not-found",
			setup: func(m *mockPostService) {
				m.getPostBySlugFunc = func(slug string) (*domain.Post, error) {
					return nil, nil
				}
			},
			containsText: []string{
				"<title>Page Not Found - My Tech Blog</title>",
				"404",
				"Page Not Found",
				">My Tech Blog</a>",
				"&copy; 2025 My Tech Blog</span>",
			},
		},
		{
			name:      "404 page with Japanese blog title",
			blogTitle: "テストブログ",
			url:       "/posts/missing",
			setup: func(m *mockPostService) {
				m.getPostBySlugFunc = func(slug string) (*domain.Post, error) {
					return nil, nil
				}
			},
			containsText: []string{
				"<title>Page Not Found - テストブログ</title>",
				">テストブログ</a>",
				"&copy; 2025 テストブログ</span>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockPostService{}
			tt.setup(mockService)

			router := NewRouterWithTemplates(mockService, nil, nil, false, nil, tt.blogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize, testPostsPerPage)

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusNotFound {
				t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
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
		})
	}
}

func TestPinnedPostsInHeader(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		pinnedPosts    []*domain.Post
		containsText   []string
		notContains    []string
	}{
		{
			name: "Pinned posts are displayed in header",
			pinnedPosts: []*domain.Post{
				{
					ID:          1,
					Title:       "自己紹介",
					Slug:        "about",
					Status:      domain.PostStatusPublished,
					IsPinned:    true,
					PublishedAt: &now,
				},
				{
					ID:          2,
					Title:       "お問い合わせ",
					Slug:        "contact",
					Status:      domain.PostStatusPublished,
					IsPinned:    true,
					PublishedAt: &now,
				},
			},
			containsText: []string{
				`href="/posts/about"`,
				">自己紹介</a>",
				`href="/posts/contact"`,
				">お問い合わせ</a>",
			},
		},
		{
			name:        "When there are no pinned posts",
			pinnedPosts: []*domain.Post{},
			containsText: []string{
				`href="/tags"`,
			},
			notContains: []string{
				"自己紹介",
				"お問い合わせ",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockPostService{
				getPublishedPostsFunc: func(limit, offset int) ([]*domain.Post, error) {
					return []*domain.Post{}, nil
				},
				getPinnedPostsFunc: func() ([]*domain.Post, error) {
					return tt.pinnedPosts, nil
				},
			}

			router := NewRouterWithTemplates(mockService, nil, nil, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize, testPostsPerPage)

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

			for _, text := range tt.notContains {
				if strings.Contains(body, text) {
					t.Errorf("expected body NOT to contain %q", text)
				}
			}
		})
	}
}

func TestLayout_SearchFormInHeader(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		containsText []string
		notContains  []string
	}{
		{
			name: "search form is present in the header on the home page",
			url:  "/",
			containsText: []string{
				`<form action="/" method="GET"`,
				`name="q"`,
				`placeholder="Search posts..."`,
			},
			notContains: []string{
				`<a href="/posts"`, // legacy nav link should be gone
			},
		},
		{
			name: "search form pre-fills with the current query",
			url:  "/?q=Go",
			containsText: []string{
				`value="Go"`,
				`<form action="/" method="GET"`,
			},
		},
		{
			name: "search form appears on the tags page too (global header)",
			url:  "/tags",
			containsText: []string{
				`<form action="/" method="GET"`,
				`name="q"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockPostService{
				getPublishedPostsFunc: func(limit, offset int) ([]*domain.Post, error) {
					return []*domain.Post{}, nil
				},
				getPublishedTagsFunc: func() (map[string]int, error) {
					return map[string]int{}, nil
				},
			}
			router := NewRouterWithTemplates(mockService, nil, nil, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize, testPostsPerPage)
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d", w.Code)
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

func TestPostsRedirectsToHome(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		wantStatus   int
		wantLocation string
	}{
		{"bare /posts redirects to /", "/posts", http.StatusMovedPermanently, "/"},
		{"/posts/ trailing slash also redirects", "/posts/", http.StatusMovedPermanently, "/"},
		{"/posts?page=2 preserves page", "/posts?page=2", http.StatusMovedPermanently, "/?page=2"},
		{"/posts?q=foo preserves query", "/posts?q=foo", http.StatusMovedPermanently, "/?q=foo"},
		{"/posts?q=foo&page=3 preserves both", "/posts?q=foo&page=3", http.StatusMovedPermanently, "/?q=foo&page=3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockPostService{}
			router := NewRouterWithTemplates(mockService, nil, nil, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize, testPostsPerPage)
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
			got := w.Header().Get("Location")
			if got != tt.wantLocation {
				t.Errorf("expected Location %q, got %q", tt.wantLocation, got)
			}
		})
	}
}

func TestPostDetailNotAffectedByRedirect(t *testing.T) {
	// /posts/{slug} must still be served as a post detail, not redirected.
	publishedAt := time.Now()
	mockService := &mockPostService{
		getPostBySlugFunc: func(slug string) (*domain.Post, error) {
			if slug != "live-post" {
				t.Errorf("unexpected slug: %q", slug)
			}
			return &domain.Post{
				ID:          1,
				Title:       "Live Post",
				Slug:        "live-post",
				Status:      domain.PostStatusPublished,
				Content:     "body",
				PublishedAt: &publishedAt,
			}, nil
		},
	}
	router := NewRouterWithTemplates(mockService, nil, nil, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize, testPostsPerPage)
	req := httptest.NewRequest(http.MethodGet, "/posts/live-post", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK for /posts/{slug}, got %d", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "" {
		t.Errorf("expected no Location header, got %q", loc)
	}
}

func TestSearchInput_EscapesQueryValue(t *testing.T) {
	mockService := &mockPostService{
		searchPublishedPostsFunc: func(query string, limit, offset int) ([]*domain.Post, error) {
			return []*domain.Post{}, nil
		},
	}
	router := NewRouterWithTemplates(mockService, nil, nil, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize, testPostsPerPage)

	req := httptest.NewRequest(http.MethodGet, "/?q=%3Cscript%3Ealert(1)%3C%2Fscript%3E", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	body := w.Body.String()

	if strings.Contains(body, "<script>alert(1)</script>") {
		t.Errorf("unescaped <script> tag found in body — XSS regression. Body: %s", body)
	}
	if !strings.Contains(body, `value="&lt;script&gt;alert(1)&lt;/script&gt;"`) {
		t.Errorf("expected HTML-escaped value attribute in search input, body: %s", body)
	}
}

func TestHandleHome_NoIndexForSearchAndDeepPages(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		wantNoIndexTag bool
	}{
		{"home page 1 no query is indexable", "/", false},
		{"page 2 emits noindex", "/?page=2", true},
		{"search results emit noindex", "/?q=Go", true},
		{"search + pagination emits noindex", "/?q=Go&page=2", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockPostService{
				getPublishedPostsFunc: func(limit, offset int) ([]*domain.Post, error) {
					return []*domain.Post{}, nil
				},
				searchPublishedPostsFunc: func(query string, limit, offset int) ([]*domain.Post, error) {
					return []*domain.Post{}, nil
				},
			}
			router := NewRouterWithTemplates(mockService, nil, nil, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize, testPostsPerPage)
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			body := w.Body.String()
			hasMeta := strings.Contains(body, `<meta name="robots" content="noindex,follow">`)
			if hasMeta != tt.wantNoIndexTag {
				t.Errorf("noindex meta tag presence = %v, want %v\nbody: %s", hasMeta, tt.wantNoIndexTag, body)
			}
		})
	}
}

func TestNotFound_RendersSearchForm(t *testing.T) {
	mockService := &mockPostService{}
	router := NewRouterWithTemplates(mockService, nil, nil, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize, testPostsPerPage)

	req := httptest.NewRequest(http.MethodGet, "/this-route-does-not-exist", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, `<form action="/" method="GET"`) {
		t.Errorf("404 page must render the global header search form, body: %s", body)
	}
	if !strings.Contains(body, `name="q"`) {
		t.Errorf("404 page search form is missing the q input, body: %s", body)
	}
}

func TestHandleHome_PostsPerPageIsConfigurable(t *testing.T) {
	const customPerPage = 5

	var gotLimit, gotOffset int
	mockService := &mockPostService{
		getPublishedPostsFunc: func(limit, offset int) ([]*domain.Post, error) {
			gotLimit = limit
			gotOffset = offset
			return []*domain.Post{}, nil
		},
	}
	router := NewRouterWithTemplates(mockService, nil, nil, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize, customPerPage)

	req := httptest.NewRequest(http.MethodGet, "/?page=3", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// hasNext look-ahead: limit should be perPage + 1
	if gotLimit != customPerPage+1 {
		t.Errorf("limit = %d, want %d (perPage+1)", gotLimit, customPerPage+1)
	}
	// offset for page 3 should be (3-1) * perPage = 10
	wantOffset := (3 - 1) * customPerPage
	if gotOffset != wantOffset {
		t.Errorf("offset = %d, want %d", gotOffset, wantOffset)
	}
}

func TestHandleTagPosts_PostsPerPageIsConfigurable(t *testing.T) {
	const customPerPage = 7

	var gotLimit, gotOffset int
	mockService := &mockPostService{
		getPublishedPostsByTagFunc: func(tag string, limit, offset int) ([]*domain.Post, error) {
			gotLimit = limit
			gotOffset = offset
			return []*domain.Post{}, nil
		},
	}
	router := NewRouterWithTemplates(mockService, nil, nil, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize, customPerPage)

	req := httptest.NewRequest(http.MethodGet, "/tags/Go?page=2", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if gotLimit != customPerPage+1 {
		t.Errorf("limit = %d, want %d (perPage+1)", gotLimit, customPerPage+1)
	}
	wantOffset := (2 - 1) * customPerPage
	if gotOffset != wantOffset {
		t.Errorf("offset = %d, want %d", gotOffset, wantOffset)
	}
}
