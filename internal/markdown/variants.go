package markdown

// Variant describes one resized WebP derivative of an original upload.
// URL is the path that the public router serves; Width is the WebP's
// intrinsic pixel width, used as the descriptor in the srcset attribute.
type Variant struct {
	URL   string
	Width int
}

// VariantsProvider returns the available WebP variants of an image
// referenced from a Markdown document. Implementations should be fast
// and safe to call from the request-rendering path. The empty slice
// means "no variants available" — the renderer treats it as a signal
// to fall back to the original src without a srcset.
//
// url is the raw Markdown destination (typically "/uploads/<uuid>.<ext>").
// Implementations may return only the variants that physically exist;
// the renderer does not assume any particular width is present.
//
// Ordering contract: returned variants MUST be sorted by ascending
// Width. The renderer relies on this to pick the middle entry as the
// legacy `src` fallback (so srcset-unaware clients fetch a sensibly
// sized image), and to keep the emitted srcset attribute in canonical
// "small-to-large" order. Implementations that aggregate from multiple
// sources should sort before returning.
type VariantsProvider interface {
	Variants(url string) []Variant
}
