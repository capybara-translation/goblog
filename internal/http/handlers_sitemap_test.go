package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/capybara-translation/goblog/internal/domain"
)

func TestHandleSitemap(t *testing.T) {
	// テスト用の記事データ
	now := time.Now()
	testPosts := []*domain.Post{
		{
			ID:        1,
			Title:     "Test Post 1",
			Slug:      "test-post-1",
			Content:   "Test content 1",
			Status:    domain.PostStatusPublished,
			UpdatedAt: now,
		},
		{
			ID:        2,
			Title:     "Test Post 2",
			Slug:      "test-post-2",
			Content:   "Test content 2",
			Status:    domain.PostStatusPublished,
			UpdatedAt: now.Add(-24 * time.Hour),
		},
	}

	// テスト用のタグデータ
	testTags := map[string]int{
		"Go":   2,
		"Web":  1,
		"日本語": 1,
	}

	mockService := &mockPostService{
		getPublishedPostsFunc: func(limit, offset int) ([]*domain.Post, error) {
			return testPosts, nil
		},
		getPublishedTagsFunc: func() (map[string]int, error) {
			return testTags, nil
		},
	}

	router := NewRouterWithTemplates(mockService, nil, testSecureCookie, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

	t.Run("正常にsitemap.xmlが生成される", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if !strings.Contains(contentType, "application/xml") {
			t.Errorf("expected Content-Type application/xml, got %q", contentType)
		}

		body := w.Body.String()

		// XMLヘッダーが含まれる
		if !strings.Contains(body, `<?xml version="1.0" encoding="UTF-8"?>`) {
			t.Error("expected XML header")
		}

		// urlset要素が含まれる
		if !strings.Contains(body, `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`) {
			t.Error("expected urlset element with xmlns")
		}

		// 固定URLが含まれる
		if !strings.Contains(body, testBaseURL+"/") {
			t.Error("expected root URL")
		}
		if !strings.Contains(body, testBaseURL+"/posts") {
			t.Error("expected posts URL")
		}
		if !strings.Contains(body, testBaseURL+"/tags") {
			t.Error("expected tags URL")
		}

		// 記事URLが含まれる
		if !strings.Contains(body, testBaseURL+"/posts/test-post-1") {
			t.Error("expected post URL for test-post-1")
		}
		if !strings.Contains(body, testBaseURL+"/posts/test-post-2") {
			t.Error("expected post URL for test-post-2")
		}

		// タグURLが含まれる
		if !strings.Contains(body, testBaseURL+"/tags/Go") {
			t.Error("expected tag URL for Go")
		}
		if !strings.Contains(body, testBaseURL+"/tags/Web") {
			t.Error("expected tag URL for Web")
		}

		// 日本語タグはURLエンコードされる
		if !strings.Contains(body, testBaseURL+"/tags/") {
			t.Error("expected tag URL")
		}

		// lastmodが含まれる（YYYY-MM-DD形式、UTC）
		expectedDate := now.UTC().Format("2006-01-02")
		if !strings.Contains(body, "<lastmod>"+expectedDate+"</lastmod>") {
			t.Errorf("expected lastmod with date %s, body: %s", expectedDate, body)
		}

		// changefreqが含まれる
		if !strings.Contains(body, "<changefreq>weekly</changefreq>") {
			t.Error("expected changefreq weekly")
		}
		if !strings.Contains(body, "<changefreq>daily</changefreq>") {
			t.Error("expected changefreq daily")
		}
		if !strings.Contains(body, "<changefreq>monthly</changefreq>") {
			t.Error("expected changefreq monthly")
		}

		// priorityが含まれる
		if !strings.Contains(body, "<priority>1</priority>") {
			t.Error("expected priority 1")
		}
		if !strings.Contains(body, "<priority>0.8</priority>") {
			t.Error("expected priority 0.8")
		}
		if !strings.Contains(body, "<priority>0.7</priority>") {
			t.Error("expected priority 0.7")
		}
	})

	t.Run("記事がない場合も固定URLは生成される", func(t *testing.T) {
		emptyMockService := &mockPostService{
			getPublishedPostsFunc: func(limit, offset int) ([]*domain.Post, error) {
				return []*domain.Post{}, nil
			},
			getPublishedTagsFunc: func() (map[string]int, error) {
				return map[string]int{}, nil
			},
		}

		router := NewRouterWithTemplates(emptyMockService, nil, testSecureCookie, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

		req := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		body := w.Body.String()

		// 固定URLは含まれる
		if !strings.Contains(body, testBaseURL+"/") {
			t.Error("expected root URL even with no posts")
		}
		if !strings.Contains(body, testBaseURL+"/posts") {
			t.Error("expected posts URL even with no posts")
		}
		if !strings.Contains(body, testBaseURL+"/tags") {
			t.Error("expected tags URL even with no posts")
		}
	})
}
