package markdown

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// ========================================
// lineIndex tests
// ========================================

func TestNewLineIndex(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		expected []int
	}{
		{"Empty", "", []int{0}},
		{"1 line", "hello", []int{0}},
		{"2 lines", "hello\nworld", []int{0, 6}},
		{"3 lines", "a\nb\nc", []int{0, 2, 4}},
		{"Trailing newline only", "hello\n", []int{0}},
		{"Multiple lines trailing newline", "a\nb\n", []int{0, 2}},
		{"Japanese 2 lines", "あ\nい", []int{0, 4}}, // UTF-8: 3bytes + 1newline
		{"Contains empty line", "a\n\nb", []int{0, 2, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			li := newLineIndex([]byte(tt.src))
			if !reflect.DeepEqual(li.starts, tt.expected) {
				t.Errorf("newLineIndex(%q).starts = %v, want %v", tt.src, li.starts, tt.expected)
			}
		})
	}
}

func TestLineFromOffset(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		offset   int
		expected int
	}{
		{"offset 0", "a\nb\nc", 0, 0},
		{"Middle of line 0", "hello\nworld", 3, 0},
		{"End of line 0", "hello\nworld", 5, 0},
		{"Start of line 1", "hello\nworld", 6, 1},
		{"Middle of line 1", "hello\nworld", 8, 1},
		{"Negative offset", "a\nb", -1, 0},
		{"Out of range offset", "a\nb", 100, 1},
		{"Line 3", "a\nb\nc", 4, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			li := newLineIndex([]byte(tt.src))
			result := li.lineFromOffset(tt.offset)
			if result != tt.expected {
				t.Errorf("lineFromOffset(%d) = %d, want %d (src=%q, starts=%v)",
					tt.offset, result, tt.expected, tt.src, li.starts)
			}
		})
	}
}

// ========================================
// Helper functions
// ========================================

// renderWithDataLine converts Markdown to HTML using the dataline extension
func renderWithDataLine(src string) string {
	srcBytes := []byte(src)
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRendererOptions(
			renderer.WithNodeRenderers(
				util.Prioritized(&practicalLineRenderer{
					li: newLineIndex(srcBytes),
				}, 100),
			),
		),
	)

	var buf bytes.Buffer
	if err := md.Convert(srcBytes, &buf); err != nil {
		return "ERROR: " + err.Error()
	}
	return buf.String()
}

// ========================================
// Block element tests
// ========================================

