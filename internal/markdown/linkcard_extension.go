package markdown

import (
	"html"
	"net/url"
	"strings"

	"github.com/capybara-translation/goblog/internal/ogp"
	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// LinkCardRenderer renders standalone URLs as link cards
type LinkCardRenderer struct {
	ogpGetter OGPGetter
	li        *lineIndex
}

// NewLinkCardRenderer creates a new link card renderer
func NewLinkCardRenderer(ogpGetter OGPGetter, li *lineIndex) *LinkCardRenderer {
	return &LinkCardRenderer{
		ogpGetter: ogpGetter,
		li:        li,
	}
}

// RegisterFuncs registers the paragraph renderer
func (r *LinkCardRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	// Override paragraph rendering to check for standalone URLs
	reg.Register(gast.KindParagraph, r.renderParagraph)
}

// renderParagraph renders a paragraph, converting standalone URLs to link cards
func (r *LinkCardRenderer) renderParagraph(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		// Check if this was rendered as a link card
		if _, ok := node.AttributeString("data-linkcard"); ok {
			return gast.WalkContinue, nil
		}
		_, _ = w.WriteString("</p>")
		return gast.WalkContinue, nil
	}

	// Check if this paragraph contains only a standalone URL
	if url := r.extractStandaloneURL(node, source); url != "" {
		// Mark this node as a link card
		node.SetAttributeString("data-linkcard", []byte("true"))
		// Render as link card
		r.renderLinkCard(w, url)
		// Skip children since we've rendered the content
		return gast.WalkSkipChildren, nil
	}

	// Regular paragraph with data-line attribute
	_, _ = w.WriteString("<p" + dataLineAttr(r.li, node) + ">")
	return gast.WalkContinue, nil
}

// extractStandaloneURL checks if the paragraph contains only a single URL
// Returns the URL if it's a standalone URL, empty string otherwise
func (r *LinkCardRenderer) extractStandaloneURL(node gast.Node, source []byte) string {
	// Must have exactly one child
	if node.ChildCount() != 1 {
		return ""
	}

	child := node.FirstChild()

	// Check for AutoLink (e.g., https://example.com automatically linkified)
	if autoLink, ok := child.(*gast.AutoLink); ok {
		return string(autoLink.URL(source))
	}

	// Check for Link node (e.g., <https://example.com> or [text](url) where text == url)
	if link, ok := child.(*gast.Link); ok {
		linkURL := string(link.Destination)

		// Check if link text equals the URL (standalone URL case)
		if link.ChildCount() == 1 {
			if textNode, ok := link.FirstChild().(*gast.Text); ok {
				linkText := string(textNode.Segment.Value(source))
				// Only treat as standalone if text matches URL
				if linkText == linkURL {
					return linkURL
				}
			}
		}
	}

	// Check for Text node that looks like a URL (handles edge cases)
	if textNode, ok := child.(*gast.Text); ok {
		text := strings.TrimSpace(string(textNode.Segment.Value(source)))
		if isValidURL(text) {
			return text
		}
	}

	return ""
}

// isValidURL checks if a string is a valid HTTP/HTTPS URL
func isValidURL(s string) bool {
	if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
		return false
	}

	u, err := url.Parse(s)
	if err != nil {
		return false
	}

	return u.Host != ""
}

// renderLinkCard renders a URL as a link card
func (r *LinkCardRenderer) renderLinkCard(w util.BufWriter, targetURL string) {
	// Get OGP data
	var data *ogp.Data
	if r.ogpGetter != nil {
		data = r.ogpGetter.Get(targetURL)
	}

	// If no service or fetch failed, create minimal data
	if data == nil {
		data = &ogp.Data{
			URL:   targetURL,
			Title: targetURL,
		}
	}

	// Extract domain for display
	domain := targetURL
	if u, err := url.Parse(targetURL); err == nil {
		domain = u.Host
	}

	// Escape all user-provided content for XSS protection
	escapedURL := html.EscapeString(targetURL)
	escapedTitle := html.EscapeString(data.Title)
	escapedDescription := html.EscapeString(data.Description)
	escapedDomain := html.EscapeString(domain)

	// Build HTML
	_, _ = w.WriteString(`<a href="`)
	_, _ = w.WriteString(escapedURL)
	_, _ = w.WriteString(`" class="link-card" target="_blank" rel="noopener noreferrer">`)

	// Content section
	_, _ = w.WriteString(`<div class="link-card-content">`)

	// Title
	_, _ = w.WriteString(`<div class="link-card-title">`)
	_, _ = w.WriteString(escapedTitle)
	_, _ = w.WriteString(`</div>`)

	// Description (if available)
	if data.Description != "" {
		_, _ = w.WriteString(`<div class="link-card-description">`)
		_, _ = w.WriteString(escapedDescription)
		_, _ = w.WriteString(`</div>`)
	}

	// Domain with link icon
	_, _ = w.WriteString(`<div class="link-card-domain">`)
	_, _ = w.WriteString(`<svg class="link-card-icon" viewBox="0 0 24 24" width="14" height="14" fill="currentColor">`)
	_, _ = w.WriteString(`<path d="M3.9 12c0-1.71 1.39-3.1 3.1-3.1h4V7H7c-2.76 0-5 2.24-5 5s2.24 5 5 5h4v-1.9H7c-1.71 0-3.1-1.39-3.1-3.1zM8 13h8v-2H8v2zm9-6h-4v1.9h4c1.71 0 3.1 1.39 3.1 3.1s-1.39 3.1-3.1 3.1h-4V17h4c2.76 0 5-2.24 5-5s-2.24-5-5-5z"/>`)
	_, _ = w.WriteString(`</svg>`)
	_, _ = w.WriteString(escapedDomain)
	_, _ = w.WriteString(`</div>`)

	_, _ = w.WriteString(`</div>`)

	// Image (if available, prefer locally cached image)
	if data.HasImage() {
		imageURL := data.ImageURL
		if data.LocalImagePath != "" {
			imageURL = data.LocalImagePath
		}
		escapedImageURL := html.EscapeString(imageURL)
		_, _ = w.WriteString(`<img class="link-card-image" src="`)
		_, _ = w.WriteString(escapedImageURL)
		_, _ = w.WriteString(`" alt="" loading="lazy">`)
	}

	_, _ = w.WriteString(`</a>`)
}

// LinkCardExtension is a Goldmark extension for link cards
type LinkCardExtension struct {
	OGPGetter OGPGetter
}

// Extend implements goldmark.Extender
func (e *LinkCardExtension) Extend(m goldmark.Markdown) {
	// This extension requires source bytes, which we don't have here
	// The actual rendering is done in the converter by using NewLinkCardRenderer directly
}

// CreateLinkCardRenderer creates a link card renderer for use in the converter
func CreateLinkCardRenderer(ogpGetter OGPGetter, src []byte) renderer.NodeRenderer {
	return NewLinkCardRenderer(ogpGetter, newLineIndex(src))
}
