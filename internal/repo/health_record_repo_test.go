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

func seedHealthRecords(t *testing.T, r HealthRecordRepository) {
	t.Helper()
	mk := func(ts, metric string, v float64) *domain.HealthRecord {
		parsed, err := time.ParseInLocation("2006-01-02 15:04:05", ts, time.Local)
		if err != nil {
			t.Fatalf("parse %s: %v", ts, err)
		}
		return &domain.HealthRecord{MeasuredAt: parsed, Metric: metric, Value: v}
	}
	records := []*domain.HealthRecord{
		mk("2026-07-20 08:00:00", domain.MetricWeight, 72.0),
		mk("2026-07-20 21:00:00", domain.MetricWeight, 73.0), // 同日2回 → 平均 72.5
		mk("2026-07-20 08:00:00", domain.MetricSystolic, 120),
		mk("2026-07-21 08:00:00", domain.MetricWeight, 71.0),
		mk("2026-07-25 08:00:00", domain.MetricPulse, 60), // 範囲外検証用
	}
	if err := r.Upsert(records); err != nil {
		t.Fatalf("seed: %v", err)
	}
}

func TestHealthRecordRepository_DailyAverages(t *testing.T) {
	db := setupTestDBWithHealthRecords(t)
	r := NewHealthRecordRepository(db)
	seedHealthRecords(t, r)

	rows, err := r.DailyAverages("2026-07-20", "2026-07-21")
	if err != nil {
		t.Fatalf("DailyAverages: %v", err)
	}
	if len(rows) != 3 { // 7/20 weight, 7/20 systolic, 7/21 weight（7/25 pulse は範囲外）
		t.Fatalf("len = %d, want 3: %+v", len(rows), rows)
	}
	byKey := map[string]float64{}
	for _, row := range rows {
		byKey[row.Date+"/"+row.Metric] = row.Avg
	}
	if byKey["2026-07-20/"+domain.MetricWeight] != 72.5 {
		t.Errorf("7/20 weight avg = %v, want 72.5", byKey["2026-07-20/"+domain.MetricWeight])
	}
	if byKey["2026-07-21/"+domain.MetricWeight] != 71.0 {
		t.Errorf("7/21 weight avg = %v, want 71.0", byKey["2026-07-21/"+domain.MetricWeight])
	}
}

func TestHealthRecordRepository_DailyAveragesByDates(t *testing.T) {
	db := setupTestDBWithHealthRecords(t)
	r := NewHealthRecordRepository(db)
	seedHealthRecords(t, r)

	rows, err := r.DailyAveragesByDates([]string{"2026-07-20", "2026-07-25"})
	if err != nil {
		t.Fatalf("DailyAveragesByDates: %v", err)
	}
	if len(rows) != 3 { // 7/20 weight+systolic, 7/25 pulse
		t.Fatalf("len = %d, want 3: %+v", len(rows), rows)
	}

	empty, err := r.DailyAveragesByDates(nil)
	if err != nil {
		t.Fatalf("empty dates: %v", err)
	}
	if empty != nil {
		t.Errorf("empty dates should return nil, got %+v", empty)
	}
}
