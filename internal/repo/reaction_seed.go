package repo

import (
	"fmt"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/jmoiron/sqlx"
)

// DefaultReactionTypes is the single source of truth for the built-in reaction
// emojis. It drives both seeding (SeedReactionTypes) and the protected-set check
// (IsSeedEmoji), so the two can never drift. These rows are "system" types: the
// admin UI may toggle/relabel them but may not change their emoji or physically
// delete them (a delete would be resurrected here on the next startup).
var DefaultReactionTypes = []domain.ReactionType{
	{Emoji: "👍", Label: "いいね", SortOrder: 10},
	{Emoji: "❤️", Label: "好き", SortOrder: 20},
	{Emoji: "🎉", Label: "おめでとう", SortOrder: 30},
	{Emoji: "👀", Label: "見た", SortOrder: 40},
	{Emoji: "🤔", Label: "なるほど", SortOrder: 50},
}

// SeedReactionTypes inserts the default reaction types. INSERT OR IGNORE keeps it
// idempotent: existing rows (matched by the emoji UNIQUE constraint) are left
// untouched, preserving later admin edits to is_active / sort_order.
func SeedReactionTypes(db *sqlx.DB) error {
	for _, rt := range DefaultReactionTypes {
		if _, err := db.Exec(
			"INSERT OR IGNORE INTO reaction_types (emoji, label, sort_order) VALUES (?, ?, ?)",
			rt.Emoji, rt.Label, rt.SortOrder,
		); err != nil {
			return fmt.Errorf("failed to seed reaction type %q: %w", rt.Emoji, err)
		}
	}
	return nil
}

// IsSeedEmoji reports whether emoji belongs to the built-in set. Such types are
// protected from emoji edits and physical deletion.
func IsSeedEmoji(emoji string) bool {
	for _, rt := range DefaultReactionTypes {
		if rt.Emoji == emoji {
			return true
		}
	}
	return false
}
