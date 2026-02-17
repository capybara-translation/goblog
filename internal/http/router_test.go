package http

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNoDirListingFS(t *testing.T) {
	// Create a temporary directory with a test file
	tempDir, err := os.MkdirTemp("", "nodirlisting_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test file
	testContent := []byte("test image content")
	if err := os.WriteFile(filepath.Join(tempDir, "test.jpg"), testContent, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create a subdirectory
	if err := os.MkdirAll(filepath.Join(tempDir, "subdir"), 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	fs := noDirListingFS{http.Dir(tempDir)}
	handler := http.FileServer(fs)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{
			name:           "existing file returns 200",
			path:           "/test.jpg",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "root directory returns 404",
			path:           "/",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "subdirectory returns 404",
			path:           "/subdir/",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "non-existent file returns 404",
			path:           "/nonexistent.jpg",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d (body: %s)", tt.expectedStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestNoDirListingFS_FileContentServed(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "nodirlisting_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testContent := []byte("hello world")
	if err := os.WriteFile(filepath.Join(tempDir, "file.txt"), testContent, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	fs := noDirListingFS{http.Dir(tempDir)}
	handler := http.FileServer(fs)

	req := httptest.NewRequest("GET", "/file.txt", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	if rr.Body.String() != "hello world" {
		t.Errorf("expected body %q, got %q", "hello world", rr.Body.String())
	}
}
