package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/capybara-translation/goblog/internal/repo"
)

// normalizeTags normalizes a tag string
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

// PostService provides business logic for posts
type PostService interface {
	// GetPublishedPosts retrieves a list of published posts
	GetPublishedPosts(limit, offset int) ([]*domain.Post, error)

	// GetPostBySlug retrieves a published post by slug
	GetPostBySlug(slug string) (*domain.Post, error)

	// GetAllPosts retrieves all posts for the admin panel
	GetAllPosts(status *domain.PostStatus, limit, offset int) ([]*domain.Post, error)

	// GetPostByID retrieves a post by ID (for admin panel)
	GetPostByID(id int64) (*domain.Post, error)

	// CreatePost creates a new post
	CreatePost(title, slug, content, tags string, isPinned bool) (*domain.Post, error)

	// UpdatePost updates a post
	UpdatePost(id int64, title, slug, content, tags string, isPinned bool) (*domain.Post, error)

	// PublishPost publishes a post
	PublishPost(id int64) (*domain.Post, error)

	// UnpublishPost reverts a post to draft status
	UnpublishPost(id int64) (*domain.Post, error)

	// DeletePost deletes a post
	DeletePost(id int64) error

	// GetPublishedPostsByTag retrieves published posts by tag
	GetPublishedPostsByTag(tag string, limit, offset int) ([]*domain.Post, error)

	// GetAllPostsByTag retrieves posts by tag for the admin panel (status can be specified)
	GetAllPostsByTag(tag string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error)

	// GetPublishedTags retrieves a list of tags from published posts
	GetPublishedTags() (map[string]int, error)

	// GetAllTags retrieves a list of tags for the admin panel (status can be specified)
	GetAllTags(status *domain.PostStatus) (map[string]int, error)

	// CountPosts retrieves the total number of posts
	CountPosts(status *domain.PostStatus) (int, error)

	// CountPostsByTag retrieves the total number of posts with a specific tag
	CountPostsByTag(tag string, status *domain.PostStatus) (int, error)

	// SearchPosts retrieves posts matching the search query (for admin panel)
	SearchPosts(query string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error)

	// CountSearchPosts retrieves the count of posts matching the search query
	CountSearchPosts(query string, status *domain.PostStatus) (int, error)

	// SearchPublishedPosts searches published posts (for public pages)
	SearchPublishedPosts(query string, limit, offset int) ([]*domain.Post, error)

	// CountSearchPublishedPosts retrieves the count of search results for published posts
	CountSearchPublishedPosts(query string) (int, error)

	// GetPinnedPosts retrieves pinned published posts (for public pages)
	GetPinnedPosts() ([]*domain.Post, error)

	// SetPinned sets the pinned status of a post
	SetPinned(id int64, pinned bool) (*domain.Post, error)

	// GetPublishedPostsBySlugs retrieves published posts for the given slugs (for
	// batch reaction lookups on listing pages).
	GetPublishedPostsBySlugs(slugs []string) ([]*domain.Post, error)
}

type postService struct {
	repo repo.PostRepository
}

// NewPostService creates a new PostService
func NewPostService(repo repo.PostRepository) PostService {
	return &postService{repo: repo}
}

// GetPublishedPosts retrieves a list of published posts
func (s *postService) GetPublishedPosts(limit, offset int) ([]*domain.Post, error) {
	status := domain.PostStatusPublished
	return s.repo.FindAll(&status, limit, offset)
}

// GetPostBySlug retrieves a published post by slug
func (s *postService) GetPostBySlug(slug string) (*domain.Post, error) {
	post, err := s.repo.FindBySlug(slug)
	if err != nil {
		return nil, err
	}

	// Return only published posts
	if post == nil || post.Status != domain.PostStatusPublished {
		return nil, nil
	}

	return post, nil
}

// GetAllPosts retrieves all posts for the admin panel
func (s *postService) GetAllPosts(status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
	return s.repo.FindAll(status, limit, offset)
}

