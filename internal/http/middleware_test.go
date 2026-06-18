package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/capybara-translation/goblog/internal/domain"
)

func TestAuthMiddleware_Success(t *testing.T) {
	now := time.Now()
	mockService := &mockAuthService{
		getUserBySessionFunc: func(sessionID string) (*domain.User, error) {
			if sessionID == "valid-session-id" {
				return &domain.User{
					ID:        123,
					Username:  "testuser",
					CreatedAt: now,
					UpdatedAt: now,
				}, nil
			}
			return nil, nil
		},
	}

	// Create a handler with middleware applied
	var capturedUserID int64
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get user ID from context
		userID, ok := GetUserIDFromContext(r.Context())
		if !ok {
			t.Error("expected user ID in context")
			return
		}
		capturedUserID = userID
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware := AuthMiddleware(NewCurrentUserHelper(mockService, false))
	protectedHandler := middleware(handler)

	// Request with valid session ID
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  sessionCookieName,
		Value: "valid-session-id",
	})
	rec := httptest.NewRecorder()

	// Execute handler
	protectedHandler.ServeHTTP(rec, req)

	// Verify status code
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Verify correct user ID is set in context
	if capturedUserID != 123 {
		t.Errorf("expected user ID %d in context, got %d", 123, capturedUserID)
	}
}

func TestAuthMiddleware_NoCookie(t *testing.T) {
	mockService := &mockAuthService{
		getUserBySessionFunc: func(sessionID string) (*domain.User, error) {
			t.Error("GetUserBySession should not be called when no cookie is present")
			return nil, nil
		},
	}

	// Create a handler with middleware applied
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called when auth fails")
		w.WriteHeader(http.StatusOK)
	})

	middleware := AuthMiddleware(NewCurrentUserHelper(mockService, false))
	protectedHandler := middleware(handler)

	// Request without cookie
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()

	protectedHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
	// AuthMiddleware emits JSON ErrorResponse, matching the rest of /api/v1/*.
	expectedError := `"error":"Unauthorized"`
	if !strings.Contains(rec.Body.String(), expectedError) {
		t.Errorf("expected body to contain %q, got %q", expectedError, rec.Body.String())
	}
}

func TestAuthMiddleware_InvalidSession(t *testing.T) {
	mockService := &mockAuthService{
		getUserBySessionFunc: func(sessionID string) (*domain.User, error) {
			// Session is invalid (user not found)
			return nil, nil
		},
	}

	// Create a handler with middleware applied
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called when auth fails")
		w.WriteHeader(http.StatusOK)
	})

	middleware := AuthMiddleware(NewCurrentUserHelper(mockService, false))
	protectedHandler := middleware(handler)

	// Request with invalid session ID (no remember cookie either → 401)
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  sessionCookieName,
		Value: "invalid-session-id",
	})
	rec := httptest.NewRecorder()

	protectedHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
	expectedError := `"error":"Unauthorized"`
	if !strings.Contains(rec.Body.String(), expectedError) {
		t.Errorf("expected body to contain %q, got %q", expectedError, rec.Body.String())
	}
}

func TestAuthMiddleware_SessionDBError_NoRememberCookie_401(t *testing.T) {
	// CurrentUserHelper swallows session-lookup DB errors to allow a
	// remember-token fallback to run. With no remember cookie, this surfaces
	// as anonymous (401) rather than 500 — a deliberate trade-off so a
	// transient session-store hiccup does not break users who have a valid
	// remember cookie. [TRADE-OFF, was 500 before remember-me feature]
	mockService := &mockAuthService{
		getUserBySessionFunc: func(sessionID string) (*domain.User, error) {
			// Simulate database error
			return nil, errors.New("database connection error")
		},
	}

	// Create a handler with middleware applied
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called when auth fails")
		w.WriteHeader(http.StatusOK)
	})

	middleware := AuthMiddleware(NewCurrentUserHelper(mockService, false))
	protectedHandler := middleware(handler)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  sessionCookieName,
		Value: "valid-session-id",
	})
	rec := httptest.NewRecorder()

	protectedHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestAuthMiddleware_RememberTokenDBError_500(t *testing.T) {
	// When a remember-token lookup fails with a real error, Optional() returns
	// it and AuthMiddleware emits a JSON 500. This is the path that still
	// acts as a "DB-error → 500" backstop after the remember-me refactor.
	mockService := &mockAuthService{
		getUserBySessionFunc: func(string) (*domain.User, error) {
			return nil, nil
		},
		restoreFromRememberTokenFunc: func(string, string, string) (*domain.User, string, error) {
			return nil, "", errors.New("database connection error")
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called when auth fails")
		w.WriteHeader(http.StatusOK)
	})

	middleware := AuthMiddleware(NewCurrentUserHelper(mockService, false))
	protectedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: rememberCookieName, Value: "rem"})
	rec := httptest.NewRecorder()

	protectedHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestGetUserIDFromContext_Success(t *testing.T) {
	now := time.Now()
	mockService := &mockAuthService{
		getUserBySessionFunc: func(sessionID string) (*domain.User, error) {
			if sessionID == "valid-session-id" {
				return &domain.User{
					ID:        456,
					Username:  "testuser",
					CreatedAt: now,
					UpdatedAt: now,
				}, nil
			}
			return nil, nil
		},
	}

	// Process request via middleware
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := GetUserIDFromContext(r.Context())
		if !ok {
			t.Error("expected user ID in context")
			return
		}

		if userID != 456 {
			t.Errorf("expected user ID %d, got %d", 456, userID)
		}
		w.WriteHeader(http.StatusOK)
	})

	middleware := AuthMiddleware(NewCurrentUserHelper(mockService, false))
	protectedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  sessionCookieName,
		Value: "valid-session-id",
	})
	rec := httptest.NewRecorder()

	protectedHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestGetUserIDFromContext_NotFound(t *testing.T) {
	// When user ID is not in context
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	userID, ok := GetUserIDFromContext(req.Context())
	if ok {
		t.Error("expected user ID not to be found in context")
	}

	if userID != 0 {
		t.Errorf("expected user ID to be 0 when not found, got %d", userID)
	}
}

