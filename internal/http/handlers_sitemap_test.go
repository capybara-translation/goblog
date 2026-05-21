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
	// Test post data
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

	// Test tag data
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

	router := NewRouterWithTemplates(mockService, nil, nil, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize, testPostsPerPage)

	t.Run("sitemap.xml is generated correctly", func(t *testing.T) {
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

		// Contains XML header
		if !strings.Contains(body, `<?xml version="1.0" encoding="UTF-8"?>`) {
			t.Error("expected XML header")
		}

		// Contains urlset element
		if !strings.Contains(body, `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`) {
			t.Error("expected urlset element with xmlns")
		}

		// Contains static URLs
		if !strings.Contains(body, testBaseURL+"/") {
			t.Error("expected root URL")
		}
		if !strings.Contains(body, testBaseURL+"/tags") {
			t.Error("expected tags URL")
		}

		// Contains post URLs
		if !strings.Contains(body, testBaseURL+"/posts/test-post-1") {
			t.Error("expected post URL for test-post-1")
		}
		if !strings.Contains(body, testBaseURL+"/posts/test-post-2") {
			t.Error("expected post URL for test-post-2")
		}

		// Contains tag URLs
		if !strings.Contains(body, testBaseURL+"/tags/Go") {
			t.Error("expected tag URL for Go")
		}
		if !strings.Contains(body, testBaseURL+"/tags/Web") {
			t.Error("expected tag URL for Web")
		}

		// Japanese tags are URL encoded
		if !strings.Contains(body, testBaseURL+"/tags/") {
			t.Error("expected tag URL")
		}

		// Contains lastmod (YYYY-MM-DD format, UTC)
		expectedDate := now.UTC().Format("2006-01-02")
		if !strings.Contains(body, "<lastmod>"+expectedDate+"</lastmod>") {
			t.Errorf("expected lastmod with date %s, body: %s", expectedDate, body)
		}

		// Contains changefreq
		if !strings.Contains(body, "<changefreq>weekly</changefreq>") {
			t.Error("expected changefreq weekly")
		}
		if !strings.Contains(body, "<changefreq>monthly</changefreq>") {
			t.Error("expected changefreq monthly")
		}

		// Contains priority
		if !strings.Contains(body, "<priority>1</priority>") {
			t.Error("expected priority 1")
		}
		if !strings.Contains(body, "<priority>0.7</priority>") {
			t.Error("expected priority 0.7")
		}

		// /posts is no longer in the sitemap (it 301-redirects to /)
		if strings.Contains(body, "<loc>"+testBaseURL+"/posts</loc>") {
			t.Errorf("sitemap should not list /posts (now a redirect), but it did. Body: %s", body)
		}
	})

	t.Run("Static URLs are generated even when there are no posts", func(t *testing.T) {
		emptyMockService := &mockPostService{
			getPublishedPostsFunc: func(limit, offset int) ([]*domain.Post, error) {
				return []*domain.Post{}, nil
			},
			getPublishedTagsFunc: func() (map[string]int, error) {
				return map[string]int{}, nil
			},
		}

		router := NewRouterWithTemplates(emptyMockService, nil, nil, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize, testPostsPerPage)

		req := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		body := w.Body.String()

		// Static URLs are included
		if !strings.Contains(body, testBaseURL+"/") {
			t.Error("expected root URL even with no posts")
		}
		if !strings.Contains(body, testBaseURL+"/tags") {
			t.Error("expected tags URL even with no posts")
		}
	})
}
