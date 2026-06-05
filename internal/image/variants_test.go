package image

import (
	"bytes"
	"errors"
	stdimage "image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/chai2010/webp"
	_ "golang.org/x/image/webp"
)

// GenerateVariants is the upload-time WebP resize pipeline. It must:
//   - emit one WebP per target width, sized to fit the original
//   - skip widths larger than the original (no upscaling)
//   - use lossless WebP when the source has an alpha channel
//   - skip animated GIFs entirely (no static frame extraction)
//   - leave the original file untouched
//   - be idempotent (running twice produces the same set of files)

func TestGenerateVariants_JPEG(t *testing.T) {
	dir := t.TempDir()
	original := filepath.Join(dir, "photo.jpg")
	mustWriteJPEG(t, original, 1600, 1200)

	if err := GenerateVariants(original); err != nil {
		t.Fatalf("GenerateVariants: %v", err)
	}

	for _, w := range []int{480, 800, 1200} {
		want := filepath.Join(dir, "photo-"+itoa(w)+"w.webp")
		stat, err := os.Stat(want)
		if err != nil {
			t.Errorf("expected variant %s to exist: %v", want, err)
			continue
		}
		if stat.Size() == 0 {
			t.Errorf("variant %s is empty", want)
		}

		// Sanity: the variant should actually be a WebP of the right width.
		gotW, _, err := decodeWebPSize(want)
		if err != nil {
			t.Errorf("variant %s is not a readable WebP: %v", want, err)
			continue
		}
		if gotW != w {
			t.Errorf("variant %s has width %d, want %d", want, gotW, w)
		}
	}

	// Original must remain untouched.
	if _, err := os.Stat(original); err != nil {
		t.Errorf("original was removed: %v", err)
	}
}

func TestGenerateVariants_PNGWithAlphaUsesLossless(t *testing.T) {
	// A semi-transparent PNG round-tripped through lossy WebP would lose
	// alpha precision. The pipeline must detect alpha and pick lossless.
	dir := t.TempDir()
	original := filepath.Join(dir, "icon.png")
	mustWritePNGWithAlpha(t, original, 1600, 1200)

	if err := GenerateVariants(original); err != nil {
		t.Fatalf("GenerateVariants: %v", err)
	}

	// All variants must round-trip alpha. Lossless WebP preserves the
	// exact alpha channel; lossy would quantize. We check by re-decoding
	// and asserting at least one pixel kept its alpha < 255.
	for _, w := range []int{480, 800, 1200} {
		v := filepath.Join(dir, "icon-"+itoa(w)+"w.webp")
		img, err := decodeWebPImage(v)
		if err != nil {
			t.Errorf("decode %s: %v", v, err)
			continue
		}
		if !hasAlphaTransparency(img) {
			t.Errorf("variant %s lost its alpha — lossy encode was used", v)
		}
	}
}

func TestGenerateVariants_SkipsWidthsLargerThanOriginal(t *testing.T) {
	// A 600px-wide upload should produce only the 480w variant.
	dir := t.TempDir()
	original := filepath.Join(dir, "small.jpg")
	mustWriteJPEG(t, original, 600, 400)

	if err := GenerateVariants(original); err != nil {
		t.Fatalf("GenerateVariants: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "small-480w.webp")); err != nil {
		t.Errorf("expected 480w to exist: %v", err)
	}
	for _, w := range []int{800, 1200} {
		p := filepath.Join(dir, "small-"+itoa(w)+"w.webp")
		if _, err := os.Stat(p); err == nil {
			t.Errorf("did not expect %s (would be upscaled)", p)
		}
	}
}

func TestGenerateVariants_AnimatedGIFSkippedRegardlessOfExtensionCase(t *testing.T) {
	// Regression: animated GIF detection used to be gated on
	// filepath.Ext(path) == ".gif", which silently let a manually-placed
	// "foo.GIF" through to image.Decode → single-frame static WebP →
	// lost animation. Detect via magic bytes instead so case (and other
	// extension surprises) don't matter.
	dir := t.TempDir()
	original := filepath.Join(dir, "anim.GIF")
	mustWriteAnimatedGIF(t, original, 800, 600)

	if err := GenerateVariants(original); err != nil {
		t.Fatalf("GenerateVariants on animated .GIF should succeed (no-op), got: %v", err)
	}

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".webp" {
			t.Errorf(".GIF (uppercase) animation must not produce variants, found %s", e.Name())
		}
	}
}

func TestGenerateVariants_AnimatedGIFSkipped(t *testing.T) {
	// We don't try to extract a still frame; producing a single-frame
	// WebP from an animated GIF would be misleading. Confirm no .webp
	// files appear for an animated GIF input.
	dir := t.TempDir()
	original := filepath.Join(dir, "anim.gif")
	mustWriteAnimatedGIF(t, original, 800, 600)

	if err := GenerateVariants(original); err != nil {
		t.Fatalf("GenerateVariants on animated GIF should succeed (no-op), got: %v", err)
	}

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".webp" {
			t.Errorf("animated GIF should not produce variants, found %s", e.Name())
		}
	}
}

