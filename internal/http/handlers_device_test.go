package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/capybara-translation/goblog/internal/auth"
	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/capybara-translation/goblog/internal/service"
	"github.com/gorilla/mux"
)

// fakeRememberStoreHTTP is a minimal auth.RememberTokenStore for handler tests.
type fakeRememberStoreHTTP struct {
	tokens map[string]*domain.RememberToken
}

func newFakeRememberStoreHTTP() *fakeRememberStoreHTTP {
	return &fakeRememberStoreHTTP{tokens: map[string]*domain.RememberToken{}}
}
func (f *fakeRememberStoreHTTP) Create(t *domain.RememberToken) error {
	f.tokens[t.Selector] = t
	return nil
}
func (f *fakeRememberStoreHTTP) FindBySelector(sel string) (*domain.RememberToken, error) {
	return f.tokens[sel], nil
}
func (f *fakeRememberStoreHTTP) Delete(sel string) error { delete(f.tokens, sel); return nil }
func (f *fakeRememberStoreHTTP) DeleteByUserID(userID int64) error {
	for s, t := range f.tokens {
		if t.UserID == userID {
			delete(f.tokens, s)
		}
	}
	return nil
}
func (f *fakeRememberStoreHTTP) RefreshOnUse(sel string, lu time.Time, exp time.Time) error {
	return nil
}
func (f *fakeRememberStoreHTTP) CleanupExpired() error { return nil }
func (f *fakeRememberStoreHTTP) FindByUserID(userID int64) ([]*domain.RememberToken, error) {
	var out []*domain.RememberToken
	for _, t := range f.tokens {
		if t.UserID == userID {
			out = append(out, t)
		}
	}
	return out, nil
}
func (f *fakeRememberStoreHTTP) DeleteByUserExceptSelector(userID int64, keep string) error {
	for s, t := range f.tokens {
		if t.UserID == userID && s != keep {
			delete(f.tokens, s)
		}
	}
	return nil
}

func newDeviceTestRouter(svc service.DeviceService) *mux.Router {
	h := NewDeviceHandlers(svc)
	r := mux.NewRouter()
	api := r.PathPrefix("/api/v1").Subrouter()
	api.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), contextKeyUserID, int64(1))
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	api.HandleFunc("/devices", h.HandleList).Methods("GET")
	api.HandleFunc("/devices/logout-others", h.HandleLogoutOthers).Methods("POST")
	api.HandleFunc("/devices/{kind:remember|session}/{id}", h.HandleRevoke).Methods("DELETE")
	return r
}

func TestHandleList_ReturnsDevices(t *testing.T) {
	sessions := auth.NewInMemorySessionStore()
	remember := newFakeRememberStoreHTTP()
	svc := service.NewDeviceService(sessions, remember)
	sessions.Create(1, time.Hour, auth.SessionMeta{UserAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 10_3_2 like Mac OS X) AppleWebKit/603.2.4 (KHTML, like Gecko) Version/10.0 Mobile/14F89 Safari/602.1"})

	r := newDeviceTestRouter(svc)
	req := httptest.NewRequest("GET", "/api/v1/devices", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var out []service.Device
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out) != 1 || out[0].Browser != "Safari" {
		t.Fatalf("unexpected devices: %+v", out)
	}
}

// An unknown kind no longer reaches the handler: the route is constrained to
// {kind:remember|session}, so an unmatched kind yields 404 at the router. The
// service-layer ErrInvalidDeviceKind (400) remains as defense-in-depth and is
// covered by a unit test in the service package.
func TestHandleRevoke_UnknownKindNotRouted(t *testing.T) {
	svc := service.NewDeviceService(auth.NewInMemorySessionStore(), newFakeRememberStoreHTTP())
	r := newDeviceTestRouter(svc)
	req := httptest.NewRequest("DELETE", "/api/v1/devices/bogus/xyz", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown kind (route not matched), got %d", rec.Code)
	}
}

func TestHandleRevoke_CurrentForbidden(t *testing.T) {
	sessions := auth.NewInMemorySessionStore()
	remember := newFakeRememberStoreHTTP()
	svc := service.NewDeviceService(sessions, remember)
	remember.Create(&domain.RememberToken{UserID: 1, Selector: "sel-cur", TokenHash: "h", ExpiresAt: time.Now().Add(time.Hour)})

	r := newDeviceTestRouter(svc)
	req := httptest.NewRequest("DELETE", "/api/v1/devices/remember/sel-cur", nil)
	// Present the remember cookie so the server considers sel-cur the current device.
	req.AddCookie(&http.Cookie{Name: rememberCookieName, Value: "sel-cur:rawtokenvalue"})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 revoking current device, got %d", rec.Code)
	}
}

func TestHandleLogoutOthers_OK(t *testing.T) {
	sessions := auth.NewInMemorySessionStore()
	remember := newFakeRememberStoreHTTP()
	svc := service.NewDeviceService(sessions, remember)
	remember.Create(&domain.RememberToken{UserID: 1, Selector: "drop", TokenHash: "h", ExpiresAt: time.Now().Add(time.Hour)})

	r := newDeviceTestRouter(svc)
	req := httptest.NewRequest("POST", "/api/v1/devices/logout-others", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if tok, _ := remember.FindBySelector("drop"); tok != nil {
		t.Fatalf("other device token should be revoked")
	}
}
