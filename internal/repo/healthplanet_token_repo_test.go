package repo

import (
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// setupTestDBWithHealthPlanetTokens opens an in-memory SQLite database with
// the healthplanet_tokens table (mirrors migrations/011_create_health_records.sql).
func setupTestDBWithHealthPlanetTokens(t *testing.T) *sqlx.DB {
	t.Helper()
	db, err := sqlx.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	db.MustExec(`
		CREATE TABLE healthplanet_tokens (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			access_token TEXT NOT NULL,
			refresh_token TEXT NOT NULL,
			expires_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return db
}

func TestHealthPlanetTokenRepository_Load_ReturnsNilWhenEmpty(t *testing.T) {
	db := setupTestDBWithHealthPlanetTokens(t)
	r := NewHealthPlanetTokenRepository(db)

	tok, err := r.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if tok != nil {
		t.Errorf("tok = %+v, want nil", tok)
	}
}

func TestHealthPlanetTokenRepository_SaveAndLoad(t *testing.T) {
	db := setupTestDBWithHealthPlanetTokens(t)
	r := NewHealthPlanetTokenRepository(db)

	expires := time.Now().Add(30 * 24 * time.Hour).Truncate(time.Second)
	if err := r.Save("AT/abc", "RT/def", expires); err != nil {
		t.Fatalf("Save: %v", err)
	}

	tok, err := r.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if tok == nil {
		t.Fatal("tok = nil, want saved token")
	}
	if tok.AccessToken != "AT/abc" || tok.RefreshToken != "RT/def" {
		t.Errorf("unexpected token: %+v", tok)
	}
	if !tok.ExpiresAt.Equal(expires) {
		t.Errorf("ExpiresAt = %v, want %v", tok.ExpiresAt, expires)
	}
}

func TestHealthPlanetTokenRepository_Save_OverwritesSingleRow(t *testing.T) {
	db := setupTestDBWithHealthPlanetTokens(t)
	r := NewHealthPlanetTokenRepository(db)

	if err := r.Save("AT/old", "RT/old", time.Now()); err != nil {
		t.Fatalf("first Save: %v", err)
	}
	newExpires := time.Now().Add(30 * 24 * time.Hour).Truncate(time.Second)
	if err := r.Save("AT/new", "RT/new", newExpires); err != nil {
		t.Fatalf("second Save: %v", err)
	}

	var count int
	if err := db.Get(&count, `SELECT COUNT(*) FROM healthplanet_tokens`); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1 (single-row table)", count)
	}
	tok, err := r.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if tok.AccessToken != "AT/new" {
		t.Errorf("AccessToken = %q, want AT/new", tok.AccessToken)
	}
}
