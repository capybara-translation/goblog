package markdown

import (
	"bytes"
	"regexp"
	"sync"

	"github.com/capybara-translation/goblog/internal/ogp"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

// OGPGetter defines the interface for retrieving OGP data
// This is a minimal interface that the markdown converter needs
type OGPGetter interface {
	Get(url string) *ogp.Data
}

// Converter is an interface for converting Markdown to HTML
type Converter interface {
	Convert(markdown string) (string, error)
}

type converter struct {
	policy    *bluemonday.Policy
	ogpGetter OGPGetter
}

var (
	instance *converter
	once     sync.Once
)

// NewConverter returns a singleton Converter (without OGP support)
func NewConverter() Converter {
	once.Do(func() {
		instance = &converter{
			policy: createPolicy(),
		}
	})
	return instance
}

// NewConverterWithOGP creates a new Converter with OGP link card support
func NewConverterWithOGP(ogpGetter OGPGetter) Converter {
	return &converter{
		policy:    createPolicy(),
		ogpGetter: ogpGetter,
	}
}

// createPolicy creates an HTML sanitization policy
func createPolicy() *bluemonday.Policy {
	policy := bluemonday.UGCPolicy()
	// Allow class attribute for syntax highlighting and link cards
	policy.AllowAttrs("class").Matching(bluemonday.SpaceSeparatedTokens).OnElements("code", "pre", "span", "div", "a", "svg", "path", "img")
	// Allow style attribute for code blocks (highlight colors)
	policy.AllowAttrs("style").OnElements("pre", "code", "span")
	// Allow only checkboxes for task lists (limited to type="checkbox" for security)
	policy.AllowAttrs("checked", "disabled").OnElements("input")
	policy.AllowAttrs("type").Matching(regexp.MustCompile(`^checkbox$`)).OnElements("input")
	// Allow class attribute for li elements in task lists
	policy.AllowAttrs("class").Matching(bluemonday.SpaceSeparatedTokens).OnElements("li", "ul")
	// Allow data-line attribute (for preview sync scrolling)
	policy.AllowAttrs("data-line").Matching(regexp.MustCompile(`^\d+$`)).Globally()
	// Allow attributes for link cards
	policy.AllowAttrs("target", "rel").OnElements("a")
	policy.AllowAttrs("alt").OnElements("img")
	// Browser-hint attributes added by imageRenderer. Constrain values to
	// the well-known token sets so a stray destination can't smuggle
	// anything surprising past the sanitizer. Matching uses MatchString
	// (unanchored), so the regexps must include ^...$ themselves.
	policy.AllowAttrs("loading").Matching(regexp.MustCompile(`^(lazy|eager|auto)$`)).OnElements("img")
	policy.AllowAttrs("decoding").Matching(regexp.MustCompile(`^(async|sync|auto)$`)).OnElements("img")
	// Allow SVG elements for link card icons
	policy.AllowElements("svg", "path")
	policy.AllowAttrs("viewBox", "width", "height", "fill").OnElements("svg")
	policy.AllowAttrs("d").OnElements("path")
	return policy
}

// createMarkdown creates a goldmark instance
func createMarkdown(src []byte, ogpGetter OGPGetter) goldmark.Markdown {
	li := newLineIndex(src)

	// Build renderer options
	rendererOpts := []renderer.Option{
		html.WithHardWraps(),
		html.WithXHTML(),
	}

	// Add link card renderer if OGP getter is available
	// Note: lower priority number = higher precedence in goldmark
	if ogpGetter != nil {
		rendererOpts = append(rendererOpts,
			renderer.WithNodeRenderers(
				util.Prioritized(NewLinkCardRenderer(ogpGetter, li), 10), // high precedence for link cards
			),
		)
	}

	// Add practical line renderer (for data-line attributes)
	rendererOpts = append(rendererOpts,
		renderer.WithNodeRenderers(
			util.Prioritized(&practicalLineRenderer{li: li}, 50),
		),
	)

	// Override <img> rendering to add browser-hint attributes
	// (loading, decoding). Priority < 1000 so this wins over goldmark's
	// default image renderer. See image_extension.go for why
	// fetchpriority is intentionally NOT emitted.
	rendererOpts = append(rendererOpts,
		renderer.WithNodeRenderers(
			util.Prioritized(newImageRenderer(), 10),
		),
	)

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
		goldmark.WithRendererOptions(rendererOpts...),
	)
}

// Convert converts Markdown to sanitized HTML
// Adds data-line attribute to each block element for preview sync scrolling
// If OGP getter is configured, standalone URLs are converted to link cards
func (c *converter) Convert(markdown string) (string, error) {
	src := []byte(markdown)
	md := createMarkdown(src, c.ogpGetter)

	var buf bytes.Buffer
	if err := md.Convert(src, &buf); err != nil {
		return "", err
	}

	// XSS protection: sanitize HTML
	return c.policy.Sanitize(buf.String()), nil
}
