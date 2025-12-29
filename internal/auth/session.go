package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

// Session はセッション情報を表します
type Session struct {
	UserID    int64
	ExpiresAt time.Time
}

// SessionStore はセッションを管理するインターフェースです
type SessionStore interface {
	// Create は新しいセッションを作成し、セッションIDを返します
	Create(userID int64, ttl time.Duration) (string, error)

	// Get はセッションIDからセッション情報を取得します
	Get(sessionID string) (*Session, error)

	// Delete はセッションを削除します
	Delete(sessionID string) error

	// CleanupExpired は期限切れのセッションを削除します
	CleanupExpired()
}

// inMemorySessionStore はインメモリでセッションを管理する実装です
type inMemorySessionStore struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

// NewInMemorySessionStore は新しいインメモリセッションストアを作成します
func NewInMemorySessionStore() SessionStore {
	store := &inMemorySessionStore{
		sessions: make(map[string]*Session),
	}

	// 定期的に期限切れセッションをクリーンアップ
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			store.CleanupExpired()
		}
	}()

	return store
}

// Create は新しいセッションを作成し、セッションIDを返します
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

// Get はセッションIDからセッション情報を取得します
func (s *inMemorySessionStore) Get(sessionID string) (*Session, error) {
	s.mu.RLock()
	session, exists := s.sessions[sessionID]
	s.mu.RUnlock()

	if !exists {
		return nil, nil
	}

	// 期限切れチェック
	if time.Now().After(session.ExpiresAt) {
		// 期限切れセッションを削除
		s.mu.Lock()
		delete(s.sessions, sessionID)
		s.mu.Unlock()
		return nil, nil
	}

	return session, nil
}

// Delete はセッションを削除します
func (s *inMemorySessionStore) Delete(sessionID string) error {
	s.mu.Lock()
	delete(s.sessions, sessionID)
	s.mu.Unlock()
	return nil
}

// CleanupExpired は期限切れのセッションを削除します
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

// generateSessionID はランダムなセッションIDを生成します
// 32バイト（256ビット）のランダムデータをBase64エンコードして返します
func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
