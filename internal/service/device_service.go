package service

import (
	"errors"
	"sort"
	"time"

	"github.com/capybara-translation/goblog/internal/auth"
	"github.com/mileusna/useragent"
)

var (
	// ErrCannotRevokeCurrent is returned when a caller tries to revoke the
	// device it is currently using (the UI disables this; the service enforces it).
	ErrCannotRevokeCurrent = errors.New("cannot revoke the current device")
	// ErrDeviceNotFound is returned when no device matches the given kind/id for
	// the user (also used for another user's id, to avoid leaking existence).
	ErrDeviceNotFound = errors.New("device not found")
	// ErrInvalidDeviceKind is returned for a kind other than remember/session.
	ErrInvalidDeviceKind = errors.New("invalid device kind")
)

// Device is one row of the logged-in devices list.
type Device struct {
	Kind        string    `json:"kind"` // "remember" | "session"
	ID          string    `json:"id"`   // selector (remember) or session handle (session)
	Device      string    `json:"device"`
	Browser     string    `json:"browser"`
	OS          string    `json:"os"`
	IP          string    `json:"ip"`
	LastUsedAt  time.Time `json:"last_used_at"`
	CreatedAt   time.Time `json:"created_at"`
	IsCurrent   bool      `json:"is_current"`
	IsEphemeral bool      `json:"is_ephemeral"` // true for in-memory (remember-me off) sessions
}

// DeviceService lists and revokes logged-in devices.
type DeviceService interface {
	ListDevices(userID int64, currentSessionID, currentSelector string) ([]Device, error)
	RevokeDevice(userID int64, kind, id, currentSessionID, currentSelector string) error
	LogoutOtherDevices(userID int64, currentSessionID, currentSelector string) error
}

type deviceService struct {
	sessions auth.SessionStore
	remember auth.RememberTokenStore
}

// NewDeviceService creates a DeviceService over the session and remember stores.
func NewDeviceService(sessions auth.SessionStore, remember auth.RememberTokenStore) DeviceService {
	return &deviceService{sessions: sessions, remember: remember}
}

func (s *deviceService) ListDevices(userID int64, currentSessionID, currentSelector string) ([]Device, error) {
	currentHandle := auth.HashSessionID(currentSessionID)

	sessionInfos := s.sessions.ListByUser(userID)
	// Index linked-session last-used by selector so a remember row reflects the
	// freshest activity (token.last_used_at only updates on restore).
	linkedLastUsed := map[string]time.Time{}
	for _, si := range sessionInfos {
		if si.RememberSelector != "" {
			if t, ok := linkedLastUsed[si.RememberSelector]; !ok || si.LastUsedAt.After(t) {
				linkedLastUsed[si.RememberSelector] = si.LastUsedAt
			}
		}
	}

	tokens, err := s.remember.FindByUserID(userID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	liveSelectors := make(map[string]bool)

	devices := make([]Device, 0, len(tokens)+len(sessionInfos))

	// Remember rows (the backbone). Skip expired tokens — an expired remember
	// token is effectively logged out and must not appear as an active device.
	for _, tok := range tokens {
		if now.After(tok.ExpiresAt) {
			continue
		}
		liveSelectors[tok.Selector] = true

		lastUsed := tok.CreatedAt
		if tok.LastUsedAt != nil && tok.LastUsedAt.After(lastUsed) {
			lastUsed = *tok.LastUsedAt
		}
		if lu, ok := linkedLastUsed[tok.Selector]; ok && lu.After(lastUsed) {
			lastUsed = lu
		}
		dev, browser, os := parseUA(tok.UserAgent)
		devices = append(devices, Device{
			Kind:       "remember",
			ID:         tok.Selector,
			Device:     dev,
			Browser:    browser,
			OS:         os,
			IP:         tok.IPAddress,
			LastUsedAt: lastUsed,
			CreatedAt:  tok.CreatedAt,
			IsCurrent:  currentSelector != "" && tok.Selector == currentSelector,
		})
	}

	// Session rows. A session linked to a LIVE remember token is already
	// represented by that token's row, so skip it. A session whose selector has
	// no live token (expired/swept) is still a live login and must be shown.
	for _, si := range sessionInfos {
		if si.RememberSelector != "" && liveSelectors[si.RememberSelector] {
			continue
		}
		dev, browser, os := parseUA(si.UserAgent)
		devices = append(devices, Device{
			Kind:        "session",
			ID:          si.Handle,
			Device:      dev,
			Browser:     browser,
			OS:          os,
			IP:          si.IP,
			LastUsedAt:  si.LastUsedAt,
			CreatedAt:   si.CreatedAt,
			IsCurrent:   si.Handle == currentHandle,
			IsEphemeral: true,
		})
	}

	sort.SliceStable(devices, func(i, j int) bool {
		if !devices[i].LastUsedAt.Equal(devices[j].LastUsedAt) {
			return devices[i].LastUsedAt.After(devices[j].LastUsedAt)
		}
		return devices[i].ID < devices[j].ID
	})

	return devices, nil
}

func (s *deviceService) RevokeDevice(userID int64, kind, id, currentSessionID, currentSelector string) error {
	switch kind {
	case "remember":
		if currentSelector != "" && id == currentSelector {
			return ErrCannotRevokeCurrent
		}
		tok, err := s.remember.FindBySelector(id)
		if err != nil {
			return err
		}
		if tok == nil || tok.UserID != userID {
			return ErrDeviceNotFound
		}
		if err := s.remember.Delete(id); err != nil {
			return err
		}
		// Kill any live session linked to this device so revocation is real.
		s.sessions.DeleteBySelector(id)
		return nil
	case "session":
		if id == auth.HashSessionID(currentSessionID) {
			return ErrCannotRevokeCurrent
		}
		found, err := s.sessions.DeleteByHandleForUser(userID, id)
		if err != nil {
			return err
		}
		if !found {
			return ErrDeviceNotFound
		}
		return nil
	default:
		return ErrInvalidDeviceKind
	}
}

func (s *deviceService) LogoutOtherDevices(userID int64, currentSessionID, currentSelector string) error {
	if err := s.remember.DeleteByUserExceptSelector(userID, currentSelector); err != nil {
		return err
	}
	s.sessions.DeleteByUserExcept(userID, currentSessionID)
	return nil
}

// parseUA derives display labels from a raw user-agent string. Empty UA (rows
// issued before the migration, or sessions with no UA) become "不明な端末".
func parseUA(raw string) (device, browser, os string) {
	if raw == "" {
		return "不明な端末", "", ""
	}
	ua := useragent.Parse(raw)
	device = ua.Device
	if device == "" {
		device = ua.OS // e.g. "Mac", "Windows"
	}
	if device == "" {
		device = "不明な端末"
	}
	return device, ua.Name, ua.OS
}
