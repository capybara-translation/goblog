package http

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/capybara-translation/goblog/internal/repo"
	"github.com/capybara-translation/goblog/internal/service"
)

// fakeHealthRecordRepo returns fixed daily averages for the page test.
type fakeHealthRecordRepo struct{}

func (f *fakeHealthRecordRepo) Upsert(records []*domain.HealthRecord) error { return nil }
func (f *fakeHealthRecordRepo) DailyAverages(fromDate, toDate string) ([]repo.DailyAverage, error) {
	return []repo.DailyAverage{
		{Date: fromDate, Metric: domain.MetricWeight, Avg: 72.1},
		{Date: toDate, Metric: domain.MetricSystolic, Avg: 119},
		{Date: toDate, Metric: domain.MetricDiastolic, Avg: 82},
	}, nil
}
func (f *fakeHealthRecordRepo) DailyAveragesByDates(dates []string) ([]repo.DailyAverage, error) {
	return nil, nil
}

var _ repo.HealthRecordRepository = (*fakeHealthRecordRepo)(nil)

// newTestPublicHandlersWithHealth builds PublicHandlers from the real
// on-disk templates (same path existing public-handler tests use via
// NewPublicHandlersFromPath) with the given HealthDisplayService injected.
func newTestPublicHandlersWithHealth(t *testing.T, healthDisplay *service.HealthDisplayService) *PublicHandlers {
	t.Helper()
	postSvc := &mockPostService{}
	return NewPublicHandlersFromPath(postSvc, nil, nil, nil, testSecureCookie, nil, testBlogTitle, testBaseURL, testTemplatePattern, testPostsPerPage, healthDisplay, nil, nil)
}

func TestHandleHealthPage_RendersChartsAndRangeLinks(t *testing.T) {
	h := newTestPublicHandlersWithHealth(t, service.NewHealthDisplayService(&fakeHealthRecordRepo{}))
	req := httptest.NewRequest("GET", "/health?range=30", nil)
	rr := httptest.NewRecorder()
	h.HandleHealthPage(rr, req)

	if rr.Code != 200 {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "<svg") {
		t.Error("page should contain inline SVG charts")
	}
	for _, link := range []string{`href="/health?range=30"`, `href="/health?range=90"`, `href="/health?range=365"`, `href="/health?range=all"`} {
		if !strings.Contains(body, link) {
			t.Errorf("range link %s missing", link)
		}
	}
	if !strings.Contains(body, "体脂肪率") {
		t.Error("chart section titles missing")
	}
	// 体脂肪率・脈拍はデータ 0 件 → 空状態メッセージ
	if !strings.Contains(body, "データがありません") {
		t.Error("empty-state message missing for metrics without data")
	}
}
