package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/capybara-translation/goblog/internal/service"
)

const testSessionID = "test-session-id"

// mockPostServiceForAPI is a mock implementation of PostService (for API)
type mockPostServiceForAPI struct {
	getAllPostsFunc             func(status *domain.PostStatus, limit, offset int) ([]*domain.Post, error)
	getAllPostsByTagFunc        func(tag string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error)
	getAllTagsFunc              func(status *domain.PostStatus) (map[string]int, error)
	getPostByIDFunc             func(id int64) (*domain.Post, error)
	createPostFunc              func(title, slug, content, tags string, isPinned bool) (*domain.Post, error)
	updatePostFunc              func(id int64, title, slug, content, tags string, isPinned bool) (*domain.Post, error)
	deletePostFunc              func(id int64) error
	publishPostFunc             func(id int64) (*domain.Post, error)
	unpublishPostFunc           func(id int64) (*domain.Post, error)
	countPostsFunc              func(status *domain.PostStatus) (int, error)
	countPostsByTagFunc         func(tag string, status *domain.PostStatus) (int, error)
	searchPostsFunc             func(query string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error)
	countSearchPostsFunc        func(query string, status *domain.PostStatus) (int, error)
	searchPublishedPostsFunc    func(query string, limit, offset int) ([]*domain.Post, error)
	countSearchPublishedFunc    func(query string) (int, error)
	getPinnedPostsFunc          func() ([]*domain.Post, error)
	setPinnedFunc               func(id int64, pinned bool) (*domain.Post, error)
}

func (m *mockPostServiceForAPI) GetPublishedPosts(limit, offset int) ([]*domain.Post, error) {
	return []*domain.Post{}, nil
}

func (m *mockPostServiceForAPI) GetPublishedPostsByTag(tag string, limit, offset int) ([]*domain.Post, error) {
	return []*domain.Post{}, nil
}

func (m *mockPostServiceForAPI) GetAllPostsByTag(tag string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
	if m.getAllPostsByTagFunc != nil {
		return m.getAllPostsByTagFunc(tag, status, limit, offset)
	}
	return []*domain.Post{}, nil
}

func (m *mockPostServiceForAPI) GetPublishedTags() (map[string]int, error) {
	return map[string]int{}, nil
}

