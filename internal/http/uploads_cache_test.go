package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

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
