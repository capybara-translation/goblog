package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

// Session represents an in-memory authenticated session. Fields beyond
// UserID/ExpiresAt are device metadata for the "logged-in devices" admin
// feature; they live only in memory and are wiped on restart.
type Session struct {
	UserID           int64
	ExpiresAt        time.Time
	UserAgent        string    // captured at login/restore
	IP               string    // captured at login/restore
	CreatedAt        time.Time // session issue time
	LastUsedAt       time.Time // bumped by Touch on each authenticated request
	RememberSelector string    // selector of the linked remember_token ("" = ephemeral / remember-me off)
}

// SessionMeta is the device metadata supplied at session creation.
type SessionMeta struct {
	UserAgent        string
	IP               string
	RememberSelector string
}

// SessionInfo is a read-only public view of a session for the device list.
// Handle is HashSessionID(rawID) — the raw session id is never exposed.
type SessionInfo struct {
	Handle           string
	UserID           int64
	UserAgent        string
	IP               string
	CreatedAt        time.Time
	LastUsedAt       time.Time
	ExpiresAt        time.Time
	RememberSelector string
}

// SessionStore is an interface for managing sessions
type SessionStore interface {
	// Create creates a new session (with device metadata) and returns the id.
	Create(userID int64, ttl time.Duration, meta SessionMeta) (string, error)
	// Get retrieves session information from the session ID
	Get(sessionID string) (*Session, error)
	// Delete deletes a session
	Delete(sessionID string) error
	// Touch updates LastUsedAt for an existing session (no-op if absent).
	Touch(sessionID string, t time.Time)
	// SetRememberSelector links an existing session to a remember token.
	SetRememberSelector(sessionID, selector string)
	// ListByUser returns public views of all live sessions for the user.
	ListByUser(userID int64) []SessionInfo
	// DeleteByHandleForUser deletes the session whose public handle matches,
	// scoped to userID. Returns the deleted session's remember selector (empty
	// if it was an ephemeral session) and whether a matching session was found.
	DeleteByHandleForUser(userID int64, handle string) (selector string, found bool, err error)
	// DeleteBySelector deletes every session linked to the given remember
	// selector. Used so revoking a remember device also kills its live session.
	DeleteBySelector(selector string)
	// DeleteByUserExcept deletes all of the user's sessions except exceptID.
	DeleteByUserExcept(userID int64, exceptID string)
	// CleanupExpired deletes expired sessions
	CleanupExpired()
}

// inMemorySessionStore is an implementation that manages sessions in memory
type inMemorySessionStore struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

// NewInMemorySessionStore creates a new in-memory session store
func NewInMemorySessionStore() SessionStore {
	store := &inMemorySessionStore{
		sessions: make(map[string]*Session),
	}

	// Periodically clean up expired sessions
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			store.CleanupExpired()
		}
	}()

	return store
}

func (s *inMemorySessionStore) Create(userID int64, ttl time.Duration, meta SessionMeta) (string, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return "", fmt.Errorf("failed to generate session id: %w", err)
	}

	now := time.Now()
	session := &Session{
		UserID:           userID,
		ExpiresAt:        now.Add(ttl),
		UserAgent:        meta.UserAgent,
		IP:               meta.IP,
		CreatedAt:        now,
		LastUsedAt:       now,
		RememberSelector: meta.RememberSelector,
	}

	s.mu.Lock()
	s.sessions[sessionID] = session
	s.mu.Unlock()

	return sessionID, nil
}

func (s *inMemorySessionStore) Get(sessionID string) (*Session, error) {
	s.mu.RLock()
	session, exists := s.sessions[sessionID]
	s.mu.RUnlock()

	if !exists {
		return nil, nil
	}

	if time.Now().After(session.ExpiresAt) {
		s.mu.Lock()
		delete(s.sessions, sessionID)
		s.mu.Unlock()
		return nil, nil
	}

	return session, nil
}

func (s *inMemorySessionStore) Delete(sessionID string) error {
	s.mu.Lock()
	delete(s.sessions, sessionID)
	s.mu.Unlock()
	return nil
}

func (s *inMemorySessionStore) Touch(sessionID string, t time.Time) {
	s.mu.Lock()
	if session, ok := s.sessions[sessionID]; ok {
		session.LastUsedAt = t
	}
	s.mu.Unlock()
}

func (s *inMemorySessionStore) SetRememberSelector(sessionID, selector string) {
	s.mu.Lock()
	if session, ok := s.sessions[sessionID]; ok {
		session.RememberSelector = selector
	}
	s.mu.Unlock()
}

func (s *inMemorySessionStore) ListByUser(userID int64) []SessionInfo {
	now := time.Now()
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []SessionInfo
	for id, session := range s.sessions {
		if session.UserID != userID || now.After(session.ExpiresAt) {
			continue
		}
		out = append(out, SessionInfo{
			Handle:           HashSessionID(id),
			UserID:           session.UserID,
			UserAgent:        session.UserAgent,
			IP:               session.IP,
			CreatedAt:        session.CreatedAt,
			LastUsedAt:       session.LastUsedAt,
			ExpiresAt:        session.ExpiresAt,
			RememberSelector: session.RememberSelector,
		})
	}
	return out
}

func (s *inMemorySessionStore) DeleteByHandleForUser(userID int64, handle string) (string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, session := range s.sessions {
		if session.UserID == userID && HashSessionID(id) == handle {
			selector := session.RememberSelector
			delete(s.sessions, id)
			return selector, true, nil
		}
	}
	return "", false, nil
}

func (s *inMemorySessionStore) DeleteBySelector(selector string) {
	if selector == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, session := range s.sessions {
		if session.RememberSelector == selector {
			delete(s.sessions, id)
		}
	}
}

func (s *inMemorySessionStore) DeleteByUserExcept(userID int64, exceptID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, session := range s.sessions {
		if session.UserID == userID && id != exceptID {
			delete(s.sessions, id)
		}
	}
}

func (s *inMemorySessionStore) CleanupExpired() {
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	for id, session := range s.sessions {
		if now.After(session.ExpiresAt) {
			delete(s.sessions, id)
		}
	}
}

// generateSessionID generates a random session ID
// Returns Base64-encoded 32 bytes (256 bits) of random data
func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
