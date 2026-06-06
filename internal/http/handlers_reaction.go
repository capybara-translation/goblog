package http

import (
	"errors"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/capybara-translation/goblog/internal/auth"
	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/capybara-translation/goblog/internal/service"
	"github.com/gorilla/mux"
)

const (
	// reactionVisitorCookieName identifies the anonymous reaction visitor.
	reactionVisitorCookieName = "reaction_visitor"
	// reactionVisitorMaxAge is 400 days (browser cap on cookie lifetime).
	reactionVisitorMaxAge = 400 * 24 * 60 * 60
	// reactionRateLimit / reactionRateWindow bound write requests per IP.
	reactionRateLimit  = 30
	reactionRateWindow = 60 * time.Second
)

// reactionListResponse is the JSON envelope for reaction summaries.
type reactionListResponse struct {
	Reactions []*domain.PostReactionSummary `json:"reactions"`
}

// ReactionHandlers groups the public reaction endpoints.
type ReactionHandlers struct {
	service        service.ReactionService
	secureCookie   bool
	trustedProxies []*net.IPNet
	// limiter is intentionally shared between HandleAdd and HandleRemove so the
	// combined write rate per IP is bounded to reactionRateLimit/reactionRateWindow
	// regardless of which verb is used.
	limiter        *ipRateLimiter
}

// NewReactionHandlers creates ReactionHandlers. trustedProxies are raw strings
// (IP/CIDR) parsed the same way as the auth handler.
func NewReactionHandlers(reactionService service.ReactionService, secureCookie bool, trustedProxies []string) *ReactionHandlers {
	return &ReactionHandlers{
		service:        reactionService,
		secureCookie:   secureCookie,
		trustedProxies: parseTrustedProxies(trustedProxies),
		limiter:        newIPRateLimiter(reactionRateLimit, reactionRateWindow),
	}
}

// visitorKey returns the SHA-256 hash of the visitor cookie, issuing a fresh
// HttpOnly cookie (random 32 bytes) when none is present. The raw cookie value
// is never stored; only its hash is used as visitor_key.
func (h *ReactionHandlers) visitorKey(w http.ResponseWriter, r *http.Request) string {
	if c, err := r.Cookie(reactionVisitorCookieName); err == nil && c.Value != "" {
		return auth.HashToken(c.Value)
	}
	raw := auth.GenerateRawToken()
	http.SetCookie(w, &http.Cookie{
		Name:     reactionVisitorCookieName,
		Value:    raw,
		Path:     "/",
		MaxAge:   reactionVisitorMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.secureCookie,
	})
	return auth.HashToken(raw)
}

// HandleGet returns the reaction summaries for a post (issuing a visitor cookie
// if needed so reacted state is correct on the first interaction).
func (h *ReactionHandlers) HandleGet(w http.ResponseWriter, r *http.Request) {
	slug := mux.Vars(r)["slug"]
	vk := h.visitorKey(w, r)
	summaries, err := h.service.GetPostReactions(slug, vk)
	if err != nil {
		h.writeError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, reactionListResponse{Reactions: nonNilSummaries(summaries)})
}

// HandleAdd records a reaction.
func (h *ReactionHandlers) HandleAdd(w http.ResponseWriter, r *http.Request) {
	if !h.allow(r) {
		respondJSON(w, http.StatusTooManyRequests, ErrorResponse{Error: "too many requests"})
		return
	}
	slug := mux.Vars(r)["slug"]
	typeID, err := h.parseTypeID(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid reaction type id"})
		return
	}
	vk := h.visitorKey(w, r)
	summaries, err := h.service.AddReaction(slug, typeID, vk)
	if err != nil {
		h.writeError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, reactionListResponse{Reactions: nonNilSummaries(summaries)})
}

// HandleRemove removes a reaction.
func (h *ReactionHandlers) HandleRemove(w http.ResponseWriter, r *http.Request) {
	if !h.allow(r) {
		respondJSON(w, http.StatusTooManyRequests, ErrorResponse{Error: "too many requests"})
		return
	}
	slug := mux.Vars(r)["slug"]
	typeID, err := h.parseTypeID(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid reaction type id"})
		return
	}
	vk := h.visitorKey(w, r)
	summaries, err := h.service.RemoveReaction(slug, typeID, vk)
	if err != nil {
		h.writeError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, reactionListResponse{Reactions: nonNilSummaries(summaries)})
}

func (h *ReactionHandlers) allow(r *http.Request) bool {
	return h.limiter.Allow(clientIP(r, h.trustedProxies))
}

func (h *ReactionHandlers) parseTypeID(r *http.Request) (int64, error) {
	return strconv.ParseInt(mux.Vars(r)["reactionTypeID"], 10, 64)
}

func (h *ReactionHandlers) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrReactionPostNotFound):
		respondJSON(w, http.StatusNotFound, ErrorResponse{Error: "post not found"})
	case errors.Is(err, service.ErrReactionTypeInactive):
		respondJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid reaction type"})
	case errors.Is(err, service.ErrReactionVisitorEmpty):
		// Unreachable from these handlers: visitorKey() always returns a non-empty
		// SHA-256 hash because GenerateRawToken panics on CSPRNG failure rather than
		// returning an empty string. Kept as a defensive mapping for any future caller.
		respondJSON(w, http.StatusBadRequest, ErrorResponse{Error: "missing visitor"})
	default:
		log.Printf("reaction handler error: %v", err)
		respondJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
	}
}

// nonNilSummaries guarantees a non-nil slice so JSON serializes "[]" not "null".
func nonNilSummaries(s []*domain.PostReactionSummary) []*domain.PostReactionSummary {
	if s == nil {
		return []*domain.PostReactionSummary{}
	}
	return s
}
