package repo

import (
	"testing"
	"time"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// setupTestDBWithHealthRecords opens an in-memory SQLite database with the
// health_records table (mirrors migrations/011_create_health_records.sql).
func setupTestDBWithHealthRecords(t *testing.T) *sqlx.DB {
	t.Helper()
	db, err := sqlx.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	db.MustExec(`
		CREATE TABLE health_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			measured_at DATETIME NOT NULL,
			metric TEXT NOT NULL,
			value REAL NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (measured_at, metric)
		)
	`)
	return db
}

func TestHealthRecordRepository_Upsert_InsertsRecords(t *testing.T) {
	db := setupTestDBWithHealthRecords(t)
	r := NewHealthRecordRepository(db)

	measured := time.Date(2026, 7, 20, 16, 24, 0, 0, time.Local)
	records := []*domain.HealthRecord{
		{MeasuredAt: measured, Metric: domain.MetricWeight, Value: 72.10},
		{MeasuredAt: measured, Metric: domain.MetricBodyFat, Value: 20.80},
	}
	if err := r.Upsert(records); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	var count int
	if err := db.Get(&count, `SELECT COUNT(*) FROM health_records`); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestHealthRecordRepository_Upsert_IsIdempotent(t *testing.T) {
	db := setupTestDBWithHealthRecords(t)
	r := NewHealthRecordRepository(db)

	measured := time.Date(2026, 7, 20, 16, 24, 0, 0, time.Local)
	records := []*domain.HealthRecord{
		{MeasuredAt: measured, Metric: domain.MetricWeight, Value: 72.10},
	}
	if err := r.Upsert(records); err != nil {
		t.Fatalf("first Upsert: %v", err)
	}
	// Same measurement fetched again on a later sync — value corrected upstream.
	records[0].Value = 72.30
	if err := r.Upsert(records); err != nil {
		t.Fatalf("second Upsert: %v", err)
	}

	var count int
	if err := db.Get(&count, `SELECT COUNT(*) FROM health_records`); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1 (no duplicate row)", count)
	}
	var value float64
	if err := db.Get(&value, `SELECT value FROM health_records`); err != nil {
		t.Fatalf("value: %v", err)
	}
	if value != 72.30 {
		t.Errorf("value = %v, want 72.30 (updated)", value)
	}
}

func TestHealthRecordRepository_Upsert_EmptyIsNoop(t *testing.T) {
	db := setupTestDBWithHealthRecords(t)
	r := NewHealthRecordRepository(db)
	if err := r.Upsert(nil); err != nil {
		t.Fatalf("Upsert(nil): %v", err)
	}
}
