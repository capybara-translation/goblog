package repo

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/jmoiron/sqlx"
)

// PostRepository は記事データへのアクセスを提供するインターフェースです
type PostRepository interface {
	// FindAll はすべての記事を取得します
	FindAll(status *domain.PostStatus, limit, offset int) ([]*domain.Post, error)

	// FindBySlug はスラッグで記事を取得します
	FindBySlug(slug string) (*domain.Post, error)

	// FindByID はIDで記事を取得します
	FindByID(id int64) (*domain.Post, error)

	// Create は新しい記事を作成します
	Create(post *domain.Post) error

	// Update は記事を更新します
	Update(post *domain.Post) error

	// Delete は記事を削除します
	Delete(id int64) error

	// FindAllByTag は特定のタグを持つ記事を取得します
	FindAllByTag(tag string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error)

	// GetAllTags はすべてのユニークなタグと記事数を取得します
	GetAllTags(status *domain.PostStatus) (map[string]int, error)

	// Count は記事の総数を取得します
	Count(status *domain.PostStatus) (int, error)

	// CountByTag は特定のタグを持つ記事の総数を取得します
	CountByTag(tag string, status *domain.PostStatus) (int, error)

	// Search は検索クエリに一致する記事を取得します
	Search(query string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error)

	// CountSearch は検索クエリに一致する記事数を取得します
	CountSearch(query string, status *domain.PostStatus) (int, error)

	// FindPinnedPublished はピン留めされた公開記事を取得します
	FindPinnedPublished() ([]*domain.Post, error)
}

// postRepository はPostRepositoryのSQLite実装です
type postRepository struct {
	db *sqlx.DB
}

// NewPostRepository は新しいPostRepositoryを作成します
func NewPostRepository(db *sqlx.DB) PostRepository {
	return &postRepository{db: db}
}

// FindAll はすべての記事を取得します
func (r *postRepository) FindAll(status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
	var posts []*domain.Post
	query := "SELECT * FROM posts"
	args := []any{}

	// ステータスでフィルタリング
	if status != nil {
		query += " WHERE status = ?"
		args = append(args, *status)
	}

	// 公開日時の降順でソート
	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	if err := r.db.Select(&posts, query, args...); err != nil {
		return nil, fmt.Errorf("failed to query posts: %w", err)
	}

	return posts, nil
}

// FindBySlug はスラッグで記事を取得します
func (r *postRepository) FindBySlug(slug string) (*domain.Post, error) {
	var post domain.Post
	query := "SELECT * FROM posts WHERE slug = ?"

	err := r.db.Get(&post, query, slug)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find post by slug: %w", err)
	}

	return &post, nil
}

// FindByID はIDで記事を取得します
func (r *postRepository) FindByID(id int64) (*domain.Post, error) {
	var post domain.Post
	query := "SELECT * FROM posts WHERE id = ?"

	err := r.db.Get(&post, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find post by id: %w", err)
	}

	return &post, nil
}