func (m *mockPostServiceForAPI) GetAllTags(status *domain.PostStatus) (map[string]int, error) {
	if m.getAllTagsFunc != nil {
		return m.getAllTagsFunc(status)
	}
	return map[string]int{}, nil
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

func (m *mockPostServiceForAPI) CreatePost(title, slug, content, tags string, isPinned bool) (*domain.Post, error) {
	if m.createPostFunc != nil {
		return m.createPostFunc(title, slug, content, tags, isPinned)
	}
	return nil, nil
}

func (m *mockPostServiceForAPI) UpdatePost(id int64, title, slug, content, tags string, isPinned bool) (*domain.Post, error) {
	if m.updatePostFunc != nil {
		return m.updatePostFunc(id, title, slug, content, tags, isPinned)
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

func (m *mockPostServiceForAPI) CountPosts(status *domain.PostStatus) (int, error) {
	if m.countPostsFunc != nil {
		return m.countPostsFunc(status)
	}
	return 0, nil
}

func (m *mockPostServiceForAPI) CountPostsByTag(tag string, status *domain.PostStatus) (int, error) {
	if m.countPostsByTagFunc != nil {
		return m.countPostsByTagFunc(tag, status)
	}
	return 0, nil
}

func (m *mockPostServiceForAPI) SearchPosts(query string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
	if m.searchPostsFunc != nil {
		return m.searchPostsFunc(query, status, limit, offset)
	}
	return []*domain.Post{}, nil
}

func (m *mockPostServiceForAPI) CountSearchPosts(query string, status *domain.PostStatus) (int, error) {
	if m.countSearchPostsFunc != nil {
		return m.countSearchPostsFunc(query, status)
	}
	return 0, nil
}

func (m *mockPostServiceForAPI) SearchPublishedPosts(query string, limit, offset int) ([]*domain.Post, error) {
	if m.searchPublishedPostsFunc != nil {
		return m.searchPublishedPostsFunc(query, limit, offset)
	}
	return []*domain.Post{}, nil
}

func (m *mockPostServiceForAPI) CountSearchPublishedPosts(query string) (int, error) {
	if m.countSearchPublishedFunc != nil {
		return m.countSearchPublishedFunc(query)
	}
	return 0, nil
}

func (m *mockPostServiceForAPI) GetPinnedPosts() ([]*domain.Post, error) {
	if m.getPinnedPostsFunc != nil {
		return m.getPinnedPostsFunc()
	}
	return []*domain.Post{}, nil
}

func (m *mockPostServiceForAPI) SetPinned(id int64, pinned bool) (*domain.Post, error) {
	if m.setPinnedFunc != nil {
		return m.setPinnedFunc(id, pinned)
	}
	return nil, nil
}

var _ service.PostService = (*mockPostServiceForAPI)(nil)

// mockAuthServiceForAPI is a mock implementation of AuthService (for API)
type mockAuthServiceForAPI struct {
	loginFunc            func(username, password, ipAddress string) (string, error)
	logoutFunc           func(sessionID string) error
	getUserBySessionFunc func(sessionID string) (*domain.User, error)
	createUserFunc       func(username, password string) (*domain.User, error)
}

func (m *mockAuthServiceForAPI) Login(username, password, ipAddress string) (string, error) {
	if m.loginFunc != nil {
		return m.loginFunc(username, password, ipAddress)
	}
	return "", nil
}

func (m *mockAuthServiceForAPI) Logout(sessionID string) error {
	if m.logoutFunc != nil {
		return m.logoutFunc(sessionID)
	}
	return nil
}

func (m *mockAuthServiceForAPI) GetUserBySession(sessionID string) (*domain.User, error) {
	if m.getUserBySessionFunc != nil {
		return m.getUserBySessionFunc(sessionID)
	}
	return nil, nil
}

func (m *mockAuthServiceForAPI) CreateUser(username, password string) (*domain.User, error) {
	if m.createUserFunc != nil {
		return m.createUserFunc(username, password)
	}
	return nil, nil
}

var _ service.AuthService = (*mockAuthServiceForAPI)(nil)

// createTestAuthService creates an authentication service for testing.
// Sessions with testSessionID are treated as valid.
func createTestAuthService() *mockAuthServiceForAPI {
	return &mockAuthServiceForAPI{
		getUserBySessionFunc: func(sessionID string) (*domain.User, error) {
			if sessionID == testSessionID {
				return &domain.User{
					ID:        1,
					Username:  "testuser",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}, nil
			}
			return nil, nil
		},
	}
}

// addSessionCookie adds an authentication session Cookie to the request
func addSessionCookie(req *http.Request) {
	req.AddCookie(&http.Cookie{
		Name:  sessionCookieName,
		Value: testSessionID,
	})
}

// addCSRFToken adds a CSRF token to the request (Cookie and Header)
func addCSRFToken(req *http.Request, token string) {
	req.AddCookie(&http.Cookie{
		Name:  csrfCookieName,
		Value: token,
	})
	req.Header.Set(csrfHeaderName, token)
}

// addAuthAndCSRF adds authentication Cookie and CSRF token to the request
func addAuthAndCSRF(req *http.Request) {
	addSessionCookie(req)
	addCSRFToken(req, "test-csrf-token")
}

func TestHandleHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	HandleHealth(w, req)

	// Verify status code
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify Content-Type
	contentType := w.Header().Get("Content-Type")
	expected := "application/json"
	if contentType != expected {
		t.Errorf("expected Content-Type %q, got %q", expected, contentType)
	}

	// Verify JSON response
	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("expected status %q, got %q", "ok", response["status"])
	}
}

func TestHandleHealth_ViaRouter(t *testing.T) {
	// Also test via router
	mockPostService := &mockPostServiceForAPI{}
	mockAuthService := createTestAuthService()
	router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

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

// Unauthenticated tests
func TestUnauthenticated_Endpoints(t *testing.T) {
	mockPostService := &mockPostServiceForAPI{}
	mockAuthService := createTestAuthService()
	router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{"GET /api/v1/posts", http.MethodGet, "/api/v1/posts", ""},
		{"GET /api/v1/posts/{id}", http.MethodGet, "/api/v1/posts/1", ""},
		{"POST /api/v1/posts", http.MethodPost, "/api/v1/posts", `{"title":"Test","slug":"test","content":"content","tags":""}`},
		{"PUT /api/v1/posts/{id}", http.MethodPut, "/api/v1/posts/1", `{"title":"Test","slug":"test","content":"content","tags":""}`},
		{"DELETE /api/v1/posts/{id}", http.MethodDelete, "/api/v1/posts/1", ""},
		{"POST /api/v1/posts/{id}/publish", http.MethodPost, "/api/v1/posts/1/publish", ""},
		{"POST /api/v1/posts/{id}/unpublish", http.MethodPost, "/api/v1/posts/1/unpublish", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
			}

			var response ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to parse JSON response: %v", err)
			}

			if response.Error != "Authentication required" {
				t.Errorf("expected error %q, got %q", "Authentication required", response.Error)
			}
		})
	}
}

func TestHandleGetPosts(t *testing.T) {
	mockPostService := &mockPostServiceForAPI{}
	mockAuthService := createTestAuthService()
	router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts", nil)
	addSessionCookie(req)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response PostsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}
}

