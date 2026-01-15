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
	"github.com/yuin/goldmark/renderer/html"
)

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

		instance = &converter{
			md:     md,
			policy: policy,
		}
	})
	return instance
}

// Convert はMarkdownをサニタイズ済みHTMLに変換する
func (c *converter) Convert(markdown string) (string, error) {
	var buf bytes.Buffer
	if err := c.md.Convert([]byte(markdown), &buf); err != nil {
		return "", err
	}
	// XSS対策: HTMLをサニタイズ
	sanitized := c.policy.Sanitize(buf.String())
	return sanitized, nil
}
