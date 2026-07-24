package repo

import (
	"fmt"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/jmoiron/sqlx"
)

// DailyAverage is one metric's average value for one calendar day.
// Date stays a string end-to-end (see measuredAtFormat comment).
type DailyAverage struct {
	Date   string  `db:"date"` // YYYY-MM-DD
	Metric string  `db:"metric"`
	Avg    float64 `db:"avg"`
}

// HealthRecordRepository persists Health Planet measurements.
type HealthRecordRepository interface {
	// Upsert inserts records, updating value on (measured_at, metric)
	// conflict so re-syncing the same window is idempotent and upstream
	// corrections win.
	Upsert(records []*domain.HealthRecord) error
	// DailyAverages returns the daily average for each metric in the date range (inclusive).
	DailyAverages(fromDate, toDate string) ([]DailyAverage, error)
	// DailyAveragesByDates returns the daily average for each metric on the given dates.
	// If dates is empty or nil, returns (nil, nil).
	DailyAveragesByDates(dates []string) ([]DailyAverage, error)
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
// regardless of the server's TZ setting. When reading this column back as
// time.Time, the SQLite driver (without a _loc DSN param) interprets it as
// UTC; callers that need local time must scan it as a string and re-parse with
// time.ParseInLocation using the desired location.
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

func (r *sqliteHealthRecordRepository) DailyAverages(fromDate, toDate string) ([]DailyAverage, error) {
	var rows []DailyAverage
	err := r.db.Select(&rows, `
		SELECT date(measured_at) AS date, metric, AVG(value) AS avg
		FROM health_records
		WHERE date(measured_at) BETWEEN ? AND ?
		GROUP BY date(measured_at), metric
		ORDER BY date
	`, fromDate, toDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query daily averages: %w", err)
	}
	return rows, nil
}

func (r *sqliteHealthRecordRepository) DailyAveragesByDates(dates []string) ([]DailyAverage, error) {
	if len(dates) == 0 {
		return nil, nil
	}
	query, args, err := sqlx.In(`
		SELECT date(measured_at) AS date, metric, AVG(value) AS avg
		FROM health_records
		WHERE date(measured_at) IN (?)
		GROUP BY date(measured_at), metric
	`, dates)
	if err != nil {
		return nil, fmt.Errorf("failed to build daily averages query: %w", err)
	}
	var rows []DailyAverage
	if err := r.db.Select(&rows, r.db.Rebind(query), args...); err != nil {
		return nil, fmt.Errorf("failed to query daily averages by dates: %w", err)
	}
	return rows, nil
}