// TestHandleGetPosts_Empty tests that an empty array is returned instead of null when there are 0 posts
func TestHandleGetPosts_Empty(t *testing.T) {
	mockPostService := &mockPostServiceForAPI{
		// Reproduce Go behavior by returning nil
		getAllPostsFunc: func(status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
			return nil, nil
		},
		countPostsFunc: func(status *domain.PostStatus) (int, error) {
			return 0, nil
		},
	}
	mockAuthService := createTestAuthService()
	router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts", nil)
	addSessionCookie(req)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify JSON response contains an empty array, not null
	body := w.Body.String()
	if !strings.Contains(body, `"posts":[]`) {
		t.Errorf("expected posts to be empty array, got: %s", body)
	}

	var response PostsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	if response.Posts == nil {
		t.Error("expected posts to be empty slice, got nil")
	}
	if len(response.Posts) != 0 {
		t.Errorf("expected 0 posts, got %d", len(response.Posts))
	}
	if response.Total != 0 {
		t.Errorf("expected total to be 0, got %d", response.Total)
	}
}

func TestHandleGetPost_NotFound(t *testing.T) {
	mockPostService := &mockPostServiceForAPI{}
	mockAuthService := createTestAuthService()
	router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts/999", nil)
	addSessionCookie(req)
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
			name:        "Get all posts",
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
			name:        "Published posts only",
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
			name:        "Draft posts only",
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
			mockPostService := &mockPostServiceForAPI{
				getAllPostsFunc: tt.mockFunc,
			}
			mockAuthService := createTestAuthService()
			router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/posts"+tt.queryParams, nil)
			addSessionCookie(req)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var response PostsResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to parse JSON response: %v", err)
			}

			if len(response.Posts) != tt.expectedCount {
				t.Errorf("expected %d posts, got %d", tt.expectedCount, len(response.Posts))
			}
		})
	}
}

