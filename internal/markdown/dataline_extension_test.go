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
// lineIndex のテスト
// ========================================

func TestNewLineIndex(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		expected []int
	}{
		{"空", "", []int{0}},
		{"1行", "hello", []int{0}},
		{"2行", "hello\nworld", []int{0, 6}},
		{"3行", "a\nb\nc", []int{0, 2, 4}},
		{"末尾改行のみ", "hello\n", []int{0}},
		{"複数行末尾改行", "a\nb\n", []int{0, 2}},
		{"日本語2行", "あ\nい", []int{0, 4}}, // UTF-8: 3bytes + 1newline
		{"空行を含む", "a\n\nb", []int{0, 2, 3}},
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
		{"行0の途中", "hello\nworld", 3, 0},
		{"行0の末尾", "hello\nworld", 5, 0},
		{"行1の開始", "hello\nworld", 6, 1},
		{"行1の途中", "hello\nworld", 8, 1},
		{"負のoffset", "a\nb", -1, 0},
		{"範囲外offset", "a\nb", 100, 1},
		{"3行目", "a\nb\nc", 4, 2},
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
// ヘルパー関数
// ========================================

// renderWithDataLine はdataline extensionを使ってMarkdownをHTMLに変換する
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
// ブロック要素のテスト
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
			name:     "単一段落",
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
	// Note: blockquote自体は node.Lines() が空のため data-line="0" になる
	// 内部の段落は正しい行番号を持つ
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "単一引用",
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
	// Note: リストとリスト項目は node.Lines() が空のため data-line="0" になる
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
	// Note: ThematicBreak は node.Lines() が空のため data-line="0" になる
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
	// Note: FencedCodeBlockはhighlighting拡張に任せるため、ここではインデントコードのみテスト
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "インデントコードブロック",
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
// 複合ドキュメントのテスト
// ========================================

func TestDataLineRenderer_ComplexDocument(t *testing.T) {
	// FencedCodeBlockはhighlighting拡張に任せるため、このテストではインデントコードを使用
	input := `# Title

Paragraph text.

- List item 1
- List item 2

> Quote

    code`

	// コンテナノード（ul, blockquote等）は node.Lines() が空のため data-line="0" になる
	// 子要素（p等）は正しい行番号を持つ
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
			name:     "日本語見出し",
			input:    "# こんにちは",
			expected: `<h1 data-line="0">こんにちは</h1>`,
		},
		{
			name:     "日本語複数行",
			input:    "あいうえお\n\nかきくけこ",
			expected: `<p data-line="0">あいうえお</p><p data-line="2">かきくけこ</p>`,
		},
		{
			name:     "日中混在",
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
// エッジケースのテスト
// ========================================

func TestDataLineRenderer_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "空のドキュメント",
			input:    "",
			expected: "",
		},
		{
			name:     "空行のみ",
			input:    "\n\n\n",
			expected: "",
		},
		{
			name:     "連続する空行後のコンテンツ",
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
