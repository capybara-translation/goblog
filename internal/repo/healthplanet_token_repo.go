package repo

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/jmoiron/sqlx"
)

// HealthPlanetTokenRepository persists the single Health Planet OAuth token.
type HealthPlanetTokenRepository interface {
	// Load returns the stored token, or (nil, nil) when none exists yet.
	Load() (*domain.HealthPlanetToken, error)
	// Save stores the token, overwriting any previous one.
	Save(accessToken, refreshToken string, expiresAt time.Time) error
}

type sqliteHealthPlanetTokenRepository struct {
	db *sqlx.DB
}

var _ HealthPlanetTokenRepository = (*sqliteHealthPlanetTokenRepository)(nil)

func NewHealthPlanetTokenRepository(db *sqlx.DB) HealthPlanetTokenRepository {
	return &sqliteHealthPlanetTokenRepository{db: db}
}

func (r *sqliteHealthPlanetTokenRepository) Load() (*domain.HealthPlanetToken, error) {
	var t domain.HealthPlanetToken
	err := r.db.Get(&t, `SELECT * FROM healthplanet_tokens WHERE id = 1`)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load healthplanet token: %w", err)
	}
	return &t, nil
}

func (r *sqliteHealthPlanetTokenRepository) Save(accessToken, refreshToken string, expiresAt time.Time) error {
	_, err := r.db.Exec(`
		INSERT INTO healthplanet_tokens (id, access_token, refresh_token, expires_at, updated_at)
		VALUES (1, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT (id) DO UPDATE SET
			access_token = excluded.access_token,
			refresh_token = excluded.refresh_token,
			expires_at = excluded.expires_at,
			updated_at = CURRENT_TIMESTAMP
	`, accessToken, refreshToken, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to save healthplanet token: %w", err)
	}
	return nil
}
