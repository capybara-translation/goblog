package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// /uploads/ is a static read-only asset endpoint. POST/PUT/DELETE etc.
// must be rejected at the router so that an accidental future
// piggyback handler doesn't end up with non-idempotent traffic getting
// the long-lived immutable Cache-Control header. This test wires up
// the full NewRouterWithTemplates path so we catch regressions in
// router.go itself, not just in middleware unit tests.
func TestUploadsRoute_RejectsNonGETHEAD(t *testing.T) {
	mockService := &mockPostService{}
	router := NewRouterWithTemplates(mockService, nil, nil, false, nil, "goblog", "http://localhost:8080", "../view/templates/*.html", "", 5*1024*1024, 20)

	for _, method := range []string{"POST", "PUT", "DELETE", "PATCH"} {
		req := httptest.NewRequest(method, "/uploads/whatever.jpg", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s /uploads/whatever.jpg: expected 405, got %d", method, rec.Code)
		}
	}
}

func TestUploadsCacheControl_AddsImmutableHeader(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	h := uploadsCacheControl(next)

	req := httptest.NewRequest("GET", "/uploads/abc.jpg", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if !called {
		t.Fatalf("expected next handler to be called")
	}
	got := rec.Header().Get("Cache-Control")
	want := "public, max-age=31536000, immutable"
	if got != want {
		t.Errorf("Cache-Control = %q, want %q", got, want)
	}
}

func TestUploadsCacheControl_NextHeaderSetBeforeWriteWins(t *testing.T) {
	// Document the order of operations: the middleware sets its
	// Cache-Control first; a downstream handler that overrides the header
	// BEFORE writing any body bytes wins. (After the first Write the
	// header is locked by net/http, which is why http.FileServer — the
	// real downstream — cannot override what we set here.)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, no-store")
		w.WriteHeader(http.StatusOK)
	})

	h := uploadsCacheControl(next)

	req := httptest.NewRequest("GET", "/uploads/abc.jpg", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	got := rec.Header().Get("Cache-Control")
	want := "private, no-store"
	if got != want {
		t.Errorf("Cache-Control = %q, want next handler's value %q to win", got, want)
	}
}