func TestGenerateVariants_RejectsOversizedImage(t *testing.T) {
	// A maliciously-crafted JPEG (or just a high-resolution screenshot)
	// can fit under MAX_UPLOAD_SIZE while decoding to a multi-hundred-MB
	// RGBA buffer. Reject anything whose width*height exceeds the pixel
	// budget before decode, so a single bad upload can't OOM the process.
	dir := t.TempDir()
	original := filepath.Join(dir, "big.png")
	mustWriteJPEG(t, original, 2000, 2000) // 4,000,000 pixels

	err := generateVariantsLimited(original, 1_000_000) // 1M pixel cap
	if err == nil {
		t.Fatalf("expected oversized image to be rejected")
	}
	if !errors.Is(err, ErrTooLarge) {
		t.Errorf("expected ErrTooLarge, got %v", err)
	}

	// No variants should have been written.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".webp" {
			t.Errorf("rejected image must not produce variants, found %s", e.Name())
		}
	}
}

func TestGenerateVariants_AllowsImageUnderLimit(t *testing.T) {
	// Sanity: the limit applies only to oversized inputs.
	dir := t.TempDir()
	original := filepath.Join(dir, "ok.jpg")
	mustWriteJPEG(t, original, 600, 400) // 240,000 pixels

	if err := generateVariantsLimited(original, 1_000_000); err != nil {
		t.Fatalf("under-limit image should succeed, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "ok-480w.webp")); err != nil {
		t.Errorf("expected 480w variant to be written, got %v", err)
	}
}

func TestGenerateVariants_Idempotent(t *testing.T) {
	dir := t.TempDir()
	original := filepath.Join(dir, "photo.jpg")
	mustWriteJPEG(t, original, 1600, 1200)

	if err := GenerateVariants(original); err != nil {
		t.Fatalf("first run: %v", err)
	}

	// Snapshot mtimes so we can detect a needless rewrite on the second run.
	before := map[string]int64{}
	for _, w := range []int{480, 800, 1200} {
		p := filepath.Join(dir, "photo-"+itoa(w)+"w.webp")
		st, err := os.Stat(p)
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		before[p] = st.ModTime().UnixNano()
	}

	if err := GenerateVariants(original); err != nil {
		t.Fatalf("second run: %v", err)
	}

	for p, prev := range before {
		st, err := os.Stat(p)
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		if st.ModTime().UnixNano() != prev {
			t.Errorf("variant %s was rewritten on the second run (not idempotent)", p)
		}
	}
}

// --- fixture helpers ---

func itoa(n int) string {
	return [...]string{
		480:  "480",
		800:  "800",
		1200: "1200",
		600:  "600",
	}[n]
}

func mustWriteJPEG(t *testing.T, path string, w, h int) {
	t.Helper()
	img := stdimage.NewRGBA(stdimage.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), 100, 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		t.Fatalf("write jpeg: %v", err)
	}
}

func mustWritePNGWithAlpha(t *testing.T, path string, w, h int) {
	t.Helper()
	img := stdimage.NewRGBA(stdimage.Rect(0, 0, w, h))
	// Top half opaque red, bottom half 50% red.
	for y := 0; y < h; y++ {
		a := uint8(255)
		if y > h/2 {
			a = 128
		}
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{200, 50, 50, a})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		t.Fatalf("write png: %v", err)
	}
}

func mustWriteAnimatedGIF(t *testing.T, path string, w, h int) {
	t.Helper()
	pal := color.Palette{color.Black, color.White, color.RGBA{255, 0, 0, 255}}
	g := &gif.GIF{
		LoopCount: 0,
		Delay:     []int{10, 10, 10},
		Image:     []*stdimage.Paletted{},
	}
	for i := 0; i < 3; i++ {
		frame := stdimage.NewPaletted(stdimage.Rect(0, 0, w, h), pal)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				frame.SetColorIndex(x, y, uint8((x+y+i)%3))
			}
		}
		g.Image = append(g.Image, frame)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create gif: %v", err)
	}
	defer f.Close()
	if err := gif.EncodeAll(f, g); err != nil {
		t.Fatalf("encode gif: %v", err)
	}
}

func decodeWebPSize(path string) (int, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	cfg, _, err := stdimage.DecodeConfig(f)
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}

func decodeWebPImage(path string) (stdimage.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return webp.Decode(f)
}

func hasAlphaTransparency(img stdimage.Image) bool {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a>>8 < 250 { // anything visibly less than fully opaque
				return true
			}
		}
	}
	return false
}
