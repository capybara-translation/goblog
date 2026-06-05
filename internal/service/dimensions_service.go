package service

import (
	"image"
	_ "image/gif"  // register decoder
	_ "image/jpeg" // register decoder
	_ "image/png"  // register decoder
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "golang.org/x/image/webp" // register WebP decoder for image.DecodeConfig
)

// uploadsPrefix is the URL prefix under which user-uploaded images are
// served; only URLs matching this prefix are eligible for caching and
// disk resolution.
const uploadsPrefix = "/uploads/"

// DiskDimensionsService resolves /uploads/<file> URLs to (width, height)
// by reading just the image header from disk (image.DecodeConfig — no
// pixel decoding). Results are memoized in an in-process map for the
// lifetime of the process. Negative results (missing/broken /uploads/
// files) are cached too, so a single bad image reference does not cause
// repeated disk hits.
//
// Only /uploads/* URLs are cached — never external URLs, dangerous
// schemes, or anything else that could be reached from Markdown. Those
// strings are rejected without touching disk anyway, and caching them
// would let an admin balloon process memory just by previewing markdown
// that references many unique non-upload URLs.
//
// The cache has no eviction. Memory pressure is bounded by the number
// of distinct /uploads/* URLs ever resolved (positive + negative), with
// ~50 bytes per entry, which is acceptable for a single-author blog.
type DiskDimensionsService struct {
	uploadDir string
	cache     sync.Map // map[string]cachedDim
}

type cachedDim struct {
	w, h int
	ok   bool
}

// NewDiskDimensionsService constructs a provider rooted at uploadDir.
// uploadDir must be the same path served at /uploads/ by the HTTP layer.
// Empty uploadDir disables all lookups (Get returns ok=false).
func NewDiskDimensionsService(uploadDir string) *DiskDimensionsService {
	return &DiskDimensionsService{uploadDir: uploadDir}
}

// Get returns (width, height, true) for known /uploads/* URLs and
// (0, 0, false) for anything else. Implements markdown.DimensionsProvider.
func (s *DiskDimensionsService) Get(url string) (int, int, bool) {
	if v, ok := s.cache.Load(url); ok {
		d := v.(cachedDim)
		return d.w, d.h, d.ok
	}
	w, h, ok := s.resolve(url)
	// Only memoize results we'd be willing to repeat: a positive lookup
	// or a /uploads/* miss. External URLs and other non-/uploads/ strings
	// short-circuit cheaply inside resolve, so skipping the Store leaves
	// no disk hit to amortize while keeping the cache bounded by the
	// upload directory's URL space.
	if ok || strings.HasPrefix(url, uploadsPrefix) {
		s.cache.Store(url, cachedDim{w: w, h: h, ok: ok})
	}
	return w, h, ok
}

// Prime populates the cache with a known result. Callers that have
// already decoded an image (e.g., the upload handler, which strips
// metadata via image.Decode) can avoid forcing a first-render disk
// hit by priming here.
func (s *DiskDimensionsService) Prime(url string, width, height int) {
	if width <= 0 || height <= 0 {
		return
	}
	s.cache.Store(url, cachedDim{w: width, h: height, ok: true})
}

// resolve does the disk lookup. Only /uploads/<safe-path> URLs are
// accepted; the resolved path must stay under uploadDir.
func (s *DiskDimensionsService) resolve(url string) (int, int, bool) {
	if !strings.HasPrefix(url, uploadsPrefix) || s.uploadDir == "" {
		return 0, 0, false
	}

	// Strip a possible query/fragment so e.g. "/uploads/x.jpg?v=2" still
	// resolves to "x.jpg" on disk.
	rel := url[len(uploadsPrefix):]
	if i := strings.IndexAny(rel, "?#"); i >= 0 {
		rel = rel[:i]
	}
	// Reject empty, "." (which would otherwise resolve to uploadDir
	// itself), absolute paths, and anything containing ".." segments.
	if rel == "" || rel == "." || strings.Contains(rel, "..") || filepath.IsAbs(rel) {
		return 0, 0, false
	}

	// Build the resolved path and double-check it sits inside uploadDir.
	rootAbs, err := filepath.Abs(s.uploadDir)
	if err != nil {
		return 0, 0, false
	}
	pathAbs, err := filepath.Abs(filepath.Join(rootAbs, filepath.Clean(rel)))
	if err != nil {
		return 0, 0, false
	}
	if pathAbs != rootAbs && !strings.HasPrefix(pathAbs, rootAbs+string(filepath.Separator)) {
		return 0, 0, false
	}

	f, err := os.Open(pathAbs)
	if err != nil {
		return 0, 0, false
	}
	defer f.Close()

	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, false
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return 0, 0, false
	}
	return cfg.Width, cfg.Height, true
}
