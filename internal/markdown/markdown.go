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

// Converter はMarkdownをHTMLに変換するインターフェース
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

// NewConverter はシングルトンのConverterを返す
func NewConverter() Converter {
	once.Do(func() {
		instance = &converter{
			policy: createPolicy(),
		}
	})
	return instance
}

// createPolicy はHTMLサニタイズポリシーを作成する
func createPolicy() *bluemonday.Policy {
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
	// data-line属性を許可（プレビュー同期スクロール用）
	policy.AllowAttrs("data-line").Matching(regexp.MustCompile(`^\d+$`)).Globally()
	return policy
}

// createMarkdown はgoldmarkインスタンスを作成する
func createMarkdown(src []byte) goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(
			extension.GFM, // テーブル、取り消し線、タスクリスト等
			highlighting.NewHighlighting(
				highlighting.WithStyle("monokai"), // シンタックスハイライトのテーマ
			),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
			// data-line属性を付与するカスタムrenderer
			renderer.WithNodeRenderers(
				util.Prioritized(&practicalLineRenderer{
					li: newLineIndex(src),
				}, 50), // highlightingより低い優先度
			),
		),
	)
}

// Convert はMarkdownをサニタイズ済みHTMLに変換する
// 各ブロック要素にdata-line属性を付与し、プレビュー同期スクロールに使用する
func (c *converter) Convert(markdown string) (string, error) {
	src := []byte(markdown)
	md := createMarkdown(src)

	var buf bytes.Buffer
	if err := md.Convert(src, &buf); err != nil {
		return "", err
	}

	// XSS対策: HTMLをサニタイズ
	return c.policy.Sanitize(buf.String()), nil
}
