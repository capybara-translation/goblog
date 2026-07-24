package http

import (
	"errors"
	"html/template"
	"log"
	"net/http"

	"github.com/capybara-translation/goblog/internal/healthchart"
	"github.com/capybara-translation/goblog/internal/service"
)

// healthChartView is one chart section on the /health page.
type healthChartView struct {
	Title string
	SVG   template.HTML
	Empty bool
}

func buildHealthCharts(s *service.HealthSeries) []healthChartView {
	toPoints := func(pts []service.HealthSeriesPoint) []healthchart.Point {
		out := make([]healthchart.Point, len(pts))
		for i, p := range pts {
			out[i] = healthchart.Point{Date: p.Date, Value: p.Value}
		}
		return out
	}
	charts := []struct {
		title, unit string
		series      []healthchart.Series
	}{
		{"体重", "kg", []healthchart.Series{{Label: "体重", Points: toPoints(s.Weight)}}},
		{"体脂肪率", "%", []healthchart.Series{{Label: "体脂肪率", Points: toPoints(s.BodyFat)}}},
		{"血圧", "mmHg", []healthchart.Series{
			{Label: "最高", Points: toPoints(s.Systolic)},
			{Label: "最低", Points: toPoints(s.Diastolic)},
		}},
		{"脈拍", "bpm", []healthchart.Series{{Label: "脈拍", Points: toPoints(s.Pulse)}}},
	}

	views := make([]healthChartView, 0, len(charts))
	for _, c := range charts {
		svg, err := healthchart.Render(healthchart.Chart{
			Title: c.title, Unit: c.unit, From: s.From, To: s.To, Series: c.series,
		})
		if errors.Is(err, healthchart.ErrNoData) {
			views = append(views, healthChartView{Title: c.title, Empty: true})
			continue
		}
		if err != nil {
			log.Printf("health chart %s: %v", c.title, err)
			views = append(views, healthChartView{Title: c.title, Empty: true})
			continue
		}
		views = append(views, healthChartView{Title: c.title, SVG: svg})
	}
	return views
}

// HandleHealthPage renders GET /health. The route is only registered when
// the Health Planet integration is enabled (healthDisplay != nil).
func (h *PublicHandlers) HandleHealthPage(w http.ResponseWriter, r *http.Request) {
	series, err := h.healthDisplay.Series(r.URL.Query().Get("range"))
	if err != nil {
		log.Printf("health page: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"SiteTitle":     h.blogTitle,
		"PinnedPosts":   h.getPinnedPosts(),
		"HealthEnabled": true,
		"Range":         series.Range,
		"Charts":        buildHealthCharts(series),
		"IsAdmin":       h.isAdminRequest(w, r),
		"Query":         "",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.healthTemplate.ExecuteTemplate(w, "health", data); err != nil {
		log.Printf("health page render: %v", err)
	}
}