func TestDataLineRenderer_Heading(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "h1",
			input:    "# Hello",
			expected: `<h1 data-line="0">Hello</h1>`,
		},
		{
			name:     "h2",
			input:    "## World",
			expected: `<h2 data-line="0">World</h2>`,
		},
		{
			name:     "h3",
			input:    "### Test",
			expected: `<h3 data-line="0">Test</h3>`,
		},
		{
			name:     "h4",
			input:    "#### Test",
			expected: `<h4 data-line="0">Test</h4>`,
		},
		{
			name:     "h5",
			input:    "##### Test",
			expected: `<h5 data-line="0">Test</h5>`,
		},
		{
			name:     "h6",
			input:    "###### Test",
			expected: `<h6 data-line="0">Test</h6>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderWithDataLine(tt.input)
			if result != tt.expected {
				t.Errorf("renderWithDataLine(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDataLineRenderer_Paragraph(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Single paragraph",
			input:    "Hello World",
			expected: `<p data-line="0">Hello World</p>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderWithDataLine(tt.input)
			if result != tt.expected {
				t.Errorf("renderWithDataLine(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDataLineRenderer_Blockquote(t *testing.T) {
	// Note: blockquote itself has data-line="0" because node.Lines() is empty
	// Inner paragraphs have correct line numbers
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Single quote",
			input:    "> Quote text",
			expected: `<blockquote data-line="0"><p data-line="0">Quote text</p></blockquote>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderWithDataLine(tt.input)
			if result != tt.expected {
				t.Errorf("renderWithDataLine(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDataLineRenderer_List(t *testing.T) {
	// Note: List and list items have data-line="0" because node.Lines() is empty
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "unordered list",
			input:    "- item 1\n- item 2",
			expected: `<ul data-line="0"><li data-line="0">item 1</li><li data-line="0">item 2</li></ul>`,
		},
		{
			name:     "ordered list",
			input:    "1. first\n2. second",
			expected: `<ol data-line="0"><li data-line="0">first</li><li data-line="0">second</li></ol>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderWithDataLine(tt.input)
			if result != tt.expected {
				t.Errorf("renderWithDataLine(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDataLineRenderer_ThematicBreak(t *testing.T) {
	// Note: ThematicBreak has data-line="0" because node.Lines() is empty
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "hr",
			input:    "---",
			expected: `<hr data-line="0"/>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderWithDataLine(tt.input)
			if result != tt.expected {
				t.Errorf("renderWithDataLine(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDataLineRenderer_CodeBlock(t *testing.T) {
	// Note: FencedCodeBlock is handled by highlighting extension, so only test indented code here
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Indented code block",
			input:    "    code line",
			expected: "<pre data-line=\"0\"><code>code line\n</code></pre>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderWithDataLine(tt.input)
			if result != tt.expected {
				t.Errorf("renderWithDataLine(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDataLineRenderer_Table(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple table",
			input:    "| H1 | H2 |\n|---|---|\n| A | B |",
			expected: `<table data-line="0"><thead data-line="0"><th data-line="0">H1</th><th data-line="0">H2</th></thead><tr data-line="0"><td data-line="2">A</td><td data-line="2">B</td></tr></table>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderWithDataLine(tt.input)
			if result != tt.expected {
				t.Errorf("renderWithDataLine(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// ========================================
// Complex document tests
// ========================================

func TestDataLineRenderer_ComplexDocument(t *testing.T) {
	// FencedCodeBlock is handled by highlighting extension, so this test uses indented code
	input := `# Title

Paragraph text.

- List item 1
- List item 2

> Quote

    code`

	// Container nodes (ul, blockquote, etc.) have data-line="0" because node.Lines() is empty
	// Child elements (p, etc.) have correct line numbers
	expected := `<h1 data-line="0">Title</h1>` +
		`<p data-line="2">Paragraph text.</p>` +
		`<ul data-line="0"><li data-line="0">List item 1</li><li data-line="0">List item 2</li></ul>` +
		`<blockquote data-line="0"><p data-line="7">Quote</p></blockquote>` +
		"<pre data-line=\"9\"><code>code\n</code></pre>"

	result := renderWithDataLine(input)
	if result != expected {
		t.Errorf("renderWithDataLine() = %q, want %q", result, expected)
	}
}

func TestDataLineRenderer_MultibyteCharacters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Japanese heading",
			input:    "# こんにちは",
			expected: `<h1 data-line="0">こんにちは</h1>`,
		},
		{
			name:     "Japanese multiple lines",
			input:    "あいうえお\n\nかきくけこ",
			expected: `<p data-line="0">あいうえお</p><p data-line="2">かきくけこ</p>`,
		},
		{
			name:     "Japanese and Chinese mixed",
			input:    "# 日本語タイトル\n\n这是中文段落",
			expected: `<h1 data-line="0">日本語タイトル</h1><p data-line="2">这是中文段落</p>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderWithDataLine(tt.input)
			if result != tt.expected {
				t.Errorf("renderWithDataLine(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// ========================================
// Edge case tests
// ========================================

func TestDataLineRenderer_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Empty document",
			input:    "",
			expected: "",
		},
		{
			name:     "Empty lines only",
			input:    "\n\n\n",
			expected: "",
		},
		{
			name:     "Content after consecutive empty lines",
			input:    "\n\n\n# Title",
			expected: `<h1 data-line="3">Title</h1>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderWithDataLine(tt.input)
			if result != tt.expected {
				t.Errorf("renderWithDataLine(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
