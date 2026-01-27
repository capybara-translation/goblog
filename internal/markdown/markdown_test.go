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

// カーソルマーカー置換のテスト

func TestConvert_CursorMarker(t *testing.T) {
	converter := NewConverter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "段落の先頭にマーカー",
			input:    CursorMarker + "Hello World",
			expected: "<p><span id=\"cursor-line-marker\"></span>Hello World</p>\n",
		},
		{
			name:     "段落の途中にマーカー",
			input:    "Hello " + CursorMarker + "World",
			expected: "<p>Hello <span id=\"cursor-line-marker\"></span>World</p>\n",
		},
		{
			name:     "見出し内にマーカー",
			input:    "# Title " + CursorMarker + "Here",
			expected: "<h1 id=\"title-here\">Title <span id=\"cursor-line-marker\"></span>Here</h1>\n",
		},
		{
			name:     "リスト項目内にマーカー",
			input:    "- Item 1\n- " + CursorMarker + "Item 2\n- Item 3",
			expected: "<ul>\n<li>Item 1</li>\n<li><span id=\"cursor-line-marker\"></span>Item 2</li>\n<li>Item 3</li>\n</ul>\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := converter.Convert(tt.input)
			if err != nil {
				t.Fatalf("Convert() error = %v", err)
			}
			if result != tt.expected {
				t.Errorf("Convert() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestConvert_CursorMarker_NotContains(t *testing.T) {
	converter := NewConverter()

	t.Run("マーカーなしの場合はマーカースパンなし", func(t *testing.T) {
		result, err := converter.Convert("Hello World")
		if err != nil {
			t.Fatalf("Convert() error = %v", err)
		}
		if strings.Contains(result, CursorMarkerID) {
			t.Errorf("Convert() should NOT contain marker span when no marker in input")
		}
	})
}

func TestConvert_CursorMarker_OnlyOne(t *testing.T) {
	converter := NewConverter()

	// 複数のマーカーがあっても、最初の1つだけがスパンに置換される
	input := "Para 1 " + CursorMarker + "\n\nPara 2 " + CursorMarker
	result, err := converter.Convert(input)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	// マーカースパンは1つだけ
	count := strings.Count(result, CursorMarkerID)
	if count != 1 {
		t.Errorf("Convert() should contain exactly 1 marker span, got %d", count)
	}
}

func TestConvert_CursorMarker_XSSSanitization(t *testing.T) {
	converter := NewConverter()

	// マーカーがあってもXSSサニタイズは適用される
	input := "# Title\n\n" + CursorMarker + "<script>alert('xss')</script>"
	result, err := converter.Convert(input)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	if strings.Contains(result, "<script>") {
		t.Error("Convert() should sanitize XSS even with cursor marker")
	}

	// マーカースパンは残っているべき
	if !strings.Contains(result, CursorMarkerID) {
		t.Error("Convert() should contain marker span after sanitization")
	}
}

func TestConvert_CursorMarker_EmptyLine(t *testing.T) {
	converter := NewConverter()

	t.Run("段落間の空行にマーカーがある場合、不要なpタグが除去される", func(t *testing.T) {
		// 空行にマーカーを挿入（段落間）
		input := "# Title\n\n" + CursorMarker + "\n\nParagraph"
		result, err := converter.Convert(input)
		if err != nil {
			t.Fatalf("Convert() error = %v", err)
		}

		// マーカーは存在するべき
		if !strings.Contains(result, CursorMarkerID) {
			t.Errorf("Convert() should contain marker")
		}

		// <p><span id="cursor-line-marker"></span></p> のようなパターンは除去されるべき
		if strings.Contains(result, "<p><span id=\""+CursorMarkerID+"\"></span></p>") {
			t.Errorf("Convert() should not contain marker wrapped in empty p tag, got: %q", result)
		}
	})

	t.Run("コンテンツ行の先頭にマーカーがある場合、brタグは保持される", func(t *testing.T) {
		// 行の先頭にマーカーを挿入（前の行との改行がある）
		input := "aaa\n" + CursorMarker + "bbb"
		result, err := converter.Convert(input)
		if err != nil {
			t.Fatalf("Convert() error = %v", err)
		}

		// マーカーは存在するべき
		if !strings.Contains(result, CursorMarkerID) {
			t.Errorf("Convert() should contain marker")
		}

		// brタグは保持されるべき（HardWrapsが有効なため）
		if !strings.Contains(result, "<br") {
			t.Errorf("Convert() should contain br tag for hard wraps, got: %q", result)
		}
	})

	t.Run("空行にマーカーがある場合、段落が分離される", func(t *testing.T) {
		// 空行にマーカーを挿入（元の入力は aaa\n\nbbb で、空行にカーソルがあった）
		// フロントエンドがマーカーを挿入すると aaa\n<MARKER>\nbbb となる
		// これは段落分離として扱われるべき
		input := "aaa\n" + CursorMarker + "\nbbb"
		result, err := converter.Convert(input)
		if err != nil {
			t.Fatalf("Convert() error = %v", err)
		}

		markerSpan := `<span id="` + CursorMarkerID + `"></span>`

		// マーカーは存在するべき
		if !strings.Contains(result, CursorMarkerID) {
			t.Errorf("Convert() should contain marker")
		}

		// 段落が分離されるべき（<p>aaa</p> と <p>bbb</p>）
		if !strings.Contains(result, "<p>aaa</p>") {
			t.Errorf("Convert() should separate paragraphs, got: %q", result)
		}
		if !strings.Contains(result, "<p>bbb</p>") {
			t.Errorf("Convert() should have separate paragraph for bbb, got: %q", result)
		}

		// マーカーは段落間にあるべき（<p>タグで囲まれていない）
		if strings.Contains(result, "<p>"+markerSpan) || strings.Contains(result, markerSpan+"</p>") {
			// ただし <p>...</p> の中にマーカーがない場合はOK
			if strings.Contains(result, "<p>"+markerSpan+"</p>") {
				t.Errorf("Convert() should not have marker wrapped in p tag, got: %q", result)
			}
		}

		// 段落間なので<br>は出力されない
		if strings.Contains(result, "<br") {
			t.Errorf("Convert() should not contain br between paragraphs, got: %q", result)
		}
	})

	t.Run("3連続改行でも段落が分離される", func(t *testing.T) {
		// 3連続改行（2つの空行）の最初にカーソルがある場合
		input := "aaa\n" + CursorMarker + "\n\nbbb"
		result, err := converter.Convert(input)
		if err != nil {
			t.Fatalf("Convert() error = %v", err)
		}

		t.Logf("Input: %q", input)
		t.Logf("Result: %q", result)

		// 段落が分離されるべき
		if !strings.Contains(result, "<p>aaa</p>") {
			t.Errorf("Convert() should separate paragraphs, got: %q", result)
		}
		if !strings.Contains(result, "<p>bbb</p>") {
			t.Errorf("Convert() should have separate paragraph for bbb, got: %q", result)
		}

		// マーカーは存在するべき
		if !strings.Contains(result, CursorMarkerID) {
			t.Errorf("Convert() should contain marker")
		}
	})
}

func TestConvert_CursorMarker_InCodeBlock(t *testing.T) {
	converter := NewConverter()

	t.Run("コードブロック内の空行にマーカーがある場合、前処理がスキップされる", func(t *testing.T) {
		// コードブロック内の空行にマーカー
		input := "```go\nfunc main() {\n" + CursorMarker + "\n}\n```"
		result, err := converter.Convert(input)
		if err != nil {
			t.Fatalf("Convert() error = %v", err)
		}

		t.Logf("Input: %q", input)
		t.Logf("Result: %q", result)

		// マーカーは存在するべき
		if !strings.Contains(result, CursorMarkerID) {
			t.Errorf("Convert() should contain marker")
		}

		// コードブロックが正しくレンダリングされるべき
		if !strings.Contains(result, "<pre") {
			t.Errorf("Convert() should contain pre tag")
		}
		// シンタックスハイライトにより "func" と "main" が別々のspanに入る可能性があるため、それぞれを確認
		if !strings.Contains(result, "func") {
			t.Errorf("Convert() should contain 'func' keyword")
		}
		if !strings.Contains(result, "main") {
			t.Errorf("Convert() should contain 'main' identifier")
		}
	})

	t.Run("コードブロック外の空行にマーカーがある場合、前処理が適用される", func(t *testing.T) {
		// コードブロックとは別の場所にマーカー
		input := "aaa\n" + CursorMarker + "\nbbb\n\n```go\ncode\n```"
		result, err := converter.Convert(input)
		if err != nil {
			t.Fatalf("Convert() error = %v", err)
		}

		// 段落が分離されるべき
		if !strings.Contains(result, "<p>aaa</p>") {
			t.Errorf("Convert() should separate paragraphs, got: %q", result)
		}
		if !strings.Contains(result, "<p>bbb</p>") {
			t.Errorf("Convert() should have separate paragraph for bbb, got: %q", result)
		}
	})
}

func TestIsMarkerInCodeBlock(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "マーカーがコードブロック内にある",
			input:    "text\n```\ncode" + CursorMarker + "\n```\nmore",
			expected: true,
		},
		{
			name:     "マーカーがコードブロック外にある",
			input:    "text" + CursorMarker + "\n```\ncode\n```\nmore",
			expected: false,
		},
		{
			name:     "マーカーがコードブロックの後にある",
			input:    "```\ncode\n```\ntext" + CursorMarker,
			expected: false,
		},
		{
			name:     "コードブロックが複数ある場合、最初のコードブロック内",
			input:    "```\n" + CursorMarker + "\n```\n\n```\ncode\n```",
			expected: true,
		},
		{
			name:     "コードブロックが複数ある場合、2番目のコードブロック内",
			input:    "```\ncode\n```\n\n```\n" + CursorMarker + "\n```",
			expected: true,
		},
		{
			name:     "コードブロックが複数ある場合、ブロック間にある",
			input:    "```\ncode\n```\n" + CursorMarker + "\n```\ncode\n```",
			expected: false,
		},
		{
			name:     "マーカーがない",
			input:    "text\n```\ncode\n```",
			expected: false,
		},
		{
			name:     "コードブロックがない",
			input:    "text" + CursorMarker + "more",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isMarkerInCodeBlock(tt.input)
			if result != tt.expected {
				t.Errorf("isMarkerInCodeBlock() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConvert_UserBrClass(t *testing.T) {
	converter := NewConverter()

	t.Run("ユーザー入力の改行にuser-brクラスが付与される", func(t *testing.T) {
		input := "aaa\nbbb\nccc"
		result, err := converter.Convert(input)
		if err != nil {
			t.Fatalf("Convert() error = %v", err)
		}

		t.Logf("Result: %q", result)

		// user-brクラスが付与されるべき
		if !strings.Contains(result, `class="user-br"`) {
			t.Errorf("Convert() should contain user-br class, got: %q", result)
		}

		// <br>タグが存在するべき
		if !strings.Contains(result, "<br") {
			t.Errorf("Convert() should contain br tag, got: %q", result)
		}
	})

	t.Run("マーカーのみの行では改行が出力されない", func(t *testing.T) {
		// マーカーのみの行（次の行との間）
		input := "aaa\n" + CursorMarker + "\nbbb"
		result, err := converter.Convert(input)
		if err != nil {
			t.Fatalf("Convert() error = %v", err)
		}

		t.Logf("Result: %q", result)

		// preprocessMarkerPositionにより段落分離として扱われるので、
		// 段落が分離されていればOK
		if !strings.Contains(result, "<p>aaa</p>") {
			t.Errorf("Convert() should have paragraph for aaa, got: %q", result)
		}
		if !strings.Contains(result, "<p>bbb</p>") {
			t.Errorf("Convert() should have paragraph for bbb, got: %q", result)
		}
	})

	t.Run("マーカーが行の途中にある場合は改行が出力される", func(t *testing.T) {
		input := "aaa" + CursorMarker + "\nbbb"
		result, err := converter.Convert(input)
		if err != nil {
			t.Fatalf("Convert() error = %v", err)
		}

		t.Logf("Result: %q", result)

		// user-brクラスが付与されるべき
		if !strings.Contains(result, `class="user-br"`) {
			t.Errorf("Convert() should contain user-br class, got: %q", result)
		}
	})
}

func TestPreprocessMarkerPosition(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "2連続改行の間にマーカー",
			input:    "aaa\n" + CursorMarker + "\nbbb",
			expected: "aaa\n\n" + CursorMarker + "\n\nbbb",
		},
		{
			name:     "3連続改行の最初にマーカー",
			input:    "aaa\n" + CursorMarker + "\n\nbbb",
			expected: "aaa\n\n" + CursorMarker + "\n\nbbb",
		},
		{
			name:     "3連続改行の2番目にマーカー",
			input:    "aaa\n\n" + CursorMarker + "\nbbb",
			expected: "aaa\n\n" + CursorMarker + "\n\nbbb",
		},
		{
			name:     "4連続改行の中間にマーカー",
			input:    "aaa\n\n" + CursorMarker + "\n\nbbb",
			expected: "aaa\n\n" + CursorMarker + "\n\nbbb",
		},
		{
			name:     "同一段落内の改行（変換されない）",
			input:    "aaa\n" + CursorMarker + "bbb",
			expected: "aaa\n" + CursorMarker + "bbb",
		},
		{
			name:     "マーカーなし（変換されない）",
			input:    "aaa\n\nbbb",
			expected: "aaa\n\nbbb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := preprocessMarkerPosition(tt.input)
			if result != tt.expected {
				t.Errorf("preprocessMarkerPosition() = %q, want %q", result, tt.expected)
			}
		})
	}
}
