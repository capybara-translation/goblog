package repo

import (
	"errors"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/capybara-translation/goblog/internal/domain"
)

func setupReactionTypeTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db, err := sqlx.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	applyMigrations(t, db,
		"../../migrations/001_create_posts.sql",
		"../../migrations/008_create_reactions.sql",
	)
	if err := SeedReactionTypes(db); err != nil {
		t.Fatalf("seed: %v", err)
	}
	db.MustExec("INSERT INTO posts (id, title, slug, content, status) VALUES (1, 'P1', 'p1', 'c', 'published')")
	return db
}

func TestReactionTypeRepo_FindAll_IncludesInactive(t *testing.T) {
	db := setupReactionTypeTestDB(t)
	defer db.Close()
	db.MustExec("UPDATE reaction_types SET is_active = 0 WHERE emoji = '🤔'")
	r := NewReactionTypeRepository(db)

	all, err := r.FindAll()
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if len(all) != len(DefaultReactionTypes) {
		t.Fatalf("expected %d rows (incl inactive), got %d", len(DefaultReactionTypes), len(all))
	}
	if all[0].Emoji != "👍" {
		t.Fatalf("expected first emoji 👍, got %s", all[0].Emoji)
	}
}

func TestReactionTypeRepo_CreateAndFindByID(t *testing.T) {
	db := setupReactionTypeTestDB(t)
	defer db.Close()
	r := NewReactionTypeRepository(db)

	id, err := r.Create(&domain.ReactionType{Emoji: "🚀", Label: "発射", SortOrder: 99, IsActive: true})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := r.FindByID(id)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got == nil || got.Emoji != "🚀" || got.Label != "発射" || got.SortOrder != 99 || !got.IsActive {
		t.Fatalf("unexpected row: %+v", got)
	}
}

func TestReactionTypeRepo_FindByID_Missing(t *testing.T) {
	db := setupReactionTypeTestDB(t)
	defer db.Close()
	r := NewReactionTypeRepository(db)
	got, err := r.FindByID(9999)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for missing id, got %+v", got)
	}
}

func TestReactionTypeRepo_Create_DuplicateEmoji(t *testing.T) {
	db := setupReactionTypeTestDB(t)
	defer db.Close()
	r := NewReactionTypeRepository(db)
	_, err := r.Create(&domain.ReactionType{Emoji: "👍", Label: "dup", SortOrder: 1, IsActive: true})
	if !errors.Is(err, ErrDuplicateEmoji) {
		t.Fatalf("expected ErrDuplicateEmoji, got %v", err)
	}
}

func TestReactionTypeRepo_Update(t *testing.T) {
	db := setupReactionTypeTestDB(t)
	defer db.Close()
	r := NewReactionTypeRepository(db)
	all, err := r.FindAll()
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	target := all[0] // 👍
	target.Label = "Good"
	target.SortOrder = 5
	target.IsActive = false
	if err := r.Update(target); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, err := r.FindByID(target.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got == nil {
		t.Fatal("expected row, got nil")
	}
	if got.Label != "Good" || got.SortOrder != 5 || got.IsActive {
		t.Fatalf("update not applied: %+v", got)
	}
}

func TestReactionTypeRepo_Update_DuplicateEmoji(t *testing.T) {
	db := setupReactionTypeTestDB(t)
	defer db.Close()
	r := NewReactionTypeRepository(db)
	all, err := r.FindAll()
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	target := all[0] // 👍
	target.Emoji = "❤️"
	if !errors.Is(r.Update(target), ErrDuplicateEmoji) {
		t.Fatal("expected ErrDuplicateEmoji on colliding update")
	}
}

func TestReactionTypeRepo_DeleteIfUnused(t *testing.T) {
	db := setupReactionTypeTestDB(t)
	defer db.Close()
	r := NewReactionTypeRepository(db)
	id, err := r.Create(&domain.ReactionType{Emoji: "🚀", Label: "発射", SortOrder: 99, IsActive: true})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	deleted, err := r.DeleteIfUnused(id)
	if err != nil {
		t.Fatalf("DeleteIfUnused: %v", err)
	}
	if !deleted {
		t.Fatal("expected deleted=true for unused type")
	}
	if got, _ := r.FindByID(id); got != nil {
		t.Fatal("row should be gone")
	}
}

func TestReactionTypeRepo_DeleteIfUnused_InUseKeepsCounts(t *testing.T) {
	db := setupReactionTypeTestDB(t)
	defer db.Close()
	r := NewReactionTypeRepository(db)
	id, err := r.Create(&domain.ReactionType{Emoji: "🚀", Label: "発射", SortOrder: 99, IsActive: true})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	db.MustExec("INSERT INTO post_reactions (post_id, reaction_type_id, visitor_key) VALUES (1, ?, 'vk')", id)

	deleted, err := r.DeleteIfUnused(id)
	if err != nil {
		t.Fatalf("DeleteIfUnused: %v", err)
	}
	if deleted {
		t.Fatal("expected deleted=false for in-use type")
	}
	if got, _ := r.FindByID(id); got == nil {
		t.Fatal("type row should survive")
	}
	var cnt int
	db.Get(&cnt, "SELECT COUNT(*) FROM post_reactions WHERE reaction_type_id = ?", id)
	if cnt != 1 {
		t.Fatalf("reaction count must be untouched, got %d", cnt)
	}
}
