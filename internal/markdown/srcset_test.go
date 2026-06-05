package markdown

import (
	"strings"
	"testing"
)

// When the VariantsProvider returns variants, the image renderer must:
//   - emit srcset with each variant + the variant URL of the middle width as src
//   - emit a sensible default sizes attribute
//   - survive bluemonday sanitization (srcset/sizes must be allowed)
//
// When no variants are present, the renderer must fall back to plain src.

type stubVariants map[string][]Variant

func (s stubVariants) Variants(url string) []Variant {
	return s[url]
}

func TestConvert_ImageSrcset_ThreeVariants(t *testing.T) {
	provider := stubVariants{
		"/uploads/photo.jpg": {
			{URL: "/uploads/photo-480w.webp", Width: 480},
			{URL: "/uploads/photo-800w.webp", Width: 800},
			{URL: "/uploads/photo-1200w.webp", Width: 1200},
		},
	}
	conv := newConverterWithVariantsForTest(provider)

	html, err := conv.Convert("![hero](/uploads/photo.jpg)")
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}

	// srcset must list every variant with its "Nw" descriptor.
	for _, want := range []string{
		"/uploads/photo-480w.webp 480w",
		"/uploads/photo-800w.webp 800w",
		"/uploads/photo-1200w.webp 1200w",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("srcset missing %q. html=%q", want, html)
		}
	}

	// src should point at the middle width (800w) so legacy clients get
	// a reasonable default — neither the absolute smallest nor largest.
	if !strings.Contains(html, `src="/uploads/photo-800w.webp"`) {
		t.Errorf("src should fall back to the 800w variant, html=%q", html)
	}

	// sizes attribute present, pinned to the article width.
	if !strings.Contains(html, `sizes="(max-width: 768px) 100vw, 672px"`) {
		t.Errorf(`sizes attribute missing or wrong. html=%q`, html)
	}
}

func TestConvert_ImageSrcset_OnlySmallVariantPicksItAsSrc(t *testing.T) {
	// A 600×400 upload only produces 480w. src must point at the only
	// variant that exists (we don't reference variants that aren't there).
	provider := stubVariants{
		"/uploads/small.jpg": {
			{URL: "/uploads/small-480w.webp", Width: 480},
		},
	}
	conv := newConverterWithVariantsForTest(provider)

	html, err := conv.Convert("![small](/uploads/small.jpg)")
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}

	if !strings.Contains(html, `src="/uploads/small-480w.webp"`) {
		t.Errorf("src must use the only available variant, html=%q", html)
	}
	if !strings.Contains(html, "/uploads/small-480w.webp 480w") {
		t.Errorf("srcset missing the only variant, html=%q", html)
	}
	// Must not reference variants that don't exist.
	for _, gone := range []string{"800w.webp", "1200w.webp"} {
		if strings.Contains(html, gone) {
			t.Errorf("html references nonexistent variant %q: %q", gone, html)
		}
	}
}

func TestConvert_ImageSrcset_NoVariantsFallsBackToOriginal(t *testing.T) {
	provider := stubVariants{} // empty
	conv := newConverterWithVariantsForTest(provider)

	html, err := conv.Convert("![hero](/uploads/photo.jpg)")
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}

	if !strings.Contains(html, `src="/uploads/photo.jpg"`) {
		t.Errorf("src must fall back to the original URL, html=%q", html)
	}
	if strings.Contains(html, "srcset") {
		t.Errorf("srcset must not appear when no variants exist, html=%q", html)
	}
	if strings.Contains(html, "sizes=") {
		t.Errorf("sizes must not appear when no variants exist, html=%q", html)
	}
}

func TestConvert_ImageSrcset_ExternalURLGetsNoSrcset(t *testing.T) {
	// External URLs are not under our control; we don't have variants
	// of remote images and shouldn't pretend to.
	provider := stubVariants{
		"/uploads/local.jpg": {{URL: "/uploads/local-800w.webp", Width: 800}},
	}
	conv := newConverterWithVariantsForTest(provider)

	html, err := conv.Convert("![ext](https://example.com/remote.jpg)")
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}
	if strings.Contains(html, "srcset") || strings.Contains(html, "sizes=") {
		t.Errorf("external URL must not get srcset/sizes, html=%q", html)
	}
}

func TestConvert_ImageSrcset_SurvivesSanitizer(t *testing.T) {
	// Regression guard: srcset/sizes must be in the allow-list, otherwise
	// the bluemonday policy would silently strip them.
	provider := stubVariants{
		"/uploads/photo.jpg": {
			{URL: "/uploads/photo-480w.webp", Width: 480},
			{URL: "/uploads/photo-800w.webp", Width: 800},
		},
	}
	conv := newConverterWithVariantsForTest(provider)

	html, err := conv.Convert("![p](/uploads/photo.jpg)")
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}
	for _, attr := range []string{"srcset=", "sizes="} {
		if !strings.Contains(html, attr) {
			t.Errorf("sanitizer stripped %s. html=%q", attr, html)
		}
	}
}

func newConverterWithVariantsForTest(provider VariantsProvider) Converter {
	return &converter{policy: createPolicy(), variants: provider}
}
