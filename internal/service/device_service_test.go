package service

import (
	"testing"
	"time"

	"github.com/capybara-translation/goblog/internal/auth"
	"github.com/capybara-translation/goblog/internal/domain"
)

// fakeRememberStore is a minimal in-memory RememberTokenStore for device tests.
type fakeRememberStore struct {
	tokens map[string]*domain.RememberToken // keyed by selector
}

func newFakeRememberStore() *fakeRememberStore {
	return &fakeRememberStore{tokens: map[string]*domain.RememberToken{}}
}
func (f *fakeRememberStore) Create(t *domain.RememberToken) error {
	f.tokens[t.Selector] = t
	return nil
}
func (f *fakeRememberStore) FindBySelector(sel string) (*domain.RememberToken, error) {
	return f.tokens[sel], nil
}
func (f *fakeRememberStore) Delete(sel string) error { delete(f.tokens, sel); return nil }
func (f *fakeRememberStore) DeleteByUserID(userID int64) error {
	for s, t := range f.tokens {
		if t.UserID == userID {
			delete(f.tokens, s)
		}
	}
	return nil
}
func (f *fakeRememberStore) RefreshOnUse(sel string, lu time.Time, exp time.Time) error { return nil }
func (f *fakeRememberStore) CleanupExpired() error                                      { return nil }
func (f *fakeRememberStore) FindByUserID(userID int64) ([]*domain.RememberToken, error) {
	var out []*domain.RememberToken
	for _, t := range f.tokens {
		if t.UserID == userID {
			out = append(out, t)
		}
	}
	return out, nil
}
func (f *fakeRememberStore) DeleteByUserExceptSelector(userID int64, keep string) error {
	for s, t := range f.tokens {
		if t.UserID == userID && s != keep {
			delete(f.tokens, s)
		}
	}
	return nil
}

func TestListDevices_MergesRememberAndEphemeral(t *testing.T) {
	sessions := auth.NewInMemorySessionStore()
	remember := newFakeRememberStore()
	svc := NewDeviceService(sessions, remember)

	remember.Create(&domain.RememberToken{
		UserID: 1, Selector: "sel-mac", TokenHash: "h",
		ExpiresAt: time.Now().Add(time.Hour),
		UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/603.3.8 (KHTML, like Gecko) Version/10.1.2 Safari/603.3.8",
		IPAddress: "203.0.113.1",
	})
	macSession, _ := sessions.Create(1, time.Hour, auth.SessionMeta{RememberSelector: "sel-mac", UserAgent: "x"})
	ephID, _ := sessions.Create(1, time.Hour, auth.SessionMeta{
		UserAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 10_3_2 like Mac OS X) AppleWebKit/603.2.4 (KHTML, like Gecko) Version/10.0 Mobile/14F89 Safari/602.1",
		IP:        "203.0.113.2",
	})

	devices, err := svc.ListDevices(1, ephID, "sel-mac")
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if len(devices) != 2 {
		t.Fatalf("expected 2 devices, got %d: %+v", len(devices), devices)
	}
	var mac, iphone *Device
	for i := range devices {
		switch devices[i].Kind {
		case "remember":
			mac = &devices[i]
		case "session":
			iphone = &devices[i]
		}
	}
	if mac == nil || mac.Browser != "Safari" || mac.IP != "203.0.113.1" {
		t.Fatalf("bad mac row: %+v", mac)
	}
	if mac.ID != "sel-mac" || !mac.IsCurrent {
		t.Fatalf("mac should be current (selector match): %+v", mac)
	}
	if iphone == nil || !iphone.IsEphemeral || iphone.IsCurrent != true {
		t.Fatalf("iphone ephemeral should be current (session match): %+v", iphone)
	}
	_ = macSession
}

func TestListDevices_ExpiredTokenHiddenLinkedSessionShown(t *testing.T) {
	sessions := auth.NewInMemorySessionStore()
	remember := newFakeRememberStore()
	svc := NewDeviceService(sessions, remember)

	expired := time.Now().Add(-time.Minute)
	remember.Create(&domain.RememberToken{
		UserID: 1, Selector: "sel-exp", TokenHash: "h",
		ExpiresAt: expired, CreatedAt: time.Now().Add(-time.Hour),
		UserAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 10_3_2 like Mac OS X) AppleWebKit/603.2.4 (KHTML, like Gecko) Version/10.0 Mobile/14F89 Safari/602.1",
	})
	// A live session still linked to that now-expired token's selector.
	linked, _ := sessions.Create(1, time.Hour, auth.SessionMeta{RememberSelector: "sel-exp", IP: "203.0.113.9"})

	devices, err := svc.ListDevices(1, "", "")
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if len(devices) != 1 {
		t.Fatalf("expected exactly 1 device (the live session; expired token hidden), got %d: %+v", len(devices), devices)
	}
	d := devices[0]
	if d.Kind != "session" || !d.IsEphemeral || d.IP != "203.0.113.9" {
		t.Fatalf("expected the orphaned live session row, got %+v", d)
	}
	_ = linked
}

func TestRevokeDevice_RememberAlsoKillsSession(t *testing.T) {
	sessions := auth.NewInMemorySessionStore()
	remember := newFakeRememberStore()
	svc := NewDeviceService(sessions, remember)

	remember.Create(&domain.RememberToken{UserID: 1, Selector: "sel-x", TokenHash: "h", ExpiresAt: time.Now().Add(time.Hour)})
	linked, _ := sessions.Create(1, time.Hour, auth.SessionMeta{RememberSelector: "sel-x"})

	current, _ := sessions.Create(1, time.Hour, auth.SessionMeta{})
	if err := svc.RevokeDevice(1, "remember", "sel-x", current, ""); err != nil {
		t.Fatalf("RevokeDevice: %v", err)
	}
	if tok, _ := remember.FindBySelector("sel-x"); tok != nil {
		t.Fatalf("remember token should be deleted")
	}
	if s, _ := sessions.Get(linked); s != nil {
		t.Fatalf("linked session must be killed so the device is truly logged out")
	}
}

