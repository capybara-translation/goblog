package service

import (
	"fmt"
	"time"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/capybara-translation/goblog/internal/repo"
)

// PostService は記事に関するビジネスロジックを提供します
type PostService interface {
	// GetPublishedPosts は公開済みの記事一覧を取得します
	GetPublishedPosts(limit, offset int) ([]*domain.Post, error)

	// GetPostBySlug はスラッグで公開済み記事を取得します
	GetPostBySlug(slug string) (*domain.Post, error)

	// GetAllPosts は管理画面用にすべての記事を取得します
	GetAllPosts(status *domain.PostStatus, limit, offset int) ([]*domain.Post, error)

	// GetPostByID はIDで記事を取得します（管理画面用）
	GetPostByID(id int64) (*domain.Post, error)

	// CreatePost は新しい記事を作成します
	CreatePost(title, slug, content, tags string) (*domain.Post, error)

	// UpdatePost は記事を更新します
	UpdatePost(id int64, title, slug, content, tags string) (*domain.Post, error)

	// PublishPost は記事を公開します
	PublishPost(id int64) (*domain.Post, error)

	// UnpublishPost は記事を下書きに戻します
	UnpublishPost(id int64) (*domain.Post, error)

	// DeletePost は記事を削除します
	DeletePost(id int64) error
}

type postService struct {
	repo repo.PostRepository
}

// NewPostService は新しいPostServiceを作成します
func NewPostService(repo repo.PostRepository) PostService {
	return &postService{repo: repo}
}

// GetPublishedPosts は公開済みの記事一覧を取得します
func (s *postService) GetPublishedPosts(limit, offset int) ([]*domain.Post, error) {
	status := domain.PostStatusPublished
	return s.repo.FindAll(&status, limit, offset)
}

// GetPostBySlug はスラッグで公開済み記事を取得します
func (s *postService) GetPostBySlug(slug string) (*domain.Post, error) {
	post, err := s.repo.FindBySlug(slug)
	if err != nil {
		return nil, err
	}

	// 公開済みの記事のみ返す
	if post == nil || post.Status != domain.PostStatusPublished {
		return nil, nil
	}

	return post, nil
}

// GetAllPosts は管理画面用にすべての記事を取得します
func (s *postService) GetAllPosts(status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
	return s.repo.FindAll(status, limit, offset)
}

// GetPostByID はIDで記事を取得します（管理画面用）
func (s *postService) GetPostByID(id int64) (*domain.Post, error) {
	return s.repo.FindByID(id)
}

// CreatePost は新しい記事を作成します
func (s *postService) CreatePost(title, slug, content, tags string) (*domain.Post, error) {
	// スラッグの重複チェック
	existing, err := s.repo.FindBySlug(slug)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("slug already exists: %s", slug)
	}

	now := time.Now()
	post := &domain.Post{
		Title:     title,
		Slug:      slug,
		Content:   content,
		Status:    domain.PostStatusDraft,
		Tags:      tags,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.Create(post); err != nil {
		return nil, err
	}

	return post, nil
}

// UpdatePost は記事を更新します
func (s *postService) UpdatePost(id int64, title, slug, content, tags string) (*domain.Post, error) {
	post, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, fmt.Errorf("post not found: %d", id)
	}

	// スラッグが変更された場合、重複チェック
	if post.Slug != slug {
		existing, err := s.repo.FindBySlug(slug)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return nil, fmt.Errorf("slug already exists: %s", slug)
		}
	}

	post.Title = title
	post.Slug = slug
	post.Content = content
	post.Tags = tags
	post.UpdatedAt = time.Now()

	if err := s.repo.Update(post); err != nil {
		return nil, err
	}

	return post, nil
}

// PublishPost は記事を公開します
func (s *postService) PublishPost(id int64) (*domain.Post, error) {
	post, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, fmt.Errorf("post not found: %d", id)
	}

	now := time.Now()
	post.Status = domain.PostStatusPublished
	post.PublishedAt = &now
	post.UpdatedAt = now

	if err := s.repo.Update(post); err != nil {
		return nil, err
	}

	return post, nil
}

// UnpublishPost は記事を下書きに戻します
func (s *postService) UnpublishPost(id int64) (*domain.Post, error) {
	post, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, fmt.Errorf("post not found: %d", id)
	}

	post.Status = domain.PostStatusDraft
	post.UpdatedAt = time.Now()

	if err := s.repo.Update(post); err != nil {
		return nil, err
	}

	return post, nil
}

// DeletePost は記事を削除します
func (s *postService) DeletePost(id int64) error {
	return s.repo.Delete(id)
}