func TestHandleGetPost_Success(t *testing.T) {
	mockPostService := &mockPostServiceForAPI{
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
	mockAuthService := createTestAuthService()
	router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts/1", nil)
	addSessionCookie(req)
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
	mockPostService := &mockPostServiceForAPI{
		createPostFunc: func(title, slug, content, tags string, isPinned bool) (*domain.Post, error) {
			return &domain.Post{
				ID:       1,
				Title:    title,
				Slug:     slug,
				Content:  content,
				Tags:     tags,
				Status:   domain.PostStatusDraft,
				IsPinned: isPinned,
			}, nil
		},
	}
	mockAuthService := createTestAuthService()
	router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

	body := `{"title":"New Post","slug":"new-post","content":"Content here","tags":"tag1,tag2"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/posts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	addAuthAndCSRF(req)
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
	mockPostService := &mockPostServiceForAPI{}
	mockAuthService := createTestAuthService()
	router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/posts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	addAuthAndCSRF(req)
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

func TestHandleCreatePost_Validation(t *testing.T) {
	mockPostService := &mockPostServiceForAPI{}
	mockAuthService := createTestAuthService()
	router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

	tests := []struct {
		name          string
		body          string
		expectedError string
	}{
		{
			name:          "Empty title",
			body:          `{"title":"","slug":"test-slug","content":"Content"}`,
			expectedError: "title is required",
		},
		{
			name:          "Title with only whitespace",
			body:          `{"title":"   ","slug":"test-slug","content":"Content"}`,
			expectedError: "title is required",
		},
		{
			name:          "Empty slug",
			body:          `{"title":"Test Title","slug":"","content":"Content"}`,
			expectedError: "slug is required",
		},
		{
			name:          "Slug with only whitespace",
			body:          `{"title":"Test Title","slug":"   ","content":"Content"}`,
			expectedError: "slug is required",
		},
		{
			name:          "Slug contains whitespace",
			body:          `{"title":"Test Title","slug":"test slug","content":"Content"}`,
			expectedError: "slug must not contain whitespace",
		},
		{
			name:          "Slug contains uppercase letters",
			body:          `{"title":"Test Title","slug":"Test-Slug","content":"Content"}`,
			expectedError: "slug must contain only lowercase letters, numbers, and hyphens",
		},
		{
			name:          "Slug contains Japanese characters",
			body:          `{"title":"Test Title","slug":"テスト","content":"Content"}`,
			expectedError: "slug must contain only lowercase letters, numbers, and hyphens",
		},
		{
			name:          "Slug contains underscore",
			body:          `{"title":"Test Title","slug":"test_slug","content":"Content"}`,
			expectedError: "slug must contain only lowercase letters, numbers, and hyphens",
		},
		{
			name:          "Slug starts with hyphen",
			body:          `{"title":"Test Title","slug":"-test-slug","content":"Content"}`,
			expectedError: "slug must contain only lowercase letters, numbers, and hyphens",
		},
		{
			name:          "Slug ends with hyphen",
			body:          `{"title":"Test Title","slug":"test-slug-","content":"Content"}`,
			expectedError: "slug must contain only lowercase letters, numbers, and hyphens",
		},
		{
			name:          "Slug contains consecutive hyphens",
			body:          `{"title":"Test Title","slug":"test--slug","content":"Content"}`,
			expectedError: "slug must contain only lowercase letters, numbers, and hyphens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/posts", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			addAuthAndCSRF(req)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
			}

			var response ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to parse JSON response: %v", err)
			}

			if response.Error != tt.expectedError {
				t.Errorf("expected error %q, got %q", tt.expectedError, response.Error)
			}
		})
	}
}

func TestHandleUpdatePost_Validation(t *testing.T) {
	mockPostService := &mockPostServiceForAPI{}
	mockAuthService := createTestAuthService()
	router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

	tests := []struct {
		name          string
		body          string
		expectedError string
	}{
		{
			name:          "Empty title",
			body:          `{"title":"","slug":"test-slug","content":"Content"}`,
			expectedError: "title is required",
		},
		{
			name:          "Empty slug",
			body:          `{"title":"Test Title","slug":"","content":"Content"}`,
			expectedError: "slug is required",
		},
		{
			name:          "Slug is not URL-safe",
			body:          `{"title":"Test Title","slug":"Test Slug!","content":"Content"}`,
			expectedError: "slug must not contain whitespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/api/v1/posts/1", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			addAuthAndCSRF(req)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
			}

			var response ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to parse JSON response: %v", err)
			}

			if response.Error != tt.expectedError {
				t.Errorf("expected error %q, got %q", tt.expectedError, response.Error)
			}
		})
	}
}

func TestHandleUpdatePost_Success(t *testing.T) {
	mockPostService := &mockPostServiceForAPI{
		updatePostFunc: func(id int64, title, slug, content, tags string, isPinned bool) (*domain.Post, error) {
			if id == 1 {
				return &domain.Post{
					ID:       id,
					Title:    title,
					Slug:     slug,
					Content:  content,
					Tags:     tags,
					Status:   domain.PostStatusDraft,
					IsPinned: isPinned,
				}, nil
			}
			return nil, nil
		},
	}
	mockAuthService := createTestAuthService()
	router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

	body := `{"title":"Updated Post","slug":"updated-post","content":"Updated content","tags":"new-tags"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/posts/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	addAuthAndCSRF(req)
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
	mockPostService := &mockPostServiceForAPI{
		deletePostFunc: func(id int64) error {
			if id == 1 {
				return nil
			}
			return nil
		},
	}
	mockAuthService := createTestAuthService()
	router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/posts/1", nil)
	addAuthAndCSRF(req)
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
	mockPostService := &mockPostServiceForAPI{
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
	mockAuthService := createTestAuthService()
	router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/posts/1/publish", nil)
	addAuthAndCSRF(req)
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
	mockPostService := &mockPostServiceForAPI{
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
	mockAuthService := createTestAuthService()
	router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/posts/1/unpublish", nil)
	addAuthAndCSRF(req)
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
	mockPostService := &mockPostServiceForAPI{}
	mockAuthService := createTestAuthService()
	router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts/invalid", nil)
	addSessionCookie(req)
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
	mockPostService := &mockPostServiceForAPI{
		getAllPostsFunc: func(status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
			callCount++
			// Verify that limit is capped at 200 or less
			if limit > 200 {
				t.Errorf("expected limit to be capped at 200, got %d", limit)
			}
			return []*domain.Post{}, nil
		},
	}
	mockAuthService := createTestAuthService()
	router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

	tests := []struct {
		name          string
		limitParam    string
		expectedLimit int
	}{
		{
			name:          "limit=300 is capped at 200",
			limitParam:    "?limit=300",
			expectedLimit: 200,
		},
		{
			name:          "limit=1000 is capped at 200",
			limitParam:    "?limit=1000",
			expectedLimit: 200,
		},
		{
			name:          "limit=50 remains unchanged",
			limitParam:    "?limit=50",
			expectedLimit: 50,
		},
		{
			name:          "limit=200 remains unchanged",
			limitParam:    "?limit=200",
			expectedLimit: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receivedLimit := 0
			mockPostService.getAllPostsFunc = func(status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
				receivedLimit = limit
				return []*domain.Post{}, nil
			}

			req := httptest.NewRequest(http.MethodGet, "/api/v1/posts"+tt.limitParam, nil)
			addSessionCookie(req)
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

func TestHandleGetPosts_WithTagFilter(t *testing.T) {
	mockAuthService := createTestAuthService()

	t.Run("Filter by tag", func(t *testing.T) {
		receivedTag := ""
		mockPostService := &mockPostServiceForAPI{
			getAllPostsByTagFunc: func(tag string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
				receivedTag = tag
				return []*domain.Post{
					{ID: 1, Title: "Go Post", Tags: "Go"},
				}, nil
			},
		}

		router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/posts?tag=Go", nil)
		addSessionCookie(req)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		if receivedTag != "Go" {
			t.Errorf("expected tag 'Go', got '%s'", receivedTag)
		}
	})

	t.Run("Filter by both tag and status", func(t *testing.T) {
		receivedTag := ""
		var receivedStatus *domain.PostStatus

		mockPostService := &mockPostServiceForAPI{
			getAllPostsByTagFunc: func(tag string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
				receivedTag = tag
				receivedStatus = status
				return []*domain.Post{}, nil
			},
		}

		router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/posts?tag=React&status=published", nil)
		addSessionCookie(req)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		if receivedTag != "React" {
			t.Errorf("expected tag 'React', got '%s'", receivedTag)
		}

		if receivedStatus == nil || *receivedStatus != domain.PostStatusPublished {
			t.Errorf("expected status 'published', got %v", receivedStatus)
		}
	})
}

func TestHandleGetPosts_InvalidStatus(t *testing.T) {
	mockPostService := &mockPostServiceForAPI{}
	mockAuthService := createTestAuthService()
	router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

	tests := []struct {
		name           string
		status         string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Invalid status value",
			status:         "invalid",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid status",
		},
		{
			name:           "Typo in status",
			status:         "publised",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid status",
		},
		{
			name:           "Uppercase status",
			status:         "PUBLISHED",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/posts?status="+tt.status, nil)
			addSessionCookie(req)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var response map[string]string
			json.NewDecoder(w.Body).Decode(&response)

			if !strings.Contains(response["error"], tt.expectedError) {
				t.Errorf("expected error to contain '%s', got '%s'", tt.expectedError, response["error"])
			}
		})
	}
}

func TestHandleGetTags(t *testing.T) {
	mockAuthService := createTestAuthService()

	t.Run("Get all tags (verify sorting)", func(t *testing.T) {
		mockPostService := &mockPostServiceForAPI{
			getAllTagsFunc: func(status *domain.PostStatus) (map[string]int, error) {
				return map[string]int{
					"Go":     10,
					"React":  8,
					"Docker": 5,
					"Python": 3,
				}, nil
			},
		}
		router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/tags", nil)
		addSessionCookie(req)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var tags []map[string]interface{}
		json.NewDecoder(w.Body).Decode(&tags)

		if len(tags) != 4 {
			t.Errorf("expected 4 tags, got %d", len(tags))
		}

		// Verify sort order: descending by post count, ascending by name if counts are equal
		if tags[0]["name"] != "Go" || int(tags[0]["count"].(float64)) != 10 {
			t.Errorf("expected first tag to be Go with count 10, got %v", tags[0])
		}

		if tags[1]["name"] != "React" || int(tags[1]["count"].(float64)) != 8 {
			t.Errorf("expected second tag to be React with count 8, got %v", tags[1])
		}

		if tags[2]["name"] != "Docker" || int(tags[2]["count"].(float64)) != 5 {
			t.Errorf("expected third tag to be Docker with count 5, got %v", tags[2])
		}

		if tags[3]["name"] != "Python" || int(tags[3]["count"].(float64)) != 3 {
			t.Errorf("expected fourth tag to be Python with count 3, got %v", tags[3])
		}
	})

	t.Run("Get tags with status filter", func(t *testing.T) {
		var receivedStatus *domain.PostStatus
		mockPostService := &mockPostServiceForAPI{
			getAllTagsFunc: func(status *domain.PostStatus) (map[string]int, error) {
				receivedStatus = status
				return map[string]int{"Go": 5}, nil
			},
		}
		router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/tags?status=published", nil)
		addSessionCookie(req)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		if receivedStatus == nil || *receivedStatus != domain.PostStatusPublished {
			t.Errorf("expected status 'published', got %v", receivedStatus)
		}
	})

	t.Run("Error on invalid status value", func(t *testing.T) {
		mockPostService := &mockPostServiceForAPI{}
		router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/tags?status=invalid", nil)
		addSessionCookie(req)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}

		var response map[string]string
		json.NewDecoder(w.Body).Decode(&response)

		if !strings.Contains(response["error"], "invalid status") {
			t.Errorf("expected error to contain 'invalid status', got '%s'", response["error"])
		}
	})

	t.Run("Service error propagation", func(t *testing.T) {
		mockPostService := &mockPostServiceForAPI{
			getAllTagsFunc: func(status *domain.PostStatus) (map[string]int, error) {
				return nil, fmt.Errorf("database error")
			},
		}
		router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/tags", nil)
		addSessionCookie(req)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
		}
	})

	t.Run("Empty array when no tags exist", func(t *testing.T) {
		mockPostService := &mockPostServiceForAPI{
			getAllTagsFunc: func(status *domain.PostStatus) (map[string]int, error) {
				return map[string]int{}, nil
			},
		}
		router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/tags", nil)
		addSessionCookie(req)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var tags []map[string]interface{}
		json.NewDecoder(w.Body).Decode(&tags)

		if len(tags) != 0 {
			t.Errorf("expected 0 tags, got %d", len(tags))
		}
	})
}

func TestHandleGetTags_Sorting(t *testing.T) {
	// Test tag sorting logic in more detail
	mockAuthService := createTestAuthService()
	mockPostService := &mockPostServiceForAPI{
		getAllTagsFunc: func(status *domain.PostStatus) (map[string]int, error) {
			// Data containing tags with the same count
			return map[string]int{
				"Zebra":  5,
				"Apple":  5,
				"Docker": 10,
				"Beta":   5,
				"Go":     10,
			}, nil
		},
	}

	router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tags", nil)
	addSessionCookie(req)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var tags []map[string]interface{}
	json.NewDecoder(w.Body).Decode(&tags)

	// Expected order:
	// 1. Docker (count: 10)
	// 2. Go (count: 10)
	// 3. Apple (count: 5)
	// 4. Beta (count: 5)
	// 5. Zebra (count: 5)

	expectedOrder := []struct {
		name  string
		count int
	}{
		{"Docker", 10},
		{"Go", 10},
		{"Apple", 5},
		{"Beta", 5},
		{"Zebra", 5},
	}

	for i, expected := range expectedOrder {
		if i >= len(tags) {
			t.Fatalf("not enough tags in response, expected at least %d", i+1)
		}

		if tags[i]["name"] != expected.name {
			t.Errorf("position %d: expected name '%s', got '%s'", i, expected.name, tags[i]["name"])
		}

		if int(tags[i]["count"].(float64)) != expected.count {
			t.Errorf("position %d: expected count %d, got %v", i, expected.count, tags[i]["count"])
		}
	}
}

func TestHandleGetPosts_WithSearchQuery(t *testing.T) {
	mockAuthService := createTestAuthService()

	t.Run("Get posts with search query", func(t *testing.T) {
		receivedQuery := ""
		mockPostService := &mockPostServiceForAPI{
			searchPostsFunc: func(query string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
				receivedQuery = query
				return []*domain.Post{
					{ID: 1, Title: "Go入門", Content: "Goの基本的な使い方"},
					{ID: 2, Title: "Go応用", Content: "Goの応用テクニック"},
				}, nil
			},
			countSearchPostsFunc: func(query string, status *domain.PostStatus) (int, error) {
				return 2, nil
			},
		}
		router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/posts?q=Go", nil)
		addSessionCookie(req)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		if receivedQuery != "Go" {
			t.Errorf("expected query 'Go', got '%s'", receivedQuery)
		}

		var response PostsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if len(response.Posts) != 2 {
			t.Errorf("expected 2 posts, got %d", len(response.Posts))
		}

		if response.Total != 2 {
			t.Errorf("expected total 2, got %d", response.Total)
		}
	})

	t.Run("Search query + status filter", func(t *testing.T) {
		receivedQuery := ""
		var receivedStatus *domain.PostStatus

		mockPostService := &mockPostServiceForAPI{
			searchPostsFunc: func(query string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
				receivedQuery = query
				receivedStatus = status
				return []*domain.Post{
					{ID: 1, Title: "Go入門", Status: domain.PostStatusPublished},
				}, nil
			},
			countSearchPostsFunc: func(query string, status *domain.PostStatus) (int, error) {
				return 1, nil
			},
		}
		router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/posts?q=Go&status=published", nil)
		addSessionCookie(req)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		if receivedQuery != "Go" {
			t.Errorf("expected query 'Go', got '%s'", receivedQuery)
		}

		if receivedStatus == nil || *receivedStatus != domain.PostStatusPublished {
			t.Errorf("expected status published, got %v", receivedStatus)
		}
	})

	t.Run("Japanese search query", func(t *testing.T) {
		receivedQuery := ""
		mockPostService := &mockPostServiceForAPI{
			searchPostsFunc: func(query string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
				receivedQuery = query
				return []*domain.Post{
					{ID: 1, Title: "プログラミング入門", Content: "プログラミングの基礎"},
				}, nil
			},
			countSearchPostsFunc: func(query string, status *domain.PostStatus) (int, error) {
				return 1, nil
			},
		}
		router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/posts?q=%E3%83%97%E3%83%AD%E3%82%B0%E3%83%A9%E3%83%9F%E3%83%B3%E3%82%B0", nil) // "プログラミング"
		addSessionCookie(req)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		if receivedQuery != "プログラミング" {
			t.Errorf("expected query 'プログラミング', got '%s'", receivedQuery)
		}
	})

	t.Run("When search results are 0", func(t *testing.T) {
		mockPostService := &mockPostServiceForAPI{
			searchPostsFunc: func(query string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
				return []*domain.Post{}, nil
			},
			countSearchPostsFunc: func(query string, status *domain.PostStatus) (int, error) {
				return 0, nil
			},
		}
		router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/posts?q=notfound", nil)
		addSessionCookie(req)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response PostsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if len(response.Posts) != 0 {
			t.Errorf("expected 0 posts, got %d", len(response.Posts))
		}

		if response.Total != 0 {
			t.Errorf("expected total 0, got %d", response.Total)
		}
	})

	t.Run("Search query + pagination", func(t *testing.T) {
		receivedLimit := 0
		receivedOffset := 0

		mockPostService := &mockPostServiceForAPI{
			searchPostsFunc: func(query string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
				receivedLimit = limit
				receivedOffset = offset
				return []*domain.Post{
					{ID: 11, Title: "Go Post 11"},
				}, nil
			},
			countSearchPostsFunc: func(query string, status *domain.PostStatus) (int, error) {
				return 25, nil
			},
		}
		router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/posts?q=Go&limit=10&offset=10", nil)
		addSessionCookie(req)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		if receivedLimit != 10 {
			t.Errorf("expected limit 10, got %d", receivedLimit)
		}

		if receivedOffset != 10 {
			t.Errorf("expected offset 10, got %d", receivedOffset)
		}

		var response PostsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if response.Total != 25 {
			t.Errorf("expected total 25, got %d", response.Total)
		}
	})

	t.Run("Search service error", func(t *testing.T) {
		mockPostService := &mockPostServiceForAPI{
			searchPostsFunc: func(query string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
				return nil, fmt.Errorf("database error")
			},
		}
		router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/posts?q=Go", nil)
		addSessionCookie(req)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
		}

		var response ErrorResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if !strings.Contains(response.Error, "Failed to get posts") {
			t.Errorf("expected error to contain 'Failed to get posts', got '%s'", response.Error)
		}
	})

	t.Run("Search count service error", func(t *testing.T) {
		mockPostService := &mockPostServiceForAPI{
			searchPostsFunc: func(query string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
				return []*domain.Post{{ID: 1, Title: "Test"}}, nil
			},
			countSearchPostsFunc: func(query string, status *domain.PostStatus) (int, error) {
				return 0, fmt.Errorf("count error")
			},
		}
		router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/posts?q=Go", nil)
		addSessionCookie(req)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
		}
	})

	t.Run("Error when search query is too long", func(t *testing.T) {
		mockPostService := &mockPostServiceForAPI{}
		router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

		// 201-character query (limit is 200 characters)
		longQuery := strings.Repeat("a", 201)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/posts?q="+longQuery, nil)
		addSessionCookie(req)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}

		var response ErrorResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if !strings.Contains(response.Error, "search query too long") {
			t.Errorf("expected error to contain 'search query too long', got '%s'", response.Error)
		}
	})

	t.Run("Search query of exactly 200 characters is allowed", func(t *testing.T) {
		mockPostService := &mockPostServiceForAPI{
			searchPostsFunc: func(query string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
				return []*domain.Post{}, nil
			},
			countSearchPostsFunc: func(query string, status *domain.PostStatus) (int, error) {
				return 0, nil
			},
		}
		router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

		// 200-character query (exactly at the limit)
		exactQuery := strings.Repeat("a", 200)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/posts?q="+exactQuery, nil)
		addSessionCookie(req)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}
	})
}