// GetPostByID retrieves a post by ID (for admin panel)
func (s *postService) GetPostByID(id int64) (*domain.Post, error) {
	return s.repo.FindByID(id)
}

// CreatePost creates a new post
func (s *postService) CreatePost(title, slug, content, tags string, isPinned bool) (*domain.Post, error) {
	// Check for slug duplication
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

// UpdatePost updates a post
func (s *postService) UpdatePost(id int64, title, slug, content, tags string, isPinned bool) (*domain.Post, error) {
	post, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, fmt.Errorf("post not found: %d", id)
	}

	// Check for slug duplication if the slug has been changed
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

// PublishPost publishes a post
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

// UnpublishPost reverts a post to draft status
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

// DeletePost deletes a post
func (s *postService) DeletePost(id int64) error {
	return s.repo.Delete(id)
}

// GetPublishedPostsByTag retrieves published posts by tag
func (s *postService) GetPublishedPostsByTag(tag string, limit, offset int) ([]*domain.Post, error) {
	if tag == "" {
		return nil, fmt.Errorf("tag cannot be empty")
	}
	status := domain.PostStatusPublished
	return s.repo.FindAllByTag(tag, &status, limit, offset)
}

// GetAllPostsByTag retrieves posts by tag for the admin panel (status can be specified)
func (s *postService) GetAllPostsByTag(tag string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
	if tag == "" {
		return nil, fmt.Errorf("tag cannot be empty")
	}
	return s.repo.FindAllByTag(tag, status, limit, offset)
}

// GetPublishedTags retrieves a list of tags from published posts
func (s *postService) GetPublishedTags() (map[string]int, error) {
	status := domain.PostStatusPublished
	return s.repo.GetAllTags(&status)
}

// GetAllTags retrieves a list of tags for the admin panel (status can be specified)
func (s *postService) GetAllTags(status *domain.PostStatus) (map[string]int, error) {
	return s.repo.GetAllTags(status)
}

// CountPosts retrieves the total number of posts
func (s *postService) CountPosts(status *domain.PostStatus) (int, error) {
	return s.repo.Count(status)
}

// CountPostsByTag retrieves the total number of posts with a specific tag
func (s *postService) CountPostsByTag(tag string, status *domain.PostStatus) (int, error) {
	if tag == "" {
		return 0, fmt.Errorf("tag cannot be empty")
	}
	return s.repo.CountByTag(tag, status)
}

// SearchPosts retrieves posts matching the search query (for admin panel)
func (s *postService) SearchPosts(query string, status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
	if query == "" {
		return s.repo.FindAll(status, limit, offset)
	}
	return s.repo.Search(query, status, limit, offset)
}

// CountSearchPosts retrieves the count of posts matching the search query
func (s *postService) CountSearchPosts(query string, status *domain.PostStatus) (int, error) {
	if query == "" {
		return s.repo.Count(status)
	}
	return s.repo.CountSearch(query, status)
}

// SearchPublishedPosts searches published posts (for public pages)
func (s *postService) SearchPublishedPosts(query string, limit, offset int) ([]*domain.Post, error) {
	status := domain.PostStatusPublished
	if query == "" {
		return s.repo.FindAll(&status, limit, offset)
	}
	return s.repo.Search(query, &status, limit, offset)
}

// CountSearchPublishedPosts retrieves the count of search results for published posts
func (s *postService) CountSearchPublishedPosts(query string) (int, error) {
	status := domain.PostStatusPublished
	if query == "" {
		return s.repo.Count(&status)
	}
	return s.repo.CountSearch(query, &status)
}

// GetPinnedPosts retrieves pinned published posts (for public pages)
func (s *postService) GetPinnedPosts() ([]*domain.Post, error) {
	return s.repo.FindPinnedPublished()
}

// GetPublishedPostsBySlugs retrieves published posts for the given slugs.
func (s *postService) GetPublishedPostsBySlugs(slugs []string) ([]*domain.Post, error) {
	return s.repo.FindPublishedBySlugs(slugs)
}

// SetPinned sets the pinned status of a post
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
