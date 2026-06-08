package service

import (
	"errors"
	"strings"
	"testing"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/capybara-translation/goblog/internal/repo"
)

type mockReactionTypeRepo struct {
	findAll  func() ([]*domain.ReactionType, error)
	findByID func(id int64) (*domain.ReactionType, error)
	create   func(rt *domain.ReactionType) (int64, error)
	update   func(rt *domain.ReactionType) error
	deleteIf func(id int64) (bool, error)
}

func (m *mockReactionTypeRepo) FindAll() ([]*domain.ReactionType, error) { return m.findAll() }
func (m *mockReactionTypeRepo) FindByID(id int64) (*domain.ReactionType, error) {
	return m.findByID(id)
}
func (m *mockReactionTypeRepo) Create(rt *domain.ReactionType) (int64, error) { return m.create(rt) }
func (m *mockReactionTypeRepo) Update(rt *domain.ReactionType) error          { return m.update(rt) }
func (m *mockReactionTypeRepo) DeleteIfUnused(id int64) (bool, error)         { return m.deleteIf(id) }

func TestReactionTypeService_Create_Valid(t *testing.T) {
	m := &mockReactionTypeRepo{
		create: func(rt *domain.ReactionType) (int64, error) {
			if !rt.IsActive {
				t.Error("created type must be active")
			}
			return 7, nil
		},
		findByID: func(id int64) (*domain.ReactionType, error) {
			if id != 7 {
				t.Errorf("FindByID called with id=%d, want 7", id)
			}
			return &domain.ReactionType{ID: 7, Emoji: "🚀", Label: "発射", SortOrder: 99, IsActive: true}, nil
		},
	}
	s := NewReactionTypeService(m)
	got, err := s.Create("🚀", "発射", 99)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.ID != 7 {
		t.Fatalf("expected id 7, got %d", got.ID)
	}
}

func TestReactionTypeService_Create_Invalid(t *testing.T) {
	s := NewReactionTypeService(&mockReactionTypeRepo{})
	if _, err := s.Create("", "label", 0); !errors.Is(err, ErrReactionTypeInvalid) {
		t.Errorf("empty emoji should be invalid, got %v", err)
	}
	if _, err := s.Create("👍", "  ", 0); !errors.Is(err, ErrReactionTypeInvalid) {
		t.Errorf("blank label should be invalid, got %v", err)
	}
}

func TestReactionTypeService_Create_DuplicateEmoji(t *testing.T) {
	m := &mockReactionTypeRepo{
		create: func(rt *domain.ReactionType) (int64, error) { return 0, repo.ErrDuplicateEmoji },
	}
	s := NewReactionTypeService(m)
	_, err := s.Create("👍", "dup", 1)
	if !errors.Is(err, ErrReactionEmojiTaken) {
		t.Fatalf("expected ErrReactionEmojiTaken, got %v", err)
	}
}

func TestReactionTypeService_Update_NotFound(t *testing.T) {
	m := &mockReactionTypeRepo{findByID: func(id int64) (*domain.ReactionType, error) { return nil, nil }}
	s := NewReactionTypeService(m)
	_, err := s.Update(1, "👍", "x", 1, true)
	if !errors.Is(err, ErrReactionTypeNotFound) {
		t.Fatalf("expected ErrReactionTypeNotFound, got %v", err)
	}
}

func TestReactionTypeService_Update_SeedEmojiLocked(t *testing.T) {
	m := &mockReactionTypeRepo{
		findByID: func(id int64) (*domain.ReactionType, error) {
			return &domain.ReactionType{ID: 1, Emoji: "👍", Label: "いいね", SortOrder: 10, IsActive: true}, nil
		},
	}
	s := NewReactionTypeService(m)
	_, err := s.Update(1, "👏", "いいね", 10, true)
	if !errors.Is(err, ErrReactionEmojiLocked) {
		t.Fatalf("expected ErrReactionEmojiLocked, got %v", err)
	}
}

