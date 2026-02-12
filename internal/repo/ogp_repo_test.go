package repo

import (
	"testing"
	"time"

	"github.com/capybara-translation/goblog/internal/ogp"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func setupOGPTestDB(t *testing.T) *sqlx.DB {
	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	// Create ogp_cache table
	schema := `
		CREATE TABLE ogp_cache (
			url TEXT PRIMARY KEY,
			title TEXT,
			description TEXT,
			image_url TEXT,
			site_name TEXT,
			fetched_at DATETIME NOT NULL,
			expires_at DATETIME NOT NULL,
			error_msg TEXT
		);
		CREATE INDEX idx_ogp_cache_expires_at ON ogp_cache(expires_at);
	`
	_, err = db.Exec(schema)
	if err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	return db
}

func TestOGPRepository_FindByURL(t *testing.T) {
	db := setupOGPTestDB(t)
	defer db.Close()

	repo := NewOGPRepository(db)

	t.Run("Not found returns nil", func(t *testing.T) {
		data, err := repo.FindByURL("https://example.com/notfound")
		if err != nil {
			t.Errorf("FindByURL() error = %v", err)
		}
		if data != nil {
			t.Errorf("FindByURL() = %v, want nil", data)
		}
	})

	t.Run("Found returns data", func(t *testing.T) {
		// Insert test data
		testData := &ogp.Data{
			URL:         "https://example.com",
			Title:       "Test Title",
			Description: "Test Description",
			ImageURL:    "https://example.com/image.jpg",
			SiteName:    "Example",
			FetchedAt:   time.Now(),
			ExpiresAt:   time.Now().Add(24 * time.Hour),
		}
		err := repo.Upsert(testData)
		if err != nil {
			t.Fatalf("Upsert() error = %v", err)
		}

		// Find
		data, err := repo.FindByURL("https://example.com")
		if err != nil {
			t.Errorf("FindByURL() error = %v", err)
		}
		if data == nil {
			t.Fatal("FindByURL() returned nil")
		}
		if data.Title != "Test Title" {
			t.Errorf("Title = %q, want %q", data.Title, "Test Title")
		}
		if data.Description != "Test Description" {
			t.Errorf("Description = %q, want %q", data.Description, "Test Description")
		}
	})
}

func TestOGPRepository_Upsert(t *testing.T) {
	db := setupOGPTestDB(t)
	defer db.Close()

	repo := NewOGPRepository(db)

	t.Run("Insert new data", func(t *testing.T) {
		data := &ogp.Data{
			URL:         "https://example.com/new",
			Title:       "New Title",
			Description: "New Description",
			FetchedAt:   time.Now(),
			ExpiresAt:   time.Now().Add(24 * time.Hour),
		}

		err := repo.Upsert(data)
		if err != nil {
			t.Errorf("Upsert() error = %v", err)
		}

		// Verify
		found, _ := repo.FindByURL("https://example.com/new")
		if found == nil || found.Title != "New Title" {
			t.Error("Upsert() failed to insert data")
		}
	})

	t.Run("Update existing data", func(t *testing.T) {
		// Insert
		data := &ogp.Data{
			URL:       "https://example.com/update",
			Title:     "Original Title",
			FetchedAt: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		repo.Upsert(data)

		// Update
		data.Title = "Updated Title"
		err := repo.Upsert(data)
		if err != nil {
			t.Errorf("Upsert() error = %v", err)
		}

		// Verify
		found, _ := repo.FindByURL("https://example.com/update")
		if found == nil || found.Title != "Updated Title" {
			t.Error("Upsert() failed to update data")
		}
	})

	t.Run("Store error message", func(t *testing.T) {
		data := &ogp.Data{
			URL:       "https://example.com/error",
			Title:     "https://example.com/error",
			FetchedAt: time.Now(),
			ExpiresAt: time.Now().Add(1 * time.Hour),
			ErrorMsg:  "fetch failed: connection timeout",
		}

		err := repo.Upsert(data)
		if err != nil {
			t.Errorf("Upsert() error = %v", err)
		}

		// Verify
		found, _ := repo.FindByURL("https://example.com/error")
		if found == nil || found.ErrorMsg != "fetch failed: connection timeout" {
			t.Error("Upsert() failed to store error message")
		}
	})
}

func TestOGPRepository_DeleteExpired(t *testing.T) {
	db := setupOGPTestDB(t)
	defer db.Close()

	repo := NewOGPRepository(db)

	// Insert expired data
	expiredData := &ogp.Data{
		URL:       "https://example.com/expired",
		Title:     "Expired",
		FetchedAt: time.Now().Add(-48 * time.Hour),
		ExpiresAt: time.Now().Add(-24 * time.Hour), // Expired 24 hours ago
	}
	repo.Upsert(expiredData)

	// Insert valid data
	validData := &ogp.Data{
		URL:       "https://example.com/valid",
		Title:     "Valid",
		FetchedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour), // Expires in 24 hours
	}
	repo.Upsert(validData)

	// Delete expired
	count, err := repo.DeleteExpired()
	if err != nil {
		t.Errorf("DeleteExpired() error = %v", err)
	}
	if count != 1 {
		t.Errorf("DeleteExpired() deleted %d rows, want 1", count)
	}

	// Verify expired is deleted
	found, _ := repo.FindByURL("https://example.com/expired")
	if found != nil {
		t.Error("DeleteExpired() did not delete expired data")
	}

	// Verify valid is still there
	found, _ = repo.FindByURL("https://example.com/valid")
	if found == nil {
		t.Error("DeleteExpired() incorrectly deleted valid data")
	}
}

func TestNewOGPRepository(t *testing.T) {
	db := setupOGPTestDB(t)
	defer db.Close()

	repo := NewOGPRepository(db)
	if repo == nil {
		t.Error("NewOGPRepository() returned nil")
	}
}
