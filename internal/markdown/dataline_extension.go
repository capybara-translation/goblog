package markdown

import (
	"html"
	"sort"
	"strconv"

	gast "github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type lineIndex struct {
	starts []int
}

func newLineIndex(src []byte) *lineIndex {
	starts := []int{0}
	for i, b := range src {
		if b == '\n' && i+1 < len(src) {
			starts = append(starts, i+1)
		}
	}
	return &lineIndex{starts: starts}
}

func (li *lineIndex) lineFromOffset(off int) int {
	if off <= 0 {
		return 0
	}
	i := sort.Search(len(li.starts), func(i int) bool { return li.starts[i] > off }) - 1
	if i < 0 {
		return 0
	}
	return i
}

func startOffsetOf(node gast.Node) int {
	if lines := node.Lines(); lines != nil && lines.Len() > 0 {
		return lines.At(0).Start
	}
	return 0
}

func dataLineAttr(li *lineIndex, node gast.Node) string {
	line := li.lineFromOffset(startOffsetOf(node))
	return ` data-line="` + strconv.Itoa(line) + `"`
}

type practicalLineRenderer struct {
	li *lineIndex
}

func (r *practicalLineRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	// --- anchors (blocks) ---
	reg.Register(gast.KindHeading, r.renderHeading)         // h1..h6
	reg.Register(gast.KindParagraph, r.wrap("p"))           // p
	reg.Register(gast.KindBlockquote, r.wrap("blockquote")) // blockquote
	reg.Register(gast.KindList, r.renderList)               // ul/ol
	reg.Register(gast.KindListItem, r.wrap("li"))           // li
	reg.Register(gast.KindThematicBreak, r.renderHR)        // hr
	reg.Register(gast.KindCodeBlock, r.renderCodeBlock)     // indented code
	// FencedCodeBlock is delegated to the highlighting extension (syntax highlighting takes priority)
	reg.Register(gast.KindHTMLBlock, r.renderHTMLBlock) // raw HTML block (optional)

	// --- tables (GFM) ---
	reg.Register(extast.KindTable, r.wrap("table"))
	reg.Register(extast.KindTableHeader, r.wrap("thead"))
	reg.Register(extast.KindTableRow, r.wrap("tr"))
	reg.Register(extast.KindTableCell, r.renderTableCell) // th/td
}

func (r *practicalLineRenderer) wrap(tag string) renderer.NodeRendererFunc {
	return func(w util.BufWriter, src []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
		if entering {
			_, _ = w.WriteString("<" + tag + dataLineAttr(r.li, node) + ">")
		} else {
			_, _ = w.WriteString("</" + tag + ">")
		}
		return gast.WalkContinue, nil
	}
}

func (r *practicalLineRenderer) renderHeading(w util.BufWriter, src []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	n := node.(*gast.Heading)
	tag := "h" + strconv.Itoa(n.Level)
	if entering {
		_, _ = w.WriteString("<" + tag + dataLineAttr(r.li, node) + ">")
	} else {
		_, _ = w.WriteString("</" + tag + ">")
	}
	return gast.WalkContinue, nil
}

func (r *practicalLineRenderer) renderList(w util.BufWriter, src []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	n := node.(*gast.List)
	tag := "ul"
	if n.IsOrdered() {
		tag = "ol"
	}
	if entering {
		_, _ = w.WriteString("<" + tag + dataLineAttr(r.li, node) + ">")
	} else {
		_, _ = w.WriteString("</" + tag + ">")
	}
	return gast.WalkContinue, nil
}

func (r *practicalLineRenderer) renderHR(w util.BufWriter, src []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString("<hr" + dataLineAttr(r.li, node) + "/>")
	}
	return gast.WalkContinue, nil
}

func (r *practicalLineRenderer) renderCodeBlock(w util.BufWriter, src []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		_, _ = w.WriteString("</code></pre>")
		return gast.WalkContinue, nil
	}
	n := node.(*gast.CodeBlock)
	_, _ = w.WriteString("<pre" + dataLineAttr(r.li, node) + "><code>")
	for i := 0; i < n.Lines().Len(); i++ {
		seg := n.Lines().At(i)
		_, _ = w.WriteString(html.EscapeString(string(seg.Value(src))))
	}
	return gast.WalkContinue, nil
}

func (r *practicalLineRenderer) renderHTMLBlock(w util.BufWriter, src []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	// Raw HTML cannot have data-line added to the tag itself, so we compromise by wrapping it
	if !entering {
		return gast.WalkContinue, nil
	}
	n := node.(*gast.HTMLBlock)
	_, _ = w.WriteString("<div" + dataLineAttr(r.li, node) + ">")
	for i := 0; i < n.Lines().Len(); i++ {
		seg := n.Lines().At(i)
		_, _ = w.Write(seg.Value(src))
	}
	_, _ = w.WriteString("</div>")
	return gast.WalkContinue, nil
}

func (r *practicalLineRenderer) renderTableCell(w util.BufWriter, src []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	n := node.(*extast.TableCell)

	// Cells directly under TableHeader are header cells
	isHeader := false
	if p := node.Parent(); p != nil && p.Kind() == extast.KindTableHeader {
		isHeader = true
	}

	tag := "td"
	if isHeader {
		tag = "th"
	}

	if entering {
		alignAttr := ""
		switch n.Alignment {
		case extast.AlignLeft:
			alignAttr = ` style="text-align:left"`
		case extast.AlignCenter:
			alignAttr = ` style="text-align:center"`
		case extast.AlignRight:
			alignAttr = ` style="text-align:right"`
			// AlignNone adds nothing
		}

		_, _ = w.WriteString("<" + tag + dataLineAttr(r.li, node) + alignAttr + ">")
	} else {
		_, _ = w.WriteString("</" + tag + ">")
	}

	return gast.WalkContinue, nil
}