func TestCSRFMiddleware_GetRequest_Skip(t *testing.T) {
	// GET requests skip CSRF check
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware := CSRFMiddleware()
	protectedHandler := middleware(handler)

	// GET request without CSRF token
	req := httptest.NewRequest(http.MethodGet, "/api/posts", nil)
	rec := httptest.NewRecorder()

	protectedHandler.ServeHTTP(rec, req)

	// GET requests skip CSRF check so OK
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestCSRFMiddleware_PostRequest_Success(t *testing.T) {
	// POST request succeeds when correct CSRF token is present
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware := CSRFMiddleware()
	protectedHandler := middleware(handler)

	csrfToken := "test-csrf-token"

	// Set CSRF token in cookie and header
	req := httptest.NewRequest(http.MethodPost, "/api/posts", nil)
	req.AddCookie(&http.Cookie{
		Name:  csrfCookieName,
		Value: csrfToken,
	})
	req.Header.Set(csrfHeaderName, csrfToken)
	rec := httptest.NewRecorder()

	protectedHandler.ServeHTTP(rec, req)

	// Succeeds because CSRF tokens match
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestCSRFMiddleware_NoCookie(t *testing.T) {
	// Return 403 when CSRF token cookie is missing
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called when CSRF cookie is missing")
		w.WriteHeader(http.StatusOK)
	})

	middleware := CSRFMiddleware()
	protectedHandler := middleware(handler)

	// POST request without CSRF token cookie
	req := httptest.NewRequest(http.MethodPost, "/api/posts", nil)
	req.Header.Set(csrfHeaderName, "some-token")
	rec := httptest.NewRecorder()

	protectedHandler.ServeHTTP(rec, req)

	// Return 403 because CSRF token cookie is missing
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}

	expectedError := "CSRF token cookie missing"
	if !strings.Contains(rec.Body.String(), expectedError) {
		t.Errorf("expected error message to contain %q, got %q", expectedError, rec.Body.String())
	}
}

func TestCSRFMiddleware_NoHeader(t *testing.T) {
	// Return 403 when CSRF token header is missing
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called when CSRF header is missing")
		w.WriteHeader(http.StatusOK)
	})

	middleware := CSRFMiddleware()
	protectedHandler := middleware(handler)

	// POST request without CSRF token header
	req := httptest.NewRequest(http.MethodPost, "/api/posts", nil)
	req.AddCookie(&http.Cookie{
		Name:  csrfCookieName,
		Value: "some-token",
	})
	rec := httptest.NewRecorder()

	protectedHandler.ServeHTTP(rec, req)

	// Return 403 because CSRF token header is missing
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}

	expectedError := "CSRF token header missing"
	if !strings.Contains(rec.Body.String(), expectedError) {
		t.Errorf("expected error message to contain %q, got %q", expectedError, rec.Body.String())
	}
}

func TestCSRFMiddleware_TokenMismatch(t *testing.T) {
	// Return 403 when CSRF tokens do not match
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called when CSRF tokens mismatch")
		w.WriteHeader(http.StatusOK)
	})

	middleware := CSRFMiddleware()
	protectedHandler := middleware(handler)

	// CSRF tokens in cookie and header are different
	req := httptest.NewRequest(http.MethodPost, "/api/posts", nil)
	req.AddCookie(&http.Cookie{
		Name:  csrfCookieName,
		Value: "token-in-cookie",
	})
	req.Header.Set(csrfHeaderName, "token-in-header")
	rec := httptest.NewRecorder()

	protectedHandler.ServeHTTP(rec, req)

	// Return 403 because CSRF tokens do not match
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}

	expectedError := "CSRF token mismatch"
	if !strings.Contains(rec.Body.String(), expectedError) {
		t.Errorf("expected error message to contain %q, got %q", expectedError, rec.Body.String())
	}
}

func TestCSRFMiddleware_PutRequest(t *testing.T) {
	// PUT, DELETE, PATCH requests also require CSRF check
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware := CSRFMiddleware()
	protectedHandler := middleware(handler)

	csrfToken := "test-csrf-token"

	tests := []struct {
		name   string
		method string
	}{
		{"PUT", http.MethodPut},
		{"DELETE", http.MethodDelete},
		{"PATCH", http.MethodPatch},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/posts/1", nil)
			req.AddCookie(&http.Cookie{
				Name:  csrfCookieName,
				Value: csrfToken,
			})
			req.Header.Set(csrfHeaderName, csrfToken)
			rec := httptest.NewRecorder()

			protectedHandler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
			}
		})
	}
}

func TestGenerateCSRFToken(t *testing.T) {
	// Test CSRF token generation
	token1, err := generateCSRFToken()
	if err != nil {
		t.Fatalf("failed to generate CSRF token: %v", err)
	}

	if token1 == "" {
		t.Error("expected non-empty CSRF token")
	}

	// Verify that a different token is generated on the second call
	token2, err := generateCSRFToken()
	if err != nil {
		t.Fatalf("failed to generate CSRF token: %v", err)
	}

	if token1 == token2 {
		t.Error("expected different CSRF tokens on each generation")
	}
}
