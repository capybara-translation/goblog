package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// buildReactionAdminRouter builds a minimal router that mirrors the production
// protected-API wiring for reaction-type admin routes, without needing the full
// embed.FS apparatus. The mockAuthService setup mirrors TestAuthMiddleware_NoCookie:
// an empty mockAuthService suffices because a no-cookie request never reaches
// GetUserBySession or RestoreFromRememberToken — AuthMiddleware rejects first.
func buildReactionAdminRouter() *mux.Router {
	authMock := &mockAuthService{}

	r := mux.NewRouter()
	api := r.PathPrefix("/api/v1").Subrouter()
	protected := api.PathPrefix("").Subrouter()
	protected.Use(AuthMiddleware(NewCurrentUserHelper(authMock, false)))

	admin := NewReactionAdminHandlers(&mockReactionTypeService{})
	protected.HandleFunc("/reaction-types", admin.HandleList).Methods("GET")
	protected.HandleFunc("/reaction-types", admin.HandleCreate).Methods("POST")
	protected.HandleFunc("/reaction-types/{id:[0-9]+}", admin.HandleUpdate).Methods("PUT")
	protected.HandleFunc("/reaction-types/{id:[0-9]+}", admin.HandleDelete).Methods("DELETE")
	return r
}

// TestReactionTypeRoutes_RequireAuth verifies all four admin reaction-type
// routes are covered by AuthMiddleware and return 401 for unauthenticated
// requests (no session or remember cookie).
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
