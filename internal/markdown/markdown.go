package markdown

import (
	"bytes"
	"regexp"
	"strings"
	"sync"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// CursorMarker はカーソル位置を示すプレースホルダー
// ゼロ幅スペース（U+200B）を使用し、シンタックスハイライトに影響されないようにする
const CursorMarker = "\u200B"

// CursorMarkerID はカーソル位置マーカーのHTML ID
const CursorMarkerID = "cursor-line-marker"

// cursorMarkerSpan はカーソルマーカーのHTML
const cursorMarkerSpan = `<span id="` + CursorMarkerID + `"></span>`

// Converter はMarkdownをHTMLに変換するインターフェース
type Converter interface {
	Convert(markdown string) (string, error)
}

type converter struct {
	md     goldmark.Markdown
	policy *bluemonday.Policy
}

var (
	instance *converter
	once     sync.Once
)

// NewConverter はシングルトンのConverterを返す
func NewConverter() Converter {
	once.Do(func() {
		md := goldmark.New(
			goldmark.WithExtensions(
				extension.GFM, // テーブル、取り消し線、タスクリスト等
				highlighting.NewHighlighting(
					highlighting.WithStyle("monokai"), // シンタックスハイライトのテーマ
				),
				// カスタムextension: ユーザー入力の改行にクラスを付与
				// マーカーのみの行では改行を出力しない
				NewMarkedHardBreakExtension(true), // XHTML mode
			),
			goldmark.WithParserOptions(
				parser.WithAutoHeadingID(),
			),
			goldmark.WithRendererOptions(
				html.WithHardWraps(),
				html.WithXHTML(),
			),
		)

		// HTMLサニタイズポリシー（UGCPolicy + コードブロック許可）
		policy := bluemonday.UGCPolicy()
		// シンタックスハイライト用のclass属性を許可
		policy.AllowAttrs("class").Matching(bluemonday.SpaceSeparatedTokens).OnElements("code", "pre", "span", "div")
		// コードブロックのstyle属性を許可（ハイライト色）
		policy.AllowAttrs("style").OnElements("pre", "code", "span")
		// タスクリスト用のチェックボックスのみを許可（セキュリティ上、type="checkbox"に限定）
		policy.AllowAttrs("checked", "disabled").OnElements("input")
		policy.AllowAttrs("type").Matching(regexp.MustCompile(`^checkbox$`)).OnElements("input")
		// タスクリスト用のli要素のclass属性を許可
		policy.AllowAttrs("class").Matching(bluemonday.SpaceSeparatedTokens).OnElements("li", "ul")
		// カーソル位置マーカー用のID属性を許可（cursor-line-marker のみ）
		policy.AllowAttrs("id").Matching(regexp.MustCompile(`^` + CursorMarkerID + `$`)).OnElements("span")
		// ユーザー入力の改行に付与するクラスを許可
		policy.AllowAttrs("class").Matching(regexp.MustCompile(`^` + UserBrClass + `$`)).OnElements("br")

		instance = &converter{
			md:     md,
			policy: policy,
		}
	})
	return instance
}

// Convert はMarkdownをサニタイズ済みHTMLに変換する
// マークダウン内にゼロ幅スペース（\u200B）が含まれている場合、HTMLに変換後、
// カーソルマーカー <span id="cursor-line-marker"></span> に置換する
func (c *converter) Convert(markdown string) (string, error) {
	// 前処理: 空行（段落分離）にマーカーがある場合、段落分離を維持するために
	// マーカーを次の段落の先頭に移動する
	// \n<MARKER>\n → \n\n<MARKER> に変換
	processed := preprocessMarkerPosition(markdown)

	var buf bytes.Buffer
	if err := c.md.Convert([]byte(processed), &buf); err != nil {
		return "", err
	}

	htmlStr := buf.String()

	// ゼロ幅スペースをマーカーに置換（最初の1つだけ）
	htmlStr = strings.Replace(htmlStr, CursorMarker, cursorMarkerSpan, 1)

	// 空行にマーカーだけがある場合の不要な<p>タグをクリーンアップ
	// 例: <p><span...></span></p> → <span...></span>（段落間の空行）
	htmlStr = strings.Replace(htmlStr, "<p>"+cursorMarkerSpan+"</p>", cursorMarkerSpan, -1)

	// XSS対策: HTMLをサニタイズ
	sanitized := c.policy.Sanitize(htmlStr)
	return sanitized, nil
}

// markerInEmptyLinePattern はマーカーが空行にあるパターンを検出する正規表現
// \n+<MARKER>\n+ （1つ以上の改行 + マーカー + 1つ以上の改行）
var markerInEmptyLinePattern = regexp.MustCompile(`\n+` + CursorMarker + `\n+`)

// codeBlockPattern はコードブロック（```...```）を検出する正規表現
var codeBlockPattern = regexp.MustCompile("(?s)```.*?```")

// preprocessMarkerPosition は空行にあるマーカーの位置を調整する
// Markdownでは2連続改行が段落分離を意味するが、マーカー（ゼロ幅スペース）が
// 間に入ると段落が分かれなくなる問題を解決する
// ただし、コードブロック内のマーカーは変換しない
func preprocessMarkerPosition(markdown string) string {
	// マーカーがコードブロック内にあるかチェック
	if isMarkerInCodeBlock(markdown) {
		return markdown
	}

	// \n+<MARKER>\n+ パターンを \n\n<MARKER>\n\n に変換
	// これによりマーカーは独立した段落になり、HTML変換後にクリーンアップされる
	// 3連続以上の改行の場合も適切に処理される
	return markerInEmptyLinePattern.ReplaceAllString(markdown, "\n\n"+CursorMarker+"\n\n")
}

// isMarkerInCodeBlock はマーカーがコードブロック内にあるかどうかを判定する
func isMarkerInCodeBlock(markdown string) bool {
	markerPos := strings.Index(markdown, CursorMarker)
	if markerPos == -1 {
		return false
	}

	// コードブロックの範囲を検出
	matches := codeBlockPattern.FindAllStringIndex(markdown, -1)
	for _, match := range matches {
		start, end := match[0], match[1]
		if markerPos >= start && markerPos < end {
			return true
		}
	}

	return false
}
