package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/capybara-translation/goblog/internal/service"
)

// mockPostServiceForAPI は PostService のモック実装です（API用）
type mockPostServiceForAPI struct {
	getAllPostsFunc   func(status *domain.PostStatus, limit, offset int) ([]*domain.Post, error)
	getPostByIDFunc   func(id int64) (*domain.Post, error)
	createPostFunc    func(title, slug, content, tags string) (*domain.Post, error)
	updatePostFunc    func(id int64, title, slug, content, tags string) (*domain.Post, error)
	deletePostFunc    func(id int64) error
	publishPostFunc   func(id int64) (*domain.Post, error)
	unpublishPostFunc func(id int64) (*domain.Post, error)
}

func (m *mockPostServiceForAPI) GetPublishedPosts(limit, offset int) ([]*domain.Post, error) {
	return []*domain.Post{}, nil
}

func (m *mockPostServiceForAPI) GetPostBySlug(slug string) (*domain.Post, error) {
	return nil, nil
}

func (m *mockPostServiceForAPI) GetAllPosts(status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
	if m.getAllPostsFunc != nil {
		return m.getAllPostsFunc(status, limit, offset)
	}
	return []*domain.Post{}, nil
}

func (m *mockPostServiceForAPI) GetPostByID(id int64) (*domain.Post, error) {
	if m.getPostByIDFunc != nil {
		return m.getPostByIDFunc(id)
	}
	return nil, nil
}

func (m *mockPostServiceForAPI) CreatePost(title, slug, content, tags string) (*domain.Post, error) {
	if m.createPostFunc != nil {
		return m.createPostFunc(title, slug, content, tags)
	}
	return nil, nil
}

func (m *mockPostServiceForAPI) UpdatePost(id int64, title, slug, content, tags string) (*domain.Post, error) {
	if m.updatePostFunc != nil {
		return m.updatePostFunc(id, title, slug, content, tags)
	}
	return nil, nil
}

func (m *mockPostServiceForAPI) PublishPost(id int64) (*domain.Post, error) {
	if m.publishPostFunc != nil {
		return m.publishPostFunc(id)
	}
	return nil, nil
}

func (m *mockPostServiceForAPI) UnpublishPost(id int64) (*domain.Post, error) {
	if m.unpublishPostFunc != nil {
		return m.unpublishPostFunc(id)
	}
	return nil, nil
}

func (m *mockPostServiceForAPI) DeletePost(id int64) error {
	if m.deletePostFunc != nil {
		return m.deletePostFunc(id)
	}
	return nil
}

var _ service.PostService = (*mockPostServiceForAPI)(nil)

func TestHandleHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	HandleHealth(w, req)

	// ステータスコードを確認
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Content-Typeを確認
	contentType := w.Header().Get("Content-Type")
	expected := "application/json"
	if contentType != expected {
		t.Errorf("expected Content-Type %q, got %q", expected, contentType)
	}

	// JSONレスポンスを確認
	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("expected status %q, got %q", "ok", response["status"])
	}
}

func TestHandleHealth_ViaRouter(t *testing.T) {
	// ルーター経由でもテスト
	mockService := &mockPostServiceForAPI{}
	router := NewRouterWithTemplates(mockService, nil, false, "../view/templates/*.html")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("expected status %q, got %q", "ok", response["status"])
	}
}
func TestHandleGetPosts(t *testing.T) {
	mockService := &mockPostServiceForAPI{}
	router := NewRouterWithTemplates(mockService, nil, false, "../view/templates/*.html")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var posts []*domain.Post
	if err := json.Unmarshal(w.Body.Bytes(), &posts); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}
}

func TestHandleGetPost_NotFound(t *testing.T) {
	mockService := &mockPostServiceForAPI{}
	router := NewRouterWithTemplates(mockService, nil, false, "../view/templates/*.html")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts/999", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	if response.Error != "Post not found" {
		t.Errorf("expected error %q, got %q", "Post not found", response.Error)
	}
}

