package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/capybara-translation/goblog/internal/service"
)

// mockPostServiceForAPI は PostService のモック実装です（API用）
type mockPostServiceForAPI struct{}

func (m *mockPostServiceForAPI) GetPublishedPosts(limit, offset int) ([]*domain.Post, error) {
	return []*domain.Post{}, nil
}

func (m *mockPostServiceForAPI) GetPostBySlug(slug string) (*domain.Post, error) {
	return nil, nil
}

func (m *mockPostServiceForAPI) GetAllPosts(status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
	return []*domain.Post{}, nil
}

func (m *mockPostServiceForAPI) GetPostByID(id int64) (*domain.Post, error) {
	return nil, nil
}

func (m *mockPostServiceForAPI) CreatePost(title, slug, content, tags string) (*domain.Post, error) {
	return nil, nil
}

func (m *mockPostServiceForAPI) UpdatePost(id int64, title, slug, content, tags string) (*domain.Post, error) {
	return nil, nil
}

func (m *mockPostServiceForAPI) PublishPost(id int64) (*domain.Post, error) {
	return nil, nil
}

func (m *mockPostServiceForAPI) UnpublishPost(id int64) (*domain.Post, error) {
	return nil, nil
}

func (m *mockPostServiceForAPI) DeletePost(id int64) error {
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
	router := NewRouterWithTemplates(mockService, "../view/templates/*.html")

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
