package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireXRequestedWith_BlocksMissingHeaderOnPost(t *testing.T) {
	called := false
	h := RequireXRequestedWith()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/api/v1/posts/p1/reactions/1", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if called {
		t.Fatal("handler should not be called without X-Requested-With")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestRequireXRequestedWith_AllowsWithHeader(t *testing.T) {
	h := RequireXRequestedWith()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/api/v1/posts/p1/reactions/1", nil)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRequireXRequestedWith_SkipsReadOnlyMethods(t *testing.T) {
	methods := []string{http.MethodGet, http.MethodHead, http.MethodOptions}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			h := RequireXRequestedWith()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(method, "/api/v1/posts/p1/reactions", nil) // no X-Requested-With
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("%s should bypass the header check, got %d", method, rec.Code)
			}
		})
	}
}
