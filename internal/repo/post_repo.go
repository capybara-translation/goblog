package repo

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/jmoiron/sqlx"
)

// PostRepository is an interface that provides access to post data
type PostRepository interface {
	// FindAll retrieves all posts
	FindAll(status *domain.PostStatus, limit, offset int) ([]*domain.Post, error)

	// FindBySlug retrieves a post by slug
	FindBySlug(slug string) (*domain.Post, error)

	// FindByID retrieves a post by ID
	FindByID(id int64) (*domain.Post, error)

	// Create creates a new post
	Create(post *domain.Post) error

	// Update updates a post
	Update(post *domain.Post) error

	// Delete deletes a post
	Delete(id int64) error

	// FindAllByTag retrieves posts with a specific tag
	FindAllByTag(tag string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error)

	// GetAllTags retrieves all unique tags and their post counts
	GetAllTags(status *domain.PostStatus) (map[string]int, error)

	// Count retrieves the total number of posts
	Count(status *domain.PostStatus) (int, error)

	// CountByTag retrieves the total number of posts with a specific tag
	CountByTag(tag string, status *domain.PostStatus) (int, error)

	// Search retrieves posts matching the search query
	Search(query string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error)

	// CountSearch retrieves the number of posts matching the search query
	CountSearch(query string, status *domain.PostStatus) (int, error)

	// FindPinnedPublished retrieves pinned published posts
	FindPinnedPublished() ([]*domain.Post, error)
}

// postRepository is the SQLite implementation of PostRepository
type postRepository struct {
	db *sqlx.DB
}

// NewPostRepository creates a new PostRepository
func NewPostRepository(db *sqlx.DB) PostRepository {
	return &postRepository{db: db}
}

// FindAll retrieves all posts
func (r *postRepository) FindAll(status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
	var posts []*domain.Post
	query := "SELECT * FROM posts"
	args := []any{}

	// Filter by status
	if status != nil {
		query += " WHERE status = ?"
		args = append(args, *status)
	}

	// Sort by published date descending
	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	if err := r.db.Select(&posts, query, args...); err != nil {
		return nil, fmt.Errorf("failed to query posts: %w", err)
	}

	return posts, nil
}

// FindBySlug retrieves a post by slug
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

// FindByID retrieves a post by ID
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

// Create creates a new post
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

// Update updates a post
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

// Delete deletes a post
func (r *postRepository) Delete(id int64) error {
	query := "DELETE FROM posts WHERE id = ?"

	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete post: %w", err)
	}

	return nil
}

// FindAllByTag retrieves posts with a specific tag
func (r *postRepository) FindAllByTag(tag string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
	var posts []*domain.Post

	// Search comma-separated tags with LIKE query
	// Ensure exact match with 4 patterns:
	// 1. tags = "Go" (single tag)
	// 2. tags LIKE "Go,%" (at beginning)
	// 3. tags LIKE "%,Go,%" (in middle)
	// 4. tags LIKE "%,Go" (at end)
	query := `SELECT * FROM posts WHERE (
		tags = ? OR
		tags LIKE ? || ',%' OR
		tags LIKE '%,' || ? || ',%' OR
		tags LIKE '%,' || ?
	)`
	args := []any{tag, tag, tag, tag}

	// Filter by status
	if status != nil {
		query += " AND status = ?"
		args = append(args, *status)
	}

	// Sort by created date descending
	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	if err := r.db.Select(&posts, query, args...); err != nil {
		return nil, fmt.Errorf("failed to query posts by tag: %w", err)
	}

	return posts, nil
}

// GetAllTags retrieves all unique tags and their post counts
func (r *postRepository) GetAllTags(status *domain.PostStatus) (map[string]int, error) {
	var posts []*domain.Post
	query := "SELECT tags FROM posts"
	args := []any{}

	// Filter by status
	if status != nil {
		query += " WHERE status = ?"
		args = append(args, *status)
	}

	if err := r.db.Select(&posts, query, args...); err != nil {
		return nil, fmt.Errorf("failed to query tags: %w", err)
	}

	// Count tags
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

// Count retrieves the total number of posts
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

// CountByTag retrieves the total number of posts with a specific tag
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

// Search retrieves posts matching the search query (searches title and body)
func (r *postRepository) Search(query string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
	var posts []*domain.Post

	// Case-insensitive partial match search
	searchPattern := "%" + strings.ToLower(query) + "%"
	sql := `SELECT * FROM posts WHERE (
		LOWER(title) LIKE ? OR LOWER(content) LIKE ?
	)`
	args := []any{searchPattern, searchPattern}

	// Filter by status
	if status != nil {
		sql += " AND status = ?"
		args = append(args, *status)
	}

	// Sort by created date descending
	sql += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	if err := r.db.Select(&posts, sql, args...); err != nil {
		return nil, fmt.Errorf("failed to search posts: %w", err)
	}

	return posts, nil
}

// CountSearch retrieves the number of posts matching the search query
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

// FindPinnedPublished retrieves pinned published posts
func (r *postRepository) FindPinnedPublished() ([]*domain.Post, error) {
	var posts []*domain.Post
	query := `SELECT * FROM posts WHERE is_pinned = 1 AND status = 'published' ORDER BY title ASC`

	if err := r.db.Select(&posts, query); err != nil {
		return nil, fmt.Errorf("failed to query pinned posts: %w", err)
	}

	return posts, nil
}
