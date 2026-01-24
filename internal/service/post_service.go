package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/capybara-translation/goblog/internal/repo"
)

// normalizeTags はタグ文字列を正規化します
// "Go, React, TypeScript" → "Go,React,TypeScript"
func normalizeTags(tags string) string {
	if tags == "" {
		return ""
	}
	parts := strings.Split(tags, ",")
	var normalized []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	return strings.Join(normalized, ",")
}

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
	CreatePost(title, slug, content, tags string, isPinned bool) (*domain.Post, error)

	// UpdatePost は記事を更新します
	UpdatePost(id int64, title, slug, content, tags string, isPinned bool) (*domain.Post, error)

	// PublishPost は記事を公開します
	PublishPost(id int64) (*domain.Post, error)

	// UnpublishPost は記事を下書きに戻します
	UnpublishPost(id int64) (*domain.Post, error)

	// DeletePost は記事を削除します
	DeletePost(id int64) error

	// GetPublishedPostsByTag は公開済み記事をタグで取得します
	GetPublishedPostsByTag(tag string, limit, offset int) ([]*domain.Post, error)

	// GetAllPostsByTag は管理画面用にタグで記事を取得します（ステータス指定可能）
	GetAllPostsByTag(tag string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error)

	// GetPublishedTags は公開済み記事のタグ一覧を取得します
	GetPublishedTags() (map[string]int, error)

	// GetAllTags は管理画面用にタグ一覧を取得します（ステータス指定可能）
	GetAllTags(status *domain.PostStatus) (map[string]int, error)

	// CountPosts は記事の総数を取得します
	CountPosts(status *domain.PostStatus) (int, error)

	// CountPostsByTag は特定のタグを持つ記事の総数を取得します
	CountPostsByTag(tag string, status *domain.PostStatus) (int, error)

	// SearchPosts は検索クエリに一致する記事を取得します（管理画面用）
	SearchPosts(query string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error)

	// CountSearchPosts は検索クエリに一致する記事数を取得します
	CountSearchPosts(query string, status *domain.PostStatus) (int, error)

	// SearchPublishedPosts は公開済み記事を検索します（公開ページ用）
	SearchPublishedPosts(query string, limit, offset int) ([]*domain.Post, error)

	// CountSearchPublishedPosts は公開済み記事の検索結果数を取得します
	CountSearchPublishedPosts(query string) (int, error)

	// GetPinnedPosts はピン留めされた公開記事を取得します（公開ページ用）
	GetPinnedPosts() ([]*domain.Post, error)

	// SetPinned は記事のピン留め状態を設定します
	SetPinned(id int64, pinned bool) (*domain.Post, error)
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
func (s *postService) CreatePost(title, slug, content, tags string, isPinned bool) (*domain.Post, error) {
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
		Tags:      normalizeTags(tags),
		IsPinned:  isPinned,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.Create(post); err != nil {
		return nil, err
	}

	return post, nil
}

// UpdatePost は記事を更新します
func (s *postService) UpdatePost(id int64, title, slug, content, tags string, isPinned bool) (*domain.Post, error) {
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
	post.Tags = normalizeTags(tags)
	post.IsPinned = isPinned
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

// GetPublishedPostsByTag は公開済み記事をタグで取得します
func (s *postService) GetPublishedPostsByTag(tag string, limit, offset int) ([]*domain.Post, error) {
	if tag == "" {
		return nil, fmt.Errorf("tag cannot be empty")
	}
	status := domain.PostStatusPublished
	return s.repo.FindAllByTag(tag, &status, limit, offset)
}

// GetAllPostsByTag は管理画面用にタグで記事を取得します（ステータス指定可能）
func (s *postService) GetAllPostsByTag(tag string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
	if tag == "" {
		return nil, fmt.Errorf("tag cannot be empty")
	}
	return s.repo.FindAllByTag(tag, status, limit, offset)
}

// GetPublishedTags は公開済み記事のタグ一覧を取得します
func (s *postService) GetPublishedTags() (map[string]int, error) {
	status := domain.PostStatusPublished
	return s.repo.GetAllTags(&status)
}

// GetAllTags は管理画面用にタグ一覧を取得します（ステータス指定可能）
func (s *postService) GetAllTags(status *domain.PostStatus) (map[string]int, error) {
	return s.repo.GetAllTags(status)
}

// CountPosts は記事の総数を取得します
func (s *postService) CountPosts(status *domain.PostStatus) (int, error) {
	return s.repo.Count(status)
}

// CountPostsByTag は特定のタグを持つ記事の総数を取得します
func (s *postService) CountPostsByTag(tag string, status *domain.PostStatus) (int, error) {
	if tag == "" {
		return 0, fmt.Errorf("tag cannot be empty")
	}
	return s.repo.CountByTag(tag, status)
}

// SearchPosts は検索クエリに一致する記事を取得します（管理画面用）
func (s *postService) SearchPosts(query string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
	if query == "" {
		return s.repo.FindAll(status, limit, offset)
	}
	return s.repo.Search(query, status, limit, offset)
}

// CountSearchPosts は検索クエリに一致する記事数を取得します
func (s *postService) CountSearchPosts(query string, status *domain.PostStatus) (int, error) {
	if query == "" {
		return s.repo.Count(status)
	}
	return s.repo.CountSearch(query, status)
}

// SearchPublishedPosts は公開済み記事を検索します（公開ページ用）
func (s *postService) SearchPublishedPosts(query string, limit, offset int) ([]*domain.Post, error) {
	status := domain.PostStatusPublished
	if query == "" {
		return s.repo.FindAll(&status, limit, offset)
	}
	return s.repo.Search(query, &status, limit, offset)
}

// CountSearchPublishedPosts は公開済み記事の検索結果数を取得します
func (s *postService) CountSearchPublishedPosts(query string) (int, error) {
	status := domain.PostStatusPublished
	if query == "" {
		return s.repo.Count(&status)
	}
	return s.repo.CountSearch(query, &status)
}

// GetPinnedPosts はピン留めされた公開記事を取得します（公開ページ用）
func (s *postService) GetPinnedPosts() ([]*domain.Post, error) {
	return s.repo.FindPinnedPublished()
}

// SetPinned は記事のピン留め状態を設定します
func (s *postService) SetPinned(id int64, pinned bool) (*domain.Post, error) {
	post, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, fmt.Errorf("post not found: %d", id)
	}

	post.IsPinned = pinned
	post.UpdatedAt = time.Now()

	if err := s.repo.Update(post); err != nil {
		return nil, err
	}

	return post, nil
}
