package http

import (
	"errors"
	"log"
	"net/http"

	"github.com/capybara-translation/goblog/internal/auth"
	"github.com/capybara-translation/goblog/internal/service"
	"github.com/gorilla/mux"
)

// DeviceHandlers serves the logged-in devices admin API. All routes run behind
// AuthMiddleware + CSRFMiddleware, so the user id is always in context.
type DeviceHandlers struct {
	service service.DeviceService
}

// NewDeviceHandlers creates DeviceHandlers.
func NewDeviceHandlers(s service.DeviceService) *DeviceHandlers {
	return &DeviceHandlers{service: s}
}

// currentIdentifiers extracts the raw session id and remember selector from the
// request cookies, used to mark/protect the current device.
func currentIdentifiers(r *http.Request) (sessionID, selector string) {
	if c, err := r.Cookie(sessionCookieName); err == nil {
		sessionID = c.Value
	}
	if c, err := r.Cookie(rememberCookieName); err == nil {
		if sel, _, ok := auth.DecodeRememberCookie(c.Value); ok {
			selector = sel
		}
	}
	return sessionID, selector
}

func (h *DeviceHandlers) HandleList(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	sid, sel := currentIdentifiers(r)
	devices, err := h.service.ListDevices(userID, sid, sel)
	if err != nil {
		log.Printf("ListDevices failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list devices")
		return
	}
	writeJSON(w, http.StatusOK, devices)
}

func (h *DeviceHandlers) HandleRevoke(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	vars := mux.Vars(r)
	kind, id := vars["kind"], vars["id"]
	sid, sel := currentIdentifiers(r)

	err := h.service.RevokeDevice(userID, kind, id, sid, sel)
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, service.ErrInvalidDeviceKind):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrCannotRevokeCurrent):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, service.ErrDeviceNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	default:
		log.Printf("RevokeDevice failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to revoke device")
	}
}

func (h *DeviceHandlers) HandleLogoutOthers(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	sid, sel := currentIdentifiers(r)
	if err := h.service.LogoutOtherDevices(userID, sid, sel); err != nil {
		log.Printf("LogoutOtherDevices failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to log out other devices")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
