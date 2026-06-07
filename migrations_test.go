package goblog

import (
	"io/fs"
	"testing"
)

func TestMigrationFiles_AllEmbedded(t *testing.T) {
	if len(MigrationFiles) == 0 {
		t.Fatal("MigrationFiles is empty")
	}
	for _, p := range MigrationFiles {
		if _, err := fs.Stat(Migrations, p); err != nil {
			t.Errorf("migration %q not embedded: %v", p, err)
		}
	}
}
