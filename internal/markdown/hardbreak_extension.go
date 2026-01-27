package markdown

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

// UserBrClass はユーザーが入力した改行に付与するクラス名
const UserBrClass = "user-br"

// markedHardBreakExtension はユーザー入力の改行にクラスを付与するextension
type markedHardBreakExtension struct {
	xhtml bool
}

// NewMarkedHardBreakExtension はmarkedHardBreakExtensionを作成する
func NewMarkedHardBreakExtension(xhtml bool) goldmark.Extender {
	return &markedHardBreakExtension{xhtml: xhtml}
}

func (e *markedHardBreakExtension) Extend(m goldmark.Markdown) {
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(&markedHardBreakRenderer{xhtml: e.xhtml}, 100),
		),
	)
}

// markedHardBreakRenderer はTextノードをレンダリングする際、改行に特別なクラスを付与する
type markedHardBreakRenderer struct {
	xhtml bool
}

func (r *markedHardBreakRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindText, r.renderText)
}

func (r *markedHardBreakRenderer) renderText(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	n := node.(*ast.Text)
	segment := n.Segment
	value := segment.Value(source)

	if n.IsRaw() {
		html.DefaultWriter.RawWrite(w, value)
	} else {
		html.DefaultWriter.Write(w, value)
	}

	// HardLineBreak または SoftLineBreak で改行を出力
	if n.HardLineBreak() || n.SoftLineBreak() {
		// テキストがカーソルマーカー（ゼロ幅スペース）のみの場合は改行を出力しない
		// これにより、空行にマーカーがある場合の余分な<br>を防ぐ
		if isOnlyCursorMarker(value) {
			return ast.WalkContinue, nil
		}

		// ユーザーが入力した改行にはクラスを付与する
		if r.xhtml {
			_, _ = w.WriteString("<br class=\"" + UserBrClass + "\" />\n")
		} else {
			_, _ = w.WriteString("<br class=\"" + UserBrClass + "\">\n")
		}
	}

	return ast.WalkContinue, nil
}

// isOnlyCursorMarker はテキストがカーソルマーカーのみかどうかを判定する
func isOnlyCursorMarker(value []byte) bool {
	// ゼロ幅スペース（U+200B）はUTF-8で3バイト: 0xE2 0x80 0x8B
	return len(value) == 3 && value[0] == 0xE2 && value[1] == 0x80 && value[2] == 0x8B
}
