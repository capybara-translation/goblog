// Package image holds image-processing utilities used by the upload
// pipeline and the Markdown renderer: deriving filenames for WebP
// variants and generating those variants from an original upload.
package image

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

// variantSuffix matches the "-<digits>w.webp" tail that VariantPath
// appends, anchored to the end of the path. Used by IsVariantPath to
// recognize a derived image without parsing its on-disk header.
var variantSuffix = regexp.MustCompile(`-[0-9]+w\.webp$`)

// VariantPath returns the URL/path of a WebP variant resized to the
// given width. The original's extension (case-insensitively) is
// replaced; subdirectories are preserved.
//
//	VariantPath("/uploads/abc.jpg", 800) == "/uploads/abc-800w.webp"
//
// This function is the single source of truth: both the upload-time
// generator and the render-time existence check must derive variant
// paths from here so they cannot drift apart.
func VariantPath(original string, width int) string {
	dir, file := path.Split(original)
	ext := path.Ext(file)
	base := strings.TrimSuffix(file, ext)
	return fmt.Sprintf("%s%s-%dw.webp", dir, base, width)
}

// IsVariantPath reports whether the given path is a derived variant
// (i.e., ends with "-<digits>w.webp"). Used by the backfill CLI to
// skip already-derived files instead of generating a variant-of-a-variant.
//
// Note: the upload pipeline only produces UUID-v4 base names, so the
// regex doesn't bother enforcing a non-empty base part. A manually-placed
// file like "-800w.webp" would also match, but the CLI's only use of
// IsVariantPath is to *skip* such files — there is no path where a false
// positive leads to a destructive write. Callers should lower-case the
// path first if they care about case-insensitive matching (CLI does).
func IsVariantPath(p string) bool {
	return variantSuffix.MatchString(p)
}
