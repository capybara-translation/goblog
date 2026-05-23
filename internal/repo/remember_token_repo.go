package repo

import (
	"database/sql"
	"errors"
	"time"

	"github.com/capybara-translation/goblog/internal/auth"
	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/jmoiron/sqlx"
)

type sqliteRememberTokenStore struct {
	db *sqlx.DB
}

// Compile-time interface assertion.
var _ auth.RememberTokenStore = (*sqliteRememberTokenStore)(nil)

// NewSQLiteRememberTokenStore returns an auth.RememberTokenStore backed by
// SQLite. The interface itself lives in internal/auth so that callers
// (notably internal/service) can depend on auth without pulling in this
// repo package or the sqlx import that comes with it.
func NewSQLiteRememberTokenStore(db *sqlx.DB) auth.RememberTokenStore {
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
