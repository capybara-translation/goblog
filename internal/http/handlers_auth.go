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
	authService       service.AuthService
	currentUserHelper *CurrentUserHelper // Resolves session-or-remember-token (used by HandleMe).
	secureCookie      bool               // Cookie secure attribute (requires HTTPS)
	trustedProxies    []*net.IPNet       // Trusted proxy networks for X-Forwarded-For
}

// NewAuthHandlers creates a new AuthHandlers
func NewAuthHandlers(authService service.AuthService, secureCookie bool, trustedProxies []string) *AuthHandlers {
	return &AuthHandlers{
		authService:       authService,
		currentUserHelper: NewCurrentUserHelper(authService, secureCookie),
		secureCookie:      secureCookie,
		trustedProxies:    parseTrustedProxies(trustedProxies),
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
		// If the browser already presents a remember_token, this is a
		// re-issue on the same device. Revoke the previous DB row up front so
		// it does not linger as an orphan (unreachable from any cookie) until
		// expires_at. Multi-device support is unaffected because each device
		// has its own cookie and selector.
		if oldCookie, err := r.Cookie(rememberCookieName); err == nil {
			if revokeErr := h.authService.RevokeRememberToken(oldCookie.Value); revokeErr != nil {
				log.Printf("HandleLogin: failed to revoke previous remember token on re-issue: %v", revokeErr)
			}
		}

		rememberValue, err := h.authService.IssueRememberToken(user.ID)
		if err != nil {
			log.Printf("HandleLogin: IssueRememberToken failed: %v", err)
			// We may have already revoked the previously-presented token's DB
			// row above, so the old cookie value is now invalid. Clear it so
			// the browser stops sending a known-bad token until a later
			// restore would otherwise lazily clear it.
			h.clearRememberCookie(w)
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
	} else if oldCookie, err := r.Cookie(rememberCookieName); err == nil {
		// "Remember me" was left unchecked but the browser still carries a
		// remember_token from a previous opt-in on this device. Honor the
		// unchecked box as an explicit opt-out: revoke the DB row and clear
		// the cookie, otherwise the user would still be auto-restored after
		// the session expires and the checkbox could never turn remember off.
		if revokeErr := h.authService.RevokeRememberToken(oldCookie.Value); revokeErr != nil {
			log.Printf("HandleLogin: failed to revoke remember token on opt-out: %v", revokeErr)
		}
		h.clearRememberCookie(w)
	}

	// Return user information (PasswordHash is excluded via json:"-")
	respondJSON(w, http.StatusOK, user)
}

// HandleLogout handles the logout process.
//
// Logout must NOT early-return when the session cookie is absent. With
// remember-me, /auth/logout sits behind AuthMiddleware, which can restore a
// session from the remember_token before this handler runs. If we bailed out
// on a missing session_id we would skip revoking the remember token — leaving
// the user able to be auto-logged-in again on the next request, which defeats
// logout entirely. So we always run the full cleanup: revoke the session if
// one was presented, revoke the remember token if one was presented, and clear
// every auth cookie regardless.
func (h *AuthHandlers) HandleLogout(w http.ResponseWriter, r *http.Request) {
	// Delete the server-side session if a session cookie was presented.
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		if logoutErr := h.authService.Logout(cookie.Value); logoutErr != nil {
			log.Printf("logout error: %v", logoutErr)
		}
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

	// Revoke the remember token if one was presented. Revocation is
	// conditional (there's nothing to delete without a cookie), but the
	// cookie deletion below is unconditional — consistent with how the
	// session and CSRF cookies are always cleared.
	if rememberCookie, err := r.Cookie(rememberCookieName); err == nil {
		if revokeErr := h.authService.RevokeRememberToken(rememberCookie.Value); revokeErr != nil {
			log.Printf("HandleLogout: RevokeRememberToken failed: %v", revokeErr)
		}
	}
	// Always clear the remember cookie.
	h.clearRememberCookie(w)

	respondJSON(w, http.StatusOK, map[string]string{"message": "Logout successful"})
}

// clearRememberCookie emits a deletion Set-Cookie for the remember_token.
// The attributes must mirror those used when the cookie is set (Path,
// HttpOnly, SameSite, Secure) or some browsers will not match-and-delete it.
// Centralizing this avoids attribute drift across the login opt-out, the
// issue-failure path, and logout.
func (h *AuthHandlers) clearRememberCookie(w http.ResponseWriter) {
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

// HandleMe returns the currently authenticated user. The session may be
// restored transparently from a remember-me cookie via CurrentUserHelper —
// when that happens, fresh session_id and csrf_token cookies are emitted on
// the response (callers see normal session cookies on subsequent requests).
func (h *AuthHandlers) HandleMe(w http.ResponseWriter, r *http.Request) {
	user, err := h.currentUserHelper.Optional(w, r)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}
	if user == nil {
		respondJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}
	respondJSON(w, http.StatusOK, user)
}

// getClientIP retrieves the client IP address from the request.
// Delegates to the shared clientIP helper (client_ip.go); trusted-proxy
// gating and XFF/X-Real-IP parsing live there to avoid duplication.
func (h *AuthHandlers) getClientIP(r *http.Request) string {
	return clientIP(r, h.trustedProxies)
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
