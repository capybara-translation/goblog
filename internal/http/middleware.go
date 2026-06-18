package http

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
)

// contextKey is a type representing context keys
type contextKey string

const (
	// contextKeyUserID is the key for storing user ID in context
	contextKeyUserID contextKey = "user_id"

	// contextKeySessionID is the key for the effective session id (the one in
	// force after any remember-me restore), used by the device-management API.
	contextKeySessionID contextKey = "session_id"

	// csrfCookieName is the name of the cookie that stores the CSRF token
	csrfCookieName = "csrf_token"

	// csrfHeaderName is the name of the HTTP header containing the CSRF token
	csrfHeaderName = "X-CSRF-Token"

	// csrfTokenLength is the byte length of the CSRF token
	csrfTokenLength = 32
)

// AuthMiddleware protects endpoints requiring authentication. It uses
// CurrentUserHelper.Optional so that a request carrying only a remember_token
// can also pass — the helper restores a fresh session under the hood, and
// subsequent requests will arrive with a normal session_id cookie. Errors are
// written as JSON ErrorResponse to match the rest of the /api/v1/* contract.
func AuthMiddleware(helper *CurrentUserHelper) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, sessionID, err := helper.resolve(w, r)
			if err != nil {
				respondJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
				return
			}
			if user == nil {
				respondJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
				return
			}
			ctx := context.WithValue(r.Context(), contextKeyUserID, user.ID)
			ctx = context.WithValue(ctx, contextKeySessionID, sessionID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserIDFromContext retrieves the user ID from context
func GetUserIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(contextKeyUserID).(int64)
	return userID, ok
}

// GetSessionIDFromContext retrieves the effective session id from context.
func GetSessionIDFromContext(ctx context.Context) (string, bool) {
	sid, ok := ctx.Value(contextKeySessionID).(string)
	return sid, ok
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

			// Compare cookie and header tokens (constant-time to prevent timing attacks)
			if subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(headerToken)) != 1 {
				respondJSON(w, http.StatusForbidden, ErrorResponse{Error: "CSRF token mismatch"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
