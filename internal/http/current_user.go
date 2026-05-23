package http

import (
	"log"
	"net/http"
	"time"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/capybara-translation/goblog/internal/service"
)

const rememberCookieName = "remember_token"

// CurrentUserHelper resolves the current user from either the session cookie
// or, if missing/expired, the remember-me cookie. On a successful remember
// restoration it sets fresh session_id and csrf_token cookies as a side
// effect — note that this means GET endpoints calling Optional() emit
// Set-Cookie headers when a restore happens.
type CurrentUserHelper struct {
	authService  service.AuthService
	secureCookie bool
}

func NewCurrentUserHelper(authService service.AuthService, secureCookie bool) *CurrentUserHelper {
	return &CurrentUserHelper{
		authService:  authService,
		secureCookie: secureCookie,
	}
}

// Optional returns the current user if one can be resolved, or nil.
// Errors from the underlying store are logged and surfaced.
func (h *CurrentUserHelper) Optional(w http.ResponseWriter, r *http.Request) (*domain.User, error) {
	// 1. Try the session cookie first.
	if c, err := r.Cookie(sessionCookieName); err == nil {
		user, err := h.authService.GetUserBySession(c.Value)
		if err == nil && user != nil {
			return user, nil
		}
		// "session not found" and ErrUserNotFound (stale) are expected — fall
		// through to the remember token. Real DB errors surface via the same
		// path; CurrentUserHelper's caller does not need to distinguish.
	}

	// 2. Fall back to the remember token.
	rememberCookie, err := r.Cookie(rememberCookieName)
	if err != nil {
		return nil, nil // no remember cookie either
	}

	user, sessionID, restoreErr := h.authService.RestoreFromRememberToken(rememberCookie.Value)
	if restoreErr != nil {
		log.Printf("CurrentUserHelper: RestoreFromRememberToken failed: %v", restoreErr)
		return nil, restoreErr
	}
	if user == nil {
		// Remember cookie was present but invalid. Clear it so the browser
		// stops re-sending the bad value on every request.
		h.clearCookie(w, rememberCookieName)
		return nil, nil
	}

	// 3. Restoration succeeded: emit fresh session + CSRF cookies. The
	// cookie MaxAge mirrors the session TTL (not the remember TTL): the
	// remember cookie itself was already set with its own MaxAge at login.
	cookieMaxAge := int(h.authService.SessionTTL() / time.Second)
	h.setCookie(w, sessionCookieName, sessionID, cookieMaxAge, true)
	csrfToken, err := generateCSRFToken()
	if err != nil {
		log.Printf("CurrentUserHelper: generateCSRFToken failed: %v", err)
	} else {
		h.setCookie(w, csrfCookieName, csrfToken, cookieMaxAge, false)
	}

	return user, nil
}

// Required is Optional plus a 401 + halt for anonymous requests.
func (h *CurrentUserHelper) Required(w http.ResponseWriter, r *http.Request) (*domain.User, bool) {
	user, err := h.Optional(w, r)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return nil, false
	}
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil, false
	}
	return user, true
}

func (h *CurrentUserHelper) setCookie(w http.ResponseWriter, name, value string, maxAge int, httpOnly bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     sessionCookiePath,
		HttpOnly: httpOnly,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
		Secure:   h.secureCookie,
	})
}

func (h *CurrentUserHelper) clearCookie(w http.ResponseWriter, name string) {
	h.setCookie(w, name, "", -1, true)
}
