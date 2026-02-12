package service

import (
	"context"
	"log"
	"time"

	"github.com/capybara-translation/goblog/internal/ogp"
	"github.com/capybara-translation/goblog/internal/repo"
)

// OGPService provides OGP data retrieval with caching
type OGPService interface {
	// Get retrieves OGP data for a URL, using cache when available
	Get(url string) *ogp.Data
}

// ogpService implements the OGPService interface
type ogpService struct {
	repo    repo.OGPRepository
	fetcher ogp.Fetcher
}

// NewOGPService creates a new OGP service with caching
func NewOGPService(ogpRepo repo.OGPRepository, fetcher ogp.Fetcher) OGPService {
	return &ogpService{
		repo:    ogpRepo,
		fetcher: fetcher,
	}
}

// Get retrieves OGP data for a URL, using cache when available
func (s *ogpService) Get(url string) *ogp.Data {
	// 1. Check SQLite cache
	if s.repo != nil {
		data, err := s.repo.FindByURL(url)
		if err != nil {
			log.Printf("failed to find OGP data in repo: %v", err)
		} else if data != nil && !data.IsExpired() {
			return data
		}
	}

	// 2. Fetch from external URL
	ctx, cancel := context.WithTimeout(context.Background(), ogp.FetchTimeout)
	defer cancel()

	fetchedData, err := s.fetcher.Fetch(ctx, url)
	if err != nil {
		log.Printf("failed to fetch OGP data for %s: %v", url, err)
		fetchedData = &ogp.Data{
			URL:       url,
			Title:     url,
			FetchedAt: time.Now(),
			ExpiresAt: time.Now().Add(ogp.ErrorTTL),
			ErrorMsg:  err.Error(),
		}
	} else {
		fetchedData.FetchedAt = time.Now()
		fetchedData.ExpiresAt = time.Now().Add(ogp.SuccessTTL)
	}

	// 3. Save to SQLite cache
	if s.repo != nil {
		if err := s.repo.Upsert(fetchedData); err != nil {
			log.Printf("failed to cache OGP data: %v", err)
		}
	}

	return fetchedData
}
