package domain

// ReactionType is a single emoji readers may react with (master data).
type ReactionType struct {
	ID        int64  `json:"id" db:"id"`
	Emoji     string `json:"emoji" db:"emoji"`
	Label     string `json:"label" db:"label"`
	SortOrder int    `json:"sort_order" db:"sort_order"`
	IsActive  bool   `json:"is_active" db:"is_active"`
}

// PostReactionSummary is the per-reaction-type aggregate shown for one post:
// the total count plus whether the current visitor has reacted. Built by the
// repository's aggregation query; no db tags because it is hand-scanned.
type PostReactionSummary struct {
	ID      int64  `json:"id"`
	Emoji   string `json:"emoji"`
	Label   string `json:"label"`
	Count   int64  `json:"count"`
	Reacted bool   `json:"reacted"`
}
