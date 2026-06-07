package repo

import (
	"fmt"
	"strings"

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
	// It is a no-op (no error) if no matching reaction exists.
	Remove(postID, reactionTypeID int64, visitorKey string) error

	// IsActiveType reports whether the reaction type exists and is active.
	IsActiveType(reactionTypeID int64) (bool, error)

	// FindSummariesByPostIDs returns reaction summaries for each given post id,
	// keyed by post id. Every requested post gets an entry containing ALL active
	// reaction types (ordered by sort_order, id), with count and per-visitor
	// reacted filled in (zero/false when the post has no such reaction). An empty
	// visitorKey yields reacted=false everywhere. Returns an empty map for empty input.
	FindSummariesByPostIDs(postIDs []int64, visitorKey string) (map[int64][]*domain.PostReactionSummary, error)
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

func (r *reactionRepository) FindSummariesByPostIDs(postIDs []int64, visitorKey string) (map[int64][]*domain.PostReactionSummary, error) {
	result := make(map[int64][]*domain.PostReactionSummary)
	if len(postIDs) == 0 {
		return result, nil
	}

	// 1. Load all active reaction types ordered by sort_order, id.
	type reactionType struct {
		ID    int64  `db:"id"`
		Emoji string `db:"emoji"`
		Label string `db:"label"`
	}
	var activeTypes []reactionType
	if err := r.db.Select(&activeTypes,
		"SELECT id, emoji, label FROM reaction_types WHERE is_active = 1 ORDER BY sort_order ASC, id ASC",
	); err != nil {
		return nil, fmt.Errorf("failed to query active reaction types: %w", err)
	}

	// 2. Aggregate per (post_id, reaction_type_id) for the requested posts.
	placeholders := make([]string, len(postIDs))
	args := make([]any, 0, 1+len(postIDs))
	args = append(args, visitorKey)
	for i, id := range postIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}
	aggQuery := fmt.Sprintf(
		`SELECT post_id, reaction_type_id, COUNT(*) AS count,
		        MAX(CASE WHEN visitor_key = ? THEN 1 ELSE 0 END) AS reacted
		 FROM post_reactions
		 WHERE post_id IN (%s)
		 GROUP BY post_id, reaction_type_id`,
		strings.Join(placeholders, ","),
	)

	type aggRow struct {
		PostID         int64 `db:"post_id"`
		ReactionTypeID int64 `db:"reaction_type_id"`
		Count          int64 `db:"count"`
		Reacted        int   `db:"reacted"`
	}
	rows, err := r.db.Queryx(aggQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query reaction aggregates: %w", err)
	}
	defer rows.Close()

	// agg[postID][reactionTypeID] = {count, reacted}
	type counts struct {
		count   int64
		reacted bool
	}
	agg := make(map[int64]map[int64]counts)
	for rows.Next() {
		var row aggRow
		if err := rows.StructScan(&row); err != nil {
			return nil, fmt.Errorf("failed to scan reaction aggregate: %w", err)
		}
		if agg[row.PostID] == nil {
			agg[row.PostID] = make(map[int64]counts)
		}
		agg[row.PostID][row.ReactionTypeID] = counts{count: row.Count, reacted: row.Reacted == 1}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate reaction aggregates: %w", err)
	}

	// 3. Assemble: for each requested postID build summaries from active types.
	for _, postID := range postIDs {
		summaries := make([]*domain.PostReactionSummary, 0, len(activeTypes))
		postAgg := agg[postID] // nil map is safe to read from
		for _, rt := range activeTypes {
			c := postAgg[rt.ID] // zero-value if absent
			summaries = append(summaries, &domain.PostReactionSummary{
				ID:      rt.ID,
				Emoji:   rt.Emoji,
				Label:   rt.Label,
				Count:   c.count,
				Reacted: c.reacted,
			})
		}
		result[postID] = summaries
	}
	return result, nil
}
