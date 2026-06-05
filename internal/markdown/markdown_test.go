package markdown

import (
	"strings"
	"testing"

	"github.com/capybara-translation/goblog/internal/ogp"
)

func TestConvert_BasicMarkdown(t *testing.T) {
	converter := NewConverter()

	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:     "Heading conversion",
			input:    "# Hello World",
			contains: []string{"<h1", "Hello World", "</h1>"},
		},
		{
			name:     "Paragraph conversion",
			input:    "This is a paragraph.",
			contains: []string{"<p ", "This is a paragraph.", "</p>"},
		},
		{
			name:     "Bold conversion",
			input:    "**bold text**",
			contains: []string{"<strong>", "bold text", "</strong>"},
		},
		{
			name:     "Italic conversion",
			input:    "*italic text*",
			contains: []string{"<em>", "italic text", "</em>"},
		},
		{
			name:     "Link conversion",
			input:    "[Google](https://google.com)",
			contains: []string{`href="https://google.com"`, "Google"},
		},
		{
			name:     "Inline code",
			input:    "Use `fmt.Println()` function",
			contains: []string{"<code>", "fmt.Println()", "</code>"},
		},
		{
			name:     "List conversion",
			input:    "- Item 1\n- Item 2\n- Item 3",
			contains: []string{"<ul ", "<li ", "Item 1", "Item 2", "Item 3", "</li>", "</ul>"},
		},
		{
			name:     "Numbered list",
			input:    "1. First\n2. Second\n3. Third",
			contains: []string{"<ol ", "<li ", "First", "Second", "Third", "</li>", "</ol>"},
		},
		{
			name:     "Blockquote",
			input:    "> This is a quote",
			contains: []string{"<blockquote ", "This is a quote", "</blockquote>"},
		},
		{
			name:     "Japanese text",
			input:    "# こんにちは世界\n\nこれは日本語のテキストです。",
			contains: []string{"<h1", "こんにちは世界", "</h1>", "<p ", "これは日本語のテキストです。", "</p>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := converter.Convert(tt.input)
			if err != nil {
				t.Fatalf("Convert() error = %v", err)
			}
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Convert() = %q, want to contain %q", result, expected)
				}
			}
		})
	}
}

func TestConvert_GFMFeatures(t *testing.T) {
	converter := NewConverter()

	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:     "Table conversion",
			input:    "| Header 1 | Header 2 |\n|----------|----------|\n| Cell 1   | Cell 2   |",
			contains: []string{"<table ", "<th ", "Header 1", "Header 2", "<td ", "Cell 1", "Cell 2", "</table>"},
		},
		{
			name:     "Strikethrough",
			input:    "~~deleted text~~",
			contains: []string{"<del>", "deleted text", "</del>"},
		},
		{
			name:     "Task list (checked)",
			input:    "- [x] Done task",
			contains: []string{"<li ", "Done task", "<input", "type=\"checkbox\"", "checked"},
		},
		{
			name:     "Task list (unchecked)",
			input:    "- [ ] Todo task",
			contains: []string{"<li ", "Todo task", "<input", "type=\"checkbox\""},
		},
		{
			name:     "Auto link",
			input:    "Visit https://example.com for more info",
			contains: []string{`href="https://example.com"`, "https://example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := converter.Convert(tt.input)
			if err != nil {
				t.Fatalf("Convert() error = %v", err)
			}
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Convert() = %q, want to contain %q", result, expected)
				}
			}
		})
	}
}

