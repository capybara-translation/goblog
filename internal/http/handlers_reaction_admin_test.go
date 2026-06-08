package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/capybara-translation/goblog/internal/service"
	"github.com/gorilla/mux"
)

type mockReactionTypeService struct {
	list   func() ([]*domain.ReactionType, error)
	create func(emoji, label string, sortOrder int) (*domain.ReactionType, error)
	update func(id int64, emoji, label string, sortOrder int, isActive bool) (*domain.ReactionType, error)
	del    func(id int64) error
}

func (m *mockReactionTypeService) List() ([]*domain.ReactionType, error) { return m.list() }
func (m *mockReactionTypeService) Create(e, l string, s int) (*domain.ReactionType, error) {
	return m.create(e, l, s)
}
func (m *mockReactionTypeService) Update(id int64, e, l string, s int, a bool) (*domain.ReactionType, error) {
	return m.update(id, e, l, s, a)
}
func (m *mockReactionTypeService) Delete(id int64) error { return m.del(id) }

func TestAdminReactions_List_IncludesIsSeed(t *testing.T) {
	svc := &mockReactionTypeService{
		list: func() ([]*domain.ReactionType, error) {
			return []*domain.ReactionType{
				{ID: 1, Emoji: "👍", Label: "いいね", SortOrder: 10, IsActive: true},
				{ID: 7, Emoji: "🚀", Label: "発射", SortOrder: 99, IsActive: true},
			}, nil
		},
	}
	h := NewReactionAdminHandlers(svc)
	req := httptest.NewRequest("GET", "/api/v1/reaction-types", nil)
	rec := httptest.NewRecorder()
	h.HandleList(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d", rec.Code)
	}
	var got []struct {
		ID     int64  `json:"id"`
		Emoji  string `json:"emoji"`
		IsSeed bool   `json:"is_seed"`
	}
	json.Unmarshal(rec.Body.Bytes(), &got)
	if len(got) != 2 || !got[0].IsSeed || got[1].IsSeed {
		t.Fatalf("unexpected is_seed flags: %+v", got)
	}
}

func TestAdminReactions_Create_OK(t *testing.T) {
	svc := &mockReactionTypeService{
		create: func(e, l string, s int) (*domain.ReactionType, error) {
			return &domain.ReactionType{ID: 7, Emoji: e, Label: l, SortOrder: s, IsActive: true}, nil
		},
	}
	h := NewReactionAdminHandlers(svc)
	body, _ := json.Marshal(map[string]any{"emoji": "🚀", "label": "発射", "sort_order": 99})
	req := httptest.NewRequest("POST", "/api/v1/reaction-types", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.HandleCreate(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status: %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminReactions_Create_Invalid400(t *testing.T) {
	svc := &mockReactionTypeService{
		create: func(e, l string, s int) (*domain.ReactionType, error) {
			return nil, service.ErrReactionTypeInvalid
		},
	}
	h := NewReactionAdminHandlers(svc)
	body, _ := json.Marshal(map[string]any{"emoji": "", "label": "x"})
	req := httptest.NewRequest("POST", "/api/v1/reaction-types", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.HandleCreate(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestAdminReactions_Create_Duplicate409(t *testing.T) {
	svc := &mockReactionTypeService{
		create: func(e, l string, s int) (*domain.ReactionType, error) {
			return nil, service.ErrReactionEmojiTaken
		},
	}
	h := NewReactionAdminHandlers(svc)
	body, _ := json.Marshal(map[string]any{"emoji": "👍", "label": "dup"})
	req := httptest.NewRequest("POST", "/api/v1/reaction-types", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.HandleCreate(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
}

func TestAdminReactions_Update_NotFound404(t *testing.T) {
	svc := &mockReactionTypeService{
		update: func(id int64, e, l string, s int, a bool) (*domain.ReactionType, error) {
			return nil, service.ErrReactionTypeNotFound
		},
	}
	h := NewReactionAdminHandlers(svc)
	body, _ := json.Marshal(map[string]any{"emoji": "👍", "label": "x", "sort_order": 1, "is_active": true})
	req := httptest.NewRequest("PUT", "/api/v1/reaction-types/1", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	rec := httptest.NewRecorder()
	h.HandleUpdate(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestAdminReactions_Delete_InUse409(t *testing.T) {
	svc := &mockReactionTypeService{del: func(id int64) error { return service.ErrReactionTypeInUse }}
	h := NewReactionAdminHandlers(svc)
	req := httptest.NewRequest("DELETE", "/api/v1/reaction-types/7", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "7"})
	rec := httptest.NewRecorder()
	h.HandleDelete(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
}

func TestAdminReactions_Delete_OK204(t *testing.T) {
	svc := &mockReactionTypeService{del: func(id int64) error { return nil }}
	h := NewReactionAdminHandlers(svc)
	req := httptest.NewRequest("DELETE", "/api/v1/reaction-types/7", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "7"})
	rec := httptest.NewRecorder()
	h.HandleDelete(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}

func TestAdminReactions_Update_OK200(t *testing.T) {
	svc := &mockReactionTypeService{
		update: func(id int64, e, l string, s int, a bool) (*domain.ReactionType, error) {
			return &domain.ReactionType{ID: id, Emoji: e, Label: l, SortOrder: s, IsActive: a}, nil
		},
	}
	h := NewReactionAdminHandlers(svc)
	body, _ := json.Marshal(map[string]any{"emoji": "👍", "label": "Good", "sort_order": 5, "is_active": false})
	req := httptest.NewRequest("PUT", "/api/v1/reaction-types/1", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	rec := httptest.NewRecorder()
	h.HandleUpdate(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var got reactionTypeDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.ID != 1 || got.Label != "Good" || got.SortOrder != 5 || got.IsActive {
		t.Fatalf("unexpected DTO: %+v", got)
	}
}

func TestAdminReactions_Create_BadJSON400(t *testing.T) {
	h := NewReactionAdminHandlers(&mockReactionTypeService{})
	req := httptest.NewRequest("POST", "/api/v1/reaction-types", bytes.NewReader([]byte("{not json")))
	rec := httptest.NewRecorder()
	h.HandleCreate(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
