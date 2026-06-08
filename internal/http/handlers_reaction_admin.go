package http

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/capybara-translation/goblog/internal/repo"
	"github.com/capybara-translation/goblog/internal/service"
	"github.com/gorilla/mux"
)

// ReactionAdminHandlers groups the protected reaction-type master endpoints.
type ReactionAdminHandlers struct {
	service service.ReactionTypeService
}

// NewReactionAdminHandlers creates ReactionAdminHandlers.
func NewReactionAdminHandlers(s service.ReactionTypeService) *ReactionAdminHandlers {
	return &ReactionAdminHandlers{service: s}
}

// reactionTypeDTO is the API row: the domain fields plus an is_seed flag so the
// UI can disable emoji-edit / delete on built-in types. is_seed has no DB column;
// it is derived from repo.IsSeedEmoji.
type reactionTypeDTO struct {
	ID        int64  `json:"id"`
	Emoji     string `json:"emoji"`
	Label     string `json:"label"`
	SortOrder int    `json:"sort_order"`
	IsActive  bool   `json:"is_active"`
	IsSeed    bool   `json:"is_seed"`
}

func toReactionTypeDTO(rt *domain.ReactionType) reactionTypeDTO {
	return reactionTypeDTO{
		ID: rt.ID, Emoji: rt.Emoji, Label: rt.Label,
		SortOrder: rt.SortOrder, IsActive: rt.IsActive,
		IsSeed: repo.IsSeedEmoji(rt.Emoji),
	}
}

type reactionTypeRequest struct {
	Emoji     string `json:"emoji"`
	Label     string `json:"label"`
	SortOrder int    `json:"sort_order"`
	IsActive  bool   `json:"is_active"`
}

// writeReactionTypeError maps service sentinels to HTTP status codes.
func writeReactionTypeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrReactionTypeInvalid):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrReactionTypeNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrReactionEmojiTaken),
		errors.Is(err, service.ErrReactionEmojiLocked),
		errors.Is(err, service.ErrReactionTypeProtected),
		errors.Is(err, service.ErrReactionTypeInUse):
		writeError(w, http.StatusConflict, err.Error())
	default:
		log.Printf("reaction admin handler error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func (h *ReactionAdminHandlers) HandleList(w http.ResponseWriter, r *http.Request) {
	types, err := h.service.List()
	if err != nil {
		log.Printf("failed to list reaction types: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list reaction types")
		return
	}
	out := make([]reactionTypeDTO, 0, len(types))
	for _, t := range types {
		out = append(out, toReactionTypeDTO(t))
	}
	writeJSON(w, http.StatusOK, out)
}

// HandleCreate creates a reaction type. New types are always created active;
// the is_active field of the request body is ignored on create (use HandleUpdate
// to deactivate). The shared request struct carries is_active only for updates.
func (h *ReactionAdminHandlers) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var req reactionTypeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	rt, err := h.service.Create(req.Emoji, req.Label, req.SortOrder)
	if err != nil {
		writeReactionTypeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toReactionTypeDTO(rt))
}

func (h *ReactionAdminHandlers) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req reactionTypeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	rt, err := h.service.Update(id, req.Emoji, req.Label, req.SortOrder, req.IsActive)
	if err != nil {
		writeReactionTypeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toReactionTypeDTO(rt))
}

func (h *ReactionAdminHandlers) HandleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.service.Delete(id); err != nil {
		writeReactionTypeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
