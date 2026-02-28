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
	deleteExpiredFunc func() ([]string, error)
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

func (m *mockOGPRepository) DeleteExpired() ([]string, error) {
	if m.deleteExpiredFunc != nil {
		return m.deleteExpiredFunc()
	}
	return nil, nil
}

// mockFetcher is a mock implementation of ogp.Fetcher
type mockFetcher struct {
	fetchFunc         func(ctx context.Context, url string) (*ogp.Data, error)
	downloadImageFunc func(ctx context.Context, imageURL string, destDir string) (string, error)
}

func (m *mockFetcher) Fetch(ctx context.Context, url string) (*ogp.Data, error) {
	if m.fetchFunc != nil {
		return m.fetchFunc(ctx, url)
	}
	return nil, nil
}

func (m *mockFetcher) DownloadImage(ctx context.Context, imageURL string, destDir string) (string, error) {
	if m.downloadImageFunc != nil {
		return m.downloadImageFunc(ctx, imageURL, destDir)
	}
	return "", nil
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

	service := NewOGPService(repo, fetcher, "")
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

	service := NewOGPService(repo, fetcher, "")
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

	service := NewOGPService(repo, fetcher, "")
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

	service := NewOGPService(repo, fetcher, "")
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

	service := NewOGPService(repo, fetcher, "")
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

	service := NewOGPService(nil, fetcher, "")
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

	service := NewOGPService(repo, fetcher, "")
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

func TestOGPService_Get_CacheHitDownloadsImage(t *testing.T) {
	// Cache hit with ImageURL but empty LocalImagePath should trigger download
	cachedData := &ogp.Data{
		URL:         "https://example.com",
		Title:       "Cached Title",
		ImageURL:    "https://example.com/image.jpg",
		FetchedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		// LocalImagePath intentionally empty
	}

	repo := &mockOGPRepository{
		findByURLFunc: func(url string) (*ogp.Data, error) {
			return cachedData, nil
		},
	}

	downloadCalled := false
	fetcher := &mockFetcher{
		downloadImageFunc: func(ctx context.Context, imageURL string, destDir string) (string, error) {
			downloadCalled = true
			if imageURL != "https://example.com/image.jpg" {
				t.Errorf("DownloadImage imageURL = %q, want %q", imageURL, "https://example.com/image.jpg")
			}
			return "ogp-cache/test-uuid.jpg", nil
		},
	}

	service := NewOGPService(repo, fetcher, "/tmp/uploads")
	result := service.Get("https://example.com")

	if !downloadCalled {
		t.Error("DownloadImage should be called when cache hit has empty LocalImagePath")
	}
	if result.LocalImagePath != "/uploads/ogp-cache/test-uuid.jpg" {
		t.Errorf("LocalImagePath = %q, want %q", result.LocalImagePath, "/uploads/ogp-cache/test-uuid.jpg")
	}
	// Verify Upsert was called to save the local image path
	if repo.upsertedData == nil {
		t.Error("Upsert should be called to save local image path")
	}
}

func TestOGPService_Get_CacheHitDownloadFailure(t *testing.T) {
	// Cache hit with ImageURL, download fails — should still return cached data
	cachedData := &ogp.Data{
		URL:       "https://example.com",
		Title:     "Cached Title",
		ImageURL:  "https://example.com/image.jpg",
		FetchedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	repo := &mockOGPRepository{
		findByURLFunc: func(url string) (*ogp.Data, error) {
			return cachedData, nil
		},
	}

	fetcher := &mockFetcher{
		downloadImageFunc: func(ctx context.Context, imageURL string, destDir string) (string, error) {
			return "", errors.New("download failed: 429 too many requests")
		},
	}

	service := NewOGPService(repo, fetcher, "/tmp/uploads")
	result := service.Get("https://example.com")

	if result.Title != "Cached Title" {
		t.Errorf("Title = %q, want %q", result.Title, "Cached Title")
	}
	if result.LocalImagePath != "" {
		t.Errorf("LocalImagePath = %q, want empty on download failure", result.LocalImagePath)
	}
}

func TestOGPService_Get_CacheHitSkipsDownloadWhenNoUploadDir(t *testing.T) {
	cachedData := &ogp.Data{
		URL:       "https://example.com",
		Title:     "Cached Title",
		ImageURL:  "https://example.com/image.jpg",
		FetchedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	repo := &mockOGPRepository{
		findByURLFunc: func(url string) (*ogp.Data, error) {
			return cachedData, nil
		},
	}

	downloadCalled := false
	fetcher := &mockFetcher{
		downloadImageFunc: func(ctx context.Context, imageURL string, destDir string) (string, error) {
			downloadCalled = true
			return "ogp-cache/test.jpg", nil
		},
	}

	// uploadDir is empty — should not attempt download
	service := NewOGPService(repo, fetcher, "")
	result := service.Get("https://example.com")

	if downloadCalled {
		t.Error("DownloadImage should not be called when uploadDir is empty")
	}
	if result.Title != "Cached Title" {
		t.Errorf("Title = %q, want %q", result.Title, "Cached Title")
	}
}

func TestOGPService_Get_CacheHitSkipsDownloadWhenAlreadyCached(t *testing.T) {
	cachedData := &ogp.Data{
		URL:            "https://example.com",
		Title:          "Cached Title",
		ImageURL:       "https://example.com/image.jpg",
		LocalImagePath: "/uploads/ogp-cache/existing.jpg",
		FetchedAt:      time.Now(),
		ExpiresAt:      time.Now().Add(24 * time.Hour),
	}

	repo := &mockOGPRepository{
		findByURLFunc: func(url string) (*ogp.Data, error) {
			return cachedData, nil
		},
	}

	downloadCalled := false
	fetcher := &mockFetcher{
		downloadImageFunc: func(ctx context.Context, imageURL string, destDir string) (string, error) {
			downloadCalled = true
			return "ogp-cache/new.jpg", nil
		},
	}

	service := NewOGPService(repo, fetcher, "/tmp/uploads")
	result := service.Get("https://example.com")

	if downloadCalled {
		t.Error("DownloadImage should not be called when LocalImagePath already exists")
	}
	if result.LocalImagePath != "/uploads/ogp-cache/existing.jpg" {
		t.Errorf("LocalImagePath = %q, want %q", result.LocalImagePath, "/uploads/ogp-cache/existing.jpg")
	}
}

func TestOGPService_Get_FreshFetchWithImageDownload(t *testing.T) {
	repo := &mockOGPRepository{
		findByURLFunc: func(url string) (*ogp.Data, error) {
			return nil, nil // Cache miss
		},
	}

	fetcher := &mockFetcher{
		fetchFunc: func(ctx context.Context, url string) (*ogp.Data, error) {
			return &ogp.Data{
				URL:      url,
				Title:    "Fetched Title",
				ImageURL: "https://example.com/image.jpg",
			}, nil
		},
		downloadImageFunc: func(ctx context.Context, imageURL string, destDir string) (string, error) {
			return "ogp-cache/new-uuid.jpg", nil
		},
	}

	service := NewOGPService(repo, fetcher, "/tmp/uploads")
	result := service.Get("https://example.com")

	if result.LocalImagePath != "/uploads/ogp-cache/new-uuid.jpg" {
		t.Errorf("LocalImagePath = %q, want %q", result.LocalImagePath, "/uploads/ogp-cache/new-uuid.jpg")
	}
	if repo.upsertedData == nil {
		t.Fatal("Data should be cached after fetch")
	}
	if repo.upsertedData.LocalImagePath != "/uploads/ogp-cache/new-uuid.jpg" {
		t.Errorf("Cached LocalImagePath = %q, want %q", repo.upsertedData.LocalImagePath, "/uploads/ogp-cache/new-uuid.jpg")
	}
}

func TestOGPService_Get_FreshFetchImageDownloadFailure(t *testing.T) {
	repo := &mockOGPRepository{
		findByURLFunc: func(url string) (*ogp.Data, error) {
			return nil, nil
		},
	}

	fetcher := &mockFetcher{
		fetchFunc: func(ctx context.Context, url string) (*ogp.Data, error) {
			return &ogp.Data{
				URL:      url,
				Title:    "Fetched Title",
				ImageURL: "https://example.com/image.jpg",
			}, nil
		},
		downloadImageFunc: func(ctx context.Context, imageURL string, destDir string) (string, error) {
			return "", errors.New("download failed")
		},
	}

	service := NewOGPService(repo, fetcher, "/tmp/uploads")
	result := service.Get("https://example.com")

	// Data should still be returned, but without LocalImagePath
	if result.Title != "Fetched Title" {
		t.Errorf("Title = %q, want %q", result.Title, "Fetched Title")
	}
	if result.LocalImagePath != "" {
		t.Errorf("LocalImagePath = %q, want empty on download failure", result.LocalImagePath)
	}
	// Should still be cached (just without local image)
	if repo.upsertedData == nil {
		t.Error("Data should still be cached even if image download fails")
	}
}

func TestNewOGPService(t *testing.T) {
	repo := &mockOGPRepository{}
	fetcher := &mockFetcher{}

	service := NewOGPService(repo, fetcher, "")
	if service == nil {
		t.Error("NewOGPService() returned nil")
	}
}
