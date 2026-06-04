package markdown

import (
	"strings"
	"testing"
)

// The image extension annotates Markdown images with browser-hint attributes
// so the public pages don't ship raw <img> tags. The contract is:
//   * The first <img> in a document does NOT carry loading="lazy" — it is a
//     probable LCP candidate and must not be deferred.
//   * Subsequent <img>s carry loading="lazy" so off-viewport images defer.
//   * Every <img> carries decoding="async".
//   * alt and src must continue to render.
//   * Attributes must survive the bluemonday sanitizer pass.
//
// fetchpriority="high" is deliberately not emitted (see image_extension.go
// for rationale: the home page renders multiple posts inline, so tagging
// each post's first image high would be an anti-pattern).
//
// These tests use NewConverterForTest (which goes through the sanitizer) so
// that a regression where decoding is silently stripped by the policy is
// caught here, not only in renderer-level unit tests.

func TestConvert_ImageAttributes_FirstImageIsNotLazy(t *testing.T) {
	conv := NewConverterForTest()

	html, err := conv.Convert("![sunset](/uploads/abc.jpg)")
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}

	mustContain(t, html, `src="/uploads/abc.jpg"`)
	mustContain(t, html, `alt="sunset"`)
	mustContain(t, html, `decoding="async"`)
	mustNotContain(t, html, `loading="lazy"`)        // first image is NOT lazy
	mustNotContain(t, html, `fetchpriority="high"`) // intentionally not emitted
}

func TestConvert_ImageAttributes_SubsequentImagesAreLazy(t *testing.T) {
	conv := NewConverterForTest()

	md := "![a](/uploads/a.jpg)\n\n![b](/uploads/b.jpg)\n\n![c](/uploads/c.jpg)"
	html, err := conv.Convert(md)
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}

	// Locate each <img> tag in order.
	imgs := extractImgTags(html)
	if len(imgs) != 3 {
		t.Fatalf("expected 3 img tags, got %d. html=%q", len(imgs), html)
	}

	// First image: NO loading=lazy.
	if strings.Contains(imgs[0], `loading="lazy"`) {
		t.Errorf("first img must not be lazy: %q", imgs[0])
	}

	// Second and third: loading=lazy.
	for i, img := range imgs[1:] {
		if !strings.Contains(img, `loading="lazy"`) {
			t.Errorf("img[%d] missing loading=lazy: %q", i+1, img)
		}
	}

	// All images: decoding=async; none with fetchpriority.
	for i, img := range imgs {
		if !strings.Contains(img, `decoding="async"`) {
			t.Errorf("img[%d] missing decoding=async: %q", i, img)
		}
		if strings.Contains(img, `fetchpriority`) {
			t.Errorf("img[%d] should not emit fetchpriority: %q", i, img)
		}
	}
}

func TestConvert_ImageAttributes_StatePerCallNotShared(t *testing.T) {
	// Regression guard: each Convert() invocation must start fresh.
	// If "first image" state leaked across calls, the first image of
	// the second document would incorrectly be marked lazy.
	conv := NewConverterForTest()

	_, _ = conv.Convert("![a](/uploads/a.jpg)\n\n![b](/uploads/b.jpg)")

	html, err := conv.Convert("![c](/uploads/c.jpg)")
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}
	mustNotContain(t, html, `loading="lazy"`)
}

func TestConvert_ImageAttributes_SurviveSanitizer(t *testing.T) {
	// Explicit guard against bluemonday silently dropping the new attributes.
	conv := NewConverterForTest()

	html, err := conv.Convert("![first](/uploads/a.jpg)\n\n![second](/uploads/b.jpg)")
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}

	for _, attr := range []string{`decoding="async"`, `loading="lazy"`} {
		if !strings.Contains(html, attr) {
			t.Errorf("sanitizer stripped %s. html=%q", attr, html)
		}
	}
}

// --- helpers ---

// NewConverterForTest returns a non-singleton converter (without OGP getter).
// The package-level NewConverter() uses sync.Once and would cache state
// across tests; for tests of stateful renderers we need a fresh instance.
func NewConverterForTest() Converter {
	return &converter{policy: createPolicy()}
}

// extractImgTags pulls each <img ...> open tag out of the rendered HTML.
// It assumes goldmark/bluemonday escape '>' inside attribute values
// (verified at the time of writing); if a future test asserts on a raw
// '>' inside an attribute value, switch this to a real HTML parser.
func extractImgTags(html string) []string {
	var out []string
	rest := html
	for {
		i := strings.Index(rest, "<img")
		if i < 0 {
			return out
		}
		j := strings.Index(rest[i:], ">")
		if j < 0 {
			return out
		}
		out = append(out, rest[i:i+j+1])
		rest = rest[i+j+1:]
	}
}

func mustContain(t *testing.T, html, want string) {
	t.Helper()
	if !strings.Contains(html, want) {
		t.Errorf("expected output to contain %q; got %q", want, html)
	}
}

func mustNotContain(t *testing.T, html, notWant string) {
	t.Helper()
	if strings.Contains(html, notWant) {
		t.Errorf("expected output NOT to contain %q; got %q", notWant, html)
	}
}
