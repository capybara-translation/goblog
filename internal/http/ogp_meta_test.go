package http

import (
	"testing"
	"time"

	"github.com/capybara-translation/goblog/internal/domain"
)

func TestExtractFirstImage(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		expected string
	}{
		{"No image", "Hello world", ""},
		{"Simple image", "![alt](/uploads/img.png)", "/uploads/img.png"},
		{"Image with URL", "![photo](https://example.com/photo.jpg)", "https://example.com/photo.jpg"},
		{"Image with text before", "Some text\n![photo](https://example.com/photo.jpg)\nMore text", "https://example.com/photo.jpg"},
		{"Multiple images returns first", "![a](/first.png) text ![b](/second.png)", "/first.png"},
		{"Link not image", "[text](https://example.com)", ""},
		{"Empty alt", "![](/uploads/img.png)", "/uploads/img.png"},
		{"Empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFirstImage(tt.markdown)
			if result != tt.expected {
				t.Errorf("extractFirstImage() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestDefaultOGP(t *testing.T) {
	h := &PublicHandlers{
		blogTitle: "My Blog",
		baseURL:   "https://example.com",
	}

	ogp := h.defaultOGP("Test Title", "/page", "Test Description")

	if ogp.Title != "Test Title" {
		t.Errorf("Title = %q, want %q", ogp.Title, "Test Title")
	}
	if ogp.Description != "Test Description" {
		t.Errorf("Description = %q, want %q", ogp.Description, "Test Description")
	}
	if ogp.URL != "https://example.com/page" {
		t.Errorf("URL = %q, want %q", ogp.URL, "https://example.com/page")
	}
	if ogp.Type != "website" {
		t.Errorf("Type = %q, want %q", ogp.Type, "website")
	}
	if ogp.SiteName != "My Blog" {
		t.Errorf("SiteName = %q, want %q", ogp.SiteName, "My Blog")
	}
}

func TestPostOGP(t *testing.T) {
	h := &PublicHandlers{
		blogTitle: "My Blog",
		baseURL:   "https://example.com",
	}

	publishedAt := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	post := &domain.Post{
		Title:       "Test Post",
		Slug:        "test-post",
		Content:     "This is a test post.\n\n![image](/uploads/test.png)\n\nMore content here.",
		Tags:        "Go,Testing",
		PublishedAt: &publishedAt,
		UpdatedAt:   publishedAt,
	}

	ogp := h.postOGP(post, "/posts/test-post")

	if ogp.Title != "Test Post" {
		t.Errorf("Title = %q, want %q", ogp.Title, "Test Post")
	}
	if ogp.URL != "https://example.com/posts/test-post" {
		t.Errorf("URL = %q, want %q", ogp.URL, "https://example.com/posts/test-post")
	}
	if ogp.Type != "article" {
		t.Errorf("Type = %q, want %q", ogp.Type, "article")
	}
	if ogp.SiteName != "My Blog" {
		t.Errorf("SiteName = %q, want %q", ogp.SiteName, "My Blog")
	}
	// Relative image URL should be made absolute
	if ogp.ImageURL != "https://example.com/uploads/test.png" {
		t.Errorf("ImageURL = %q, want %q", ogp.ImageURL, "https://example.com/uploads/test.png")
	}
	if ogp.PublishedAt == nil {
		t.Error("PublishedAt should not be nil")
	}
	if ogp.UpdatedAt == nil {
		t.Error("UpdatedAt should not be nil")
	}
	if len(ogp.Tags) != 2 || ogp.Tags[0] != "Go" || ogp.Tags[1] != "Testing" {
		t.Errorf("Tags = %v, want [Go Testing]", ogp.Tags)
	}
}

func TestPostOGP_AbsoluteImageURL(t *testing.T) {
	h := &PublicHandlers{
		blogTitle: "My Blog",
		baseURL:   "https://example.com",
	}

	post := &domain.Post{
		Title:   "Test",
		Slug:    "test",
		Content: "![image](https://cdn.example.com/photo.jpg)",
	}

	ogp := h.postOGP(post, "/posts/test")

	// Absolute URL should remain unchanged
	if ogp.ImageURL != "https://cdn.example.com/photo.jpg" {
		t.Errorf("ImageURL = %q, want %q", ogp.ImageURL, "https://cdn.example.com/photo.jpg")
	}
}

func TestPostOGP_NoImage(t *testing.T) {
	h := &PublicHandlers{
		blogTitle: "My Blog",
		baseURL:   "https://example.com",
	}

	post := &domain.Post{
		Title:   "Test",
		Slug:    "test",
		Content: "No images in this post.",
	}

	ogp := h.postOGP(post, "/posts/test")

	if ogp.ImageURL != "" {
		t.Errorf("ImageURL = %q, want empty", ogp.ImageURL)
	}
}
