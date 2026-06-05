package markdown

import (
	"strings"
	"testing"

	"github.com/capybara-translation/goblog/internal/ogp"
)

func TestIsValidURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Valid HTTPS URL",
			input:    "https://example.com",
			expected: true,
		},
		{
			name:     "Valid HTTP URL",
			input:    "http://example.com",
			expected: true,
		},
		{
			name:     "Valid URL with path",
			input:    "https://example.com/path/to/page",
			expected: true,
		},
		{
			name:     "Valid URL with query",
			input:    "https://example.com/search?q=test",
			expected: true,
		},
		{
			name:     "Valid URL with port",
			input:    "https://example.com:8080/path",
			expected: true,
		},
		{
			name:     "Invalid - no scheme",
			input:    "example.com",
			expected: false,
		},
		{
			name:     "Invalid - ftp scheme",
			input:    "ftp://example.com",
			expected: false,
		},
		{
			name:     "Invalid - javascript scheme",
			input:    "javascript:alert('xss')",
			expected: false,
		},
		{
			name:     "Invalid - empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "Invalid - just text",
			input:    "not a url",
			expected: false,
		},
		{
			name:     "Invalid - no host",
			input:    "https://",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidURL(tt.input)
			if result != tt.expected {
				t.Errorf("isValidURL(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLinkCardRenderer_ExtractStandaloneURL(t *testing.T) {
	// Test cases for standalone URL extraction by running through the full converter
	ogpGetter := newMockOGPGetter()
	converter := NewConverterFor(ogpGetter, nil)

	tests := []struct {
		name           string
		input          string
		shouldLinkCard bool
	}{
		{
			name:           "Standalone URL on single line",
			input:          "https://example.com",
			shouldLinkCard: true,
		},
		{
			name:           "URL with angle brackets",
			input:          "<https://example.com>",
			shouldLinkCard: true,
		},
		{
			name:           "URL mixed with text",
			input:          "Check out https://example.com for more",
			shouldLinkCard: false,
		},
		{
			name:           "URL as markdown link",
			input:          "[Click here](https://example.com)",
			shouldLinkCard: false,
		},
		{
			name:           "URL in code block",
			input:          "```\nhttps://example.com\n```",
			shouldLinkCard: false,
		},
		{
			name:           "URL in inline code",
			input:          "`https://example.com`",
			shouldLinkCard: false,
		},
		{
			name:           "Multiple URLs on same line",
			input:          "https://example.com https://google.com",
			shouldLinkCard: false,
		},
		{
			name:           "URL followed by text on next line (same paragraph)",
			input:          "https://example.com\nSome text here",
			shouldLinkCard: false, // Single line break doesn't create new paragraph
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := converter.Convert(tt.input)
			if err != nil {
				t.Fatalf("Convert() error = %v", err)
			}

			hasLinkCard := strings.Contains(result, `class="link-card"`)
			if hasLinkCard != tt.shouldLinkCard {
				t.Errorf("Convert(%q) link card presence = %v, want %v\nResult: %s",
					tt.input, hasLinkCard, tt.shouldLinkCard, result)
			}
		})
	}
}

func TestLinkCardRenderer_HTMLStructure(t *testing.T) {
	ogpGetter := newMockOGPGetter()
	converter := NewConverterFor(ogpGetter, nil)

	result, err := converter.Convert("https://example.com")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	// Verify the HTML structure of link card
	requiredElements := []string{
		// Main container
		`<a href="https://example.com"`,
		`class="link-card"`,
		`target="_blank"`,

		// Content section
		`class="link-card-content"`,
		`class="link-card-title"`,
		`class="link-card-description"`,
		`class="link-card-domain"`,

		// SVG icon
		`<svg`,
		`class="link-card-icon"`,

		// Image
		`<img`,
		`class="link-card-image"`,
		`loading="lazy"`,
	}

	for _, element := range requiredElements {
		if !strings.Contains(result, element) {
			t.Errorf("Link card HTML should contain %q\nResult: %s", element, result)
		}
	}
}

func TestLinkCardRenderer_LocalImagePathPreferred(t *testing.T) {
	// When LocalImagePath is set, it should be used instead of ImageURL
	ogpGetter := &mockOGPGetter{
		data: map[string]*ogp.Data{
			"https://example.com/local-cached": {
				URL:            "https://example.com/local-cached",
				Title:          "Locally Cached Page",
				Description:    "Page with locally cached image",
				ImageURL:       "https://example.com/image.jpg",
				LocalImagePath: "/uploads/ogp-cache/abc123.jpg",
				SiteName:       "example.com",
			},
		},
	}
	converter := NewConverterFor(ogpGetter, nil)

	result, err := converter.Convert("https://example.com/local-cached")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	// Should use local path, not external URL
	if !strings.Contains(result, `/uploads/ogp-cache/abc123.jpg`) {
		t.Errorf("Link card should use LocalImagePath, got: %s", result)
	}
	if strings.Contains(result, `https://example.com/image.jpg`) {
		t.Errorf("Link card should NOT use external ImageURL when LocalImagePath is set, got: %s", result)
	}
}

func TestLinkCardRenderer_FallsBackToImageURL(t *testing.T) {
	// When LocalImagePath is empty, should fall back to ImageURL
	ogpGetter := &mockOGPGetter{
		data: map[string]*ogp.Data{
			"https://example.com/no-local": {
				URL:         "https://example.com/no-local",
				Title:       "No Local Cache",
				Description: "Page without locally cached image",
				ImageURL:    "https://example.com/fallback.jpg",
				SiteName:    "example.com",
			},
		},
	}
	converter := NewConverterFor(ogpGetter, nil)

	result, err := converter.Convert("https://example.com/no-local")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	if !strings.Contains(result, `https://example.com/fallback.jpg`) {
		t.Errorf("Link card should use ImageURL when LocalImagePath is empty, got: %s", result)
	}
}

func TestLinkCardRenderer_NoOGPGetter(t *testing.T) {
	// When OGP getter is nil, URLs should be rendered as regular links
	converter := NewConverter()

	result, err := converter.Convert("https://example.com")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	if strings.Contains(result, `class="link-card"`) {
		t.Error("Without OGP getter, URL should not become link card")
	}

	if !strings.Contains(result, `href="https://example.com"`) {
		t.Error("URL should still be converted to regular link")
	}
}