func TestConvert_SyntaxHighlighting(t *testing.T) {
	converter := NewConverter()

	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:     "Go code block",
			input:    "```go\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```",
			contains: []string{"<pre", "<code", "func", "main"},
		},
		{
			name:     "JavaScript code block",
			input:    "```javascript\nconst x = 1;\nconsole.log(x);\n```",
			contains: []string{"<pre", "<code", "const", "console"},
		},
		{
			name:     "Code block without language",
			input:    "```\nplain code\n```",
			contains: []string{"<pre", "<code", "plain code"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := converter.Convert(tt.input)
			if err != nil {
				t.Fatalf("Convert() error = %v", err)
			}
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Convert() = %q, want to contain %q", result, expected)
				}
			}
		})
	}

	// Verify syntax highlighting generates style attributes
	t.Run("Syntax highlighting generates style attribute", func(t *testing.T) {
		input := "```go\nfunc main() {}\n```"
		result, err := converter.Convert(input)
		if err != nil {
			t.Fatalf("Convert() error = %v", err)
		}
		// highlighting extension adds style attribute to span elements
		if !strings.Contains(result, "style=") {
			t.Errorf("Expected style attribute if syntax highlighting is enabled, got: %q", result)
		}
	})
}

func TestConvert_XSSSanitization(t *testing.T) {
	converter := NewConverter()

	tests := []struct {
		name        string
		input       string
		notContains []string
	}{
		{
			name:        "Remove script tag",
			input:       "<script>alert('xss')</script>",
			notContains: []string{"<script>", "alert("},
		},
		{
			name:        "Remove onclick event",
			input:       `<a href="#" onclick="alert('xss')">Click</a>`,
			notContains: []string{"onclick"},
		},
		{
			name:        "Remove javascript protocol",
			input:       `<a href="javascript:alert('xss')">Click</a>`,
			notContains: []string{"javascript:"},
		},
		{
			name:        "Remove iframe tag",
			input:       `<iframe src="https://evil.com"></iframe>`,
			notContains: []string{"<iframe"},
		},
		{
			name:        "Remove onerror on img tag",
			input:       `<img src="x" onerror="alert('xss')">`,
			notContains: []string{"onerror"},
		},
		{
			name:        "Remove style tag (root)",
			input:       `<style>body { display: none; }</style>`,
			notContains: []string{"<style>"},
		},
		{
			name:        "Script in Markdown code (escaped)",
			input:       "```html\n<script>alert('xss')</script>\n```",
			notContains: []string{"<script>alert"},
		},
		{
			name:        "Remove input type=text",
			input:       `<input type="text" value="phishing">`,
			notContains: []string{`type="text"`},
		},
		{
			name:        "Remove input type=password",
			input:       `<input type="password">`,
			notContains: []string{`type="password"`},
		},
		{
			name:        "Remove input type=hidden",
			input:       `<input type="hidden" value="secret">`,
			notContains: []string{`type="hidden"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := converter.Convert(tt.input)
			if err != nil {
				t.Fatalf("Convert() error = %v", err)
			}
			for _, notExpected := range tt.notContains {
				if strings.Contains(result, notExpected) {
					t.Errorf("Convert() = %q, should NOT contain %q (XSS vulnerability)", result, notExpected)
				}
			}
		})
	}
}

func TestConvert_AllowedHTMLTags(t *testing.T) {
	converter := NewConverter()

	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:     "img tag is allowed (src attribute only)",
			input:    `![alt text](https://example.com/image.png)`,
			contains: []string{"<img", `src="https://example.com/image.png"`, `alt="alt text"`},
		},
		{
			name:     "a tag is allowed",
			input:    `[link](https://example.com)`,
			contains: []string{"<a", `href="https://example.com"`, "link", "</a>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := converter.Convert(tt.input)
			if err != nil {
				t.Fatalf("Convert() error = %v", err)
			}
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Convert() = %q, want to contain %q", result, expected)
				}
			}
		})
	}
}

func TestConvert_Singleton(t *testing.T) {
	c1 := NewConverter()
	c2 := NewConverter()

	if c1 != c2 {
		t.Error("NewConverter() should return the same singleton instance")
	}
}

func TestConvert_EmptyInput(t *testing.T) {
	converter := NewConverter()

	result, err := converter.Convert("")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	if result != "" {
		t.Errorf("Convert(\"\") = %q, want empty string", result)
	}
}

