package repo

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/jmoiron/sqlx"
	"github.com/mattn/go-sqlite3"
)

// ErrDuplicateEmoji is returned by Create/Update when the emoji UNIQUE
// constraint is violated. The service layer maps it to a 409.
var ErrDuplicateEmoji = errors.New("repo: reaction type emoji already exists")

// ReactionTypeRepository provides admin CRUD over the reaction_types master.
type ReactionTypeRepository interface {
	// FindAll returns all reaction types (active and inactive), ordered by
	// sort_order then id.
	FindAll() ([]*domain.ReactionType, error)
	// FindByID returns the type or (nil, nil) when it does not exist.
	FindByID(id int64) (*domain.ReactionType, error)
	// Create inserts a new type and returns its id. Emoji UNIQUE violation -> ErrDuplicateEmoji.
	Create(rt *domain.ReactionType) (int64, error)
	// Update writes emoji/label/sort_order/is_active by id. Emoji UNIQUE violation -> ErrDuplicateEmoji.
	Update(rt *domain.ReactionType) error
	// DeleteIfUnused physically deletes the type only when no post_reactions
	// reference it, returning whether a row was deleted. The NOT EXISTS guard
	// means a referenced type is never deleted, so ON DELETE CASCADE never fires.
	DeleteIfUnused(id int64) (bool, error)
}

type reactionTypeRepository struct {
	db *sqlx.DB
}

// NewReactionTypeRepository creates a new ReactionTypeRepository.
func NewReactionTypeRepository(db *sqlx.DB) ReactionTypeRepository {
	return &reactionTypeRepository{db: db}
}

func isUniqueViolation(err error) bool {
	var se sqlite3.Error
	if errors.As(err, &se) {
		return se.ExtendedCode == sqlite3.ErrConstraintUnique
	}
	return false
}

func (r *reactionTypeRepository) FindAll() ([]*domain.ReactionType, error) {
	types := make([]*domain.ReactionType, 0)
	err := r.db.Select(&types,
		"SELECT id, emoji, label, sort_order, is_active FROM reaction_types ORDER BY sort_order ASC, id ASC")
	if err != nil {
		return nil, fmt.Errorf("failed to list reaction types: %w", err)
	}
	return types, nil
}

func (r *reactionTypeRepository) FindByID(id int64) (*domain.ReactionType, error) {
	var rt domain.ReactionType
	err := r.db.Get(&rt,
		"SELECT id, emoji, label, sort_order, is_active FROM reaction_types WHERE id = ?", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get reaction type: %w", err)
	}
	return &rt, nil
}

func (r *reactionTypeRepository) Create(rt *domain.ReactionType) (int64, error) {
	res, err := r.db.Exec(
		"INSERT INTO reaction_types (emoji, label, sort_order, is_active) VALUES (?, ?, ?, ?)",
		rt.Emoji, rt.Label, rt.SortOrder, rt.IsActive,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return 0, ErrDuplicateEmoji
		}
		return 0, fmt.Errorf("failed to create reaction type: %w", err)
	}
	return res.LastInsertId()
}

func (r *reactionTypeRepository) Update(rt *domain.ReactionType) error {
	_, err := r.db.Exec(
		"UPDATE reaction_types SET emoji = ?, label = ?, sort_order = ?, is_active = ? WHERE id = ?",
		rt.Emoji, rt.Label, rt.SortOrder, rt.IsActive, rt.ID,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrDuplicateEmoji
		}
		return fmt.Errorf("failed to update reaction type: %w", err)
	}
	return nil
}

func (r *reactionTypeRepository) DeleteIfUnused(id int64) (bool, error) {
	res, err := r.db.Exec(
		`DELETE FROM reaction_types
		 WHERE id = ?
		   AND NOT EXISTS (SELECT 1 FROM post_reactions WHERE reaction_type_id = ?)`,
		id, id,
	)
	if err != nil {
		return false, fmt.Errorf("failed to delete reaction type: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to read rows affected: %w", err)
	}
	return n == 1, nil
}
