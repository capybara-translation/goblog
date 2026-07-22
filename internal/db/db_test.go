package db

import (
	"path/filepath"
	"testing"
)

// TestOpen_EnablesForeignKeys verifies that connections opened via Open have
// SQLite foreign-key enforcement turned on. Without it, ON DELETE CASCADE
// clauses in the migrations are silently ignored.
func TestOpen_EnablesForeignKeys(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "fk_test.db")
	database, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer database.Close()

	var fk int
	if err := database.Get(&fk, "PRAGMA foreign_keys"); err != nil {
		t.Fatalf("PRAGMA foreign_keys: %v", err)
	}
	if fk != 1 {
		t.Errorf("PRAGMA foreign_keys = %d, want 1 (enforcement enabled)", fk)
	}
}

// TestOpen_CascadeDeleteWorks is an end-to-end check that ON DELETE CASCADE
// actually fires through Open's connections: deleting a parent row removes the
// child rows that reference it.
func TestOpen_CascadeDeleteWorks(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "cascade_test.db")
	database, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer database.Close()

	schema := `
	CREATE TABLE parent (id INTEGER PRIMARY KEY);
	CREATE TABLE child (
		id INTEGER PRIMARY KEY,
		parent_id INTEGER NOT NULL,
		FOREIGN KEY (parent_id) REFERENCES parent(id) ON DELETE CASCADE
	);
	INSERT INTO parent (id) VALUES (1);
	INSERT INTO child (id, parent_id) VALUES (10, 1);
	`
	if _, err := database.Exec(schema); err != nil {
		t.Fatalf("schema: %v", err)
	}

	if _, err := database.Exec(`DELETE FROM parent WHERE id = 1`); err != nil {
		t.Fatalf("delete parent: %v", err)
	}

	var childCount int
	if err := database.Get(&childCount, `SELECT COUNT(*) FROM child WHERE parent_id = 1`); err != nil {
		t.Fatalf("count children: %v", err)
	}
	if childCount != 0 {
		t.Errorf("child rows after parent delete = %d, want 0 (cascade did not fire)", childCount)
	}
}

// TestOpen_SetsBusyTimeout verifies connections wait (rather than failing
// immediately with SQLITE_BUSY) when another process holds the write lock.
// The driver defaults to 5000ms so this test guards the explicit DSN pin
// (it cannot distinguish DSN from driver default). hpsync writes to the same
// DB file as the running server, so this matters.
func TestOpen_SetsBusyTimeout(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "busy_test.db")
	database, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer database.Close()

	var timeout int
	if err := database.Get(&timeout, "PRAGMA busy_timeout"); err != nil {
		t.Fatalf("PRAGMA busy_timeout: %v", err)
	}
	if timeout != 5000 {
		t.Errorf("busy_timeout = %d, want 5000", timeout)
	}
}
