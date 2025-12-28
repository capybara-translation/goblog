package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/capybara-translation/goblog/internal/service"
)

// mockPostService は PostService のモック実装です
type mockPostService struct{}

func (m *mockPostService) GetPublishedPosts(limit, offset int) ([]*domain.Post, error) {
	return []*domain.Post{}, nil
}

func (m *mockPostService) GetPostBySlug(slug string) (*domain.Post, error) {
	return nil, nil
}

func (m *mockPostService) GetAllPosts(status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
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
	mockService := &mockPostService{}
	router := NewRouterWithTemplates(mockService, "../view/templates/*.html")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// ステータスコードを確認
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Content-Typeを確認
	contentType := w.Header().Get("Content-Type")
	expected := "text/html; charset=utf-8"
	if contentType != expected {
		t.Errorf("expected Content-Type %q, got %q", expected, contentType)
	}

	// レスポンスボディを確認（実際のテンプレート内容）
	body := w.Body.String()
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("expected body to contain '<!DOCTYPE html>'")
	}
	if !strings.Contains(body, "ようこそgoblogへ") {
		t.Error("expected body to contain 'ようこそgoblogへ'")
	}
	if !strings.Contains(body, "まだ記事がありません") {
		t.Error("expected body to contain 'まだ記事がありません'")
	}
}

func TestHandlePosts(t *testing.T) {
	mockService := &mockPostService{}
	router := NewRouterWithTemplates(mockService, "../view/templates/*.html")

	req := httptest.NewRequest(http.MethodGet, "/posts", nil)
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

	// レスポンスボディを確認（実際のテンプレート内容）
	body := w.Body.String()
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("expected body to contain '<!DOCTYPE html>'")
	}
	if !strings.Contains(body, "記事一覧") {
		t.Error("expected body to contain '記事一覧'")
	}
	if !strings.Contains(body, "まだ記事がありません") {
		t.Error("expected body to contain 'まだ記事がありません'")
	}
}

func TestHandlePostDetail(t *testing.T) {
	tests := []struct {
		name         string
		slug         string
		expectedSlug string
	}{
		{
			name:         "英語のスラッグ",
			slug:         "hello-world",
			expectedSlug: "hello-world",
		},
		{
			name:         "数字を含むスラッグ",
			slug:         "post-123",
			expectedSlug: "post-123",
		},
		{
			name:         "ハイフン複数",
			slug:         "my-first-go-blog",
			expectedSlug: "my-first-go-blog",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// gorilla/muxを使ってルーティング経由でテスト
			// 直接ハンドラーを呼ぶとmux.Vars()が使えないため、ルーター経由でテスト
			mockService := &mockPostService{}
			router := NewRouterWithTemplates(mockService, "../view/templates/*.html")

			req := httptest.NewRequest(http.MethodGet, "/posts/"+tt.slug, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// 記事が見つからないので404が返る（mockが nil を返すため）
			if w.Code != http.StatusNotFound {
				t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
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
			router := NewRouterWithTemplates(mockService, "../view/templates/*.html")

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
