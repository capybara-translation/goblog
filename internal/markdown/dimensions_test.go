package markdown

import (
	"strings"
	"testing"
)

// stubDimensions is a fake DimensionsProvider used to drive the
// image-renderer's width/height output without touching disk.
type stubDimensions map[string]struct{ w, h int }

func (s stubDimensions) Get(url string) (int, int, bool) {
	d, ok := s[url]
	if !ok {
		return 0, 0, false
	}
	return d.w, d.h, true
}

func TestConvert_ImageWidthHeight_FromProvider(t *testing.T) {
	provider := stubDimensions{
		"/uploads/known.jpg": {w: 1600, h: 900},
	}
	conv := newConverterWithDimensionsForTest(provider)

	html, err := conv.Convert("![hero](/uploads/known.jpg)")
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}

	mustContain(t, html, `src="/uploads/known.jpg"`)
	mustContain(t, html, `width="1600"`)
	mustContain(t, html, `height="900"`)
}

func TestConvert_ImageWidthHeight_OmittedWhenProviderHasNoEntry(t *testing.T) {
	provider := stubDimensions{} // empty
	conv := newConverterWithDimensionsForTest(provider)

	html, err := conv.Convert("![hero](/uploads/unknown.jpg)")
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}

	if strings.Contains(html, `width=`) || strings.Contains(html, `height=`) {
		t.Errorf("expected no width/height when provider returns ok=false, got: %q", html)
	}
}

func TestConvert_ImageWidthHeight_NilProviderBehavesLikeAbsent(t *testing.T) {
	// Backwards compatibility: a converter built without a provider must
	// still emit the existing attributes and never crash on Convert.
	conv := NewConverterForTest()

	html, err := conv.Convert("![hero](/uploads/whatever.jpg)")
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}

	mustContain(t, html, `decoding="async"`)
	if strings.Contains(html, `width=`) || strings.Contains(html, `height=`) {
		t.Errorf("expected no width/height with nil provider, got: %q", html)
	}
}

func TestConvert_ImageWidthHeight_ExternalURLIgnored(t *testing.T) {
	// Provider lookup is keyed on the raw destination URL. External URLs
	// won't be in the cache (because the disk-backed implementation will
	// gate on /uploads/ prefix), so they should simply render without
	// width/height — exactly as if the entry were missing.
	provider := stubDimensions{
		"/uploads/local.jpg": {w: 100, h: 50},
	}
	conv := newConverterWithDimensionsForTest(provider)

	html, err := conv.Convert("![ext](https://example.com/remote.jpg)")
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}
	if strings.Contains(html, `width=`) || strings.Contains(html, `height=`) {
		t.Errorf("external URL must not carry width/height, got: %q", html)
	}
}

// newConverterWithDimensionsForTest mirrors NewConverterForTest but threads
// a DimensionsProvider through the converter. Like NewConverterForTest it
// must NOT use the package-level singleton.
func newConverterWithDimensionsForTest(provider DimensionsProvider) Converter {
	return &converter{policy: createPolicy(), dimensions: provider}
}
