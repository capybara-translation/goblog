package auth

import (
	"database/sql"
	"errors"
	"time"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/jmoiron/sqlx"
)

type sqliteRememberTokenStore struct {
	db *sqlx.DB
}

// Compile-time interface assertion.
var _ RememberTokenStore = (*sqliteRememberTokenStore)(nil)

// NewSQLiteRememberTokenStore returns a RememberTokenStore backed by SQLite.
func NewSQLiteRememberTokenStore(db *sqlx.DB) RememberTokenStore {
	return &sqliteRememberTokenStore{db: db}
}

func (s *sqliteRememberTokenStore) Create(token *domain.RememberToken) error {
	_, err := s.db.Exec(`
		INSERT INTO remember_tokens (user_id, selector, token_hash, expires_at)
		VALUES (?, ?, ?, ?)
	`, token.UserID, token.Selector, token.TokenHash, token.ExpiresAt)
	return err
}

func (s *sqliteRememberTokenStore) FindBySelector(selector string) (*domain.RememberToken, error) {
	var t domain.RememberToken
	err := s.db.Get(&t, `SELECT * FROM remember_tokens WHERE selector = ?`, selector)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *sqliteRememberTokenStore) Delete(selector string) error {
	_, err := s.db.Exec(`DELETE FROM remember_tokens WHERE selector = ?`, selector)
	return err
}

func (s *sqliteRememberTokenStore) DeleteByUserID(userID int64) error {
	_, err := s.db.Exec(`DELETE FROM remember_tokens WHERE user_id = ?`, userID)
	return err
}

func (s *sqliteRememberTokenStore) RefreshOnUse(selector string, lastUsed time.Time, newExpiresAt time.Time) error {
	_, err := s.db.Exec(
		`UPDATE remember_tokens SET last_used_at = ?, expires_at = ? WHERE selector = ?`,
		lastUsed, newExpiresAt, selector,
	)
	return err
}

func (s *sqliteRememberTokenStore) CleanupExpired() error {
	_, err := s.db.Exec(`DELETE FROM remember_tokens WHERE expires_at < ?`, time.Now())
	return err
}

// StartRememberTokenCleanupLoop runs CleanupExpired() periodically. The
// returned channel can be closed to stop the loop (useful in tests; production
// callers may simply leak it on shutdown).
func StartRememberTokenCleanupLoop(store RememberTokenStore, interval time.Duration) chan<- struct{} {
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_ = store.CleanupExpired()
			case <-stop:
				return
			}
		}
	}()
	return stop
}