func TestHandleGetPosts_WithFilters(t *testing.T) {
	publishedStatus := domain.PostStatusPublished
	draftStatus := domain.PostStatusDraft

	tests := []struct {
		name           string
		queryParams    string
		mockFunc       func(status *domain.PostStatus, limit, offset int) ([]*domain.Post, error)
		expectedStatus int
		expectedCount  int
	}{
		{
			name:        "全記事取得",
			queryParams: "",
			mockFunc: func(status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
				return []*domain.Post{
					{ID: 1, Title: "Post 1", Status: domain.PostStatusPublished},
					{ID: 2, Title: "Post 2", Status: domain.PostStatusDraft},
				}, nil
			},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:        "公開記事のみ",
			queryParams: "?status=published",
			mockFunc: func(status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
				if status != nil && *status == publishedStatus {
					return []*domain.Post{
						{ID: 1, Title: "Post 1", Status: domain.PostStatusPublished},
					}, nil
				}
				return []*domain.Post{}, nil
			},
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:        "下書きのみ",
			queryParams: "?status=draft",
			mockFunc: func(status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
				if status != nil && *status == draftStatus {
					return []*domain.Post{
						{ID: 2, Title: "Post 2", Status: domain.PostStatusDraft},
					}, nil
				}
				return []*domain.Post{}, nil
			},
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockPostServiceForAPI{
				getAllPostsFunc: tt.mockFunc,
			}
			router := NewRouterWithTemplates(mockService, nil, false, "../view/templates/*.html")

			req := httptest.NewRequest(http.MethodGet, "/api/v1/posts"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var posts []*domain.Post
			if err := json.Unmarshal(w.Body.Bytes(), &posts); err != nil {
				t.Fatalf("failed to parse JSON response: %v", err)
			}

			if len(posts) != tt.expectedCount {
				t.Errorf("expected %d posts, got %d", tt.expectedCount, len(posts))
			}
		})
	}
}

func TestHandleGetPost_Success(t *testing.T) {
	mockService := &mockPostServiceForAPI{
		getPostByIDFunc: func(id int64) (*domain.Post, error) {
			if id == 1 {
				return &domain.Post{
					ID:      1,
					Title:   "Test Post",
					Slug:    "test-post",
					Content: "Test content",
					Status:  domain.PostStatusPublished,
				}, nil
			}
			return nil, nil
		},
	}
	router := NewRouterWithTemplates(mockService, nil, false, "../view/templates/*.html")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts/1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var post domain.Post
	if err := json.Unmarshal(w.Body.Bytes(), &post); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	if post.ID != 1 {
		t.Errorf("expected post ID 1, got %d", post.ID)
	}
	if post.Title != "Test Post" {
		t.Errorf("expected title 'Test Post', got %q", post.Title)
	}
}

func TestHandleCreatePost_Success(t *testing.T) {
	mockService := &mockPostServiceForAPI{
		createPostFunc: func(title, slug, content, tags string) (*domain.Post, error) {
			return &domain.Post{
				ID:      1,
				Title:   title,
				Slug:    slug,
				Content: content,
				Tags:    tags,
				Status:  domain.PostStatusDraft,
			}, nil
		},
	}
	router := NewRouterWithTemplates(mockService, nil, false, "../view/templates/*.html")

	body := `{"title":"New Post","slug":"new-post","content":"Content here","tags":"tag1,tag2"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/posts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var post domain.Post
	if err := json.Unmarshal(w.Body.Bytes(), &post); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	if post.Title != "New Post" {
		t.Errorf("expected title 'New Post', got %q", post.Title)
	}
	if post.Status != domain.PostStatusDraft {
		t.Errorf("expected status 'draft', got %q", post.Status)
	}
}

func TestHandleCreatePost_InvalidJSON(t *testing.T) {
	mockService := &mockPostServiceForAPI{}
	router := NewRouterWithTemplates(mockService, nil, false, "../view/templates/*.html")

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/posts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	if response.Error != "Invalid request body" {
		t.Errorf("expected error 'Invalid request body', got %q", response.Error)
	}
}

func TestHandleUpdatePost_Success(t *testing.T) {
	mockService := &mockPostServiceForAPI{
		updatePostFunc: func(id int64, title, slug, content, tags string) (*domain.Post, error) {
			if id == 1 {
				return &domain.Post{
					ID:      id,
					Title:   title,
					Slug:    slug,
					Content: content,
					Tags:    tags,
					Status:  domain.PostStatusDraft,
				}, nil
			}
			return nil, nil
		},
	}
	router := NewRouterWithTemplates(mockService, nil, false, "../view/templates/*.html")

	body := `{"title":"Updated Post","slug":"updated-post","content":"Updated content","tags":"new-tags"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/posts/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var post domain.Post
	if err := json.Unmarshal(w.Body.Bytes(), &post); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	if post.Title != "Updated Post" {
		t.Errorf("expected title 'Updated Post', got %q", post.Title)
	}
}

func TestHandleDeletePost_Success(t *testing.T) {
	mockService := &mockPostServiceForAPI{
		deletePostFunc: func(id int64) error {
			if id == 1 {
				return nil
			}
			return nil
		},
	}
	router := NewRouterWithTemplates(mockService, nil, false, "../view/templates/*.html")

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/posts/1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}

	if w.Body.Len() != 0 {
		t.Errorf("expected empty body, got %q", w.Body.String())
	}
}

