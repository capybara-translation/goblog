package image

import "testing"

// VariantPath / OriginalFromVariant are the single source of truth for
// where derived images live on disk and at what URL. Both sides (the
// upload-time generator and the render-time existence check) compute
// the same paths from these functions, so any drift here breaks the
// srcset pipeline.

func TestVariantPath(t *testing.T) {
	cases := []struct {
		name     string
		original string
		width    int
		want     string
	}{
		{
			name:     "JPEG to WebP variant",
			original: "/uploads/abc123.jpg",
			width:    800,
			want:     "/uploads/abc123-800w.webp",
		},
		{
			name:     "PNG to WebP variant",
			original: "/uploads/photo.png",
			width:    1200,
			want:     "/uploads/photo-1200w.webp",
		},
		{
			name:     "JPEG with uppercase extension",
			original: "/uploads/big.JPG",
			width:    480,
			want:     "/uploads/big-480w.webp",
		},
		{
			name:     "Existing WebP also gets a resized WebP variant",
			original: "/uploads/already.webp",
			width:    800,
			want:     "/uploads/already-800w.webp",
		},
		{
			name:     "Path with subdirectory",
			original: "/uploads/ogp-cache/foo.jpg",
			width:    800,
			want:     "/uploads/ogp-cache/foo-800w.webp",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := VariantPath(tc.original, tc.width)
			if got != tc.want {
				t.Errorf("VariantPath(%q, %d) = %q, want %q", tc.original, tc.width, got, tc.want)
			}
		})
	}
}

func TestIsVariantPath(t *testing.T) {
	cases := []struct {
		name string
		path string
		want bool
	}{
		{"original JPEG", "/uploads/abc.jpg", false},
		{"original PNG", "/uploads/abc.png", false},
		{"480w variant", "/uploads/abc-480w.webp", true},
		{"800w variant", "/uploads/abc-800w.webp", true},
		{"1200w variant", "/uploads/abc-1200w.webp", true},
		{"plain WebP (not a variant)", "/uploads/abc.webp", false},
		{"non-WebP with -800w suffix (defensive)", "/uploads/abc-800w.jpg", false},
		{"name with hyphens preserved", "/uploads/foo-bar-baz.jpg", false},
		{"variant of name with hyphens", "/uploads/foo-bar-baz-800w.webp", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsVariantPath(tc.path)
			if got != tc.want {
				t.Errorf("IsVariantPath(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}
