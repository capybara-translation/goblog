package domain

import "time"

// Health metrics stored in health_records.metric. They correspond 1:1 to
// Health Planet measurement tags (see internal/healthplanet).
const (
	MetricWeight    = "weight"    // kg
	MetricBodyFat   = "body_fat"  // %
	MetricSystolic  = "systolic"  // mmHg
	MetricDiastolic = "diastolic" // mmHg
	MetricPulse     = "pulse"     // bpm
)

// HealthRecord is one measured value at a point in time, imported from
// Health Planet. A blood-pressure reading spans three records (systolic,
// diastolic, pulse) sharing the same MeasuredAt.
type HealthRecord struct {
	ID         int64     `db:"id"`
	MeasuredAt time.Time `db:"measured_at"` // minute precision, local time
	Metric     string    `db:"metric"`
	Value      float64   `db:"value"`
	CreatedAt  time.Time `db:"created_at"`
}

// HealthPlanetToken is the OAuth token state for the Health Planet API.
// The table holds a single row (id = 1).
type HealthPlanetToken struct {
	ID           int64     `db:"id"`
	AccessToken  string    `db:"access_token"`
	RefreshToken string    `db:"refresh_token"`
	ExpiresAt    time.Time `db:"expires_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}
