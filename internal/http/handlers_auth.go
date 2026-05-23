package http

import (
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/capybara-translation/goblog/internal/service"
)

const (
	sessionCookieName = "session_id"
	sessionCookiePath = "/"
)

// AuthHandlers is a struct that groups authentication-related HTTP handlers
type AuthHandlers struct {
	authService    service.AuthService
	secureCookie   bool       // Cookie secure attribute (requires HTTPS)
	trustedProxies []*net.IPNet // Trusted proxy networks for X-Forwarded-For
}

// NewAuthHandlers creates a new AuthHandlers
func NewAuthHandlers(authService service.AuthService, secureCookie bool, trustedProxies []string) *AuthHandlers {
	return &AuthHandlers{
		authService:    authService,
		secureCookie:   secureCookie,
		trustedProxies: parseTrustedProxies(trustedProxies),
	}
}

// parseTrustedProxies parses IP addresses and CIDR ranges into net.IPNet slices
func parseTrustedProxies(proxies []string) []*net.IPNet {
	var nets []*net.IPNet
	for _, p := range proxies {
		// Try parsing as CIDR first
		if strings.Contains(p, "/") {
			_, ipNet, err := net.ParseCIDR(p)
			if err != nil {
				log.Printf("warning: invalid CIDR in TRUSTED_PROXIES: %s", p)
				continue
			}
			nets = append(nets, ipNet)
		} else {
			// Single IP address — convert to /32 or /128
			ip := net.ParseIP(p)
			if ip == nil {
				log.Printf("warning: invalid IP in TRUSTED_PROXIES: %s", p)
				continue
			}
			mask := net.CIDRMask(128, 128)
			if ip.To4() != nil {
				mask = net.CIDRMask(32, 32)
			}
			nets = append(nets, &net.IPNet{IP: ip, Mask: mask})
		}
	}
	return nets
}

// isTrustedProxy checks if the given IP address is in the trusted proxy list
func (h *AuthHandlers) isTrustedProxy(ipStr string) bool {
	if len(h.trustedProxies) == 0 {
		return false
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, n := range h.trustedProxies {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// LoginRequest is the request body for login
type LoginRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	RememberMe bool   `json:"remember_me"`
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
	ipAddress := h.getClientIP(r)

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

	// Cookie MaxAge mirrors the server-side session TTL so the cookie and
	// the session expire together. Reading via the service avoids two
	// independent "24h" constants drifting apart. Integer division keeps
	// the conversion exact for all values reachable through SESSION_TTL
	// (the config layer rejects anything below 1m, so sub-second values
	// never reach this path).
	cookieMaxAge := int(h.authService.SessionTTL() / time.Second)

	// Set session ID in cookie
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Path:     sessionCookiePath,
		HttpOnly: true,                 // Not accessible from JavaScript
		SameSite: http.SameSiteLaxMode, // CSRF protection
		MaxAge:   cookieMaxAge,
		Secure:   h.secureCookie, // Requires HTTPS (true in production)
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
		MaxAge:   cookieMaxAge,
		Secure:   h.secureCookie, // Requires HTTPS (true in production)
	})

	// If the user opted into "remember me", issue a long-lived token and set
	// its cookie. Failure here logs but does not abort the login — the user
	// still has a valid session_id cookie and CSRF.
	if req.RememberMe {
		rememberValue, err := h.authService.IssueRememberToken(user.ID)
		if err != nil {
			log.Printf("HandleLogin: IssueRememberToken failed: %v", err)
		} else {
			// Remember cookie has its own (longer) MaxAge — derived from the
			// service's remember TTL, not the session TTL.
			http.SetCookie(w, &http.Cookie{
				Name:     rememberCookieName,
				Value:    rememberValue,
				Path:     sessionCookiePath,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
				MaxAge:   int(h.authService.RememberTTL() / time.Second),
				Secure:   h.secureCookie,
			})
		}
	}

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

	// Revoke the remember token and clear its cookie. Both must happen even
	// when one of them fails: the cookie clear keeps the browser from
	// re-presenting a token whose DB row failed to delete, and vice versa.
	if rememberCookie, err := r.Cookie(rememberCookieName); err == nil {
		if revokeErr := h.authService.RevokeRememberToken(rememberCookie.Value); revokeErr != nil {
			log.Printf("HandleLogout: RevokeRememberToken failed: %v", revokeErr)
		}
		http.SetCookie(w, &http.Cookie{
			Name:     rememberCookieName,
			Value:    "",
			Path:     sessionCookiePath,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
			Secure:   h.secureCookie,
		})
	}

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

// getClientIP retrieves the client IP address from the request.
// Only trusts X-Forwarded-For / X-Real-IP headers when the direct connection
// comes from a trusted proxy (configured via TRUSTED_PROXIES env var).
func (h *AuthHandlers) getClientIP(r *http.Request) string {
	// Extract IP from RemoteAddr (always available, "IP:port" format)
	remoteIP := extractIP(r.RemoteAddr)

	// Only use proxy headers if the direct connection is from a trusted proxy
	if h.isTrustedProxy(remoteIP) {
		// Check X-Forwarded-For header
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
	}

	return remoteIP
}

// extractIP extracts the IP address from an "IP:port" string
func extractIP(remoteAddr string) string {
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		return host
	}
	return remoteAddr
}

// respondJSON is a helper function that returns a JSON response
func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("failed to encode JSON response: %v", err)
	}
}