func TestRevokeDevice_CannotRevokeCurrent(t *testing.T) {
	sessions := auth.NewInMemorySessionStore()
	remember := newFakeRememberStore()
	svc := NewDeviceService(sessions, remember)
	remember.Create(&domain.RememberToken{UserID: 1, Selector: "sel-cur", TokenHash: "h", ExpiresAt: time.Now().Add(time.Hour)})

	err := svc.RevokeDevice(1, "remember", "sel-cur", "anything", "sel-cur")
	if err != ErrCannotRevokeCurrent {
		t.Fatalf("expected ErrCannotRevokeCurrent, got %v", err)
	}
}

func TestRevokeDevice_OtherUserNotFound(t *testing.T) {
	sessions := auth.NewInMemorySessionStore()
	remember := newFakeRememberStore()
	svc := NewDeviceService(sessions, remember)
	remember.Create(&domain.RememberToken{UserID: 2, Selector: "sel-foreign", TokenHash: "h", ExpiresAt: time.Now().Add(time.Hour)})

	err := svc.RevokeDevice(1, "remember", "sel-foreign", "cur", "")
	if err != ErrDeviceNotFound {
		t.Fatalf("expected ErrDeviceNotFound for another user's token, got %v", err)
	}
}

func TestListDevices_SortedByLastUsedDesc(t *testing.T) {
	sessions := auth.NewInMemorySessionStore()
	remember := newFakeRememberStore()
	svc := NewDeviceService(sessions, remember)

	now := time.Now()
	older := now.Add(-2 * time.Hour)
	newer := now.Add(-1 * time.Hour)

	remember.Create(&domain.RememberToken{
		UserID: 1, Selector: "sel-old", TokenHash: "h",
		ExpiresAt: now.Add(time.Hour), CreatedAt: older, LastUsedAt: &older,
		UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/603.3.8 (KHTML, like Gecko) Version/10.1.2 Safari/603.3.8",
	})
	remember.Create(&domain.RememberToken{
		UserID: 1, Selector: "sel-new", TokenHash: "h",
		ExpiresAt: now.Add(time.Hour), CreatedAt: newer, LastUsedAt: &newer,
		UserAgent: "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/59.0.3071.115 Safari/537.36",
	})

	devices, err := svc.ListDevices(1, "", "")
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if len(devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(devices))
	}
	if devices[0].ID != "sel-new" || devices[1].ID != "sel-old" {
		t.Fatalf("expected most-recently-used first, got %s then %s", devices[0].ID, devices[1].ID)
	}
}

func TestLogoutOtherDevices_KeepsCurrent(t *testing.T) {
	sessions := auth.NewInMemorySessionStore()
	remember := newFakeRememberStore()
	svc := NewDeviceService(sessions, remember)

	remember.Create(&domain.RememberToken{UserID: 1, Selector: "keep", TokenHash: "h", ExpiresAt: time.Now().Add(time.Hour)})
	remember.Create(&domain.RememberToken{UserID: 1, Selector: "drop", TokenHash: "h", ExpiresAt: time.Now().Add(time.Hour)})
	curSession, _ := sessions.Create(1, time.Hour, auth.SessionMeta{RememberSelector: "keep"})
	otherSession, _ := sessions.Create(1, time.Hour, auth.SessionMeta{})

	if err := svc.LogoutOtherDevices(1, curSession, "keep"); err != nil {
		t.Fatalf("LogoutOtherDevices: %v", err)
	}
	if tok, _ := remember.FindBySelector("keep"); tok == nil {
		t.Fatalf("current remember token must survive")
	}
	if tok, _ := remember.FindBySelector("drop"); tok != nil {
		t.Fatalf("other remember token must be deleted")
	}
	if s, _ := sessions.Get(curSession); s == nil {
		t.Fatalf("current session must survive")
	}
	if s, _ := sessions.Get(otherSession); s != nil {
		t.Fatalf("other session must be deleted")
	}
}

// When the current device is an ephemeral session (no remember token, so
// currentSelector == ""), "log out other devices" must revoke ALL remember
// tokens for the user (the current ephemeral session owns none to keep) while
// preserving the current session itself. This locks in the empty-selector
// semantics of DeleteByUserExceptSelector(userID, "").
func TestLogoutOtherDevices_EphemeralCurrentRevokesAllRememberTokens(t *testing.T) {
	sessions := auth.NewInMemorySessionStore()
	remember := newFakeRememberStore()
	svc := NewDeviceService(sessions, remember)

	remember.Create(&domain.RememberToken{UserID: 1, Selector: "a", TokenHash: "h", ExpiresAt: time.Now().Add(time.Hour)})
	remember.Create(&domain.RememberToken{UserID: 1, Selector: "b", TokenHash: "h", ExpiresAt: time.Now().Add(time.Hour)})
	// Current device is an ephemeral session: it has no linked selector.
	curEphemeral, _ := sessions.Create(1, time.Hour, auth.SessionMeta{})

	if err := svc.LogoutOtherDevices(1, curEphemeral, ""); err != nil {
		t.Fatalf("LogoutOtherDevices: %v", err)
	}
	if toks, _ := remember.FindByUserID(1); len(toks) != 0 {
		t.Fatalf("all remember tokens should be revoked, got %d", len(toks))
	}
	if s, _ := sessions.Get(curEphemeral); s == nil {
		t.Fatalf("current ephemeral session must survive")
	}
}
