package service

import (
	"fmt"
	"math"
	"time"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/capybara-translation/goblog/internal/repo"
)

// HealthSeriesPoint is one day on a chart (pre-rounded daily average).
type HealthSeriesPoint struct {
	Date  string // YYYY-MM-DD
	Value float64
}

// HealthSeries is everything the /health page charts for one range.
type HealthSeries struct {
	Range     string // the applied range: "30" | "90" | "365" | "all"
	From, To  string // x-axis domain (YYYY-MM-DD, inclusive)
	Weight    []HealthSeriesPoint
	BodyFat   []HealthSeriesPoint
	Systolic  []HealthSeriesPoint
	Diastolic []HealthSeriesPoint
	Pulse     []HealthSeriesPoint
}

const defaultHealthRange = "90"

var healthRangeDays = map[string]int{"30": 30, "90": 90, "365": 365}

// HealthDisplayService prepares health_records data for public display.
// Rounding happens here — charts and article badges must show identical
// numbers: weight/body-fat to 1 decimal, blood pressure/pulse to integers.
type HealthDisplayService struct {
	recordRepo repo.HealthRecordRepository
}

func NewHealthDisplayService(recordRepo repo.HealthRecordRepository) *HealthDisplayService {
	return &HealthDisplayService{recordRepo: recordRepo}
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }
func round0(v float64) float64 { return math.Round(v) }

// roundFor applies the per-metric display rounding.
func roundFor(metric string, v float64) float64 {
	switch metric {
	case domain.MetricWeight, domain.MetricBodyFat:
		return round1(v)
	default: // systolic / diastolic / pulse
		return round0(v)
	}
}

// Series returns the daily-average series for the requested range.
// Unknown range values silently fall back to "90" (same convention as
// POSTS_PER_PAGE). For "all", From is the earliest data date.
//
// The window boundaries are derived from time.Now() in the server process's
// local TZ, and this is assumed to match the local calendar dates stored in
// measured_at (production runs TZ=Asia/Tokyo, matching the Health Planet
// account's locale). On a server configured with a different TZ, "today"'s
// rows can be excluded from the window for up to the TZ offset — e.g. a UTC
// server computing `to` a few hours before JST midnight would miss records
// whose measured_at date, interpreted in JST, is already "today".
func (s *HealthDisplayService) Series(rangeParam string) (*HealthSeries, error) {
	applied := rangeParam
	if _, ok := healthRangeDays[applied]; !ok && applied != "all" {
		applied = defaultHealthRange
	}

	to := time.Now().Format("2006-01-02")
	from := "0001-01-01"
	if days, ok := healthRangeDays[applied]; ok {
		// N-day window including today.
		from = time.Now().AddDate(0, 0, -(days - 1)).Format("2006-01-02")
	}

	rows, err := s.recordRepo.DailyAverages(from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to load health series: %w", err)
	}

	series := &HealthSeries{Range: applied, From: from, To: to}
	for _, row := range rows {
		p := HealthSeriesPoint{Date: row.Date, Value: roundFor(row.Metric, row.Avg)}
		switch row.Metric {
		case domain.MetricWeight:
			series.Weight = append(series.Weight, p)
		case domain.MetricBodyFat:
			series.BodyFat = append(series.BodyFat, p)
		case domain.MetricSystolic:
			series.Systolic = append(series.Systolic, p)
		case domain.MetricDiastolic:
			series.Diastolic = append(series.Diastolic, p)
		case domain.MetricPulse:
			series.Pulse = append(series.Pulse, p)
		}
	}

	if applied == "all" {
		// The x-axis should start at the first datum, not year 1.
		series.From = to
		for _, pts := range [][]HealthSeriesPoint{series.Weight, series.BodyFat, series.Systolic, series.Diastolic, series.Pulse} {
			if len(pts) > 0 && pts[0].Date < series.From {
				series.From = pts[0].Date // rows are ORDER BY date: first is min
			}
		}
	}
	return series, nil
}

// SummariesForDates returns pre-rounded daily summaries keyed by date.
// Dates with no data are absent from the map. Input dates are deduplicated.
func (s *HealthDisplayService) SummariesForDates(dates []string) (map[string]*domain.HealthSummary, error) {
	seen := make(map[string]bool, len(dates))
	unique := make([]string, 0, len(dates))
	for _, d := range dates {
		if d != "" && !seen[d] {
			seen[d] = true
			unique = append(unique, d)
		}
	}
	result := map[string]*domain.HealthSummary{}
	if len(unique) == 0 {
		return result, nil
	}

	rows, err := s.recordRepo.DailyAveragesByDates(unique)
	if err != nil {
		return nil, fmt.Errorf("failed to load health summaries: %w", err)
	}
	for _, row := range rows {
		sum := result[row.Date]
		if sum == nil {
			sum = &domain.HealthSummary{}
			result[row.Date] = sum
		}
		v := roundFor(row.Metric, row.Avg)
		switch row.Metric {
		case domain.MetricWeight:
			sum.Weight = &v
		case domain.MetricBodyFat:
			sum.BodyFat = &v
		case domain.MetricSystolic:
			sum.Systolic = &v
		case domain.MetricDiastolic:
			sum.Diastolic = &v
		case domain.MetricPulse:
			sum.Pulse = &v
		}
	}
	return result, nil
}
