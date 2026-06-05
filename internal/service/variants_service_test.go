package service

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	imagepkg "github.com/capybara-translation/goblog/internal/image"
	"github.com/capybara-translation/goblog/internal/markdown"
)

// DiskVariantsService inspects uploadDir for "-<width>w.webp" siblings
// of a given /uploads/* URL. It must:
//   - return only widths whose files actually exist
//   - cache results so the same URL doesn't restat the disk
//   - reject non-/uploads/ URLs (defense in depth, same as dimensions)
//   - normalize cache keys past ?... and #... query/fragment

func TestDiskVariants_ListsExistingFiles(t *testing.T) {
	dir := t.TempDir()
	mustTouch(t, filepath.Join(dir, "abc-480w.webp"))
	mustTouch(t, filepath.Join(dir, "abc-1200w.webp"))
	// 800w intentionally missing.

	d := NewDiskVariantsService(dir)
	got := d.Variants("/uploads/abc.jpg")

	wantURLs := []string{"/uploads/abc-480w.webp", "/uploads/abc-1200w.webp"}
	if len(got) != len(wantURLs) {
		t.Fatalf("got %d variants, want %d. got=%+v", len(got), len(wantURLs), got)
	}
	gotURLs := []string{got[0].URL, got[1].URL}
	for i, want := range wantURLs {
		if gotURLs[i] != want {
			t.Errorf("variant[%d] URL = %q, want %q", i, gotURLs[i], want)
		}
	}
	if got[0].Width != 480 || got[1].Width != 1200 {
		t.Errorf("widths = (%d, %d), want (480, 1200)", got[0].Width, got[1].Width)
	}
}

func TestDiskVariants_ReturnsEmptyForExternalURL(t *testing.T) {
	d := NewDiskVariantsService(t.TempDir())
	if got := d.Variants("https://example.com/foo.jpg"); len(got) != 0 {
		t.Errorf("external URL must have no variants, got %+v", got)
	}
}

func TestDiskVariants_ReturnsEmptyWhenNoFilesExist(t *testing.T) {
	d := NewDiskVariantsService(t.TempDir())
	if got := d.Variants("/uploads/nothing.jpg"); len(got) != 0 {
		t.Errorf("missing files must produce no variants, got %+v", got)
	}
}

func TestDiskVariants_CachesAcrossLookups(t *testing.T) {
	dir := t.TempDir()
	v := filepath.Join(dir, "x-800w.webp")
	mustTouch(t, v)

	d := NewDiskVariantsService(dir)

	first := d.Variants("/uploads/x.jpg")
	if len(first) != 1 {
		t.Fatalf("first lookup: got %d, want 1", len(first))
	}

	if err := os.Remove(v); err != nil {
		t.Fatalf("remove: %v", err)
	}

	second := d.Variants("/uploads/x.jpg")
	if len(second) != 1 {
		t.Errorf("second lookup should hit cache despite missing file, got %d", len(second))
	}
}

func TestDiskVariants_QueryStringNormalized(t *testing.T) {
	dir := t.TempDir()
	mustTouch(t, filepath.Join(dir, "x-800w.webp"))

	d := NewDiskVariantsService(dir)

	for _, u := range []string{
		"/uploads/x.jpg",
		"/uploads/x.jpg?v=1",
		"/uploads/x.jpg#frag",
	} {
		got := d.Variants(u)
		if len(got) != 1 || got[0].URL != "/uploads/x-800w.webp" {
			t.Errorf("Variants(%q) = %+v, want 1 entry with /uploads/x-800w.webp", u, got)
		}
	}

	count := 0
	d.cache.Range(func(_, _ any) bool { count++; return true })
	if count != 1 {
		t.Errorf("cache should hold 1 normalized key, found %d", count)
	}
}

func TestDiskVariants_DisabledWhenUploadDirEmpty(t *testing.T) {
	d := NewDiskVariantsService("")
	if got := d.Variants("/uploads/abc.jpg"); len(got) != 0 {
		t.Errorf("uploadDir=\"\" disables lookups, got %+v", got)
	}
}

func TestDiskVariants_DoesNotCacheNonUploadURLs(t *testing.T) {
	d := NewDiskVariantsService(t.TempDir())

	for _, u := range []string{
		"https://example.com/a.jpg",
		"data:image/png;base64,abc",
		"javascript:alert(1)",
		"",
	} {
		d.Variants(u)
	}

	count := 0
	d.cache.Range(func(_, _ any) bool { count++; return true })
	if count != 0 {
		t.Errorf("non-/uploads/ URLs must not pollute the cache, found %d entries", count)
	}
}

func TestDiskVariants_ImplementsMarkdownVariantsProvider(t *testing.T) {
	// Compile-time check: the service is the production implementation
	// of markdown.VariantsProvider. If this stops compiling, the wiring
	// in router.go would break.
	var _ markdown.VariantsProvider = (*DiskVariantsService)(nil)
}

func TestDiskVariants_ConcurrentLookups(t *testing.T) {
	dir := t.TempDir()
	mustTouch(t, filepath.Join(dir, "shared-800w.webp"))

	d := NewDiskVariantsService(dir)

	const goroutines = 64
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			got := d.Variants("/uploads/shared.jpg")
			if len(got) != 1 {
				t.Errorf("concurrent: got %d variants, want 1", len(got))
			}
		}()
	}
	wg.Wait()
}

func TestDiskVariants_UsesVariantPathFromImagePackage(t *testing.T) {
	// Belt-and-suspenders: the variant URLs produced by the service must
	// match what image.VariantPath generates, so the upload-time write
	// and the render-time read agree on filenames.
	for _, w := range []int{480, 800, 1200} {
		want := imagepkg.VariantPath("/uploads/abc.jpg", w)
		// Simulate the service's URL composition by creating the file
		// it should be looking for, then checking it's surfaced.
		dir := t.TempDir()
		filename := filepath.Base(want)
		mustTouch(t, filepath.Join(dir, filename))

		d := NewDiskVariantsService(dir)
		got := d.Variants("/uploads/abc.jpg")
		if len(got) != 1 || got[0].URL != want {
			t.Errorf("width=%d: got %+v, want one entry %q", w, got, want)
		}
	}
}

func mustTouch(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte{}, 0644); err != nil {
		t.Fatalf("touch %s: %v", path, err)
	}
}
