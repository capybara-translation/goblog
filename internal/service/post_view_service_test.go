package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/capybara-translation/goblog/internal/domain"
)

// mockPostViewRepository is a mock implementation of PostViewRepository
type mockPostViewRepository struct {
	recordFunc         func(postID int64, ipAddress, userAgent string, dedup time.Duration) (bool, error)
	countByPostIDFunc  func(postID int64) (int64, error)
	countByPostIDsFunc func(postIDs []int64) (map[int64]int64, error)
}

func (m *mockPostViewRepository) Record(postID int64, ipAddress, userAgent string, dedup time.Duration) (bool, error) {
	if m.recordFunc != nil {
		return m.recordFunc(postID, ipAddress, userAgent, dedup)
	}
	return true, nil
}

func (m *mockPostViewRepository) CountByPostID(postID int64) (int64, error) {
	if m.countByPostIDFunc != nil {
		return m.countByPostIDFunc(postID)
	}
	return 0, nil
}

func (m *mockPostViewRepository) CountByPostIDs(postIDs []int64) (map[int64]int64, error) {
	if m.countByPostIDsFunc != nil {
		return m.countByPostIDsFunc(postIDs)
	}
	return map[int64]int64{}, nil
}

func TestRecordView(t *testing.T) {
	t.Run("records view with deduplication window", func(t *testing.T) {
		var calledPostID int64
		var calledIP, calledUA string
		var calledDedup time.Duration
		mockRepo := &mockPostViewRepository{
			recordFunc: func(postID int64, ipAddress, userAgent string, dedup time.Duration) (bool, error) {
				calledPostID = postID
				calledIP = ipAddress
				calledUA = userAgent
				calledDedup = dedup
				return true, nil
			},
		}

		svc := NewPostViewService(mockRepo)
		err := svc.RecordView(1, "192.168.1.1", "Mozilla/5.0")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if calledPostID != 1 {
			t.Errorf("expected postID 1, got %d", calledPostID)
		}
		if calledIP != "192.168.1.1" {
			t.Errorf("expected IP 192.168.1.1, got %s", calledIP)
		}
		if calledUA != "Mozilla/5.0" {
			t.Errorf("expected UA Mozilla/5.0, got %s", calledUA)
		}
		if calledDedup != DeduplicationWindow {
			t.Errorf("expected dedup window %v, got %v", DeduplicationWindow, calledDedup)
		}
	})

	t.Run("returns error from repository", func(t *testing.T) {
		mockRepo := &mockPostViewRepository{
			recordFunc: func(postID int64, ipAddress, userAgent string, dedup time.Duration) (bool, error) {
				return false, fmt.Errorf("database error")
			},
		}

		svc := NewPostViewService(mockRepo)
		err := svc.RecordView(1, "192.168.1.1", "Mozilla/5.0")

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("skips bots with 'bot' in user agent", func(t *testing.T) {
		called := false
		mockRepo := &mockPostViewRepository{
			recordFunc: func(postID int64, ipAddress, userAgent string, dedup time.Duration) (bool, error) {
				called = true
				return true, nil
			},
		}

		svc := NewPostViewService(mockRepo)
		err := svc.RecordView(1, "10.0.0.1", "Googlebot/2.1")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if called {
			t.Error("expected Record not to be called for bot UA")
		}
	})

	t.Run("skips bots with 'crawl' in user agent", func(t *testing.T) {
		called := false
		mockRepo := &mockPostViewRepository{
			recordFunc: func(postID int64, ipAddress, userAgent string, dedup time.Duration) (bool, error) {
				called = true
				return true, nil
			},
		}

		svc := NewPostViewService(mockRepo)
		svc.RecordView(1, "10.0.0.1", "AhrefsCrawler/7.0")

		if called {
			t.Error("expected Record not to be called for crawler UA")
		}
	})

	t.Run("skips empty user agent", func(t *testing.T) {
		called := false
		mockRepo := &mockPostViewRepository{
			recordFunc: func(postID int64, ipAddress, userAgent string, dedup time.Duration) (bool, error) {
				called = true
				return true, nil
			},
		}

		svc := NewPostViewService(mockRepo)
		svc.RecordView(1, "10.0.0.1", "")

		if called {
			t.Error("expected Record not to be called for empty UA")
		}
	})

	t.Run("allows normal browser user agents", func(t *testing.T) {
		called := false
		mockRepo := &mockPostViewRepository{
			recordFunc: func(postID int64, ipAddress, userAgent string, dedup time.Duration) (bool, error) {
				called = true
				return true, nil
			},
		}

		svc := NewPostViewService(mockRepo)
		svc.RecordView(1, "10.0.0.1", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

		if !called {
			t.Error("expected Record to be called for normal browser UA")
		}
	})
}

func TestIsBot(t *testing.T) {
	tests := []struct {
		ua       string
		expected bool
	}{
		{"", true},
		{"Googlebot/2.1", true},
		{"Mozilla/5.0 (compatible; Bingbot/2.0)", true},
		{"Twitterbot/1.0", true},
		{"facebookexternalhit/1.1", true},
		{"AhrefsCrawler/7.0", true},
		{"Python-urllib/3.9", false},
		{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)", false},
		{"Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X)", false},
		{"HeadlessChrome/100.0", true},
		{"Slurp", true},
	}

	for _, tt := range tests {
		t.Run(tt.ua, func(t *testing.T) {
			got := isBot(tt.ua)
			if got != tt.expected {
				t.Errorf("isBot(%q) = %v, want %v", tt.ua, got, tt.expected)
			}
		})
	}
}

func TestGetViewCount(t *testing.T) {
	t.Run("returns count for a post", func(t *testing.T) {
		mockRepo := &mockPostViewRepository{
			countByPostIDFunc: func(postID int64) (int64, error) {
				if postID == 1 {
					return 42, nil
				}
				return 0, nil
			},
		}

		svc := NewPostViewService(mockRepo)
		count, err := svc.GetViewCount(1)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 42 {
			t.Errorf("expected 42, got %d", count)
		}
	})

	t.Run("returns error from repository", func(t *testing.T) {
		mockRepo := &mockPostViewRepository{
			countByPostIDFunc: func(postID int64) (int64, error) {
				return 0, fmt.Errorf("database error")
			},
		}

		svc := NewPostViewService(mockRepo)
		_, err := svc.GetViewCount(1)

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestAttachViewCounts(t *testing.T) {
	t.Run("attaches counts to posts", func(t *testing.T) {
		mockRepo := &mockPostViewRepository{
			countByPostIDsFunc: func(postIDs []int64) (map[int64]int64, error) {
				return map[int64]int64{
					1: 10,
					2: 25,
					3: 0,
				}, nil
			},
		}

		posts := []*domain.Post{
			{ID: 1},
			{ID: 2},
			{ID: 3},
		}

		svc := NewPostViewService(mockRepo)
		err := svc.AttachViewCounts(posts)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if posts[0].ViewCount != 10 {
			t.Errorf("expected post 1 view count 10, got %d", posts[0].ViewCount)
		}
		if posts[1].ViewCount != 25 {
			t.Errorf("expected post 2 view count 25, got %d", posts[1].ViewCount)
		}
		if posts[2].ViewCount != 0 {
			t.Errorf("expected post 3 view count 0, got %d", posts[2].ViewCount)
		}
	})

	t.Run("sets zero for posts not in count result", func(t *testing.T) {
		mockRepo := &mockPostViewRepository{
			countByPostIDsFunc: func(postIDs []int64) (map[int64]int64, error) {
				return map[int64]int64{1: 5}, nil
			},
		}

		posts := []*domain.Post{
			{ID: 1},
			{ID: 2},
		}

		svc := NewPostViewService(mockRepo)
		err := svc.AttachViewCounts(posts)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if posts[0].ViewCount != 5 {
			t.Errorf("expected post 1 view count 5, got %d", posts[0].ViewCount)
		}
		if posts[1].ViewCount != 0 {
			t.Errorf("expected post 2 view count 0, got %d", posts[1].ViewCount)
		}
	})

	t.Run("handles empty posts slice", func(t *testing.T) {
		mockRepo := &mockPostViewRepository{
			countByPostIDsFunc: func(postIDs []int64) (map[int64]int64, error) {
				t.Fatal("should not be called for empty posts")
				return nil, nil
			},
		}

		svc := NewPostViewService(mockRepo)
		err := svc.AttachViewCounts([]*domain.Post{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns error from repository", func(t *testing.T) {
		mockRepo := &mockPostViewRepository{
			countByPostIDsFunc: func(postIDs []int64) (map[int64]int64, error) {
				return nil, fmt.Errorf("database error")
			},
		}

		posts := []*domain.Post{{ID: 1}}

		svc := NewPostViewService(mockRepo)
		err := svc.AttachViewCounts(posts)

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