func TestHandlePreview(t *testing.T) {
	mockPostService := &mockPostServiceForAPI{}
	mockAuthService := createTestAuthService()
	router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

	t.Run("Convert Markdown to HTML", func(t *testing.T) {
		body := `{"content":"# Hello\n\nThis is **bold** text."}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/markdown/preview", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		addAuthAndCSRF(req)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response PreviewResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse JSON response: %v", err)
		}

		// Verify conversion to HTML
		if !strings.Contains(response.HTML, "<h1") {
			t.Errorf("expected HTML to contain <h1>, got %q", response.HTML)
		}
		if !strings.Contains(response.HTML, "<strong>bold</strong>") {
			t.Errorf("expected HTML to contain <strong>bold</strong>, got %q", response.HTML)
		}
	})

	t.Run("Empty content", func(t *testing.T) {
		body := `{"content":""}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/markdown/preview", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		addAuthAndCSRF(req)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response PreviewResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse JSON response: %v", err)
		}

		// Empty string HTML is returned
		if response.HTML != "" {
			t.Errorf("expected empty HTML, got %q", response.HTML)
		}
	})

	t.Run("Invalid JSON request", func(t *testing.T) {
		body := `{invalid json}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/markdown/preview", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		addAuthAndCSRF(req)
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
	})

	t.Run("XSS sanitization", func(t *testing.T) {
		body := `{"content":"<script>alert('xss')</script>"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/markdown/preview", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		addAuthAndCSRF(req)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response PreviewResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse JSON response: %v", err)
		}

		// Verify script tags are sanitized
		if strings.Contains(response.HTML, "<script>") {
			t.Errorf("expected script tag to be sanitized, got %q", response.HTML)
		}
	})

	t.Run("Syntax highlighting for code blocks", func(t *testing.T) {
		body := "{\"content\":\"```go\\nfunc main() {}\\n```\"}"
		req := httptest.NewRequest(http.MethodPost, "/api/v1/markdown/preview", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		addAuthAndCSRF(req)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response PreviewResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse JSON response: %v", err)
		}

		// Verify pre tag is included
		if !strings.Contains(response.HTML, "<pre") {
			t.Errorf("expected HTML to contain <pre>, got %q", response.HTML)
		}
	})

	t.Run("data-line attribute is added", func(t *testing.T) {
		body := `{"content":"# Hello\n\nParagraph text"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/markdown/preview", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		addAuthAndCSRF(req)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response PreviewResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse JSON response: %v", err)
		}

		// Verify data-line attribute is added
		if !strings.Contains(response.HTML, `data-line="`) {
			t.Errorf("expected HTML to contain data-line attribute, got %q", response.HTML)
		}
		// Verify heading has data-line="0"
		if !strings.Contains(response.HTML, `<h1`) || !strings.Contains(response.HTML, `data-line="0"`) {
			t.Errorf("expected HTML to contain h1 with data-line=\"0\", got %q", response.HTML)
		}
		// Verify paragraph has data-line="2"
		if !strings.Contains(response.HTML, `<p `) || !strings.Contains(response.HTML, `data-line="2"`) {
			t.Errorf("expected HTML to contain p with data-line=\"2\", got %q", response.HTML)
		}
	})

	t.Run("Error when unauthenticated", func(t *testing.T) {
		body := `{"content":"# Test"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/markdown/preview", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		// No authentication
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	t.Run("Error when CSRF token is missing", func(t *testing.T) {
		body := `{"content":"# Test"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/markdown/preview", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		addSessionCookie(req) // Only authentication Cookie, no CSRF token
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
		}
	})
}

