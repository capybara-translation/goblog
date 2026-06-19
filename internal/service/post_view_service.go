package service

import (
	"log"
	"strings"
	"time"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/capybara-translation/goblog/internal/repo"
)

// DeduplicationWindow is the time window for deduplicating views
// from the same IP + User-Agent combination.
const DeduplicationWindow = 30 * time.Minute

// IPRetentionWindow is how long a view's ip_address/user_agent are retained
// before being scrubbed. It MUST be >= DeduplicationWindow (those fields are
// needed for dedup within that window); the extra margin avoids scrubbing a row
// a concurrent dedup check could still consult. After this window the fields are
// reader PII with no purpose, so a background loop blanks them while keeping the
// row (cumulative view counts are preserved).
//
// NOTE: StartPostViewCleanupLoop does not scrub at startup and its first tick is
// one interval in (matching StartRememberTokenCleanupLoop), so the EFFECTIVE
// retention is approximately IPRetentionWindow + the loop interval.
const IPRetentionWindow = 1 * time.Hour

// botPatterns contains substrings used to identify bot User-Agents.
var botPatterns = []string{
	"bot", "crawl", "spider", "slurp",
	"facebook", "twitter", "linkedin",
	"headless", "puppet", "phantom",
}

// isBot returns true if the User-Agent string looks like a bot.
func isBot(ua string) bool {
	if ua == "" {
		return true
	}
	lower := strings.ToLower(ua)
	for _, p := range botPatterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

// PostViewService provides business logic for post view tracking
type PostViewService interface {
	// RecordView records a page view for a post
	RecordView(postID int64, ipAddress, userAgent string) error

	// GetViewCount returns the view count for a single post
	GetViewCount(postID int64) (int64, error)

	// GetViewCounts returns view counts for multiple posts
	GetViewCounts(postIDs []int64) (map[int64]int64, error)

	// AttachViewCounts populates the ViewCount field on each post
	AttachViewCounts(posts []*domain.Post) error

	// ScrubOldDeviceInfo blanks ip_address/user_agent on views older than
	// IPRetentionWindow. Called periodically by StartPostViewCleanupLoop.
	ScrubOldDeviceInfo() error
}

type postViewService struct {
	repo repo.PostViewRepository
}

// NewPostViewService creates a new PostViewService
func NewPostViewService(repo repo.PostViewRepository) PostViewService {
	return &postViewService{repo: repo}
}

// ScrubOldDeviceInfo blanks ip_address/user_agent on views older than IPRetentionWindow.
func (s *postViewService) ScrubOldDeviceInfo() error {
	_, err := s.repo.ScrubDeviceInfoOlderThan(time.Now().UTC().Add(-IPRetentionWindow))
	return err
}

// StartPostViewCleanupLoop periodically scrubs old view device info. The
// returned channel can be closed to stop the loop (useful in tests; production
// callers may simply leak it on shutdown). Mirrors StartRememberTokenCleanupLoop.
func StartPostViewCleanupLoop(s PostViewService, interval time.Duration) chan<- struct{} {
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := s.ScrubOldDeviceInfo(); err != nil {
					log.Printf("post-view device-info scrub failed: %v", err)
				}
			case <-stop:
				return
			}
		}
	}()
	return stop
}

func (s *postViewService) RecordView(postID int64, ipAddress, userAgent string) error {
	if isBot(userAgent) {
		return nil
	}
	_, err := s.repo.Record(postID, ipAddress, userAgent, DeduplicationWindow)
	return err
}

func (s *postViewService) GetViewCount(postID int64) (int64, error) {
	return s.repo.CountByPostID(postID)
}

func (s *postViewService) GetViewCounts(postIDs []int64) (map[int64]int64, error) {
	return s.repo.CountByPostIDs(postIDs)
}

func (s *postViewService) AttachViewCounts(posts []*domain.Post) error {
	if len(posts) == 0 {
		return nil
	}

	ids := make([]int64, len(posts))
	for i, p := range posts {
		ids[i] = p.ID
	}

	counts, err := s.repo.CountByPostIDs(ids)
	if err != nil {
		return err
	}

	for _, p := range posts {
		p.ViewCount = counts[p.ID]
	}

	return nil
}
