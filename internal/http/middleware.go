package http

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"net/http"

	"github.com/capybara-translation/goblog/internal/service"
)

// contextKey is a type representing context keys
type contextKey string

const (
	// contextKeyUserID is the key for storing user ID in context
	contextKeyUserID contextKey = "user_id"

	// csrfCookieName is the name of the cookie that stores the CSRF token
	csrfCookieName = "csrf_token"

	// csrfHeaderName is the name of the HTTP header containing the CSRF token
	csrfHeaderName = "X-CSRF-Token"

	// csrfTokenLength is the byte length of the CSRF token
	csrfTokenLength = 32
)

// AuthMiddleware is a middleware that protects endpoints requiring authentication
func AuthMiddleware(authService service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get session ID from cookie
			cookie, err := r.Cookie(sessionCookieName)
			if err != nil {
				respondJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "Authentication required"})
				return
			}

			// Get user information from session
			user, err := authService.GetUserBySession(cookie.Value)
			if err != nil {
				// Return 401 if user was deleted
				if errors.Is(err, service.ErrUserNotFound) {
					respondJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "Session expired or invalid"})
					return
				}
				log.Printf("get user by session error: %v", err)
				respondJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
				return
			}

			if user == nil {
				respondJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "Session expired or invalid"})
				return
			}

			// Set user ID in context
			ctx := context.WithValue(r.Context(), contextKeyUserID, user.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserIDFromContext retrieves the user ID from context
func GetUserIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(contextKeyUserID).(int64)
	return userID, ok
}

// generateCSRFToken generates a random CSRF token
func generateCSRFToken() (string, error) {
	b := make([]byte, csrfTokenLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// CSRFMiddleware is a middleware that prevents CSRF attacks (Double Submit Cookie method)
// Validates CSRF token for state-changing methods (POST, PUT, DELETE, PATCH)
func CSRFMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip CSRF check for GET and HEAD methods (idempotent operations)
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			// Get CSRF token from cookie
			cookie, err := r.Cookie(csrfCookieName)
			if err != nil {
				respondJSON(w, http.StatusForbidden, ErrorResponse{Error: "CSRF token cookie missing"})
				return
			}

			// Get CSRF token from HTTP header
			headerToken := r.Header.Get(csrfHeaderName)
			if headerToken == "" {
				respondJSON(w, http.StatusForbidden, ErrorResponse{Error: "CSRF token header missing"})
				return
			}

			// Compare cookie and header tokens
			if cookie.Value != headerToken {
				respondJSON(w, http.StatusForbidden, ErrorResponse{Error: "CSRF token mismatch"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
