package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/gorilla/mux"
)

// buildReactionRoutesRouter builds a minimal router that mirrors the
// production reaction-route wiring from NewRouter without needing the full
// embed.FS apparatus. This lets us test the RequireXRequestedWith middleware
// integration without pulling in the whole router.
func buildReactionRoutesRouter() *mux.Router {
	svc := &mockReactionService{
		get: func(slug, vk string) ([]*domain.PostReactionSummary, error) {
			return []*domain.PostReactionSummary{}, nil
		},
		add: func(slug string, typeID int64, vk string) ([]*domain.PostReactionSummary, error) {
			return []*domain.PostReactionSummary{}, nil
		},
		remove: func(slug string, typeID int64, vk string) ([]*domain.PostReactionSummary, error) {
			return []*domain.PostReactionSummary{}, nil
		},
	}

	r := mux.NewRouter()
	api := r.PathPrefix("/api/v1").Subrouter()

	reactionHandlers := NewReactionHandlers(svc, false, nil)
	reactions := api.PathPrefix("/posts/{slug}/reactions").Subrouter()
	reactions.Use(RequireXRequestedWith())
	reactions.HandleFunc("", reactionHandlers.HandleGet).Methods("GET")
	reactions.HandleFunc("/{reactionTypeID:[0-9]+}", reactionHandlers.HandleAdd).Methods("POST")
	reactions.HandleFunc("/{reactionTypeID:[0-9]+}", reactionHandlers.HandleRemove).Methods("DELETE")

	return r
}

// TestReactionRoutes_POST_WithoutXRequestedWith_Returns403 verifies the
// RequireXRequestedWith middleware is actually wired: a POST without the
// header must be rejected with 403.
func TestReactionRoutes_POST_WithoutXRequestedWith_Returns403(t *testing.T) {
	router := buildReactionRoutesRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/posts/p1/reactions/1", nil)
	// No X-Requested-With header — must be blocked.
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without X-Requested-With, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

// TestReactionRoutes_POST_WithXRequestedWith_NotForbidden verifies that a
// POST WITH the X-Requested-With header passes the middleware and reaches
// the handler (200 from the mock).
func TestReactionRoutes_POST_WithXRequestedWith_NotForbidden(t *testing.T) {
	router := buildReactionRoutesRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/posts/p1/reactions/1", nil)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Assert 200 (not merely "not 403") so a broken route (e.g. 404 from a
	// wiring mistake) fails the test instead of silently passing.
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with X-Requested-With header (request should pass middleware and reach the handler), got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

// TestReactionRoutes_GET_WithoutXRequestedWith_NotForbidden verifies that the
// RequireXRequestedWith middleware still runs for GET but exempts it (read-only
// methods are allowed through without the header).
func TestReactionRoutes_GET_WithoutXRequestedWith_NotForbidden(t *testing.T) {
	router := buildReactionRoutesRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts/p1/reactions", nil)
	// No X-Requested-With header — GET should not be blocked.
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code == http.StatusForbidden {
		t.Fatalf("expected GET to bypass X-Requested-With check, got 403 (body: %s)", rec.Body.String())
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for GET, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}
