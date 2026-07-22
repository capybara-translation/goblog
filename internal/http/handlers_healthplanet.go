package http

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/capybara-translation/goblog/internal/service"
)

// HealthPlanetHandlers serves the admin-panel Health Planet endpoints.
// A nil service means the feature is disabled (HEALTHPLANET_ENABLED=false):
// only the status endpoint is registered then, and it reports enabled=false
// so the SPA can hide the UI.
type HealthPlanetHandlers struct {
	svc *service.HealthPlanetAdminService
}

func NewHealthPlanetHandlers(svc *service.HealthPlanetAdminService) *HealthPlanetHandlers {
	return &HealthPlanetHandlers{svc: svc}
}

type healthPlanetStatusResponse struct {
	Enabled         bool    `json:"enabled"`
	Authorized      bool    `json:"authorized"`
	TokenExpiresAt  *string `json:"token_expires_at"`
	LastRefreshedAt *string `json:"last_refreshed_at"`
}

func rfc3339OrNil(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}

// HandleStatus returns the integration state for the admin panel.
func (h *HealthPlanetHandlers) HandleStatus(w http.ResponseWriter, r *http.Request) {
	resp := healthPlanetStatusResponse{}
	if h.svc != nil {
		resp.Enabled = true
		st, err := h.svc.Status()
		if err != nil {
			log.Printf("healthplanet status: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load status"})
			return
		}
		resp.Authorized = st.Authorized
		resp.TokenExpiresAt = rfc3339OrNil(st.ExpiresAt)
		resp.LastRefreshedAt = rfc3339OrNil(st.LastRefreshedAt)
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleAuthURL returns the Health Planet authorization URL.
func (h *HealthPlanetHandlers) HandleAuthURL(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"url": h.svc.AuthCodeURL()})
}

// HandleExchange trades the pasted/redirected authorization code for a token.
func (h *HealthPlanetHandlers) HandleExchange(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "code is required"})
		return
	}
	if err := h.svc.Exchange(req.Code); err != nil {
		// Includes expired/invalid codes; surface the reason to the SPA.
		log.Printf("healthplanet exchange: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
