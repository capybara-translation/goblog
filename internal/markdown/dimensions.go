package markdown

// DimensionsProvider returns the intrinsic pixel dimensions of an image
// referenced from a Markdown document. Implementations should be fast
// and safe to call from the request-rendering path (e.g., backed by an
// in-process cache so disk I/O happens at most once per URL).
//
// url is the raw href as it appears in the Markdown source (typically
// "/uploads/<uuid>.<ext>"). Implementations decide which URL schemes
// they can resolve; anything they cannot resolve must return ok=false
// rather than guessing.
type DimensionsProvider interface {
	Get(url string) (width, height int, ok bool)
}
