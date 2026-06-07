package service

import (
	"errors"
	"testing"

	"github.com/capybara-translation/goblog/internal/domain"
)

type mockReactionPostLookup struct {
	post                    *domain.Post
	err                     error
	publishedBySlugsFn      func(slugs []string) ([]*domain.Post, error)
}

func (m *mockReactionPostLookup) GetPostBySlug(slug string) (*domain.Post, error) {
	return m.post, m.err
}

func (m *mockReactionPostLookup) GetPublishedPostsBySlugs(slugs []string) ([]*domain.Post, error) {
	if m.publishedBySlugsFn != nil {
		return m.publishedBySlugsFn(slugs)
	}
	return nil, nil
}

type mockReactionRepo struct {
	addCalled      bool
	removeCalled   bool
	isActive       bool
	summaries      []*domain.PostReactionSummary
	summariesByIDs map[int64][]*domain.PostReactionSummary
}

func (m *mockReactionRepo) FindSummariesByPostID(postID int64, visitorKey string) ([]*domain.PostReactionSummary, error) {
	return m.summaries, nil
}
func (m *mockReactionRepo) FindSummariesByPostIDs(postIDs []int64, visitorKey string) (map[int64][]*domain.PostReactionSummary, error) {
	if m.summariesByIDs != nil {
		return m.summariesByIDs, nil
	}
	return make(map[int64][]*domain.PostReactionSummary), nil
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

func reactionTestPost() *domain.Post {
	return &domain.Post{ID: 1, Slug: "p1", Status: domain.PostStatusPublished}
}

func TestReactionService_AddReaction_Success(t *testing.T) {
	repo := &mockReactionRepo{isActive: true, summaries: []*domain.PostReactionSummary{{ID: 1, Count: 1, Reacted: true}}}
	svc := NewReactionService(&mockReactionPostLookup{post: reactionTestPost()}, repo)

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
	svc := NewReactionService(&mockReactionPostLookup{post: reactionTestPost()}, repo)

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
	svc := NewReactionService(&mockReactionPostLookup{post: reactionTestPost()}, repo)

	_, err := svc.AddReaction("p1", 1, "")
	if !errors.Is(err, ErrReactionVisitorEmpty) {
		t.Fatalf("expected ErrReactionVisitorEmpty, got %v", err)
	}
}

func TestReactionService_RemoveReaction_Success(t *testing.T) {
	repo := &mockReactionRepo{summaries: []*domain.PostReactionSummary{{ID: 1, Count: 0}}}
	svc := NewReactionService(&mockReactionPostLookup{post: reactionTestPost()}, repo)

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
	svc := NewReactionService(&mockReactionPostLookup{post: reactionTestPost()}, repo)

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

func TestReactionService_AttachReactions(t *testing.T) {
	s1 := &domain.PostReactionSummary{ID: 1, Emoji: "👍", Count: 3, Reacted: false}
	s2 := &domain.PostReactionSummary{ID: 2, Emoji: "❤️", Count: 0, Reacted: false}

	repo := &mockReactionRepo{
		summariesByIDs: map[int64][]*domain.PostReactionSummary{
			1: {s1, s2},
			2: {},
		},
	}
	svc := NewReactionService(&mockReactionPostLookup{}, repo)

	posts := []*domain.Post{
		{ID: 1, Slug: "p1"},
		{ID: 2, Slug: "p2"},
	}
	if err := svc.AttachReactions(posts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts[0].Reactions) != 2 {
		t.Fatalf("post 1 expected 2 reactions, got %d", len(posts[0].Reactions))
	}
	if posts[0].Reactions[0].Count != 3 {
		t.Errorf("post 1 first reaction count expected 3, got %d", posts[0].Reactions[0].Count)
	}
	if len(posts[1].Reactions) != 0 {
		t.Fatalf("post 2 expected 0 reactions, got %d", len(posts[1].Reactions))
	}
}

func TestReactionService_AttachReactions_Empty(t *testing.T) {
	repo := &mockReactionRepo{}
	svc := NewReactionService(&mockReactionPostLookup{}, repo)

	if err := svc.AttachReactions([]*domain.Post{}); err != nil {
		t.Fatalf("unexpected error for empty posts: %v", err)
	}
}

func TestReactionService_GetReactionsBySlugs(t *testing.T) {
	p1 := &domain.Post{ID: 10, Slug: "hello"}
	p2 := &domain.Post{ID: 20, Slug: "world"}
	s1 := &domain.PostReactionSummary{ID: 1, Emoji: "👍", Count: 5, Reacted: true}

	repo := &mockReactionRepo{
		summariesByIDs: map[int64][]*domain.PostReactionSummary{
			10: {s1},
			20: {},
		},
	}
	lookup := &mockReactionPostLookup{
		publishedBySlugsFn: func(slugs []string) ([]*domain.Post, error) {
			return []*domain.Post{p1, p2}, nil
		},
	}
	svc := NewReactionService(lookup, repo)

	got, err := svc.GetReactionsBySlugs([]string{"hello", "world"}, "visitor-A")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 slug entries, got %d", len(got))
	}
	if len(got["hello"]) != 1 || got["hello"][0].Count != 5 {
		t.Errorf("unexpected summaries for 'hello': %+v", got["hello"])
	}
	if len(got["world"]) != 0 {
		t.Errorf("expected empty summaries for 'world', got %+v", got["world"])
	}
}

func TestReactionService_GetReactionsBySlugs_Empty(t *testing.T) {
	repo := &mockReactionRepo{}
	svc := NewReactionService(&mockReactionPostLookup{}, repo)

	got, err := svc.GetReactionsBySlugs([]string{}, "vk")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty map, got %d entries", len(got))
	}
}
