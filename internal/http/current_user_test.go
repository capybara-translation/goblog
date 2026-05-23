package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/capybara-translation/goblog/internal/domain"
)

func TestCurrentUserHelper_Optional_NoCookies(t *testing.T) {
	helper := NewCurrentUserHelper(&mockAuthService{}, false)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	user, err := helper.Optional(rec, req)
	if err != nil || user != nil {
		t.Errorf("expected nil user with no error, got user=%+v err=%v", user, err)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Errorf("no cookies should be set: %+v", rec.Result().Cookies())
	}
}

func TestCurrentUserHelper_Optional_ValidSession(t *testing.T) {
	helper := NewCurrentUserHelper(&mockAuthService{
		getUserBySessionFunc: func(id string) (*domain.User, error) {
			if id == "good" {
				return &domain.User{ID: 1, Username: "u"}, nil
			}
			return nil, nil
		},
	}, false)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "good"})
	rec := httptest.NewRecorder()

	user, err := helper.Optional(rec, req)
	if err != nil {
		t.Fatalf("Optional: %v", err)
	}
	if user == nil || user.ID != 1 {
		t.Errorf("expected user id 1, got %+v", user)
	}
	// Valid session must NOT trigger any cookie writes.
	if len(rec.Result().Cookies()) != 0 {
		t.Errorf("no new cookies expected on valid session, got: %+v", rec.Result().Cookies())
	}
}

func TestCurrentUserHelper_Optional_RestoresFromRememberToken(t *testing.T) {
	helper := NewCurrentUserHelper(&mockAuthService{
		// session lookup fails
		getUserBySessionFunc: func(string) (*domain.User, error) {
			return nil, nil
		},
		restoreFromRememberTokenFunc: func(cookie string) (*domain.User, string, error) {
			if cookie == "rem-cookie" {
				return &domain.User{ID: 7, Username: "admin"}, "fresh-session", nil
			}
			return nil, "", nil
		},
		rememberTTL: 30 * 24 * time.Hour,
	}, false)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: rememberCookieName, Value: "rem-cookie"})
	rec := httptest.NewRecorder()

	user, err := helper.Optional(rec, req)
	if err != nil {
		t.Fatalf("Optional: %v", err)
	}
	if user == nil || user.ID != 7 {
		t.Fatalf("expected user id 7, got %+v", user)
	}

	// Both session_id and csrf_token must be set as a side effect.
	var sawSession, sawCSRF bool
	for _, c := range rec.Result().Cookies() {
		switch c.Name {
		case sessionCookieName:
			sawSession = true
			if c.Value != "fresh-session" {
				t.Errorf("session cookie value = %q, want fresh-session", c.Value)
			}
		case csrfCookieName:
			sawCSRF = true
		}
	}
	if !sawSession || !sawCSRF {
		t.Fatalf("expected restoration to set both cookies; session=%v csrf=%v", sawSession, sawCSRF)
	}
}

func TestCurrentUserHelper_Optional_InvalidRememberCookieIsCleared(t *testing.T) {
	helper := NewCurrentUserHelper(&mockAuthService{
		getUserBySessionFunc: func(string) (*domain.User, error) {
			return nil, nil
		},
		restoreFromRememberTokenFunc: func(string) (*domain.User, string, error) {
			return nil, "", nil
		},
	}, false)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: rememberCookieName, Value: "garbage"})
	rec := httptest.NewRecorder()

	user, _ := helper.Optional(rec, req)
	if user != nil {
		t.Errorf("expected nil user")
	}

	// The bad remember cookie should be cleared via MaxAge=-1.
	var cleared bool
	for _, c := range rec.Result().Cookies() {
		if c.Name == rememberCookieName && c.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Errorf("expected remember cookie to be cleared, got: %+v", rec.Result().Cookies())
	}
}

func TestCurrentUserHelper_Optional_SetsCacheControlPrivateOnRestore(t *testing.T) {
	helper := NewCurrentUserHelper(&mockAuthService{
		getUserBySessionFunc: func(string) (*domain.User, error) { return nil, nil },
		restoreFromRememberTokenFunc: func(string) (*domain.User, string, error) {
			return &domain.User{ID: 1, Username: "u"}, "new-sid", nil
		},
		rememberTTL: 30 * 24 * time.Hour,
	}, false)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: rememberCookieName, Value: "rem"})
	rec := httptest.NewRecorder()

	_, _ = helper.Optional(rec, req)

	cc := rec.Result().Header.Get("Cache-Control")
	if cc == "" {
		t.Fatal("expected Cache-Control header to be set on restoration response")
	}
	if !strings.Contains(cc, "no-store") {
		t.Errorf("Cache-Control = %q, want it to include no-store so CDN/proxies do not cache the response", cc)
	}
	if !strings.Contains(cc, "private") {
		t.Errorf("Cache-Control = %q, want it to include private (user-specific response)", cc)
	}
}

func TestCurrentUserHelper_Optional_NoCacheControlWhenSessionIsValid(t *testing.T) {
	helper := NewCurrentUserHelper(&mockAuthService{
		getUserBySessionFunc: func(string) (*domain.User, error) {
			return &domain.User{ID: 1, Username: "u"}, nil
		},
	}, false)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "good"})
	rec := httptest.NewRecorder()

	_, _ = helper.Optional(rec, req)

	if cc := rec.Result().Header.Get("Cache-Control"); cc != "" {
		t.Errorf("no Set-Cookie was emitted; expected no Cache-Control override, got %q", cc)
	}
}

func TestCurrentUserHelper_Required_Returns401WhenAnonymous(t *testing.T) {
	helper := NewCurrentUserHelper(&mockAuthService{}, false)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	user, ok := helper.Required(rec, req)
	if ok || user != nil {
		t.Errorf("expected (nil, false), got (%+v, %v)", user, ok)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}
