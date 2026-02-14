package http

import (
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/capybara-translation/goblog/internal/domain"
)

// OGPMeta holds Open Graph Protocol metadata for a page
type OGPMeta struct {
	Title       string
	Description string
	URL         string
	Type        string // "website" or "article"
	SiteName    string
	ImageURL    string
	PublishedAt *time.Time
	UpdatedAt   *time.Time
	Tags        []string
}

// markdownImagePattern matches markdown image syntax: ![alt](url)
var markdownImagePattern = regexp.MustCompile(`!\[[^\]]*\]\(([^)]+)\)`)

// extractFirstImage extracts the first image URL from markdown content
func extractFirstImage(markdown string) string {
	matches := markdownImagePattern.FindStringSubmatch(markdown)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// absURL safely joins the base URL with a path
func (h *PublicHandlers) absURL(path string) string {
	u, err := url.JoinPath(h.baseURL, path)
	if err != nil {
		return h.baseURL + path
	}
	return u
}

// defaultOGP returns OGP metadata for non-article pages
func (h *PublicHandlers) defaultOGP(title, path, description string) *OGPMeta {
	return &OGPMeta{
		Title:       title,
		Description: description,
		URL:         h.absURL(path),
		Type:        "website",
		SiteName:    h.blogTitle,
	}
}

// postOGP returns OGP metadata for a post page
func (h *PublicHandlers) postOGP(post *domain.Post, path string) *OGPMeta {
	description := markdownExcerpt(post.Content, 160)
	imageURL := extractFirstImage(post.Content)

	// Make relative image URLs absolute
	if imageURL != "" && !strings.HasPrefix(imageURL, "http") {
		imageURL = h.absURL(imageURL)
	}

	return &OGPMeta{
		Title:       post.Title,
		Description: description,
		URL:         h.absURL(path),
		Type:        "article",
		SiteName:    h.blogTitle,
		ImageURL:    imageURL,
		PublishedAt: post.PublishedAt,
		UpdatedAt:   &post.UpdatedAt,
		Tags:        splitTags(post.Tags),
	}
}
