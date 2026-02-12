package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/capybara-translation/goblog/internal/ogp"
)

// mockOGPRepository is a mock implementation of repo.OGPRepository
type mockOGPRepository struct {
	findByURLFunc    func(url string) (*ogp.Data, error)
	upsertFunc       func(data *ogp.Data) error
	deleteExpiredFunc func() (int64, error)
	upsertedData     *ogp.Data
}

func (m *mockOGPRepository) FindByURL(url string) (*ogp.Data, error) {
	if m.findByURLFunc != nil {
		return m.findByURLFunc(url)
	}
	return nil, nil
}

func (m *mockOGPRepository) Upsert(data *ogp.Data) error {
	m.upsertedData = data
	if m.upsertFunc != nil {
		return m.upsertFunc(data)
	}
	return nil
}

func (m *mockOGPRepository) DeleteExpired() (int64, error) {
	if m.deleteExpiredFunc != nil {
		return m.deleteExpiredFunc()
	}
	return 0, nil
}

// mockFetcher is a mock implementation of ogp.Fetcher
type mockFetcher struct {
	fetchFunc func(ctx context.Context, url string) (*ogp.Data, error)
}

func (m *mockFetcher) Fetch(ctx context.Context, url string) (*ogp.Data, error) {
	if m.fetchFunc != nil {
		return m.fetchFunc(ctx, url)
	}
	return nil, nil
}

func TestOGPService_Get_CacheHit(t *testing.T) {
	cachedData := &ogp.Data{
		URL:         "https://example.com",
		Title:       "Cached Title",
		Description: "Cached Description",
		FetchedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(24 * time.Hour), // Not expired
	}

	repo := &mockOGPRepository{
		findByURLFunc: func(url string) (*ogp.Data, error) {
			return cachedData, nil
		},
	}

	fetcherCalled := false
	fetcher := &mockFetcher{
		fetchFunc: func(ctx context.Context, url string) (*ogp.Data, error) {
			fetcherCalled = true
			return nil, errors.New("should not be called")
		},
	}

	service := NewOGPService(repo, fetcher)
	result := service.Get("https://example.com")

	if fetcherCalled {
		t.Error("Fetcher should not be called when cache hit")
	}
	if result.Title != "Cached Title" {
		t.Errorf("Title = %q, want %q", result.Title, "Cached Title")
	}
}

func TestOGPService_Get_CacheMiss(t *testing.T) {
	repo := &mockOGPRepository{
		findByURLFunc: func(url string) (*ogp.Data, error) {
			return nil, nil // Cache miss
		},
	}

	fetcher := &mockFetcher{
		fetchFunc: func(ctx context.Context, url string) (*ogp.Data, error) {
			return &ogp.Data{
				URL:         url,
				Title:       "Fetched Title",
				Description: "Fetched Description",
			}, nil
		},
	}

	service := NewOGPService(repo, fetcher)
	result := service.Get("https://example.com")

	if result.Title != "Fetched Title" {
		t.Errorf("Title = %q, want %q", result.Title, "Fetched Title")
	}

	// Verify data was cached
	if repo.upsertedData == nil {
		t.Error("Data should be cached after fetch")
	}
	if repo.upsertedData.Title != "Fetched Title" {
		t.Errorf("Cached title = %q, want %q", repo.upsertedData.Title, "Fetched Title")
	}
}

func TestOGPService_Get_CacheExpired(t *testing.T) {
	expiredData := &ogp.Data{
		URL:       "https://example.com",
		Title:     "Old Title",
		FetchedAt: time.Now().Add(-48 * time.Hour),
		ExpiresAt: time.Now().Add(-24 * time.Hour), // Expired
	}

	repo := &mockOGPRepository{
		findByURLFunc: func(url string) (*ogp.Data, error) {
			return expiredData, nil
		},
	}

	fetcherCalled := false
	fetcher := &mockFetcher{
		fetchFunc: func(ctx context.Context, url string) (*ogp.Data, error) {
			fetcherCalled = true
			return &ogp.Data{
				URL:   url,
				Title: "Fresh Title",
			}, nil
		},
	}

	service := NewOGPService(repo, fetcher)
	result := service.Get("https://example.com")

	if !fetcherCalled {
		t.Error("Fetcher should be called when cache is expired")
	}
	if result.Title != "Fresh Title" {
		t.Errorf("Title = %q, want %q", result.Title, "Fresh Title")
	}
}

func TestOGPService_Get_FetchError(t *testing.T) {
	repo := &mockOGPRepository{
		findByURLFunc: func(url string) (*ogp.Data, error) {
			return nil, nil
		},
	}

	fetcher := &mockFetcher{
		fetchFunc: func(ctx context.Context, url string) (*ogp.Data, error) {
			return nil, errors.New("connection timeout")
		},
	}

	service := NewOGPService(repo, fetcher)
	result := service.Get("https://example.com")

	// Should return data with URL as title (fallback)
	if result.Title != "https://example.com" {
		t.Errorf("Title = %q, want %q", result.Title, "https://example.com")
	}
	if result.ErrorMsg == "" {
		t.Error("ErrorMsg should be set on fetch error")
	}

	// Should still cache the error
	if repo.upsertedData == nil {
		t.Error("Error data should be cached")
	}
}

func TestOGPService_Get_RepoError(t *testing.T) {
	repo := &mockOGPRepository{
		findByURLFunc: func(url string) (*ogp.Data, error) {
			return nil, errors.New("database error")
		},
	}

	fetcher := &mockFetcher{
		fetchFunc: func(ctx context.Context, url string) (*ogp.Data, error) {
			return &ogp.Data{
				URL:   url,
				Title: "Fetched Title",
			}, nil
		},
	}

	service := NewOGPService(repo, fetcher)
	result := service.Get("https://example.com")

	// Should still work by fetching
	if result.Title != "Fetched Title" {
		t.Errorf("Title = %q, want %q", result.Title, "Fetched Title")
	}
}

func TestOGPService_Get_NilRepo(t *testing.T) {
	fetcher := &mockFetcher{
		fetchFunc: func(ctx context.Context, url string) (*ogp.Data, error) {
			return &ogp.Data{
				URL:   url,
				Title: "Fetched Title",
			}, nil
		},
	}

	service := NewOGPService(nil, fetcher)
	result := service.Get("https://example.com")

	if result.Title != "Fetched Title" {
		t.Errorf("Title = %q, want %q", result.Title, "Fetched Title")
	}
}

func TestOGPService_Get_SetsTTL(t *testing.T) {
	repo := &mockOGPRepository{}

	fetcher := &mockFetcher{
		fetchFunc: func(ctx context.Context, url string) (*ogp.Data, error) {
			return &ogp.Data{
				URL:   url,
				Title: "Title",
			}, nil
		},
	}

	service := NewOGPService(repo, fetcher)
	result := service.Get("https://example.com")

	// Check that FetchedAt and ExpiresAt are set
	if result.FetchedAt.IsZero() {
		t.Error("FetchedAt should be set")
	}
	if result.ExpiresAt.IsZero() {
		t.Error("ExpiresAt should be set")
	}
	if result.ExpiresAt.Before(result.FetchedAt) {
		t.Error("ExpiresAt should be after FetchedAt")
	}
}

func TestNewOGPService(t *testing.T) {
	repo := &mockOGPRepository{}
	fetcher := &mockFetcher{}

	service := NewOGPService(repo, fetcher)
	if service == nil {
		t.Error("NewOGPService() returned nil")
	}
}
