package repo

import (
	"fmt"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/jmoiron/sqlx"
)

// HealthRecordRepository persists Health Planet measurements.
type HealthRecordRepository interface {
	// Upsert inserts records, updating value on (measured_at, metric)
	// conflict so re-syncing the same window is idempotent and upstream
	// corrections win.
	Upsert(records []*domain.HealthRecord) error
}

type sqliteHealthRecordRepository struct {
	db *sqlx.DB
}

var _ HealthRecordRepository = (*sqliteHealthRecordRepository)(nil)

func NewHealthRecordRepository(db *sqlx.DB) HealthRecordRepository {
	return &sqliteHealthRecordRepository{db: db}
}

// measuredAtFormat pins the stored measured_at text. A single fixed layout
// (no timezone suffix) keeps the UNIQUE(measured_at, metric) key stable
// regardless of the server's TZ setting.
const measuredAtFormat = "2006-01-02 15:04:05"

func (r *sqliteHealthRecordRepository) Upsert(records []*domain.HealthRecord) error {
	if len(records) == 0 {
		return nil
	}
	tx, err := r.db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin health record upsert: %w", err)
	}
	defer tx.Rollback()

	for _, rec := range records {
		if _, err := tx.Exec(`
			INSERT INTO health_records (measured_at, metric, value)
			VALUES (?, ?, ?)
			ON CONFLICT (measured_at, metric) DO UPDATE SET value = excluded.value
		`, rec.MeasuredAt.Format(measuredAtFormat), rec.Metric, rec.Value); err != nil {
			return fmt.Errorf("failed to upsert health record (%s, %s): %w",
				rec.MeasuredAt.Format(measuredAtFormat), rec.Metric, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit health record upsert: %w", err)
	}
	return nil
}
