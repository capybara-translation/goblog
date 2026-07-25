package http

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/capybara-translation/goblog/internal/service"
)

// newTestHealthSummaryHandlers builds a HealthSummaryHandlers backed by the
// same fakeHealthRecordRepoForBadges used by handlers_health_page_test.go,
// which returns data for 2026-07-20 only.
func newTestHealthSummaryHandlers() *HealthSummaryHandlers {
	display := service.NewHealthDisplayService(&fakeHealthRecordRepoForBadges{})
	return NewHealthSummaryHandlers(display)
}

func TestHealthSummary_ValidDateWithData(t *testing.T) {
	h := newTestHealthSummaryHandlers()
	req := httptest.NewRequest("GET", "/api/v1/healthplanet/summary?date=2026-07-20", nil)
	rr := httptest.NewRecorder()
	h.HandleGetSummary(rr, req)

	if rr.Code != 200 {
		t.Fatalf("status = %d, want 200 (body: %s)", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("body not JSON: %v", err)
	}
	if body["found"] != true {
		t.Errorf("found = %v, want true", body["found"])
	}
	if body["weight"] != 72.1 {
		t.Errorf("weight = %v, want 72.1", body["weight"])
	}
	for _, key := range []string{"body_fat", "systolic", "diastolic", "pulse"} {
		if v, ok := body[key]; !ok || v != nil {
			t.Errorf("%s = %v, want null", key, v)
		}
	}
}

func TestHealthSummary_DateWithoutData(t *testing.T) {
	h := newTestHealthSummaryHandlers()
	req := httptest.NewRequest("GET", "/api/v1/healthplanet/summary?date=2026-07-22", nil)
	rr := httptest.NewRecorder()
	h.HandleGetSummary(rr, req)

	if rr.Code != 200 {
		t.Fatalf("status = %d, want 200 (body: %s)", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("body not JSON: %v", err)
	}
	if body["found"] != false {
		t.Errorf("found = %v, want false", body["found"])
	}
}

func TestHealthSummary_BadDate(t *testing.T) {
	h := newTestHealthSummaryHandlers()
	req := httptest.NewRequest("GET", "/api/v1/healthplanet/summary?date=not-a-date", nil)
	rr := httptest.NewRecorder()
	h.HandleGetSummary(rr, req)

	if rr.Code != 400 {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("body not JSON: %v", err)
	}
	if body["error"] != "date must be YYYY-MM-DD" {
		t.Errorf("error = %q, want %q", body["error"], "date must be YYYY-MM-DD")
	}
}

func TestHealthSummary_MissingDate(t *testing.T) {
	h := newTestHealthSummaryHandlers()
	req := httptest.NewRequest("GET", "/api/v1/healthplanet/summary", nil)
	rr := httptest.NewRecorder()
	h.HandleGetSummary(rr, req)

	if rr.Code != 400 {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}
