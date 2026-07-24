package http

import (
	"bytes"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/capybara-translation/goblog/internal/repo"
	"github.com/capybara-translation/goblog/internal/service"
)

// stripTagsNormalize strips HTML tags and collapses whitespace, so template
// output can be compared against a plain-text expectation regardless of
// exact markup/indentation.
func stripTagsNormalize(html string) string {
	noTags := regexp.MustCompile(`<[^>]+>`).ReplaceAllString(html, " ")
	return strings.Join(strings.Fields(noTags), " ")
}

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
	if !strings.Contains(body, `<meta property="og:title" content="Healthcare - `) {
		t.Error("og:title meta with Healthcare title missing")
	}
}

// range クエリ指定（デフォルト以外の URL バリエーション）は home の ?q/page と同様に
// noindex にすべき — 同一データの別 URL が重複コンテンツとして評価されるのを防ぐ。
func TestHandleHealthPage_NoIndexForNonDefaultRange(t *testing.T) {
	h := newTestPublicHandlersWithHealth(t, service.NewHealthDisplayService(&fakeHealthRecordRepo{}))

	reqDefault := httptest.NewRequest("GET", "/health", nil)
	rrDefault := httptest.NewRecorder()
	h.HandleHealthPage(rrDefault, reqDefault)
	if strings.Contains(rrDefault.Body.String(), `name="robots" content="noindex,follow"`) {
		t.Error("default /health (no range param) should not have noindex meta")
	}

	reqRange := httptest.NewRequest("GET", "/health?range=30", nil)
	rrRange := httptest.NewRecorder()
	h.HandleHealthPage(rrRange, reqRange)
	if !strings.Contains(rrRange.Body.String(), `name="robots" content="noindex,follow"`) {
		t.Error("/health?range=30 should have noindex meta")
	}
}

func TestAttachHealthSummaries(t *testing.T) {
	h := newTestPublicHandlersWithHealth(t, service.NewHealthDisplayService(&fakeHealthRecordRepoForBadges{}))
	hd := "2026-07-20"
	noData := "2026-07-22"
	posts := []*domain.Post{
		{ID: 1, HealthDate: &hd},
		{ID: 2, HealthDate: nil},
		{ID: 3, HealthDate: &noData},
	}
	h.attachHealthSummaries(posts)

	if posts[0].HealthSummary == nil || posts[0].HealthSummary.Weight == nil || *posts[0].HealthSummary.Weight != 72.1 {
		t.Errorf("post 1 summary = %+v, want weight 72.1", posts[0].HealthSummary)
	}
	if posts[1].HealthSummary != nil {
		t.Error("post without HealthDate should have nil summary")
	}
	if posts[2].HealthSummary != nil {
		t.Error("post whose date has no data should have nil summary")
	}
}

// fakeHealthRecordRepoForBadges: 2026-07-20 のみデータを返す。DailyAveragesByDates
// の呼び出し回数を数えて、バッジ付与が投稿ごとに N+1 クエリを打っていないことを検証する。
type fakeHealthRecordRepoForBadges struct {
	fakeHealthRecordRepo
	calls int
}

func (f *fakeHealthRecordRepoForBadges) DailyAveragesByDates(dates []string) ([]repo.DailyAverage, error) {
	f.calls++
	return []repo.DailyAverage{{Date: "2026-07-20", Metric: domain.MetricWeight, Avg: 72.1}}, nil
}

func TestAttachHealthSummaries_QueryBatching(t *testing.T) {
	t.Run("no HealthDate on any post issues zero queries", func(t *testing.T) {
		fake := &fakeHealthRecordRepoForBadges{}
		h := newTestPublicHandlersWithHealth(t, service.NewHealthDisplayService(fake))
		posts := []*domain.Post{{ID: 1, HealthDate: nil}, {ID: 2, HealthDate: nil}}

		h.attachHealthSummaries(posts)

		if fake.calls != 0 {
			t.Errorf("DailyAveragesByDates calls = %d, want 0 (no dates to look up)", fake.calls)
		}
	})

	t.Run("two posts sharing one date issue exactly one batched query", func(t *testing.T) {
		fake := &fakeHealthRecordRepoForBadges{}
		h := newTestPublicHandlersWithHealth(t, service.NewHealthDisplayService(fake))
		hd := "2026-07-20"
		posts := []*domain.Post{{ID: 1, HealthDate: &hd}, {ID: 2, HealthDate: &hd}}

		h.attachHealthSummaries(posts)

		if fake.calls != 1 {
			t.Errorf("DailyAveragesByDates calls = %d, want 1 (single batched query, not one per post)", fake.calls)
		}
	})
}

func TestHealthBadges_Template(t *testing.T) {
	h := newTestPublicHandlersWithHealth(t, service.NewHealthDisplayService(&fakeHealthRecordRepoForBadges{}))

	render := func(p *domain.Post) string {
		var buf bytes.Buffer
		// homeTemplate は layout.html を含むセットなので healthBadges partial を持つ
		if err := h.homeTemplate.ExecuteTemplate(&buf, "healthBadges", p); err != nil {
			t.Fatalf("render healthBadges: %v", err)
		}
		return buf.String()
	}

	w, sys := 72.1, 119.0
	full := &domain.Post{HealthSummary: &domain.HealthSummary{Weight: &w}}
	if got := render(full); !strings.Contains(got, "体重 72.1kg") {
		t.Errorf("weight badge missing: %q", got)
	}
	if got := render(full); strings.Contains(got, "・") {
		t.Errorf("single-metric summary must not contain a ・ separator: %q", got)
	}

	halfBP := &domain.Post{HealthSummary: &domain.HealthSummary{Systolic: &sys}} // Diastolic なし
	if got := render(halfBP); strings.Contains(got, "血圧") {
		t.Errorf("one-sided blood pressure must not render: %q", got)
	}

	none := &domain.Post{HealthSummary: nil}
	if got := strings.TrimSpace(render(none)); got != "" {
		t.Errorf("nil summary should render nothing, got %q", got)
	}

	weight, pulse := 72.1, 63.0
	twoMetrics := &domain.Post{HealthSummary: &domain.HealthSummary{Weight: &weight, Pulse: &pulse}}
	if got := stripTagsNormalize(render(twoMetrics)); got != "体重 72.1kg ・ 脈拍 63bpm" {
		t.Errorf("two-metric badges = %q, want %q", got, "体重 72.1kg ・ 脈拍 63bpm")
	}
}
