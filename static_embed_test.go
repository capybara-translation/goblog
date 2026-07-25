package goblog

import (
	"io/fs"
	"testing"
)

// TestStaticEmbedsReactionsJS ensures the embedded StaticFiles FS includes the
// reactions.js asset (a file in a subdirectory of internal/view/static), which
// the //go:embed directive must pick up for it to be served at /static/js/.
func TestStaticEmbedsReactionsJS(t *testing.T) {
	if _, err := fs.Stat(StaticFiles, "internal/view/static/js/reactions.js"); err != nil {
		t.Fatalf("reactions.js is not embedded in StaticFiles: %v", err)
	}
}

// TestStaticEmbedsHealthChartJS is the same check for healthchart.js (the
// /health page's tap-to-show progressive enhancement, referenced by
// health.html as /static/js/healthchart.js).
func TestStaticEmbedsHealthChartJS(t *testing.T) {
	if _, err := fs.Stat(StaticFiles, "internal/view/static/js/healthchart.js"); err != nil {
		t.Fatalf("healthchart.js is not embedded in StaticFiles: %v", err)
	}
}
