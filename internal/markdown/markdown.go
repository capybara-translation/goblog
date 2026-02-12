package markdown

import (
	"bytes"
	"regexp"
	"sync"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

// Converter is an interface for converting Markdown to HTML
type Converter interface {
	Convert(markdown string) (string, error)
}

type converter struct {
	policy *bluemonday.Policy
}

var (
	instance *converter
	once     sync.Once
)

// NewConverter returns a singleton Converter
func NewConverter() Converter {
	once.Do(func() {
		instance = &converter{
			policy: createPolicy(),
		}
	})
	return instance
}

// createPolicy creates an HTML sanitization policy
func createPolicy() *bluemonday.Policy {
	policy := bluemonday.UGCPolicy()
	// Allow class attribute for syntax highlighting
	policy.AllowAttrs("class").Matching(bluemonday.SpaceSeparatedTokens).OnElements("code", "pre", "span", "div")
	// Allow style attribute for code blocks (highlight colors)
	policy.AllowAttrs("style").OnElements("pre", "code", "span")
	// Allow only checkboxes for task lists (limited to type="checkbox" for security)
	policy.AllowAttrs("checked", "disabled").OnElements("input")
	policy.AllowAttrs("type").Matching(regexp.MustCompile(`^checkbox$`)).OnElements("input")
	// Allow class attribute for li elements in task lists
	policy.AllowAttrs("class").Matching(bluemonday.SpaceSeparatedTokens).OnElements("li", "ul")
	// Allow data-line attribute (for preview sync scrolling)
	policy.AllowAttrs("data-line").Matching(regexp.MustCompile(`^\d+$`)).Globally()
	return policy
}

// createMarkdown creates a goldmark instance
func createMarkdown(src []byte) goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(
			extension.GFM, // tables, strikethrough, task lists, etc.
			highlighting.NewHighlighting(
				highlighting.WithStyle("monokai"), // syntax highlighting theme
			),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
			// Custom renderer that adds data-line attribute
			renderer.WithNodeRenderers(
				util.Prioritized(&practicalLineRenderer{
					li: newLineIndex(src),
				}, 50), // lower priority than highlighting
			),
		),
	)
}

// Convert converts Markdown to sanitized HTML
// Adds data-line attribute to each block element for preview sync scrolling
func (c *converter) Convert(markdown string) (string, error) {
	src := []byte(markdown)
	md := createMarkdown(src)

	var buf bytes.Buffer
	if err := md.Convert(src, &buf); err != nil {
		return "", err
	}

	// XSS protection: sanitize HTML
	return c.policy.Sanitize(buf.String()), nil
}
