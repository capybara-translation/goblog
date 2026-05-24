package repo

import (
	"database/sql"
	"errors"
	"fmt"
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
	if err != nil {
		return fmt.Errorf("failed to create remember token: %w", err)
	}
	return nil
}

func (s *sqliteRememberTokenStore) FindBySelector(selector string) (*domain.RememberToken, error) {
	var t domain.RememberToken
	err := s.db.Get(&t, `SELECT * FROM remember_tokens WHERE selector = ?`, selector)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find remember token by selector: %w", err)
	}
	return &t, nil
}

func (s *sqliteRememberTokenStore) Delete(selector string) error {
	if _, err := s.db.Exec(`DELETE FROM remember_tokens WHERE selector = ?`, selector); err != nil {
		return fmt.Errorf("failed to delete remember token: %w", err)
	}
	return nil
}

func (s *sqliteRememberTokenStore) DeleteByUserID(userID int64) error {
	if _, err := s.db.Exec(`DELETE FROM remember_tokens WHERE user_id = ?`, userID); err != nil {
		return fmt.Errorf("failed to delete remember tokens by user id: %w", err)
	}
	return nil
}

func (s *sqliteRememberTokenStore) RefreshOnUse(selector string, lastUsed time.Time, newExpiresAt time.Time) error {
	_, err := s.db.Exec(
		`UPDATE remember_tokens SET last_used_at = ?, expires_at = ? WHERE selector = ?`,
		lastUsed, newExpiresAt, selector,
	)
	if err != nil {
		return fmt.Errorf("failed to refresh remember token: %w", err)
	}
	return nil
}

func (s *sqliteRememberTokenStore) CleanupExpired() error {
	if _, err := s.db.Exec(`DELETE FROM remember_tokens WHERE expires_at < ?`, time.Now()); err != nil {
		return fmt.Errorf("failed to clean up expired remember tokens: %w", err)
	}
	return nil
}
