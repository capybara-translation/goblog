package service

import (
	"errors"
	"fmt"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/capybara-translation/goblog/internal/repo"
)

// Sentinel errors mapped by the HTTP layer to status codes.
var (
	// ErrReactionPostNotFound is returned when the slug resolves to no
	// published post (missing or draft). -> 404
	ErrReactionPostNotFound = errors.New("reaction: post not found or not published")
	// ErrReactionTypeInactive is returned when the reaction type does not
	// exist or is deactivated. -> 400
	ErrReactionTypeInactive = errors.New("reaction: reaction type is not active")
	// ErrReactionVisitorEmpty is returned when the visitor key is empty. -> 400
	ErrReactionVisitorEmpty = errors.New("reaction: visitor key is empty")
)

// ReactionPostLookup is the narrow slice of PostService that ReactionService
// needs: resolving a slug to a published post (returns nil for missing/draft).
// service.PostService satisfies this.
type ReactionPostLookup interface {
	GetPostBySlug(slug string) (*domain.Post, error)
}

// ReactionService provides business logic for post reactions.
type ReactionService interface {
	// GetReactionsForPost returns summaries for an already-resolved post id.
	// Used by SSR (the post is already loaded) and skips slug resolution.
	GetReactionsForPost(postID int64, visitorKey string) ([]*domain.PostReactionSummary, error)

	// GetPostReactions resolves the slug (published only) and returns summaries.
	GetPostReactions(slug, visitorKey string) ([]*domain.PostReactionSummary, error)

	// AddReaction validates and records a reaction, returning fresh summaries.
	AddReaction(slug string, reactionTypeID int64, visitorKey string) ([]*domain.PostReactionSummary, error)

	// RemoveReaction removes a reaction, returning fresh summaries.
	RemoveReaction(slug string, reactionTypeID int64, visitorKey string) ([]*domain.PostReactionSummary, error)
}

type reactionService struct {
	posts ReactionPostLookup
	repo  repo.ReactionRepository
}

// NewReactionService creates a new ReactionService.
func NewReactionService(posts ReactionPostLookup, repo repo.ReactionRepository) ReactionService {
	return &reactionService{posts: posts, repo: repo}
}

// resolvePublished returns the published post for slug or ErrReactionPostNotFound.
func (s *reactionService) resolvePublished(slug string) (*domain.Post, error) {
	post, err := s.posts.GetPostBySlug(slug)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve post: %w", err)
	}
	if post == nil {
		return nil, ErrReactionPostNotFound
	}
	return post, nil
}

func (s *reactionService) GetReactionsForPost(postID int64, visitorKey string) ([]*domain.PostReactionSummary, error) {
	return s.repo.FindSummariesByPostID(postID, visitorKey)
}

func (s *reactionService) GetPostReactions(slug, visitorKey string) ([]*domain.PostReactionSummary, error) {
	post, err := s.resolvePublished(slug)
	if err != nil {
		return nil, err
	}
	return s.repo.FindSummariesByPostID(post.ID, visitorKey)
}

func (s *reactionService) AddReaction(slug string, reactionTypeID int64, visitorKey string) ([]*domain.PostReactionSummary, error) {
	if visitorKey == "" {
		return nil, ErrReactionVisitorEmpty
	}
	post, err := s.resolvePublished(slug)
	if err != nil {
		return nil, err
	}
	active, err := s.repo.IsActiveType(reactionTypeID)
	if err != nil {
		return nil, err
	}
	if !active {
		return nil, ErrReactionTypeInactive
	}
	if err := s.repo.Add(post.ID, reactionTypeID, visitorKey); err != nil {
		return nil, err
	}
	return s.repo.FindSummariesByPostID(post.ID, visitorKey)
}

func (s *reactionService) RemoveReaction(slug string, reactionTypeID int64, visitorKey string) ([]*domain.PostReactionSummary, error) {
	if visitorKey == "" {
		return nil, ErrReactionVisitorEmpty
	}
	post, err := s.resolvePublished(slug)
	if err != nil {
		return nil, err
	}
	// No active check on removal: a visitor may un-react even if the type was
	// later deactivated (deactivated types are simply hidden from summaries).
	if err := s.repo.Remove(post.ID, reactionTypeID, visitorKey); err != nil {
		return nil, err
	}
	return s.repo.FindSummariesByPostID(post.ID, visitorKey)
}
