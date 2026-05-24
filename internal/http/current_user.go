package http

import (
	"errors"
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
//
// Session-lookup errors are logged but NOT returned: they are swallowed so a
// transient session-store hiccup still falls through to the remember-token
// path. Only an error from RestoreFromRememberToken is surfaced to the caller
// (and also logged). So a non-nil error from Optional always originates from
// the remember-token path, never from the session lookup.
func (h *CurrentUserHelper) Optional(w http.ResponseWriter, r *http.Request) (*domain.User, error) {
	// 1. Try the session cookie first.
	if c, err := r.Cookie(sessionCookieName); err == nil {
		user, lookupErr := h.authService.GetUserBySession(c.Value)
		if lookupErr == nil && user != nil {
			return user, nil
		}
		// Distinguish expected stale-session conditions from real DB errors:
		// "session not found" returns (nil, nil); a deleted user returns
		// ErrUserNotFound; anything else (e.g. DB outage) is operationally
		// interesting and worth logging so we don't lose the breadcrumb.
		if lookupErr != nil && !errors.Is(lookupErr, service.ErrUserNotFound) {
			log.Printf("CurrentUserHelper: GetUserBySession failed: %v", lookupErr)
		}
		// Fall through to the remember token regardless.
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

func (h *CurrentUserHelper) setCookie(w http.ResponseWriter, name, value string, maxAge int, httpOnly bool) {
	// Any response that emits a session/CSRF/remember Set-Cookie via this
	// helper is user-specific and must not be cached by a CDN or shared
	// proxy. We force the no-store directive here so the policy is owned by
	// the helper itself rather than scattered across every public handler.
	if w.Header().Get("Cache-Control") == "" {
		w.Header().Set("Cache-Control", "private, no-store")
	}
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
