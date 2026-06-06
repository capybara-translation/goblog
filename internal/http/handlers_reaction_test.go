package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/capybara-translation/goblog/internal/service"
	"github.com/gorilla/mux"
)

type mockReactionService struct {
	getForPost func(postID int64, visitorKey string) ([]*domain.PostReactionSummary, error)
	get        func(slug, visitorKey string) ([]*domain.PostReactionSummary, error)
	add        func(slug string, typeID int64, visitorKey string) ([]*domain.PostReactionSummary, error)
	remove     func(slug string, typeID int64, visitorKey string) ([]*domain.PostReactionSummary, error)
}

func (m *mockReactionService) GetReactionsForPost(postID int64, visitorKey string) ([]*domain.PostReactionSummary, error) {
	return m.getForPost(postID, visitorKey)
}
func (m *mockReactionService) GetPostReactions(slug, visitorKey string) ([]*domain.PostReactionSummary, error) {
	return m.get(slug, visitorKey)
}
func (m *mockReactionService) AddReaction(slug string, typeID int64, visitorKey string) ([]*domain.PostReactionSummary, error) {
	return m.add(slug, typeID, visitorKey)
}
func (m *mockReactionService) RemoveReaction(slug string, typeID int64, visitorKey string) ([]*domain.PostReactionSummary, error) {
	return m.remove(slug, typeID, visitorKey)
}

func reactionTestRouter(svc service.ReactionService) *mux.Router {
	h := NewReactionHandlers(svc, false, nil)
	r := mux.NewRouter()
	sub := r.PathPrefix("/api/v1/posts/{slug}/reactions").Subrouter()
	sub.HandleFunc("", h.HandleGet).Methods("GET")
	sub.HandleFunc("/{reactionTypeID:[0-9]+}", h.HandleAdd).Methods("POST")
	sub.HandleFunc("/{reactionTypeID:[0-9]+}", h.HandleRemove).Methods("DELETE")
	return r
}

func TestReactionHandler_Get_SetsVisitorCookie(t *testing.T) {
	svc := &mockReactionService{
		get: func(slug, vk string) ([]*domain.PostReactionSummary, error) {
			if vk == "" {
				t.Fatal("visitor key should be non-empty (cookie issued)")
			}
			return []*domain.PostReactionSummary{{ID: 1, Emoji: "👍", Count: 2, Reacted: false}}, nil
		},
	}
	router := reactionTestRouter(svc)

	req := httptest.NewRequest("GET", "/api/v1/posts/p1/reactions", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Set-Cookie"), "reaction_visitor=") {
		t.Fatalf("expected reaction_visitor cookie to be set, headers: %v", rec.Header())
	}
	if !strings.Contains(rec.Body.String(), `"reactions"`) {
		t.Fatalf("expected reactions in body, got %s", rec.Body.String())
	}
}

func TestReactionHandler_Get_ReusesExistingCookie(t *testing.T) {
	var seenKey string
	svc := &mockReactionService{
		get: func(slug, vk string) ([]*domain.PostReactionSummary, error) {
			seenKey = vk
			return []*domain.PostReactionSummary{}, nil
		},
	}
	router := reactionTestRouter(svc)

	req := httptest.NewRequest("GET", "/api/v1/posts/p1/reactions", nil)
	req.AddCookie(&http.Cookie{Name: "reaction_visitor", Value: "raw-token-value"})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Header().Get("Set-Cookie") != "" {
		t.Fatal("existing cookie must not be re-issued")
	}
	if seenKey == "" || seenKey == "raw-token-value" {
		t.Fatalf("visitor key must be the hash of the cookie, got %q", seenKey)
	}
}

func TestReactionHandler_Add_Success(t *testing.T) {
	svc := &mockReactionService{
		add: func(slug string, typeID int64, vk string) ([]*domain.PostReactionSummary, error) {
			if slug != "p1" || typeID != 1 {
				t.Fatalf("unexpected args slug=%s typeID=%d", slug, typeID)
			}
			return []*domain.PostReactionSummary{{ID: 1, Emoji: "👍", Count: 1, Reacted: true}}, nil
		},
	}
	router := reactionTestRouter(svc)

	req := httptest.NewRequest("POST", "/api/v1/posts/p1/reactions/1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"reacted":true`) {
		t.Fatalf("expected reacted=true, got %s", rec.Body.String())
	}
}

func TestReactionHandler_Add_PostNotFound_404(t *testing.T) {
	svc := &mockReactionService{
		add: func(slug string, typeID int64, vk string) ([]*domain.PostReactionSummary, error) {
			return nil, service.ErrReactionPostNotFound
		},
	}
	router := reactionTestRouter(svc)

	req := httptest.NewRequest("POST", "/api/v1/posts/missing/reactions/1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestReactionHandler_Add_InactiveType_400(t *testing.T) {
	svc := &mockReactionService{
		add: func(slug string, typeID int64, vk string) ([]*domain.PostReactionSummary, error) {
			return nil, service.ErrReactionTypeInactive
		},
	}
	router := reactionTestRouter(svc)

	req := httptest.NewRequest("POST", "/api/v1/posts/p1/reactions/999", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestReactionHandler_Remove_Success(t *testing.T) {
	svc := &mockReactionService{
		remove: func(slug string, typeID int64, vk string) ([]*domain.PostReactionSummary, error) {
			return []*domain.PostReactionSummary{{ID: 1, Emoji: "👍", Count: 0, Reacted: false}}, nil
		},
	}
	router := reactionTestRouter(svc)

	req := httptest.NewRequest("DELETE", "/api/v1/posts/p1/reactions/1", nil)
	req.AddCookie(&http.Cookie{Name: "reaction_visitor", Value: "raw-token-value"})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestReactionHandler_Add_RateLimited_429(t *testing.T) {
	svc := &mockReactionService{
		add: func(slug string, typeID int64, vk string) ([]*domain.PostReactionSummary, error) {
			return []*domain.PostReactionSummary{}, nil
		},
	}
	h := NewReactionHandlers(svc, false, nil)
	// Non-zero window so the second call within the window is deterministically blocked.
	h.limiter = newIPRateLimiter(1, time.Hour)
	r := mux.NewRouter()
	sub := r.PathPrefix("/api/v1/posts/{slug}/reactions").Subrouter()
	sub.HandleFunc("/{reactionTypeID:[0-9]+}", h.HandleAdd).Methods("POST")

	doPost := func() int {
		req := httptest.NewRequest("POST", "/api/v1/posts/p1/reactions/1", nil)
		req.RemoteAddr = "9.9.9.9:1111"
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec.Code
	}

	if code := doPost(); code != http.StatusOK {
		t.Fatalf("first request expected 200, got %d", code)
	}
	if code := doPost(); code != http.StatusTooManyRequests {
		t.Fatalf("second request expected 429, got %d", code)
	}
}
