package markdown

import (
	"strings"
	"testing"
)

func TestConvert_BasicMarkdown(t *testing.T) {
	converter := NewConverter()

	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:     "見出し変換",
			input:    "# Hello World",
			contains: []string{"<h1", "Hello World", "</h1>"},
		},
		{
			name:     "段落変換",
			input:    "This is a paragraph.",
			contains: []string{"<p>", "This is a paragraph.", "</p>"},
		},
		{
			name:     "太字変換",
			input:    "**bold text**",
			contains: []string{"<strong>", "bold text", "</strong>"},
		},
		{
			name:     "斜体変換",
			input:    "*italic text*",
			contains: []string{"<em>", "italic text", "</em>"},
		},
		{
			name:     "リンク変換",
			input:    "[Google](https://google.com)",
			contains: []string{`href="https://google.com"`, "Google"},
		},
		{
			name:     "インラインコード",
			input:    "Use `fmt.Println()` function",
			contains: []string{"<code>", "fmt.Println()", "</code>"},
		},
		{
			name:     "リスト変換",
			input:    "- Item 1\n- Item 2\n- Item 3",
			contains: []string{"<ul>", "<li>", "Item 1", "Item 2", "Item 3", "</li>", "</ul>"},
		},
		{
			name:     "番号付きリスト",
			input:    "1. First\n2. Second\n3. Third",
			contains: []string{"<ol>", "<li>", "First", "Second", "Third", "</li>", "</ol>"},
		},
		{
			name:     "引用",
			input:    "> This is a quote",
			contains: []string{"<blockquote>", "This is a quote", "</blockquote>"},
		},
		{
			name:     "日本語テキスト",
			input:    "# こんにちは世界\n\nこれは日本語のテキストです。",
			contains: []string{"<h1", "こんにちは世界", "</h1>", "<p>", "これは日本語のテキストです。", "</p>"},
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
			name:     "テーブル変換",
			input:    "| Header 1 | Header 2 |\n|----------|----------|\n| Cell 1   | Cell 2   |",
			contains: []string{"<table>", "<th>", "Header 1", "Header 2", "<td>", "Cell 1", "Cell 2", "</table>"},
		},
		{
			name:     "取り消し線",
			input:    "~~deleted text~~",
			contains: []string{"<del>", "deleted text", "</del>"},
		},
		{
			name:     "タスクリスト（チェック済み）",
			input:    "- [x] Done task",
			contains: []string{"<li>", "Done task", "<input", "type=\"checkbox\"", "checked"},
		},
		{
			name:     "タスクリスト（未チェック）",
			input:    "- [ ] Todo task",
			contains: []string{"<li>", "Todo task", "<input", "type=\"checkbox\""},
		},
		{
			name:     "自動リンク",
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
			name:     "Goコードブロック",
			input:    "```go\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```",
			contains: []string{"<pre", "<code", "func", "main"},
		},
		{
			name:     "JavaScriptコードブロック",
			input:    "```javascript\nconst x = 1;\nconsole.log(x);\n```",
			contains: []string{"<pre", "<code", "const", "console"},
		},
		{
			name:     "言語指定なしコードブロック",
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
}

func TestConvert_XSSSanitization(t *testing.T) {
	converter := NewConverter()

	tests := []struct {
		name        string
		input       string
		notContains []string
	}{
		{
			name:        "scriptタグ除去",
			input:       "<script>alert('xss')</script>",
			notContains: []string{"<script>", "alert("},
		},
		{
			name:        "onclickイベント除去",
			input:       `<a href="#" onclick="alert('xss')">Click</a>`,
			notContains: []string{"onclick"},
		},
		{
			name:        "javascriptプロトコル除去",
			input:       `<a href="javascript:alert('xss')">Click</a>`,
			notContains: []string{"javascript:"},
		},
		{
			name:        "iframeタグ除去",
			input:       `<iframe src="https://evil.com"></iframe>`,
			notContains: []string{"<iframe"},
		},
		{
			name:        "imgタグのonerror除去",
			input:       `<img src="x" onerror="alert('xss')">`,
			notContains: []string{"onerror"},
		},
		{
			name:        "styleタグ除去（ルート）",
			input:       `<style>body { display: none; }</style>`,
			notContains: []string{"<style>"},
		},
		{
			name:        "Markdownコード内のscript（エスケープ）",
			input:       "```html\n<script>alert('xss')</script>\n```",
			notContains: []string{"<script>alert"},
		},
		{
			name:        "input type=text は除去",
			input:       `<input type="text" value="phishing">`,
			notContains: []string{`type="text"`},
		},
		{
			name:        "input type=password は除去",
			input:       `<input type="password">`,
			notContains: []string{`type="password"`},
		},
		{
			name:        "input type=hidden は除去",
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
			name:     "imgタグは許可（src属性のみ）",
			input:    `![alt text](https://example.com/image.png)`,
			contains: []string{"<img", `src="https://example.com/image.png"`, `alt="alt text"`},
		},
		{
			name:     "aタグは許可",
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
		"<ul>", "<li>", "記事管理", "</li>", "</ul>",
		"<strong>", "認証機能", "</strong>",
		"<pre", "<code",
		"<table>", "<th>", "機能", "ステータス", "<td>", "記事作成", "完了",
		"<blockquote>", "注意", "</blockquote>",
		`href="https://example.com/docs"`, "ドキュメント",
	}

	for _, expected := range expectedElements {
		if !strings.Contains(result, expected) {
			t.Errorf("Convert() should contain %q in result:\n%s", expected, result)
		}
	}
}
