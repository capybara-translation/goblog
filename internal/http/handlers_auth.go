package http

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/capybara-translation/goblog/internal/service"
)

const (
	sessionCookieName = "session_id"
	sessionCookiePath = "/"
)

// AuthHandlers is a struct that groups authentication-related HTTP handlers
type AuthHandlers struct {
	authService  service.AuthService
	secureCookie bool // Cookie secure attribute (requires HTTPS)
}

// NewAuthHandlers creates a new AuthHandlers
func NewAuthHandlers(authService service.AuthService, secureCookie bool) *AuthHandlers {
	return &AuthHandlers{
		authService:  authService,
		secureCookie: secureCookie,
	}
}

// LoginRequest is the request body for login
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// HandleLogin handles the login process
func (h *AuthHandlers) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	// Validation
	if req.Username == "" || req.Password == "" {
		respondJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Username and password are required"})
		return
	}

	// Get client IP address (for brute force protection)
	ipAddress := getClientIP(r)

	// Authentication
	sessionID, err := h.authService.Login(req.Username, req.Password, ipAddress)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			respondJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "Invalid username or password"})
			return
		}
		log.Printf("login error: %v", err)
		respondJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	// Get user information from session
	user, err := h.authService.GetUserBySession(sessionID)
	if err != nil {
		// Treat as authentication failure if user was deleted
		if errors.Is(err, service.ErrUserNotFound) {
			respondJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "Authentication failed"})
			return
		}
		log.Printf("get user by session error: %v", err)
		respondJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	if user == nil {
		respondJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Failed to get user information"})
		return
	}

	// Set session ID in cookie
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Path:     sessionCookiePath,
		HttpOnly: true,                 // Not accessible from JavaScript
		SameSite: http.SameSiteLaxMode, // CSRF protection
		MaxAge:   24 * 60 * 60,         // 24 hours
		Secure:   h.secureCookie,       // Requires HTTPS (true in production)
	})

	// Generate CSRF token and set in cookie
	csrfToken, err := generateCSRFToken()
	if err != nil {
		log.Printf("failed to generate CSRF token: %v", err)
		respondJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    csrfToken,
		Path:     sessionCookiePath,
		HttpOnly: false,                // Accessible from JavaScript (for Double Submit Cookie method)
		SameSite: http.SameSiteLaxMode, // CSRF protection
		MaxAge:   24 * 60 * 60,         // 24 hours
		Secure:   h.secureCookie,       // Requires HTTPS (true in production)
	})

	// Return user information (PasswordHash is excluded via json:"-")
	respondJSON(w, http.StatusOK, user)
}

// HandleLogout handles the logout process
func (h *AuthHandlers) HandleLogout(w http.ResponseWriter, r *http.Request) {
	// Get session ID from cookie
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		// Do nothing if session cookie doesn't exist
		respondJSON(w, http.StatusOK, map[string]string{"message": "Logout successful"})
		return
	}

	// Delete session
	if err := h.authService.Logout(cookie.Value); err != nil {
		log.Printf("logout error: %v", err)
	}

	// Delete session cookie (must specify the same attributes as when it was set)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     sessionCookiePath,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,             // Delete immediately
		Secure:   h.secureCookie, // Same as when set
	})

	// Also delete CSRF token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    "",
		Path:     sessionCookiePath,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,             // Delete immediately
		Secure:   h.secureCookie, // Same as when set
	})

	respondJSON(w, http.StatusOK, map[string]string{"message": "Logout successful"})
}

// HandleMe returns the currently logged-in user information
func (h *AuthHandlers) HandleMe(w http.ResponseWriter, r *http.Request) {
	// Get session ID from cookie
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		respondJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "Not authenticated"})
		return
	}

	// Get user information from session
	user, err := h.authService.GetUserBySession(cookie.Value)
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

	// Return user information (PasswordHash is excluded via json:"-")
	respondJSON(w, http.StatusOK, user)
}

// getClientIP retrieves the client IP address from the request
// For requests via proxy/load balancer, retrieves from X-Forwarded-For or X-Real-IP headers
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (for requests via proxy)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For format is "client, proxy1, proxy2"
		// The first IP address is the client's IP
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP header (alternative proxy header)
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Get from RemoteAddr (fallback)
	// RemoteAddr is in "IP:port" format, so extract only the IP
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		return ip[:idx]
	}
	return ip
}

// respondJSON is a helper function that returns a JSON response
func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("failed to encode JSON response: %v", err)
	}
}
