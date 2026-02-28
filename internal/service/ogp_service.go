package service

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
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
	repo      repo.OGPRepository
	fetcher   ogp.Fetcher
	uploadDir string
}

// NewOGPService creates a new OGP service with caching
func NewOGPService(ogpRepo repo.OGPRepository, fetcher ogp.Fetcher, uploadDir string) OGPService {
	s := &ogpService{
		repo:      ogpRepo,
		fetcher:   fetcher,
		uploadDir: uploadDir,
	}

	// Periodically clean up expired OGP cache entries and their local images
	if ogpRepo != nil {
		go func() {
			ticker := time.NewTicker(1 * time.Hour)
			defer ticker.Stop()
			for range ticker.C {
				s.cleanupExpired()
			}
		}()
	}

	return s
}

// cleanupExpired removes expired OGP cache entries and their local image files
func (s *ogpService) cleanupExpired() {
	paths, err := s.repo.DeleteExpired()
	if err != nil {
		log.Printf("failed to cleanup expired OGP cache: %v", err)
		return
	}
	for _, path := range paths {
		s.removeLocalImage(path)
	}
	if len(paths) > 0 {
		log.Printf("cleaned up %d expired OGP cache images", len(paths))
	}
}

// Get retrieves OGP data for a URL, using cache when available
func (s *ogpService) Get(url string) *ogp.Data {
	// 1. Check SQLite cache
	var oldLocalImagePath string
	if s.repo != nil {
		data, err := s.repo.FindByURL(url)
		if err != nil {
			log.Printf("failed to find OGP data in repo: %v", err)
		} else if data != nil && !data.IsExpired() {
			// Cache hit — but download image if not yet cached locally
			if data.ImageURL != "" && data.LocalImagePath == "" && s.uploadDir != "" {
				imgCtx, imgCancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer imgCancel()

				relativePath, dlErr := s.fetcher.DownloadImage(imgCtx, data.ImageURL, s.uploadDir)
				if dlErr != nil {
					log.Printf("failed to download OGP image for %s: %v", url, dlErr)
				} else {
					data.LocalImagePath = "/uploads/" + relativePath
					if uErr := s.repo.Upsert(data); uErr != nil {
						log.Printf("failed to update local image path: %v", uErr)
					}
				}
			}
			return data
		} else if data != nil {
			// Cache expired — remember old local image for cleanup
			oldLocalImagePath = data.LocalImagePath
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

		// 2.5. Download OGP image locally
		if fetchedData.ImageURL != "" && s.uploadDir != "" {
			imgCtx, imgCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer imgCancel()

			relativePath, err := s.fetcher.DownloadImage(imgCtx, fetchedData.ImageURL, s.uploadDir)
			if err != nil {
				log.Printf("failed to download OGP image for %s: %v", url, err)
			} else {
				fetchedData.LocalImagePath = "/uploads/" + relativePath
			}
		}
	}

	// 3. Save to SQLite cache
	if s.repo != nil {
		if err := s.repo.Upsert(fetchedData); err != nil {
			log.Printf("failed to cache OGP data: %v", err)
		}
	}

	// 4. Clean up old local image if it was replaced
	if oldLocalImagePath != "" && oldLocalImagePath != fetchedData.LocalImagePath {
		s.removeLocalImage(oldLocalImagePath)
	}

	return fetchedData
}

// removeLocalImage deletes a cached OGP image file from disk
func (s *ogpService) removeLocalImage(localImagePath string) {
	if s.uploadDir == "" || localImagePath == "" {
		return
	}
	// localImagePath is like "/uploads/ogp-cache/uuid.jpg"
	// Convert to filesystem path: "{uploadDir}/ogp-cache/uuid.jpg"
	relativePath := strings.TrimPrefix(localImagePath, "/uploads/")
	filePath := filepath.Join(s.uploadDir, relativePath)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		log.Printf("failed to remove old OGP image %s: %v", filePath, err)
	}
}
