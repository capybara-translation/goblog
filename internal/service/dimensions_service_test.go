package service

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/chai2010/webp"
)

func TestDiskDimensions_DecodesJPEG_PNG_WebP(t *testing.T) {
	dir := t.TempDir()
	mustWriteJPEG(t, filepath.Join(dir, "photo.jpg"), 320, 240)
	mustWritePNG(t, filepath.Join(dir, "icon.png"), 64, 48)
	mustWriteWebP(t, filepath.Join(dir, "modern.webp"), 800, 600)

	d := NewDiskDimensionsService(dir)

	cases := []struct {
		url   string
		w, h  int
	}{
		{"/uploads/photo.jpg", 320, 240},
		{"/uploads/icon.png", 64, 48},
		{"/uploads/modern.webp", 800, 600},
	}
	for _, tc := range cases {
		gotW, gotH, ok := d.Get(tc.url)
		if !ok {
			t.Errorf("Get(%q): ok=false, want true", tc.url)
			continue
		}
		if gotW != tc.w || gotH != tc.h {
			t.Errorf("Get(%q) = (%d, %d), want (%d, %d)", tc.url, gotW, gotH, tc.w, tc.h)
		}
	}
}

func TestDiskDimensions_RejectsExternalURL(t *testing.T) {
	d := NewDiskDimensionsService(t.TempDir())
	if _, _, ok := d.Get("https://example.com/foo.jpg"); ok {
		t.Errorf("external URL must return ok=false")
	}
}

func TestDiskDimensions_RejectsPathTraversal(t *testing.T) {
	d := NewDiskDimensionsService(t.TempDir())
	for _, evil := range []string{
		"/uploads/../secret.jpg",
		"/uploads/..%2Fsecret.jpg",
		"/uploads/",
		"/uploads/sub/../../etc/passwd",
		"/uploads/.",          // would otherwise os.Open the upload dir itself
		"/uploads//etc/passwd", // filepath.IsAbs catches the embedded absolute path
	} {
		if _, _, ok := d.Get(evil); ok {
			t.Errorf("traversal-y URL %q must return ok=false", evil)
		}
	}
}

func TestDiskDimensions_NonExistentReturnsOkFalse(t *testing.T) {
	d := NewDiskDimensionsService(t.TempDir())
	if _, _, ok := d.Get("/uploads/missing.jpg"); ok {
		t.Errorf("missing file must return ok=false")
	}
}

func TestDiskDimensions_CachesAcrossLookups(t *testing.T) {
	// Second Get must not depend on the file existing — that proves the
	// first result was memoized. Test by deleting the file between calls.
	dir := t.TempDir()
	path := filepath.Join(dir, "ephemeral.png")
	mustWritePNG(t, path, 12, 34)

	d := NewDiskDimensionsService(dir)
	w1, h1, ok1 := d.Get("/uploads/ephemeral.png")
	if !ok1 || w1 != 12 || h1 != 34 {
		t.Fatalf("first lookup: (%d, %d, %v) want (12, 34, true)", w1, h1, ok1)
	}
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove: %v", err)
	}
	w2, h2, ok2 := d.Get("/uploads/ephemeral.png")
	if !ok2 || w2 != 12 || h2 != 34 {
		t.Errorf("after delete: (%d, %d, %v) want (12, 34, true) — cache miss?", w2, h2, ok2)
	}
}

func TestDiskDimensions_CachesNegativeResult(t *testing.T) {
	// Same idea for ok=false: missing now → still missing later, even if
	// the file appears between the two lookups. This prevents a hot-loop
	// of disk hits for genuinely-broken image refs.
	dir := t.TempDir()
	d := NewDiskDimensionsService(dir)

	if _, _, ok := d.Get("/uploads/notyet.png"); ok {
		t.Fatalf("first lookup should miss")
	}
	mustWritePNG(t, filepath.Join(dir, "notyet.png"), 100, 100)
	if _, _, ok := d.Get("/uploads/notyet.png"); ok {
		t.Errorf("expected cached negative result, got ok=true (cache miss?)")
	}
}

func TestDiskDimensions_DoesNotCacheNonUploadURLs(t *testing.T) {
	// The cache must only grow with /uploads/* URLs (positive or negative).
	// External URLs, dangerous schemes, and other strings come from
	// Markdown destinations under attacker influence; caching every distinct
	// one would let an admin balloon process memory just by previewing
	// markdown that references many unique external URLs.
	d := NewDiskDimensionsService(t.TempDir())

	for _, u := range []string{
		"https://example.com/foo.jpg",
		"http://example.com/bar.png",
		"data:image/png;base64,abc",
		"javascript:alert(1)",
		"",
		"/static/css/site.css",
	} {
		if _, _, ok := d.Get(u); ok {
			t.Errorf("Get(%q) returned ok=true unexpectedly", u)
		}
	}

	count := 0
	d.cache.Range(func(_, _ any) bool { count++; return true })
	if count != 0 {
		t.Errorf("cache should be empty for non-/uploads/ URLs, found %d entries", count)
	}
}

func TestDiskDimensions_CachesNegativeForUploadsPrefix(t *testing.T) {
	// Conversely, missing /uploads/* files SHOULD still cache a negative
	// result so a single bad reference doesn't cause repeated disk hits.
	dir := t.TempDir()
	d := NewDiskDimensionsService(dir)

	if _, _, ok := d.Get("/uploads/missing.jpg"); ok {
		t.Fatalf("missing /uploads/ file should return ok=false")
	}

	count := 0
	d.cache.Range(func(_, _ any) bool { count++; return true })
	if count != 1 {
		t.Errorf("expected 1 negative cache entry for /uploads/ miss, got %d", count)
	}
}

func TestDiskDimensions_PrimePopulatesCache(t *testing.T) {
	// Prime is the path used after upload: the upload handler already
	// has the bytes in memory and shouldn't pay for a re-decode at
	// first-render time.
	d := NewDiskDimensionsService(t.TempDir())
	d.Prime("/uploads/just-saved.jpg", 1920, 1080)

	w, h, ok := d.Get("/uploads/just-saved.jpg")
	if !ok || w != 1920 || h != 1080 {
		t.Errorf("after Prime: (%d, %d, %v) want (1920, 1080, true)", w, h, ok)
	}
}

// Without an explicit singleflight, a stampede on a cold cache lets every
// concurrent goroutine race to decode the same file. That's documented
// behavior (the Get path is intentionally lock-free); this test pins it
// down: every caller MUST converge on the same result, and `go test -race`
// MUST stay clean.
func TestDiskDimensions_ConcurrentGet(t *testing.T) {
	dir := t.TempDir()
	mustWritePNG(t, filepath.Join(dir, "shared.png"), 320, 240)

	d := NewDiskDimensionsService(dir)

	const goroutines = 64
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			w, h, ok := d.Get("/uploads/shared.png")
			if !ok || w != 320 || h != 240 {
				t.Errorf("concurrent Get: (%d, %d, %v) want (320, 240, true)", w, h, ok)
			}
		}()
	}
	wg.Wait()
}

// --- fixture writers ---

func mustWriteJPEG(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		img.Set(x, 0, color.RGBA{200, 100, 50, 255})
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		t.Fatalf("write jpeg: %v", err)
	}
}

func mustWritePNG(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		t.Fatalf("write png: %v", err)
	}
}

func mustWriteWebP(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	data, err := webp.EncodeRGBA(img, 80)
	if err != nil {
		t.Fatalf("encode webp: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write webp: %v", err)
	}
}
