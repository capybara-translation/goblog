package image

import (
	"bytes"
	"errors"
	"fmt"
	stdimage "image"
	"image/gif"
	_ "image/jpeg" // register JPEG decoder
	_ "image/png"  // register PNG decoder
	"os"
	"path/filepath"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
	_ "golang.org/x/image/webp" // register WebP decoder
)

// VariantWidths is the canonical set of widths we resize uploads to.
// Kept short on purpose: each extra width is another goroutine of CPU
// at upload time and another disk file per upload.
var VariantWidths = []int{480, 800, 1200}

const webpQuality = 80

// MaxPixels caps the input resolution accepted by GenerateVariants.
// 50 megapixels covers any phone-camera output anyone realistically
// uploads (~24 MP for current iPhones); anything beyond it is either
// a zip-bomb-style image (low compression ratio hiding a vast canvas
// inside the 5 MB upload limit) or a misconfiguration. Reject before
// stdimage.Decode allocates the RGBA buffer (width*height*4 bytes).
const MaxPixels = 50_000_000

// ErrTooLarge is returned when the original image's declared dimensions
// exceed MaxPixels. Callers can recover by skipping variant generation
// for that file (the upload itself is still saved).
var ErrTooLarge = errors.New("image exceeds pixel budget")

// GenerateVariants resizes the image at originalPath into WebP variants
// at every width in VariantWidths that the original can downsize to.
// Variants larger than the original are skipped (no upscaling); existing
// variant files are left untouched (the operation is idempotent).
// Animated GIFs are skipped entirely — they need different handling than
// a single-frame resize and producing a still WebP would silently lose
// information.
//
// Failures generating any one width are returned; the caller (the upload
// goroutine or the backfill CLI) decides whether to log-and-continue.
// The original file is never modified or removed.
func GenerateVariants(originalPath string) error {
	return generateVariantsLimited(originalPath, MaxPixels)
}

// generateVariantsLimited is the implementation behind GenerateVariants;
// the maxPixels seam lets tests pin the budget without writing
// 50-megapixel fixtures.
func generateVariantsLimited(originalPath string, maxPixels int) error {
	data, err := os.ReadFile(originalPath)
	if err != nil {
		return fmt.Errorf("read original: %w", err)
	}

	// Skip animated GIFs: gif.DecodeAll succeeds on still GIFs too, so
	// we use frame count to disambiguate. Detect GIFs from magic bytes
	// rather than the path extension so a manually-placed "foo.GIF" (or
	// a file served by a misconfigured upstream) still trips the
	// animation check; otherwise the renderer would silently flatten
	// the animation into a single-frame WebP.
	if isGIFBytes(data) && isAnimatedGIFBytes(data) {
		return nil
	}

	// Gate on declared resolution before the full Decode allocates a
	// width*height*4 RGBA buffer. DecodeConfig reads only the header,
	// so the cost is microseconds. The multiplication is widened to
	// int64 so a malicious header with extremely large dimensions
	// cannot wrap to a small/negative int and bypass the budget on
	// 32-bit builds.
	cfg, _, err := stdimage.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("decode config: %w", err)
	}
	if cfg.Width > 0 && cfg.Height > 0 && int64(cfg.Width)*int64(cfg.Height) > int64(maxPixels) {
		return fmt.Errorf("%w: %dx%d exceeds %d-pixel budget", ErrTooLarge, cfg.Width, cfg.Height, maxPixels)
	}

	img, _, err := stdimage.Decode(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("decode original: %w", err)
	}

	origW := img.Bounds().Dx()
	lossless := imageHasTransparency(img)

	dir, file := filepath.Split(originalPath)
	ext := filepath.Ext(file)
	base := file[:len(file)-len(ext)]

	for _, w := range VariantWidths {
		if w > origW {
			// Don't upscale; a 600px upload doesn't get a 1200w variant.
			continue
		}

		outPath := filepath.Join(dir, fmt.Sprintf("%s-%dw.webp", base, w))
		if _, err := os.Stat(outPath); err == nil {
			// Idempotent: skip if the variant already exists.
			continue
		}

		resized := imaging.Resize(img, w, 0, imaging.Lanczos)

		opts := &webp.Options{Quality: webpQuality}
		if lossless {
			opts.Lossless = true
		}

		var buf bytes.Buffer
		if err := webp.Encode(&buf, resized, opts); err != nil {
			return fmt.Errorf("encode webp w=%d: %w", w, err)
		}
		// Write to a temp file in the same directory and atomically
		// rename into place. POSIX guarantees rename atomicity within a
		// single filesystem, so SIGKILL / ENOSPC / OOM mid-write leaves
		// either the old state (no file) or the new state (complete
		// file) — never a partial WebP that os.Stat would treat as
		// "already done" on the next run.
		tmp := outPath + ".tmp"
		if err := os.WriteFile(tmp, buf.Bytes(), 0644); err != nil {
			return fmt.Errorf("write %s: %w", tmp, err)
		}
		if err := os.Rename(tmp, outPath); err != nil {
			_ = os.Remove(tmp)
			return fmt.Errorf("rename %s -> %s: %w", tmp, outPath, err)
		}
	}
	return nil
}

// isGIFBytes recognizes a GIF87a or GIF89a header. Extension-agnostic
// so renames or uppercase paths don't change the answer.
func isGIFBytes(data []byte) bool {
	if len(data) < 6 {
		return false
	}
	return bytes.Equal(data[:6], []byte("GIF87a")) || bytes.Equal(data[:6], []byte("GIF89a"))
}

// isAnimatedGIFBytes reports whether the bytes encode a multi-frame GIF.
func isAnimatedGIFBytes(data []byte) bool {
	g, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		return false
	}
	return len(g.Image) > 1
}

// imageHasTransparency reports whether the image carries any pixels
// with alpha < fully-opaque. We pick the answer up-front from the
// concrete type when we can: JPEG decodes to YCbCr / NYCbCrA without
// alpha, Gray has none by definition. That short-circuits a 12 MP+
// scan in the common phone-camera path. For RGBA-shaped types we still
// have to look at the pixels, but the walk exits at the first
// transparent pixel.
func imageHasTransparency(img stdimage.Image) bool {
	switch img.(type) {
	case *stdimage.YCbCr, *stdimage.Gray, *stdimage.Gray16, *stdimage.CMYK:
		return false
	}
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a < 0xffff {
				return true
			}
		}
	}
	return false
}
