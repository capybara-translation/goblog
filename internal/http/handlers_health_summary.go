package http

import (
	"log"
	"net/http"
	"time"

	"github.com/capybara-translation/goblog/internal/service"
)

// HealthSummaryHandlers serves the admin-panel "which day's health data"
// lookup the PostEdit editor uses to preview the badges a health_date will
// produce on the public page.
type HealthSummaryHandlers struct {
	display *service.HealthDisplayService
}

func NewHealthSummaryHandlers(display *service.HealthDisplayService) *HealthSummaryHandlers {
	return &HealthSummaryHandlers{display: display}
}

type healthSummaryResponse struct {
	Found     bool     `json:"found"`
	Weight    *float64 `json:"weight"`
	BodyFat   *float64 `json:"body_fat"`
	Systolic  *float64 `json:"systolic"`
	Diastolic *float64 `json:"diastolic"`
	Pulse     *float64 `json:"pulse"`
}

// HandleGetSummary returns the daily health summary for ?date=YYYY-MM-DD.
func (h *HealthSummaryHandlers) HandleGetSummary(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")
	if _, err := time.Parse("2006-01-02", date); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "date must be YYYY-MM-DD"})
		return
	}

	summaries, err := h.display.SummariesForDates([]string{date})
	if err != nil {
		log.Printf("healthplanet summary: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load health summary"})
		return
	}

	sum, ok := summaries[date]
	if !ok {
		writeJSON(w, http.StatusOK, healthSummaryResponse{Found: false})
		return
	}
	writeJSON(w, http.StatusOK, healthSummaryResponse{
		Found:     true,
		Weight:    sum.Weight,
		BodyFat:   sum.BodyFat,
		Systolic:  sum.Systolic,
		Diastolic: sum.Diastolic,
		Pulse:     sum.Pulse,
	})
}