func TestConvert_ComplexDocument(t *testing.T) {
	converter := NewConverter()

	input := `# プロジェクトの概要

このプロジェクトはGoで書かれています。

## 機能一覧

- 記事管理
- タグ機能
- **認証機能**

## コード例

` + "```go\n" + `package main

func main() {
    fmt.Println("Hello, World!")
}
` + "```\n" + `

| 機能 | ステータス |
|------|----------|
| 記事作成 | 完了 |
| タグ | 完了 |

> 注意: このプロジェクトは開発中です。

詳細は [ドキュメント](https://example.com/docs) を参照してください。
`

	result, err := converter.Convert(input)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	expectedElements := []string{
		"<h1", "プロジェクトの概要", "</h1>",
		"<h2", "機能一覧", "</h2>",
		"<ul ", "<li ", "記事管理", "</li>", "</ul>",
		"<strong>", "認証機能", "</strong>",
		"<pre", "<code",
		"<table ", "<th ", "機能", "ステータス", "<td ", "記事作成", "完了",
		"<blockquote ", "注意", "</blockquote>",
		`href="https://example.com/docs"`, "ドキュメント",
	}

	for _, expected := range expectedElements {
		if !strings.Contains(result, expected) {
			t.Errorf("Convert() should contain %q in result:\n%s", expected, result)
		}
	}
}

// mockOGPGetter implements OGPGetter for testing
type mockOGPGetter struct {
	data map[string]*ogp.Data
}

func newMockOGPGetter() *mockOGPGetter {
	return &mockOGPGetter{
		data: map[string]*ogp.Data{
			"https://example.com": {
				URL:         "https://example.com",
				Title:       "Example Title",
				Description: "Example Description",
				ImageURL:    "https://example.com/image.jpg",
				SiteName:    "example.com",
			},
			"https://example.com/no-image": {
				URL:         "https://example.com/no-image",
				Title:       "No Image Page",
				Description: "Page without image",
				SiteName:    "example.com",
			},
		},
	}
}

func (m *mockOGPGetter) Get(url string) *ogp.Data {
	if data, ok := m.data[url]; ok {
		return data
	}
	return &ogp.Data{
		URL:   url,
		Title: url,
	}
}

func TestConvert_LinkCard(t *testing.T) {
	ogpGetter := newMockOGPGetter()
	converter := NewConverterFor(ogpGetter, nil)

	tests := []struct {
		name        string
		input       string
		contains    []string
		notContains []string
	}{
		{
			name:  "Standalone URL becomes link card",
			input: "https://example.com",
			contains: []string{
				`class="link-card"`,
				`href="https://example.com"`,
				`class="link-card-title"`,
				"Example Title",
				`class="link-card-description"`,
				"Example Description",
				`class="link-card-image"`,
				`src="https://example.com/image.jpg"`,
			},
		},
		{
			name:  "Link card without image",
			input: "https://example.com/no-image",
			contains: []string{
				`class="link-card"`,
				"No Image Page",
			},
			notContains: []string{
				`class="link-card-image"`,
			},
		},
		{
			name:  "URL in text is not a link card",
			input: "Visit https://example.com for more info",
			contains: []string{
				`href="https://example.com"`,
			},
			notContains: []string{
				`class="link-card"`,
			},
		},
		{
			name:  "Markdown link is not a link card",
			input: "[Example](https://example.com)",
			contains: []string{
				`href="https://example.com"`,
				">Example<",
			},
			notContains: []string{
				`class="link-card"`,
			},
		},
		{
			name:  "Link card has target blank and rel noopener",
			input: "https://example.com",
			contains: []string{
				`target="_blank"`,
				`rel="noopener noreferrer`,
			},
		},
		{
			name:  "Link card shows domain",
			input: "https://example.com",
			contains: []string{
				`class="link-card-domain"`,
				"example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := converter.Convert(tt.input)
			if err != nil {
				t.Fatalf("Convert() error = %v", err)
			}
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Convert() = %q, want to contain %q", result, expected)
				}
			}
			for _, notExpected := range tt.notContains {
				if strings.Contains(result, notExpected) {
					t.Errorf("Convert() = %q, should NOT contain %q", result, notExpected)
				}
			}
		})
	}
}

