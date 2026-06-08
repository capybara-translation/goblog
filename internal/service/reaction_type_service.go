package service

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/capybara-translation/goblog/internal/repo"
)

// Sentinel errors mapped by the HTTP layer to status codes.
var (
	ErrReactionTypeNotFound  = errors.New("reaction type: not found")                                  // -> 404
	ErrReactionTypeInvalid   = errors.New("reaction type: invalid emoji or label")                     // -> 400
	ErrReactionEmojiTaken    = errors.New("reaction type: emoji already exists")                        // -> 409
	ErrReactionEmojiLocked   = errors.New("reaction type: emoji of a built-in type cannot be changed") // -> 409
	ErrReactionTypeProtected = errors.New("reaction type: built-in types cannot be deleted")           // -> 409
	ErrReactionTypeInUse     = errors.New("reaction type: has reactions and cannot be deleted")         // -> 409
)

const (
	maxReactionEmojiRunes = 16
	maxReactionLabelRunes = 50
)

// ReactionTypeService provides admin CRUD over the reaction emoji master.
type ReactionTypeService interface {
	List() ([]*domain.ReactionType, error)
	Create(emoji, label string, sortOrder int) (*domain.ReactionType, error)
	Update(id int64, emoji, label string, sortOrder int, isActive bool) (*domain.ReactionType, error)
	Delete(id int64) error
}

type reactionTypeService struct {
	repo repo.ReactionTypeRepository
}

// NewReactionTypeService creates a new ReactionTypeService.
func NewReactionTypeService(r repo.ReactionTypeRepository) ReactionTypeService {
	return &reactionTypeService{repo: r}
}

// validateReactionType trims and checks emoji/label, returning the trimmed values.
func validateReactionType(emoji, label string) (string, string, error) {
	emoji = strings.TrimSpace(emoji)
	label = strings.TrimSpace(label)
	if emoji == "" || label == "" {
		return "", "", ErrReactionTypeInvalid
	}
	if utf8.RuneCountInString(emoji) > maxReactionEmojiRunes ||
		utf8.RuneCountInString(label) > maxReactionLabelRunes {
		return "", "", ErrReactionTypeInvalid
	}
	return emoji, label, nil
}

func (s *reactionTypeService) List() ([]*domain.ReactionType, error) {
	return s.repo.FindAll()
}

func (s *reactionTypeService) Create(emoji, label string, sortOrder int) (*domain.ReactionType, error) {
	emoji, label, err := validateReactionType(emoji, label)
	if err != nil {
		return nil, err
	}
	id, err := s.repo.Create(&domain.ReactionType{
		Emoji: emoji, Label: label, SortOrder: sortOrder, IsActive: true,
	})
	if err != nil {
		if errors.Is(err, repo.ErrDuplicateEmoji) {
			return nil, ErrReactionEmojiTaken
		}
		return nil, err
	}
	rt, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if rt == nil {
		// Should never happen: the row was just inserted.
		return nil, fmt.Errorf("reaction type %d not found after create", id)
	}
	return rt, nil
}

func (s *reactionTypeService) Update(id int64, emoji, label string, sortOrder int, isActive bool) (*domain.ReactionType, error) {
	emoji, label, err := validateReactionType(emoji, label)
	if err != nil {
		return nil, err
	}
	existing, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrReactionTypeNotFound
	}
	// Built-in (seed) types may not change their emoji: a renamed seed emoji
	// would be resurrected as a duplicate row on the next startup seed.
	if repo.IsSeedEmoji(existing.Emoji) && emoji != existing.Emoji {
		return nil, ErrReactionEmojiLocked
	}
	existing.Emoji = emoji
	existing.Label = label
	existing.SortOrder = sortOrder
	existing.IsActive = isActive
	if err := s.repo.Update(existing); err != nil {
		if errors.Is(err, repo.ErrDuplicateEmoji) {
			return nil, ErrReactionEmojiTaken
		}
		return nil, err
	}
	return existing, nil
}

func (s *reactionTypeService) Delete(id int64) error {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrReactionTypeNotFound
	}
	if repo.IsSeedEmoji(existing.Emoji) {
		return ErrReactionTypeProtected
	}
	deleted, err := s.repo.DeleteIfUnused(id)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrReactionTypeInUse
	}
	return nil
}
