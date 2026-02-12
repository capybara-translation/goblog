package repo

import (
	"os"
	"testing"
	"time"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// setupTestDBWithUsers sets up an in-memory SQLite database for testing (including users table)
func setupTestDBWithUsers(t *testing.T) *sqlx.DB {
	t.Helper()

	// Open in-memory SQLite
	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	// Execute users table migration
	schema, err := os.ReadFile("../../migrations/002_create_users.sql")
	if err != nil {
		t.Fatalf("failed to read migration file: %v", err)
	}

	if _, err := db.Exec(string(schema)); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	return db
}

func TestUserRepository_Create(t *testing.T) {
	db := setupTestDBWithUsers(t)
	defer db.Close()

	repo := NewUserRepository(db)

	now := time.Now()
	user := &domain.User{
		Username:     "testuser",
		PasswordHash: "$2a$10$abcdefghijklmnopqrstuvwxyz", // Dummy bcrypt hash
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	err := repo.Create(user)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	if user.ID == 0 {
		t.Error("expected user ID to be set, got 0")
	}

	// Retrieve and verify created user
	found, err := repo.FindByID(user.ID)
	if err != nil {
		t.Fatalf("failed to find user: %v", err)
	}

	if found == nil {
		t.Fatal("expected to find user, got nil")
	}

	if found.Username != user.Username {
		t.Errorf("expected username %q, got %q", user.Username, found.Username)
	}

	if found.PasswordHash != user.PasswordHash {
		t.Errorf("expected password hash %q, got %q", user.PasswordHash, found.PasswordHash)
	}
}

func TestUserRepository_FindByUsername(t *testing.T) {
	db := setupTestDBWithUsers(t)
	defer db.Close()

	repo := NewUserRepository(db)

	now := time.Now()
	user := &domain.User{
		Username:     "findme",
		PasswordHash: "$2a$10$abcdefghijklmnopqrstuvwxyz",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := repo.Create(user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Search by username
	found, err := repo.FindByUsername("findme")
	if err != nil {
		t.Fatalf("failed to find user by username: %v", err)
	}

	if found == nil {
		t.Fatal("expected to find user, got nil")
	}

	if found.ID != user.ID {
		t.Errorf("expected user ID %d, got %d", user.ID, found.ID)
	}

	if found.Username != user.Username {
		t.Errorf("expected username %q, got %q", user.Username, found.Username)
	}

	// Search with non-existent username
	notFound, err := repo.FindByUsername("nonexistent")
	if err != nil {
		t.Fatalf("failed to query user: %v", err)
	}

	if notFound != nil {
		t.Error("expected nil for nonexistent user, got a user")
	}
}

func TestUserRepository_FindByID(t *testing.T) {
	db := setupTestDBWithUsers(t)
	defer db.Close()

	repo := NewUserRepository(db)

	now := time.Now()
	user := &domain.User{
		Username:     "testuser",
		PasswordHash: "$2a$10$abcdefghijklmnopqrstuvwxyz",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := repo.Create(user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Search by ID
	found, err := repo.FindByID(user.ID)
	if err != nil {
		t.Fatalf("failed to find user by ID: %v", err)
	}

	if found == nil {
		t.Fatal("expected to find user, got nil")
	}

	if found.Username != user.Username {
		t.Errorf("expected username %q, got %q", user.Username, found.Username)
	}

	// Search with non-existent ID
	notFound, err := repo.FindByID(99999)
	if err != nil {
		t.Fatalf("failed to query user: %v", err)
	}

	if notFound != nil {
		t.Error("expected nil for nonexistent user, got a user")
	}
}

func TestUserRepository_Update(t *testing.T) {
	db := setupTestDBWithUsers(t)
	defer db.Close()

	repo := NewUserRepository(db)

	now := time.Now()
	user := &domain.User{
		Username:     "oldname",
		PasswordHash: "$2a$10$oldhash",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := repo.Create(user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Update user information
	user.Username = "newname"
	user.PasswordHash = "$2a$10$newhash"
	user.UpdatedAt = time.Now()

	if err := repo.Update(user); err != nil {
		t.Fatalf("failed to update user: %v", err)
	}

	// Retrieve updated user
	updated, err := repo.FindByID(user.ID)
	if err != nil {
		t.Fatalf("failed to find user: %v", err)
	}

	if updated.Username != "newname" {
		t.Errorf("expected username %q, got %q", "newname", updated.Username)
	}

	if updated.PasswordHash != "$2a$10$newhash" {
		t.Errorf("expected password hash %q, got %q", "$2a$10$newhash", updated.PasswordHash)
	}
}

func TestUserRepository_Delete(t *testing.T) {
	db := setupTestDBWithUsers(t)
	defer db.Close()

	repo := NewUserRepository(db)

	now := time.Now()
	user := &domain.User{
		Username:     "deleteme",
		PasswordHash: "$2a$10$abcdefghijklmnopqrstuvwxyz",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := repo.Create(user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Delete user
	if err := repo.Delete(user.ID); err != nil {
		t.Fatalf("failed to delete user: %v", err)
	}

	// Search after deletion
	found, err := repo.FindByID(user.ID)
	if err != nil {
		t.Fatalf("failed to query user: %v", err)
	}

	if found != nil {
		t.Error("expected user to be deleted, but it still exists")
	}
}