func TestConvert_LinkCardXSS(t *testing.T) {
	// Test that OGP data is properly escaped
	ogpGetter := &mockOGPGetter{
		data: map[string]*ogp.Data{
			"https://example.com": {
				URL:         "https://example.com",
				Title:       `<script>alert('xss')</script>`,
				Description: `<img src=x onerror=alert('xss')>`,
				ImageURL:    `javascript:alert('xss')`,
				SiteName:    `<a onclick="alert('xss')">evil</a>`,
			},
		},
	}
	converter := NewConverterFor(ogpGetter, nil)

	result, err := converter.Convert("https://example.com")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	// Check that dangerous HTML tags are escaped (angle brackets become &lt; &gt;)
	// The key security concern is that raw HTML tags should not render
	notContains := []string{
		"<script>",   // Script tag should be escaped
		"<img src=x", // Img tag with onerror attack should be escaped
		"<a onclick", // Anchor tag with onclick should be escaped
	}

	for _, notExpected := range notContains {
		if strings.Contains(result, notExpected) {
			t.Errorf("Convert() = %q, should NOT contain %q (XSS vulnerability)", result, notExpected)
		}
	}

	// Check that the content is HTML-escaped (contains &lt; instead of <)
	mustContain := []string{
		"&lt;script&gt;", // Escaped script tag
		"&lt;img",        // Escaped img tag
	}

	for _, expected := range mustContain {
		if !strings.Contains(result, expected) {
			t.Errorf("Convert() = %q, should contain escaped %q", result, expected)
		}
	}

	// javascript: protocol should not be used as image src
	if strings.Contains(result, `src="javascript:`) {
		t.Errorf("Convert() = %q, should NOT contain javascript: protocol in src attribute", result)
	}

	// Verify the legitimate link-card-image element exists
	if !strings.Contains(result, `class="link-card-image"`) {
		t.Errorf("Convert() = %q, should contain link-card-image", result)
	}
}

func TestNewConverterFor(t *testing.T) {
	ogpGetter := newMockOGPGetter()

	// NewConverterFor should return a new instance each time (not singleton)
	c1 := NewConverterFor(ogpGetter, nil)
	c2 := NewConverterFor(ogpGetter, nil)

	if c1 == c2 {
		t.Error("NewConverterFor() should return different instances")
	}
}

func TestConvert_DataLineAttributes(t *testing.T) {
	converter := NewConverter()

	t.Run("data-line attribute remains after sanitization", func(t *testing.T) {
		input := "# Title\n\nParagraph"
		result, err := converter.Convert(input)
		if err != nil {
			t.Fatalf("Convert() error = %v", err)
		}

		// data-line attribute should exist
		if !strings.Contains(result, `data-line="`) {
			t.Errorf("Convert() should contain data-line attribute, got: %q", result)
		}
	})

	t.Run("data-line attribute is added to heading", func(t *testing.T) {
		input := "# Title"
		result, err := converter.Convert(input)
		if err != nil {
			t.Fatalf("Convert() error = %v", err)
		}

		if !strings.Contains(result, `<h1`) || !strings.Contains(result, `data-line="0"`) {
			t.Errorf("Convert() should contain h1 with data-line=\"0\", got: %q", result)
		}
	})

	t.Run("data-line attribute is added to paragraph", func(t *testing.T) {
		input := "# Title\n\nParagraph"
		result, err := converter.Convert(input)
		if err != nil {
			t.Fatalf("Convert() error = %v", err)
		}

		// Paragraph is on line 2 (0-indexed)
		if !strings.Contains(result, `<p`) || !strings.Contains(result, `data-line="2"`) {
			t.Errorf("Convert() should contain p with data-line=\"2\", got: %q", result)
		}
	})

	t.Run("Invalid data-line attribute is sanitized", func(t *testing.T) {
		// Direct HTML input (simulating XSS attack)
		input := `<p data-line="alert('xss')">text</p>`
		result, err := converter.Convert(input)
		if err != nil {
			t.Fatalf("Convert() error = %v", err)
		}

		// Invalid values are sanitized (only numbers allowed)
		if strings.Contains(result, `alert`) {
			t.Errorf("Convert() should sanitize invalid data-line values, got: %q", result)
		}
	})
}
