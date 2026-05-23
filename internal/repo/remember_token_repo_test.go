package repo

import (
	"os"
	"testing"
	"time"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// setupTestDBWithRememberTokens opens an in-memory SQLite database with the
// users + remember_tokens schemas applied (via the real migration files) and
// seeds one admin user so foreign keys validate.
func setupTestDBWithRememberTokens(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	migrations := []string{
		"../../migrations/002_create_users.sql",
		"../../migrations/007_create_remember_tokens.sql",
	}
	for _, path := range migrations {
		schema, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read migration file %s: %v", path, err)
		}
		if _, err := db.Exec(string(schema)); err != nil {
			t.Fatalf("failed to execute migration %s: %v", path, err)
		}
	}

	if _, err := db.Exec(`INSERT INTO users (id, username, password_hash) VALUES (1, 'admin', 'x')`); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	return db
}

func TestSQLiteRememberTokenStore_CreateAndFind(t *testing.T) {
	db := setupTestDBWithRememberTokens(t)
	store := NewSQLiteRememberTokenStore(db)

	token := &domain.RememberToken{
		UserID:    1,
		Selector:  "sel-a",
		TokenHash: "hash-a",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := store.Create(token); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := store.FindBySelector("sel-a")
	if err != nil {
		t.Fatalf("FindBySelector: %v", err)
	}
	if got == nil {
		t.Fatal("expected row, got nil")
	}
	if got.UserID != 1 || got.Selector != "sel-a" || got.TokenHash != "hash-a" {
		t.Errorf("unexpected row: %+v", got)
	}
}

func TestSQLiteRememberTokenStore_FindBySelector_NotFound(t *testing.T) {
	db := setupTestDBWithRememberTokens(t)
	store := NewSQLiteRememberTokenStore(db)

	got, err := store.FindBySelector("missing")
	if err != nil {
		t.Errorf("expected nil error for missing selector, got %v", err)
	}
	if got != nil {
		t.Errorf("expected nil row for missing selector, got %+v", got)
	}
}

func TestSQLiteRememberTokenStore_Delete(t *testing.T) {
	db := setupTestDBWithRememberTokens(t)
	store := NewSQLiteRememberTokenStore(db)
	_ = store.Create(&domain.RememberToken{
		UserID: 1, Selector: "sel-b", TokenHash: "h", ExpiresAt: time.Now().Add(time.Hour),
	})

	if err := store.Delete("sel-b"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	got, _ := store.FindBySelector("sel-b")
	if got != nil {
		t.Errorf("expected deletion, row still present: %+v", got)
	}
}

func TestSQLiteRememberTokenStore_DeleteByUserID(t *testing.T) {
	db := setupTestDBWithRememberTokens(t)
	store := NewSQLiteRememberTokenStore(db)
	for _, sel := range []string{"a", "b", "c"} {
		_ = store.Create(&domain.RememberToken{
			UserID: 1, Selector: sel, TokenHash: "h", ExpiresAt: time.Now().Add(time.Hour),
		})
	}

	if err := store.DeleteByUserID(1); err != nil {
		t.Fatalf("DeleteByUserID: %v", err)
	}
	for _, sel := range []string{"a", "b", "c"} {
		if got, _ := store.FindBySelector(sel); got != nil {
			t.Errorf("selector %q still present", sel)
		}
	}
}

func TestSQLiteRememberTokenStore_RefreshOnUse(t *testing.T) {
	db := setupTestDBWithRememberTokens(t)
	store := NewSQLiteRememberTokenStore(db)
	originalExpiry := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	_ = store.Create(&domain.RememberToken{
		UserID: 1, Selector: "sel-c", TokenHash: "h", ExpiresAt: originalExpiry,
	})

	now := time.Now().UTC().Truncate(time.Second)
	extendedExpiry := now.Add(48 * time.Hour)
	if err := store.RefreshOnUse("sel-c", now, extendedExpiry); err != nil {
		t.Fatalf("RefreshOnUse: %v", err)
	}
	got, _ := store.FindBySelector("sel-c")
	if got.LastUsedAt == nil {
		t.Fatal("LastUsedAt not set")
	}
	if !got.LastUsedAt.UTC().Truncate(time.Second).Equal(now) {
		t.Errorf("LastUsedAt = %v, want %v", got.LastUsedAt, now)
	}
	if !got.ExpiresAt.UTC().Truncate(time.Second).Equal(extendedExpiry) {
		t.Errorf("ExpiresAt = %v, want %v (sliding expiration not applied)", got.ExpiresAt, extendedExpiry)
	}
}

func TestSQLiteRememberTokenStore_CleanupExpired(t *testing.T) {
	db := setupTestDBWithRememberTokens(t)
	store := NewSQLiteRememberTokenStore(db)
	_ = store.Create(&domain.RememberToken{
		UserID: 1, Selector: "alive", TokenHash: "h", ExpiresAt: time.Now().Add(time.Hour),
	})
	_ = store.Create(&domain.RememberToken{
		UserID: 1, Selector: "dead", TokenHash: "h", ExpiresAt: time.Now().Add(-time.Hour),
	})

	if err := store.CleanupExpired(); err != nil {
		t.Fatalf("CleanupExpired: %v", err)
	}
	if got, _ := store.FindBySelector("alive"); got == nil {
		t.Error("alive token should remain")
	}
	if got, _ := store.FindBySelector("dead"); got != nil {
		t.Error("expired token should be deleted")
	}
}
