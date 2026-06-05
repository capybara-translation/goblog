// regenerate-variants walks UPLOAD_DIR and runs the same WebP variant
// pipeline that handlers_image.go runs on each upload, but for every
// original image already on disk. Use it once after deploying Tier 3
// to backfill variants for images that were uploaded before the
// pipeline existed; running it again is a no-op for images whose
// variants are already present (GenerateVariants is idempotent).
//
// Usage:
//
//	UPLOAD_DIR=data/uploads go run cmd/regenerate-variants/main.go
//
// Per-file errors are reported to stderr; one bad image does not stop
// the rest. The command exits 0 only if every file processed cleanly,
// and exits non-zero (1) on any partial failure so CI / cron can
// detect the regression without parsing stdout.
package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/capybara-translation/goblog/internal/image"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	defaultDir := os.Getenv("UPLOAD_DIR")
	if defaultDir == "" {
		defaultDir = "data/uploads"
	}

	dir := flag.String("dir", defaultDir, "upload directory to scan (defaults to $UPLOAD_DIR or data/uploads)")
	dryRun := flag.Bool("dry-run", false, "list what would be processed without writing variants")
	flag.Parse()

	var processed, skipped, failed int

	// WalkDir so files in subdirectories (e.g. /uploads/ogp-cache/<uuid>)
	// also get backfilled — DiskVariantsService and VariantPath both
	// support subdirectories, and treating only immediate children
	// special would leave the OGP cache forever without variants.
	walkErr := filepath.WalkDir(*dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "walk %s: %v\n", path, err)
			return nil // continue with siblings
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		// Normalize the extension to lowercase so case-only variations
		// of the suffix (.JPG, .Webp, …) feed into the same predicates
		// as the canonical .jpg/.webp output of the upload pipeline.
		ext := strings.ToLower(filepath.Ext(name))
		switch ext {
		case ".jpg", ".jpeg", ".png", ".webp", ".gif":
		default:
			return nil
		}
		// Skip derived variants (those produced by VariantPath). Match on
		// the lowercased name so e.g. "foo-800w.WEBP" is also skipped.
		if image.IsVariantPath(strings.ToLower(name)) {
			skipped++
			return nil
		}

		if *dryRun {
			fmt.Printf("would process %s\n", path)
			processed++
			return nil
		}
		if err := image.GenerateVariants(path); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL %s: %v\n", path, err)
			failed++
			return nil
		}
		processed++
		return nil
	})
	if walkErr != nil {
		fmt.Fprintf(os.Stderr, "walk %s: %v\n", *dir, walkErr)
		os.Exit(1)
	}

	fmt.Printf("done: %d processed, %d skipped (variants), %d failed\n", processed, skipped, failed)
	// Surface partial-failure as a non-zero exit so CI / cron can detect
	// the regression rather than relying on a human reading the summary.
	if failed > 0 {
		os.Exit(1)
	}
}