func TestHandlePinPost(t *testing.T) {
	t.Run("Pin a post", func(t *testing.T) {
		mockPostService := &mockPostServiceForAPI{
			setPinnedFunc: func(id int64, pinned bool) (*domain.Post, error) {
				if id != 1 {
					t.Errorf("expected id 1, got %d", id)
				}
				if !pinned {
					t.Errorf("expected pinned to be true")
				}
				return &domain.Post{
					ID:       1,
					Title:    "Test Post",
					Slug:     "test-post",
					IsPinned: true,
					Status:   domain.PostStatusPublished,
				}, nil
			},
		}
		mockAuthService := createTestAuthService()
		router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/posts/1/pin", nil)
		addAuthAndCSRF(req)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var post domain.Post
		if err := json.Unmarshal(w.Body.Bytes(), &post); err != nil {
			t.Fatalf("failed to parse JSON response: %v", err)
		}

		if !post.IsPinned {
			t.Errorf("expected is_pinned to be true")
		}
	})

	t.Run("Error on invalid ID", func(t *testing.T) {
		mockPostService := &mockPostServiceForAPI{}
		mockAuthService := createTestAuthService()
		router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/posts/invalid/pin", nil)
		addAuthAndCSRF(req)
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
	})

	t.Run("Service error", func(t *testing.T) {
		mockPostService := &mockPostServiceForAPI{
			setPinnedFunc: func(id int64, pinned bool) (*domain.Post, error) {
				return nil, fmt.Errorf("post not found: %d", id)
			},
		}
		mockAuthService := createTestAuthService()
		router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/posts/999/pin", nil)
		addAuthAndCSRF(req)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
		}
	})

	t.Run("Error when unauthenticated", func(t *testing.T) {
		mockPostService := &mockPostServiceForAPI{}
		mockAuthService := createTestAuthService()
		router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/posts/1/pin", nil)
		// No authentication
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})
}

