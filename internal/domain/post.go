package domain

import (
	"fmt"
	"time"
)

// PostStatus represents the status of a post
type PostStatus string

const (
	PostStatusDraft     PostStatus = "draft"
	PostStatusPublished PostStatus = "published"
)

// ParsePostStatus converts a string to PostStatus
// Returns an error for invalid values
func ParsePostStatus(s string) (PostStatus, error) {
	switch s {
	case string(PostStatusDraft):
		return PostStatusDraft, nil
	case string(PostStatusPublished):
		return PostStatusPublished, nil
	default:
		return "", fmt.Errorf("invalid status: '%s'. Must be 'draft' or 'published'", s)
	}
}

// Post is the domain model representing a blog post
type Post struct {
	ID          int64      `json:"id" db:"id"`
	Title       string     `json:"title" db:"title"`
	Slug        string     `json:"slug" db:"slug"`
	Content     string     `json:"content" db:"content"`
	Status      PostStatus `json:"status" db:"status"`
	Tags        string     `json:"tags" db:"tags"`           // Stored as comma-separated (for simplicity)
	IsPinned    bool       `json:"is_pinned" db:"is_pinned"` // Pinned to header display
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	PublishedAt *time.Time `json:"published_at,omitempty" db:"published_at"` // Published date (nil if unpublished)
	ViewCount   int64      `json:"view_count" db:"-"`                        // Populated from post_views table
	// Reactions is populated by the service layer for SSR rendering of reaction
	// buttons; not persisted and excluded from JSON API output.
	Reactions []*PostReactionSummary `json:"-" db:"-"`
}
