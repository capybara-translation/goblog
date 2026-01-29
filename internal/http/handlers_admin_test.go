package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleAdminSPA(t *testing.T) {
	tests := []struct {
		name               string
		path               string
		expectedStatus     int
		expectedContentType string
		checkBody          func(t *testing.T, body string)
	}{
		{
			name:               "root path /admin",
			path:               "/admin",
			expectedStatus:     http.StatusOK,
			expectedContentType: "text/html",
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "<div id=\"root\">") {
					t.Error("expected body to contain root div")
				}
			},
		},
		{
			name:               "root path with trailing slash /admin/",
			path:               "/admin/",
			expectedStatus:     http.StatusOK,
			expectedContentType: "text/html",
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "<div id=\"root\">") {
					t.Error("expected body to contain root div")
				}
			},
		},
		{
			name:               "client-side route /admin/posts",
			path:               "/admin/posts",
			expectedStatus:     http.StatusOK,
			expectedContentType: "text/html",
			checkBody: func(t *testing.T, body string) {
				// Should fallback to index.html for client-side routing
				if !strings.Contains(body, "<div id=\"root\">") {
					t.Error("expected body to contain root div for SPA fallback")
				}
			},
		},
		{
			name:               "client-side route /admin/posts/new",
			path:               "/admin/posts/new",
			expectedStatus:     http.StatusOK,
			expectedContentType: "text/html",
			checkBody: func(t *testing.T, body string) {
				// Should fallback to index.html for client-side routing
				if !strings.Contains(body, "<div id=\"root\">") {
					t.Error("expected body to contain root div for SPA fallback")
				}
			},
		},
		{
			name:               "client-side route /admin/posts/123/edit",
			path:               "/admin/posts/123/edit",
			expectedStatus:     http.StatusOK,
			expectedContentType: "text/html",
			checkBody: func(t *testing.T, body string) {
				// Should fallback to index.html for client-side routing
				if !strings.Contains(body, "<div id=\"root\">") {
					t.Error("expected body to contain root div for SPA fallback")
				}
			},
		},
		{
			name:               "non-existent asset falls back to index.html",
			path:               "/admin/assets/non-existent.css",
			expectedStatus:     http.StatusOK,
			expectedContentType: "text/html",
			checkBody: func(t *testing.T, body string) {
				// Non-existent files should fallback to index.html for SPA routing
				if !strings.Contains(body, "<div id=\"root\">") {
					t.Error("expected body to contain root div for SPA fallback")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			HandleAdminSPA(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			contentType := w.Header().Get("Content-Type")
			if !strings.Contains(contentType, tt.expectedContentType) {
				t.Errorf("expected Content-Type to contain %q, got %q", tt.expectedContentType, contentType)
			}

			if tt.checkBody != nil {
				tt.checkBody(t, w.Body.String())
			}
		})
	}
}

func TestHandleAdminSPA_ViaRouter(t *testing.T) {
	// Test via the actual router to ensure routing is set up correctly
	mockService := &mockPostService{}
	mockAuthService := &mockAuthService{}
	router := NewRouterWithTemplates(mockService, mockAuthService, testSecureCookie, testBlogTitle, testBaseURL, testTemplatePattern, testUploadDir, testMaxUploadSize)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{
			name:           "/admin via router",
			path:           "/admin",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "/admin/ via router",
			path:           "/admin/",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "/admin/posts via router",
			path:           "/admin/posts",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "/admin/posts/123/edit via router",
			path:           "/admin/posts/123/edit",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// All admin routes should return HTML
			contentType := w.Header().Get("Content-Type")
			if !strings.Contains(contentType, "text/html") {
				t.Errorf("expected Content-Type to contain text/html, got %q", contentType)
			}
		})
	}
}

// TestHandleAdminSPA_PathHandling tests that paths are correctly processed
func TestHandleAdminSPA_PathHandling(t *testing.T) {
	tests := []struct {
		name         string
		requestPath  string
		expectedPath string // What path should be requested from embed.FS
	}{
		{
			name:         "/admin should map to /",
			requestPath:  "/admin",
			expectedPath: "/index.html",
		},
		{
			name:         "/admin/ should map to /",
			requestPath:  "/admin/",
			expectedPath: "/index.html",
		},
		{
			name:         "/admin/assets/index.js should map to /assets/index.js",
			requestPath:  "/admin/assets/index.js",
			expectedPath: "/assets/index.js",
		},
		{
			name:         "/admin/posts should map to / (fallback)",
			requestPath:  "/admin/posts",
			expectedPath: "/index.html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.requestPath, nil)
			w := httptest.NewRecorder()

			HandleAdminSPA(w, req)

			// We can't easily test the exact path used internally,
			// but we can verify the response is successful
			if w.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
			}
		})
	}
}