func TestHandleUnpinPost(t *testing.T) {
	t.Run("Unpin a post", func(t *testing.T) {
		mockPostService := &mockPostServiceForAPI{
			setPinnedFunc: func(id int64, pinned bool) (*domain.Post, error) {
				if id != 1 {
					t.Errorf("expected id 1, got %d", id)
				}
				if pinned {
					t.Errorf("expected pinned to be false")
				}
				return &domain.Post{
					ID:       1,
					Title:    "Test Post",
					Slug:     "test-post",
					IsPinned: false,
					Status:   domain.PostStatusPublished,
				}, nil
			},
		}
		mockAuthService := createTestAuthService()
		router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/posts/1/unpin", nil)
		addAuthAndCSRF(req)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var post domain.Post
		if err := json.Unmarshal(w.Body.Bytes(), &post); err != nil {
			t.Fatalf("failed to parse JSON response: %v", err)
		}

		if post.IsPinned {
			t.Errorf("expected is_pinned to be false")
		}
	})

	t.Run("Error on invalid ID", func(t *testing.T) {
		mockPostService := &mockPostServiceForAPI{}
		mockAuthService := createTestAuthService()
		router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/posts/invalid/unpin", nil)
		addAuthAndCSRF(req)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("Service error", func(t *testing.T) {
		mockPostService := &mockPostServiceForAPI{
			setPinnedFunc: func(id int64, pinned bool) (*domain.Post, error) {
				return nil, fmt.Errorf("post not found: %d", id)
			},
		}
		mockAuthService := createTestAuthService()
		router := NewRouterWithTemplates(mockPostService, nil, mockAuthService, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/posts/999/unpin", nil)
		addAuthAndCSRF(req)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
		}
	})
}
