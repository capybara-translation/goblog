package repo

import (
	"fmt"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/jmoiron/sqlx"
)

// ReactionRepository provides access to reaction data.
type ReactionRepository interface {
	// FindSummariesByPostID returns, for each active reaction type, the total
	// count for the post plus whether visitorKey has reacted. An empty
	// visitorKey yields reacted=false everywhere (used for visitor-independent
	// SSR rendering).
	FindSummariesByPostID(postID int64, visitorKey string) ([]*domain.PostReactionSummary, error)

	// Add records a reaction. Idempotent: a duplicate (post, type, visitor) is
	// ignored via the UNIQUE constraint.
	Add(postID, reactionTypeID int64, visitorKey string) error

	// Remove deletes a visitor's reaction of the given type on the post.
	Remove(postID, reactionTypeID int64, visitorKey string) error

	// IsActiveType reports whether the reaction type exists and is active.
	IsActiveType(reactionTypeID int64) (bool, error)
}

type reactionRepository struct {
	db *sqlx.DB
}

// NewReactionRepository creates a new ReactionRepository.
func NewReactionRepository(db *sqlx.DB) ReactionRepository {
	return &reactionRepository{db: db}
}

func (r *reactionRepository) FindSummariesByPostID(postID int64, visitorKey string) ([]*domain.PostReactionSummary, error) {
	// reacted is computed via MAX(CASE ...) over the LEFT JOIN so no correlated
	// subquery is needed. With no matching post_reactions row the joined pr.*
	// are NULL, CASE evaluates to 0, and MAX over the single NULL-joined row is
	// 0 (not NULL) — so reacted scans cleanly into an int.
	const q = `
SELECT rt.id, rt.emoji, rt.label,
       COUNT(pr.id) AS count,
       MAX(CASE WHEN pr.visitor_key = ? THEN 1 ELSE 0 END) AS reacted
FROM reaction_types rt
LEFT JOIN post_reactions pr
       ON pr.reaction_type_id = rt.id AND pr.post_id = ?
WHERE rt.is_active = 1
GROUP BY rt.id, rt.emoji, rt.label, rt.sort_order
ORDER BY rt.sort_order ASC, rt.id ASC`

	rows, err := r.db.Queryx(q, visitorKey, postID)
	if err != nil {
		return nil, fmt.Errorf("failed to query reaction summaries: %w", err)
	}
	defer rows.Close()

	summaries := []*domain.PostReactionSummary{}
	for rows.Next() {
		var s domain.PostReactionSummary
		var reacted int
		if err := rows.Scan(&s.ID, &s.Emoji, &s.Label, &s.Count, &reacted); err != nil {
			return nil, fmt.Errorf("failed to scan reaction summary: %w", err)
		}
		s.Reacted = reacted == 1
		summaries = append(summaries, &s)
	}
	return summaries, rows.Err()
}

func (r *reactionRepository) Add(postID, reactionTypeID int64, visitorKey string) error {
	_, err := r.db.Exec(
		"INSERT OR IGNORE INTO post_reactions (post_id, reaction_type_id, visitor_key) VALUES (?, ?, ?)",
		postID, reactionTypeID, visitorKey,
	)
	if err != nil {
		return fmt.Errorf("failed to add reaction: %w", err)
	}
	return nil
}

func (r *reactionRepository) Remove(postID, reactionTypeID int64, visitorKey string) error {
	_, err := r.db.Exec(
		"DELETE FROM post_reactions WHERE post_id = ? AND reaction_type_id = ? AND visitor_key = ?",
		postID, reactionTypeID, visitorKey,
	)
	if err != nil {
		return fmt.Errorf("failed to remove reaction: %w", err)
	}
	return nil
}

func (r *reactionRepository) IsActiveType(reactionTypeID int64) (bool, error) {
	var count int
	err := r.db.Get(&count, "SELECT COUNT(*) FROM reaction_types WHERE id = ? AND is_active = 1", reactionTypeID)
	if err != nil {
		return false, fmt.Errorf("failed to check reaction type: %w", err)
	}
	return count > 0, nil
}