// Create は新しい記事を作成します
func (r *postRepository) Create(post *domain.Post) error {
	query := `
		INSERT INTO posts (title, slug, content, status, tags, is_pinned, created_at, updated_at, published_at)
		VALUES (:title, :slug, :content, :status, :tags, :is_pinned, :created_at, :updated_at, :published_at)
	`

	result, err := r.db.NamedExec(query, post)
	if err != nil {
		return fmt.Errorf("failed to create post: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	post.ID = id
	return nil
}

// Update は記事を更新します
func (r *postRepository) Update(post *domain.Post) error {
	query := `
		UPDATE posts
		SET title = :title, slug = :slug, content = :content, status = :status,
		    tags = :tags, is_pinned = :is_pinned, updated_at = :updated_at, published_at = :published_at
		WHERE id = :id
	`

	_, err := r.db.NamedExec(query, post)
	if err != nil {
		return fmt.Errorf("failed to update post: %w", err)
	}

	return nil
}

// Delete は記事を削除します
func (r *postRepository) Delete(id int64) error {
	query := "DELETE FROM posts WHERE id = ?"

	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete post: %w", err)
	}

	return nil
}

// FindAllByTag は特定のタグを持つ記事を取得します
func (r *postRepository) FindAllByTag(tag string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
	var posts []*domain.Post

	// LIKEクエリでカンマ区切りタグを検索
	// 4つのパターンで完全一致を確保:
	// 1. tags = "Go" (単一タグ)
	// 2. tags LIKE "Go,%" (先頭)
	// 3. tags LIKE "%,Go,%" (中間)
	// 4. tags LIKE "%,Go" (末尾)
	query := `SELECT * FROM posts WHERE (
		tags = ? OR
		tags LIKE ? || ',%' OR
		tags LIKE '%,' || ? || ',%' OR
		tags LIKE '%,' || ?
	)`
	args := []any{tag, tag, tag, tag}

	// ステータスでフィルタリング
	if status != nil {
		query += " AND status = ?"
		args = append(args, *status)
	}

	// 作成日時の降順でソート
	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	if err := r.db.Select(&posts, query, args...); err != nil {
		return nil, fmt.Errorf("failed to query posts by tag: %w", err)
	}

	return posts, nil
}

// GetAllTags はすべてのユニークなタグと記事数を取得します
func (r *postRepository) GetAllTags(status *domain.PostStatus) (map[string]int, error) {
	var posts []*domain.Post
	query := "SELECT tags FROM posts"
	args := []any{}

	// ステータスでフィルタリング
	if status != nil {
		query += " WHERE status = ?"
		args = append(args, *status)
	}

	if err := r.db.Select(&posts, query, args...); err != nil {
		return nil, fmt.Errorf("failed to query tags: %w", err)
	}

	// タグをカウント
	tagCount := make(map[string]int)
	for _, post := range posts {
		if post.Tags == "" {
			continue
		}
		tags := strings.Split(post.Tags, ",")
		for _, tag := range tags {
			trimmed := strings.TrimSpace(tag)
			if trimmed != "" {
				tagCount[trimmed]++
			}
		}
	}

	return tagCount, nil
}

// Count は記事の総数を取得します
func (r *postRepository) Count(status *domain.PostStatus) (int, error) {
	query := "SELECT COUNT(*) FROM posts"
	args := []any{}

	if status != nil {
		query += " WHERE status = ?"
		args = append(args, *status)
	}

	var count int
	if err := r.db.Get(&count, query, args...); err != nil {
		return 0, fmt.Errorf("failed to count posts: %w", err)
	}

	return count, nil
}

// CountByTag は特定のタグを持つ記事の総数を取得します
func (r *postRepository) CountByTag(tag string, status *domain.PostStatus) (int, error) {
	query := `SELECT COUNT(*) FROM posts WHERE (
		tags = ? OR
		tags LIKE ? || ',%' OR
		tags LIKE '%,' || ? || ',%' OR
		tags LIKE '%,' || ?
	)`
	args := []any{tag, tag, tag, tag}

	if status != nil {
		query += " AND status = ?"
		args = append(args, *status)
	}

	var count int
	if err := r.db.Get(&count, query, args...); err != nil {
		return 0, fmt.Errorf("failed to count posts by tag: %w", err)
	}

	return count, nil
}

// Search は検索クエリに一致する記事を取得します（タイトルと本文を検索）
func (r *postRepository) Search(query string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
	var posts []*domain.Post

	// 大文字小文字を区別しない部分一致検索
	searchPattern := "%" + strings.ToLower(query) + "%"
	sql := `SELECT * FROM posts WHERE (
		LOWER(title) LIKE ? OR LOWER(content) LIKE ?
	)`
	args := []any{searchPattern, searchPattern}

	// ステータスでフィルタリング
	if status != nil {
		sql += " AND status = ?"
		args = append(args, *status)
	}

	// 作成日時の降順でソート
	sql += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	if err := r.db.Select(&posts, sql, args...); err != nil {
		return nil, fmt.Errorf("failed to search posts: %w", err)
	}

	return posts, nil
}

// CountSearch は検索クエリに一致する記事数を取得します
func (r *postRepository) CountSearch(query string, status *domain.PostStatus) (int, error) {
	searchPattern := "%" + strings.ToLower(query) + "%"
	sql := `SELECT COUNT(*) FROM posts WHERE (
		LOWER(title) LIKE ? OR LOWER(content) LIKE ?
	)`
	args := []any{searchPattern, searchPattern}

	if status != nil {
		sql += " AND status = ?"
		args = append(args, *status)
	}

	var count int
	if err := r.db.Get(&count, sql, args...); err != nil {
		return 0, fmt.Errorf("failed to count search results: %w", err)
	}

	return count, nil
}

// FindPinnedPublished はピン留めされた公開記事を取得します
func (r *postRepository) FindPinnedPublished() ([]*domain.Post, error) {
	var posts []*domain.Post
	query := `SELECT * FROM posts WHERE is_pinned = 1 AND status = 'published' ORDER BY title ASC`

	if err := r.db.Select(&posts, query); err != nil {
		return nil, fmt.Errorf("failed to query pinned posts: %w", err)
	}

	return posts, nil
}