func TestHandlePublishPost_Success(t *testing.T) {
	mockService := &mockPostServiceForAPI{
		publishPostFunc: func(id int64) (*domain.Post, error) {
			if id == 1 {
				return &domain.Post{
					ID:     1,
					Title:  "Test Post",
					Status: domain.PostStatusPublished,
				}, nil
			}
			return nil, nil
		},
	}
	router := NewRouterWithTemplates(mockService, nil, false, "../view/templates/*.html")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/posts/1/publish", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var post domain.Post
	if err := json.Unmarshal(w.Body.Bytes(), &post); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	if post.Status != domain.PostStatusPublished {
		t.Errorf("expected status 'published', got %q", post.Status)
	}
}

func TestHandleUnpublishPost_Success(t *testing.T) {
	mockService := &mockPostServiceForAPI{
		unpublishPostFunc: func(id int64) (*domain.Post, error) {
			if id == 1 {
				return &domain.Post{
					ID:     1,
					Title:  "Test Post",
					Status: domain.PostStatusDraft,
				}, nil
			}
			return nil, nil
		},
	}
	router := NewRouterWithTemplates(mockService, nil, false, "../view/templates/*.html")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/posts/1/unpublish", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var post domain.Post
	if err := json.Unmarshal(w.Body.Bytes(), &post); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	if post.Status != domain.PostStatusDraft {
		t.Errorf("expected status 'draft', got %q", post.Status)
	}
}

func TestHandleGetPost_InvalidID(t *testing.T) {
	mockService := &mockPostServiceForAPI{}
	router := NewRouterWithTemplates(mockService, nil, false, "../view/templates/*.html")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts/invalid", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	if response.Error != "Invalid post ID" {
		t.Errorf("expected error 'Invalid post ID', got %q", response.Error)
	}
}

func TestHandleGetPosts_LimitMax(t *testing.T) {
	callCount := 0
	mockService := &mockPostServiceForAPI{
		getAllPostsFunc: func(status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
			callCount++
			// limit が 200 以下に制限されていることを確認
			if limit > 200 {
				t.Errorf("expected limit to be capped at 200, got %d", limit)
			}
			return []*domain.Post{}, nil
		},
	}
	router := NewRouterWithTemplates(mockService, nil, false, "../view/templates/*.html")

	tests := []struct {
		name          string
		limitParam    string
		expectedLimit int
	}{
		{
			name:          "limit=300 は 200 に制限される",
			limitParam:    "?limit=300",
			expectedLimit: 200,
		},
		{
			name:          "limit=1000 は 200 に制限される",
			limitParam:    "?limit=1000",
			expectedLimit: 200,
		},
		{
			name:          "limit=50 はそのまま",
			limitParam:    "?limit=50",
			expectedLimit: 50,
		},
		{
			name:          "limit=200 はそのまま",
			limitParam:    "?limit=200",
			expectedLimit: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receivedLimit := 0
			mockService.getAllPostsFunc = func(status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
				receivedLimit = limit
				return []*domain.Post{}, nil
			}

			req := httptest.NewRequest(http.MethodGet, "/api/v1/posts"+tt.limitParam, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
			}

			if receivedLimit != tt.expectedLimit {
				t.Errorf("expected service to receive limit %d, got %d", tt.expectedLimit, receivedLimit)
			}
		})
	}
}
