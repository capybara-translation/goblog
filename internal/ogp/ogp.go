// Package ogp provides Open Graph Protocol data fetching and caching
package ogp

import (
	"time"
)

// Data represents Open Graph Protocol metadata for a URL
type Data struct {
	URL         string    `db:"url" json:"url"`
	Title       string    `db:"title" json:"title"`
	Description string    `db:"description" json:"description"`
	ImageURL    string    `db:"image_url" json:"image_url"`
	SiteName    string    `db:"site_name" json:"site_name"`
	FetchedAt   time.Time `db:"fetched_at" json:"fetched_at"`
	ExpiresAt   time.Time `db:"expires_at" json:"expires_at"`
	ErrorMsg    string    `db:"error_msg" json:"error_msg,omitempty"`
}

// IsError returns true if the OGP fetch resulted in an error
func (d *Data) IsError() bool {
	return d.ErrorMsg != ""
}

// IsExpired returns true if the cached data has expired
func (d *Data) IsExpired() bool {
	return time.Now().After(d.ExpiresAt)
}

// HasImage returns true if the OGP data includes an image
func (d *Data) HasImage() bool {
	return d.ImageURL != ""
}

// Constants for cache TTL and fetch settings
const (
	// SuccessTTL is the cache duration for successful fetches
	SuccessTTL = 7 * 24 * time.Hour // 7 days
	// ErrorTTL is the cache duration for failed fetches
	ErrorTTL = 1 * time.Hour // 1 hour
	// FetchTimeout is the maximum time to wait for a fetch
	FetchTimeout = 5 * time.Second
)
