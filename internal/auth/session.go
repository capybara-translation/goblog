package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

// Session represents session information
type Session struct {
	UserID    int64
	ExpiresAt time.Time
}

// SessionStore is an interface for managing sessions
type SessionStore interface {
	// Create creates a new session and returns the session ID
	Create(userID int64, ttl time.Duration) (string, error)

	// Get retrieves session information from the session ID
	Get(sessionID string) (*Session, error)

	// Delete deletes a session
	Delete(sessionID string) error

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

// Create creates a new session and returns the session ID
func (s *inMemorySessionStore) Create(userID int64, ttl time.Duration) (string, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return "", fmt.Errorf("failed to generate session id: %w", err)
	}

	session := &Session{
		UserID:    userID,
		ExpiresAt: time.Now().Add(ttl),
	}

	s.mu.Lock()
	s.sessions[sessionID] = session
	s.mu.Unlock()

	return sessionID, nil
}

// Get retrieves session information from the session ID
func (s *inMemorySessionStore) Get(sessionID string) (*Session, error) {
	s.mu.RLock()
	session, exists := s.sessions[sessionID]
	s.mu.RUnlock()

	if !exists {
		return nil, nil
	}

	// Check for expiration
	if time.Now().After(session.ExpiresAt) {
		// Delete expired session
		s.mu.Lock()
		delete(s.sessions, sessionID)
		s.mu.Unlock()
		return nil, nil
	}

	return session, nil
}

// Delete deletes a session
func (s *inMemorySessionStore) Delete(sessionID string) error {
	s.mu.Lock()
	delete(s.sessions, sessionID)
	s.mu.Unlock()
	return nil
}

// CleanupExpired deletes expired sessions
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
