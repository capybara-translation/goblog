package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleHome(t *testing.T) {
	// リクエストを作成
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// ResponseRecorderを作成（レスポンスを記録）
	w := httptest.NewRecorder()

	// ハンドラーを実行
	HandleHome(w, req)

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

	// レスポンスボディを確認
	body := w.Body.String()
	if !strings.Contains(body, "goblog") {
		t.Error("expected body to contain 'goblog'")
	}
	if !strings.Contains(body, "トップページ") {
		t.Error("expected body to contain 'トップページ'")
	}
}

func TestHandlePosts(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/posts", nil)
	w := httptest.NewRecorder()

	HandlePosts(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	expected := "text/html; charset=utf-8"
	if contentType != expected {
		t.Errorf("expected Content-Type %q, got %q", expected, contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, "記事一覧") {
		t.Error("expected body to contain '記事一覧'")
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
			router := NewRouter()

			req := httptest.NewRequest(http.MethodGet, "/posts/"+tt.slug, nil)
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
			if !strings.Contains(body, "記事詳細") {
				t.Error("expected body to contain '記事詳細'")
			}
			if !strings.Contains(body, tt.expectedSlug) {
				t.Errorf("expected body to contain slug %q, got: %s", tt.expectedSlug, body)
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
			router := NewRouter()

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
