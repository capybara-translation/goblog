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
