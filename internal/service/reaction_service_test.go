package service

import (
	"errors"
	"testing"

	"github.com/capybara-translation/goblog/internal/domain"
)

type mockReactionPostLookup struct {
	post *domain.Post
	err  error
}

func (m *mockReactionPostLookup) GetPostBySlug(slug string) (*domain.Post, error) {
	return m.post, m.err
}

type mockReactionRepo struct {
	addCalled    bool
	removeCalled bool
	isActive     bool
	summaries    []*domain.PostReactionSummary
}

func (m *mockReactionRepo) FindSummariesByPostID(postID int64, visitorKey string) ([]*domain.PostReactionSummary, error) {
	return m.summaries, nil
}
func (m *mockReactionRepo) Add(postID, reactionTypeID int64, visitorKey string) error {
	m.addCalled = true
	return nil
}
func (m *mockReactionRepo) Remove(postID, reactionTypeID int64, visitorKey string) error {
	m.removeCalled = true
	return nil
}
func (m *mockReactionRepo) IsActiveType(reactionTypeID int64) (bool, error) {
	return m.isActive, nil
}

func publishedPost() *domain.Post {
	return &domain.Post{ID: 1, Slug: "p1", Status: domain.PostStatusPublished}
}

func TestReactionService_AddReaction_Success(t *testing.T) {
	repo := &mockReactionRepo{isActive: true, summaries: []*domain.PostReactionSummary{{ID: 1, Count: 1, Reacted: true}}}
	svc := NewReactionService(&mockReactionPostLookup{post: publishedPost()}, repo)

	got, err := svc.AddReaction("p1", 1, "visitor-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.addCalled {
		t.Fatal("expected repo.Add to be called")
	}
	if len(got) != 1 || !got[0].Reacted {
		t.Fatalf("unexpected summaries: %+v", got)
	}
}

func TestReactionService_AddReaction_PostNotFound(t *testing.T) {
	repo := &mockReactionRepo{isActive: true}
	svc := NewReactionService(&mockReactionPostLookup{post: nil}, repo)

	_, err := svc.AddReaction("missing", 1, "visitor-key")
	if !errors.Is(err, ErrReactionPostNotFound) {
		t.Fatalf("expected ErrReactionPostNotFound, got %v", err)
	}
	if repo.addCalled {
		t.Fatal("repo.Add must not be called for missing post")
	}
}

func TestReactionService_AddReaction_InactiveType(t *testing.T) {
	repo := &mockReactionRepo{isActive: false}
	svc := NewReactionService(&mockReactionPostLookup{post: publishedPost()}, repo)

	_, err := svc.AddReaction("p1", 999, "visitor-key")
	if !errors.Is(err, ErrReactionTypeInactive) {
		t.Fatalf("expected ErrReactionTypeInactive, got %v", err)
	}
	if repo.addCalled {
		t.Fatal("repo.Add must not be called for inactive type")
	}
}

func TestReactionService_AddReaction_EmptyVisitor(t *testing.T) {
	repo := &mockReactionRepo{isActive: true}
	svc := NewReactionService(&mockReactionPostLookup{post: publishedPost()}, repo)

	_, err := svc.AddReaction("p1", 1, "")
	if !errors.Is(err, ErrReactionVisitorEmpty) {
		t.Fatalf("expected ErrReactionVisitorEmpty, got %v", err)
	}
}

func TestReactionService_RemoveReaction_Success(t *testing.T) {
	repo := &mockReactionRepo{summaries: []*domain.PostReactionSummary{{ID: 1, Count: 0}}}
	svc := NewReactionService(&mockReactionPostLookup{post: publishedPost()}, repo)

	_, err := svc.RemoveReaction("p1", 1, "visitor-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.removeCalled {
		t.Fatal("expected repo.Remove to be called")
	}
}

func TestReactionService_RemoveReaction_EmptyVisitor(t *testing.T) {
	repo := &mockReactionRepo{}
	svc := NewReactionService(&mockReactionPostLookup{post: publishedPost()}, repo)

	_, err := svc.RemoveReaction("p1", 1, "")
	if !errors.Is(err, ErrReactionVisitorEmpty) {
		t.Fatalf("expected ErrReactionVisitorEmpty, got %v", err)
	}
	if repo.removeCalled {
		t.Fatal("repo.Remove must not be called with empty visitor")
	}
}

func TestReactionService_GetReactionsForPost_NoSlugLookup(t *testing.T) {
	repo := &mockReactionRepo{summaries: []*domain.PostReactionSummary{{ID: 1, Count: 3}}}
	// post lookup deliberately errors to prove GetReactionsForPost never calls it.
	svc := NewReactionService(&mockReactionPostLookup{err: errors.New("must not be called")}, repo)

	got, err := svc.GetReactionsForPost(1, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Count != 3 {
		t.Fatalf("unexpected summaries: %+v", got)
	}
}

func TestReactionService_GetPostReactions_PostNotFound(t *testing.T) {
	repo := &mockReactionRepo{}
	svc := NewReactionService(&mockReactionPostLookup{post: nil}, repo)

	_, err := svc.GetPostReactions("missing", "vk")
	if !errors.Is(err, ErrReactionPostNotFound) {
		t.Fatalf("expected ErrReactionPostNotFound, got %v", err)
	}
}
