package repo

import (
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

// PostViewRepository is an interface that provides access to post view data
type PostViewRepository interface {
	// Record records a page view if no duplicate exists within the given window
	Record(postID int64, ipAddress, userAgent string, dedup time.Duration) (recorded bool, err error)

	// CountByPostID returns the view count for a single post
	CountByPostID(postID int64) (int64, error)

	// CountByPostIDs returns view counts for multiple posts
	CountByPostIDs(postIDs []int64) (map[int64]int64, error)

	// ScrubDeviceInfoOlderThan blanks ip_address/user_agent on rows older than
	// cutoff. These fields are only consulted within the dedup window, so beyond
	// it they are reader PII with no purpose; rows are kept so cumulative view
	// counts (COUNT(*)) are unaffected. Returns the number of rows updated.
	ScrubDeviceInfoOlderThan(cutoff time.Time) (int64, error)
}

// postViewRepository is the SQLite implementation of PostViewRepository
type postViewRepository struct {
	db *sqlx.DB
}

// NewPostViewRepository creates a new PostViewRepository
func NewPostViewRepository(db *sqlx.DB) PostViewRepository {
	return &postViewRepository{db: db}
}

// ScrubDeviceInfoOlderThan blanks ip_address/user_agent on rows older than cutoff.
func (r *postViewRepository) ScrubDeviceInfoOlderThan(cutoff time.Time) (int64, error) {
	res, err := r.db.Exec(
		`UPDATE post_views SET ip_address = '', user_agent = '' WHERE viewed_at < ? AND (ip_address != '' OR user_agent != '')`,
		cutoff.UTC(),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to scrub post_view device info: %w", err)
	}
	return res.RowsAffected()
}

// Record records a page view if no duplicate exists within the given window.
// A duplicate is defined as the same post_id + ip_address + user_agent within the window.
// Returns true if a new record was inserted.
func (r *postViewRepository) Record(postID int64, ipAddress, userAgent string, dedup time.Duration) (bool, error) {
	if dedup > 0 {
		var count int
		err := r.db.Get(&count,
			"SELECT COUNT(*) FROM post_views WHERE post_id = ? AND ip_address = ? AND user_agent = ? AND viewed_at > ?",
			postID, ipAddress, userAgent, time.Now().UTC().Add(-dedup),
		)
		if err != nil {
			return false, err
		}
		if count > 0 {
			return false, nil
		}
	}

	_, err := r.db.Exec(
		"INSERT INTO post_views (post_id, ip_address, user_agent) VALUES (?, ?, ?)",
		postID, ipAddress, userAgent,
	)
	return err == nil, err
}

// CountByPostID returns the view count for a single post
func (r *postViewRepository) CountByPostID(postID int64) (int64, error) {
	var count int64
	err := r.db.Get(&count, "SELECT COUNT(*) FROM post_views WHERE post_id = ?", postID)
	return count, err
}

// CountByPostIDs returns view counts for multiple posts
func (r *postViewRepository) CountByPostIDs(postIDs []int64) (map[int64]int64, error) {
	result := make(map[int64]int64)
	if len(postIDs) == 0 {
		return result, nil
	}

	// Build query with IN clause
	placeholders := make([]string, len(postIDs))
	args := make([]any, len(postIDs))
	for i, id := range postIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(
		"SELECT post_id, COUNT(*) as count FROM post_views WHERE post_id IN (%s) GROUP BY post_id",
		strings.Join(placeholders, ","),
	)

	rows, err := r.db.Queryx(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var postID, count int64
		if err := rows.Scan(&postID, &count); err != nil {
			return nil, err
		}
		result[postID] = count
	}

	return result, rows.Err()
}
