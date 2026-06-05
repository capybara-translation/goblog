package service

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	imagepkg "github.com/capybara-translation/goblog/internal/image"
	"github.com/capybara-translation/goblog/internal/markdown"
)

// DiskVariantsService reports which WebP variants of a given
// /uploads/<file> URL physically exist on disk. The set of widths it
// checks is image.VariantWidths — the same widths the upload pipeline
// and the backfill CLI emit, so the renderer can never look for a
// width that the writers don't produce.
//
// Results are memoized in a sync.Map keyed on the URL with any
// query/fragment stripped, so e.g. "/uploads/x.jpg?v=2" shares the
// cache entry with "/uploads/x.jpg". Only /uploads/* URLs are cached;
// external URLs return an empty slice without a Store call.
//
// The cache holds no negative-versus-positive distinction beyond the
// slice it returns: a URL whose original has no variants caches the
// empty slice, which is the right answer for every subsequent render.
// If variants are generated later (e.g., by the backfill CLI on a
// running server), restart the process to clear the cache.
type DiskVariantsService struct {
	uploadDir string
	cache     sync.Map // map[string][]markdown.Variant
}

// NewDiskVariantsService constructs a provider rooted at uploadDir.
// Empty uploadDir disables all lookups (Variants returns an empty slice).
func NewDiskVariantsService(uploadDir string) *DiskVariantsService {
	return &DiskVariantsService{uploadDir: uploadDir}
}

// Variants implements markdown.VariantsProvider. The returned slice is
// ordered by ascending width, so callers can pick the smallest acceptable
// candidate first when composing srcset.
func (s *DiskVariantsService) Variants(url string) []markdown.Variant {
	if s.uploadDir == "" {
		return nil
	}
	key := variantsCacheKey(url)
	if !strings.HasPrefix(key, uploadsPrefix) {
		return nil
	}
	if v, ok := s.cache.Load(key); ok {
		return v.([]markdown.Variant)
	}
	v := s.resolveVariants(key)
	s.cache.Store(key, v)
	return v
}

// resolveVariants stats one candidate path per VariantWidth and returns
// only those that exist. The expected path under uploadDir is computed
// from imagepkg.VariantPath so the writer and reader agree.
func (s *DiskVariantsService) resolveVariants(uploadURL string) []markdown.Variant {
	rel := strings.TrimPrefix(uploadURL, uploadsPrefix)
	if rel == "" || strings.Contains(rel, "..") || filepath.IsAbs(rel) {
		return nil
	}

	rootAbs, err := filepath.Abs(s.uploadDir)
	if err != nil {
		return nil
	}

	out := make([]markdown.Variant, 0, len(imagepkg.VariantWidths))
	for _, w := range imagepkg.VariantWidths {
		variantURL := imagepkg.VariantPath(uploadURL, w)
		variantRel := strings.TrimPrefix(variantURL, uploadsPrefix)

		pathAbs, err := filepath.Abs(filepath.Join(rootAbs, filepath.Clean(variantRel)))
		if err != nil {
			continue
		}
		if pathAbs != rootAbs && !strings.HasPrefix(pathAbs, rootAbs+string(filepath.Separator)) {
			continue
		}

		if _, err := os.Stat(pathAbs); err != nil {
			continue
		}
		out = append(out, markdown.Variant{URL: variantURL, Width: w})
	}
	return out
}

// variantsCacheKey mirrors cacheKey in dimensions_service: strip query
// and fragment so equivalent URLs share an entry.
func variantsCacheKey(url string) string {
	if i := strings.IndexAny(url, "?#"); i >= 0 {
		return url[:i]
	}
	return url
}