func TestReactionTypeService_Update_SeedNonEmojiFieldsAllowed(t *testing.T) {
	updated := false
	m := &mockReactionTypeRepo{
		findByID: func(id int64) (*domain.ReactionType, error) {
			return &domain.ReactionType{ID: 1, Emoji: "👍", Label: "いいね", SortOrder: 10, IsActive: true}, nil
		},
		update: func(rt *domain.ReactionType) error { updated = true; return nil },
	}
	s := NewReactionTypeService(m)
	if _, err := s.Update(1, "👍", "Good", 5, false); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !updated {
		t.Fatal("expected repo.Update to be called")
	}
}

func TestReactionTypeService_Delete_SeedProtected(t *testing.T) {
	m := &mockReactionTypeRepo{
		findByID: func(id int64) (*domain.ReactionType, error) {
			return &domain.ReactionType{ID: 1, Emoji: "👍"}, nil
		},
	}
	s := NewReactionTypeService(m)
	if !errors.Is(s.Delete(1), ErrReactionTypeProtected) {
		t.Fatal("expected seed delete to be protected")
	}
}

func TestReactionTypeService_Delete_InUse(t *testing.T) {
	m := &mockReactionTypeRepo{
		findByID: func(id int64) (*domain.ReactionType, error) {
			return &domain.ReactionType{ID: 7, Emoji: "🚀"}, nil
		},
		deleteIf: func(id int64) (bool, error) { return false, nil },
	}
	s := NewReactionTypeService(m)
	if !errors.Is(s.Delete(7), ErrReactionTypeInUse) {
		t.Fatal("expected in-use delete to be rejected")
	}
}

func TestReactionTypeService_Delete_OK(t *testing.T) {
	m := &mockReactionTypeRepo{
		findByID: func(id int64) (*domain.ReactionType, error) {
			return &domain.ReactionType{ID: 7, Emoji: "🚀"}, nil
		},
		deleteIf: func(id int64) (bool, error) { return true, nil },
	}
	s := NewReactionTypeService(m)
	if err := s.Delete(7); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestReactionTypeService_Update_DuplicateEmoji(t *testing.T) {
	m := &mockReactionTypeRepo{
		findByID: func(id int64) (*domain.ReactionType, error) {
			return &domain.ReactionType{ID: 9, Emoji: "🚀"}, nil // not a seed
		},
		update: func(rt *domain.ReactionType) error { return repo.ErrDuplicateEmoji },
	}
	s := NewReactionTypeService(m)
	_, err := s.Update(9, "👍", "renamed", 1, true)
	if !errors.Is(err, ErrReactionEmojiTaken) {
		t.Fatalf("expected ErrReactionEmojiTaken, got %v", err)
	}
}

func TestReactionTypeService_List(t *testing.T) {
	want := []*domain.ReactionType{{ID: 1, Emoji: "👍"}}
	m := &mockReactionTypeRepo{findAll: func() ([]*domain.ReactionType, error) { return want, nil }}
	s := NewReactionTypeService(m)
	got, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 || got[0].Emoji != "👍" {
		t.Fatalf("unexpected List result: %+v", got)
	}
}

func TestReactionTypeService_Create_LengthBoundaries(t *testing.T) {
	m := &mockReactionTypeRepo{
		create:   func(rt *domain.ReactionType) (int64, error) { return 1, nil },
		findByID: func(id int64) (*domain.ReactionType, error) { return &domain.ReactionType{ID: 1}, nil },
	}
	s := NewReactionTypeService(m)

	// emoji: 8 runes OK, 9 runes invalid.
	if _, err := s.Create(strings.Repeat("a", 8), "label", 0); err != nil {
		t.Errorf("emoji of 8 runes should be valid, got %v", err)
	}
	if _, err := s.Create(strings.Repeat("a", 9), "label", 0); !errors.Is(err, ErrReactionTypeInvalid) {
		t.Errorf("emoji of 9 runes should be invalid, got %v", err)
	}
	// label: 50 runes OK, 51 runes invalid.
	if _, err := s.Create("👍", strings.Repeat("x", 50), 0); err != nil {
		t.Errorf("label of 50 runes should be valid, got %v", err)
	}
	if _, err := s.Create("👍", strings.Repeat("x", 51), 0); !errors.Is(err, ErrReactionTypeInvalid) {
		t.Errorf("label of 51 runes should be invalid, got %v", err)
	}
}
