package repo

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/capybara-translation/goblog/internal/ogp"
	"github.com/jmoiron/sqlx"
)

// OGPRepository is an interface that provides access to OGP cache data
type OGPRepository interface {
	// FindByURL retrieves cached OGP data for a URL
	FindByURL(url string) (*ogp.Data, error)
	// Upsert inserts or updates OGP data
	Upsert(data *ogp.Data) error
	// DeleteExpired removes expired entries from the cache
	DeleteExpired() (int64, error)
}

// ogpRepository is the SQLite implementation of OGPRepository
type ogpRepository struct {
	db *sqlx.DB
}

// NewOGPRepository creates a new OGPRepository
func NewOGPRepository(db *sqlx.DB) OGPRepository {
	return &ogpRepository{db: db}
}

// FindByURL retrieves cached OGP data for a URL
func (r *ogpRepository) FindByURL(url string) (*ogp.Data, error) {
	var data ogp.Data
	query := "SELECT * FROM ogp_cache WHERE url = ?"

	err := r.db.Get(&data, query, url)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find OGP data by url: %w", err)
	}

	return &data, nil
}

// Upsert inserts or updates OGP data
func (r *ogpRepository) Upsert(data *ogp.Data) error {
	query := `
		INSERT INTO ogp_cache (url, title, description, image_url, site_name, fetched_at, expires_at, error_msg)
		VALUES (:url, :title, :description, :image_url, :site_name, :fetched_at, :expires_at, :error_msg)
		ON CONFLICT(url) DO UPDATE SET
			title = excluded.title,
			description = excluded.description,
			image_url = excluded.image_url,
			site_name = excluded.site_name,
			fetched_at = excluded.fetched_at,
			expires_at = excluded.expires_at,
			error_msg = excluded.error_msg
	`

	_, err := r.db.NamedExec(query, data)
	if err != nil {
		return fmt.Errorf("failed to upsert OGP data: %w", err)
	}

	return nil
}

// DeleteExpired removes expired entries from the cache
func (r *ogpRepository) DeleteExpired() (int64, error) {
	query := "DELETE FROM ogp_cache WHERE expires_at < ?"

	result, err := r.db.Exec(query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired OGP data: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return count, nil
}
