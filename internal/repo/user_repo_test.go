package repo

import (
	"os"
	"testing"
	"time"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// setupTestDBWithUsers はテスト用のインメモリSQLiteデータベースをセットアップします（usersテーブル含む）
func setupTestDBWithUsers(t *testing.T) *sqlx.DB {
	t.Helper()

	// インメモリSQLiteを開く
	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	// usersテーブルのマイグレーションを実行
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
		PasswordHash: "$2a$10$abcdefghijklmnopqrstuvwxyz", // bcryptハッシュのダミー
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

	// 作成されたユーザーを取得して検証
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

	// ユーザー名で検索
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

	// 存在しないユーザー名で検索
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

	// IDで検索
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

	// 存在しないIDで検索
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

	// ユーザー情報を更新
	user.Username = "newname"
	user.PasswordHash = "$2a$10$newhash"
	user.UpdatedAt = time.Now()

	if err := repo.Update(user); err != nil {
		t.Fatalf("failed to update user: %v", err)
	}

	// 更新後のユーザーを取得
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

	// ユーザーを削除
	if err := repo.Delete(user.ID); err != nil {
		t.Fatalf("failed to delete user: %v", err)
	}

	// 削除後に検索
	found, err := repo.FindByID(user.ID)
	if err != nil {
		t.Fatalf("failed to query user: %v", err)
	}

	if found != nil {
		t.Error("expected user to be deleted, but it still exists")
	}
}
