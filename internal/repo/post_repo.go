package repo

import (
	"database/sql"
	"fmt"

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
	args := []interface{}{}

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
	if err == sql.ErrNoRows {
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
	if err == sql.ErrNoRows {
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
		INSERT INTO posts (title, slug, content, status, tags, created_at, updated_at, published_at)
		VALUES (:title, :slug, :content, :status, :tags, :created_at, :updated_at, :published_at)
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
		    tags = :tags, updated_at = :updated_at, published_at = :published_at
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
