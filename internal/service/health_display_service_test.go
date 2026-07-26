package service

import (
	"testing"
	"time"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/capybara-translation/goblog/internal/repo"
)

type mockHealthRecordReadRepo struct {
	mockHealthRecordRepo     // Task 5(hpsync) の既存モックを埋め込み（Upsert 用）
	dailyAveragesFunc        func(fromDate, toDate string) ([]repo.DailyAverage, error)
	dailyAveragesByDatesFunc func(dates []string) ([]repo.DailyAverage, error)
	gotFrom, gotTo           string
	gotDates                 []string
}

func (m *mockHealthRecordReadRepo) DailyAverages(fromDate, toDate string) ([]repo.DailyAverage, error) {
	m.gotFrom, m.gotTo = fromDate, toDate
	if m.dailyAveragesFunc != nil {
		return m.dailyAveragesFunc(fromDate, toDate)
	}
	return nil, nil
}

func (m *mockHealthRecordReadRepo) DailyAveragesByDates(dates []string) ([]repo.DailyAverage, error) {
	m.gotDates = dates
	if m.dailyAveragesByDatesFunc != nil {
		return m.dailyAveragesByDatesFunc(dates)
	}
	return nil, nil
}

func TestHealthDisplay_Series_RangeFallbackAndWindow(t *testing.T) {
	m := &mockHealthRecordReadRepo{}
	svc := NewHealthDisplayService(m)

	s, err := svc.Series("bogus")
	if err != nil {
		t.Fatalf("Series: %v", err)
	}
	if s.Range != "30" {
		t.Errorf("Range = %q, want 30 (fallback)", s.Range)
	}
	today := time.Now().Format("2006-01-02")
	if s.To != today || m.gotTo != today {
		t.Errorf("To = %q / repo to = %q, want %q", s.To, m.gotTo, today)
	}
	wantFrom := time.Now().AddDate(0, 0, -29).Format("2006-01-02")
	if s.From != wantFrom || m.gotFrom != wantFrom {
		t.Errorf("From = %q / repo from = %q, want %q (30-day window incl. today)", s.From, m.gotFrom, wantFrom)
	}
}

func TestHealthDisplay_Series_RoundsAndSplitsMetrics(t *testing.T) {
	m := &mockHealthRecordReadRepo{
		dailyAveragesFunc: func(_, _ string) ([]repo.DailyAverage, error) {
			return []repo.DailyAverage{
				{Date: "2026-07-20", Metric: domain.MetricWeight, Avg: 72.456},
				{Date: "2026-07-20", Metric: domain.MetricSystolic, Avg: 119.5},
				{Date: "2026-07-21", Metric: domain.MetricWeight, Avg: 71.04},
			}, nil
		},
	}
	svc := NewHealthDisplayService(m)
	s, err := svc.Series("30")
	if err != nil {
		t.Fatalf("Series: %v", err)
	}
	if len(s.Weight) != 2 || s.Weight[0].Value != 72.5 || s.Weight[1].Value != 71.0 {
		t.Errorf("Weight = %+v, want [72.5 71.0]", s.Weight)
	}
	if len(s.Systolic) != 1 || s.Systolic[0].Value != 120 {
		t.Errorf("Systolic = %+v, want [120] (integer rounding)", s.Systolic)
	}
	if len(s.BodyFat) != 0 || len(s.Pulse) != 0 || len(s.Diastolic) != 0 {
		t.Errorf("metrics without data should be empty")
	}
}

func TestHealthDisplay_Series_AllRangeUsesDataMinAsFrom(t *testing.T) {
	m := &mockHealthRecordReadRepo{
		dailyAveragesFunc: func(_, _ string) ([]repo.DailyAverage, error) {
			return []repo.DailyAverage{
				{Date: "2026-06-01", Metric: domain.MetricWeight, Avg: 70},
				{Date: "2026-07-01", Metric: domain.MetricWeight, Avg: 71},
			}, nil
		},
	}
	svc := NewHealthDisplayService(m)
	s, err := svc.Series("all")
	if err != nil {
		t.Fatalf("Series: %v", err)
	}
	if m.gotFrom != "0001-01-01" {
		t.Errorf("repo from = %q, want 0001-01-01", m.gotFrom)
	}
	if s.From != "2026-06-01" {
		t.Errorf("From = %q, want earliest data date", s.From)
	}
}

func TestHealthDisplay_SummariesForDates(t *testing.T) {
	m := &mockHealthRecordReadRepo{
		dailyAveragesByDatesFunc: func(dates []string) ([]repo.DailyAverage, error) {
			return []repo.DailyAverage{
				{Date: "2026-07-20", Metric: domain.MetricWeight, Avg: 72.46},
				{Date: "2026-07-20", Metric: domain.MetricSystolic, Avg: 119.4},
				{Date: "2026-07-20", Metric: domain.MetricDiastolic, Avg: 81.5},
			}, nil
		},
	}
	svc := NewHealthDisplayService(m)

	got, err := svc.SummariesForDates([]string{"2026-07-20", "2026-07-20", "2026-07-22"})
	if err != nil {
		t.Fatalf("SummariesForDates: %v", err)
	}
	if len(m.gotDates) != 2 {
		t.Errorf("dates should be deduplicated: %v", m.gotDates)
	}
	s := got["2026-07-20"]
	if s == nil {
		t.Fatal("summary for 2026-07-20 missing")
	}
	if s.Weight == nil || *s.Weight != 72.5 {
		t.Errorf("Weight = %v, want 72.5", s.Weight)
	}
	if s.Systolic == nil || *s.Systolic != 119 || s.Diastolic == nil || *s.Diastolic != 82 {
		t.Errorf("BP = %v/%v, want 119/82", s.Systolic, s.Diastolic)
	}
	if s.Pulse != nil || s.BodyFat != nil {
		t.Errorf("missing metrics should stay nil")
	}
	if _, ok := got["2026-07-22"]; ok {
		t.Error("date without data should not appear in map")
	}
}

func TestHealthDisplay_SummariesForDates_Empty(t *testing.T) {
	m := &mockHealthRecordReadRepo{}
	svc := NewHealthDisplayService(m)
	got, err := svc.SummariesForDates(nil)
	if err != nil {
		t.Fatalf("SummariesForDates(nil): %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %v, want empty map", got)
	}
}
