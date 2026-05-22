package domain

import "time"

// RememberToken represents a long-lived "remember me" token row.
// The raw token never lives in the database; only its SHA-256 hash does.
type RememberToken struct {
	ID         int64      `db:"id"`
	UserID     int64      `db:"user_id"`
	Selector   string     `db:"selector"`
	TokenHash  string     `db:"token_hash"`
	ExpiresAt  time.Time  `db:"expires_at"`
	CreatedAt  time.Time  `db:"created_at"`
	LastUsedAt *time.Time `db:"last_used_at"`
}
