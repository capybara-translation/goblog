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
// The command exits 0 on partial success and reports per-file errors
// to stderr; one bad image does not stop the rest.
package main

import (
	"flag"
	"fmt"
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

	entries, err := os.ReadDir(*dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read dir %s: %v\n", *dir, err)
		os.Exit(1)
	}

	var processed, skipped, failed int
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		// Normalize the extension to lowercase so case-only variations
		// of the suffix (.JPG, .Webp, …) feed into the same predicates
		// as the canonical .jpg/.webp output of the upload pipeline.
		ext := strings.ToLower(filepath.Ext(name))
		// Only process supported originals.
		switch ext {
		case ".jpg", ".jpeg", ".png", ".webp", ".gif":
		default:
			continue
		}
		// Skip derived variants (those produced by VariantPath). Match on
		// the lowercased name so e.g. "foo-800w.WEBP" is also skipped.
		if image.IsVariantPath(strings.ToLower(name)) {
			skipped++
			continue
		}

		full := filepath.Join(*dir, name)
		if *dryRun {
			fmt.Printf("would process %s\n", full)
			processed++
			continue
		}
		if err := image.GenerateVariants(full); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL %s: %v\n", name, err)
			failed++
			continue
		}
		processed++
	}

	fmt.Printf("done: %d processed, %d skipped (variants), %d failed\n", processed, skipped, failed)
	// Surface partial-failure as a non-zero exit so CI / cron can detect
	// the regression rather than relying on a human reading the summary.
	if failed > 0 {
		os.Exit(1)
	}
}
