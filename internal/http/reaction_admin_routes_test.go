package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/gorilla/mux"
)

// buildReactionAdminRouter builds a minimal router that mirrors the production
// protectedAPI wiring in router.go: AuthMiddleware then CSRFMiddleware (in that
// order), without needing the full embed.FS apparatus.
//
// IMPORTANT: this function must stay in sync with the protectedAPI block in
// router.go (AuthMiddleware → CSRFMiddleware → reaction-type routes). Any change
// to that wiring must be reflected here so the regression tests remain valid.
func buildReactionAdminRouter() *mux.Router {
	authMock := &mockAuthService{}

	r := mux.NewRouter()
	api := r.PathPrefix("/api/v1").Subrouter()
	protected := api.PathPrefix("").Subrouter()
	// Mirror production order: AuthMiddleware first, then CSRFMiddleware.
	protected.Use(AuthMiddleware(NewCurrentUserHelper(authMock, false)))
	protected.Use(CSRFMiddleware())

	admin := NewReactionAdminHandlers(&mockReactionTypeService{})
	protected.HandleFunc("/reaction-types", admin.HandleList).Methods("GET")
	protected.HandleFunc("/reaction-types", admin.HandleCreate).Methods("POST")
	protected.HandleFunc("/reaction-types/{id:[0-9]+}", admin.HandleUpdate).Methods("PUT")
	protected.HandleFunc("/reaction-types/{id:[0-9]+}", admin.HandleDelete).Methods("DELETE")
	return r
}

// buildAuthenticatedReactionAdminRouter builds a router whose mockAuthService
// resolves "valid-session-id" to a real user, so requests carrying that session
// cookie pass AuthMiddleware. CSRFMiddleware is still applied, enabling tests
// that verify CSRF enforcement for authenticated users.
func buildAuthenticatedReactionAdminRouter() *mux.Router {
	now := time.Now()
	authMock := &mockAuthService{
		getUserBySessionFunc: func(sessionID string) (*domain.User, error) {
			if sessionID == "valid-session-id" {
				return &domain.User{ID: 1, Username: "admin", CreatedAt: now, UpdatedAt: now}, nil
			}
			return nil, nil
		},
	}

	r := mux.NewRouter()
	api := r.PathPrefix("/api/v1").Subrouter()
	protected := api.PathPrefix("").Subrouter()
	// Mirror production order: AuthMiddleware first, then CSRFMiddleware.
	protected.Use(AuthMiddleware(NewCurrentUserHelper(authMock, false)))
	protected.Use(CSRFMiddleware())

	admin := NewReactionAdminHandlers(&mockReactionTypeService{})
	protected.HandleFunc("/reaction-types", admin.HandleList).Methods("GET")
	protected.HandleFunc("/reaction-types", admin.HandleCreate).Methods("POST")
	protected.HandleFunc("/reaction-types/{id:[0-9]+}", admin.HandleUpdate).Methods("PUT")
	protected.HandleFunc("/reaction-types/{id:[0-9]+}", admin.HandleDelete).Methods("DELETE")
	return r
}

// TestReactionTypeRoutes_RequireAuth verifies all four admin reaction-type
// routes are covered by AuthMiddleware and return 401 for unauthenticated
// requests (no session or remember cookie). AuthMiddleware runs before
// CSRFMiddleware, so the 401 is returned before CSRF is evaluated.
func TestReactionTypeRoutes_RequireAuth(t *testing.T) {
	r := buildReactionAdminRouter()
	for _, tc := range []struct{ method, path string }{
		{"GET", "/api/v1/reaction-types"},
		{"POST", "/api/v1/reaction-types"},
		{"PUT", "/api/v1/reaction-types/1"},
		{"DELETE", "/api/v1/reaction-types/1"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: expected 401, got %d", tc.method, tc.path, rec.Code)
		}
	}
}

// TestReactionTypeRoutes_RequireCSRF verifies that state-changing routes
// (POST, PUT, DELETE) return 403 for an authenticated request that is missing
// a valid X-CSRF-Token header. GET is exempt from CSRF and is not tested here.
func TestReactionTypeRoutes_RequireCSRF(t *testing.T) {
	r := buildAuthenticatedReactionAdminRouter()

	for _, tc := range []struct{ method, path string }{
		{"POST", "/api/v1/reaction-types"},
		{"PUT", "/api/v1/reaction-types/1"},
		{"DELETE", "/api/v1/reaction-types/1"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		// Attach a valid session so AuthMiddleware passes.
		req.AddCookie(&http.Cookie{
			Name:  sessionCookieName,
			Value: "valid-session-id",
		})
		// Deliberately omit X-CSRF-Token header → CSRFMiddleware must reject with 403.
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s %s: expected 403, got %d", tc.method, tc.path, rec.Code)
		}
	}
}
