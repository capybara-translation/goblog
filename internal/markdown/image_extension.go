package markdown

import (
	"bytes"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	ghtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

// imageRenderer overrides goldmark's default <img> output with attributes
// that improve image-heavy page performance:
//   - The first image in a document does NOT receive loading="lazy" so a
//     probable LCP candidate isn't deferred.
//   - Subsequent images get loading="lazy" so they don't compete with
//     above-the-fold content.
//   - All images get decoding="async".
//
// fetchpriority="high" is intentionally NOT emitted here. The home page
// renders multiple posts inline (home.html), so the "first image of each
// post" set would all be tagged high — which is the opposite of what
// fetchpriority is for. If a per-page-type variant is needed later, the
// caller can compose it on top of this baseline.
//
// A new instance is created per Convert() invocation (via createMarkdown),
// so the "first image seen" flag is per-document and never leaks across
// requests.
type imageRenderer struct {
	seenFirst bool
}

func newImageRenderer() *imageRenderer {
	return &imageRenderer{}
}

func (r *imageRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindImage, r.renderImage)
}

func (r *imageRenderer) renderImage(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.Image)

	_, _ = w.WriteString(`<img src="`)
	// Mirror goldmark's safety: only emit the URL when it's not classified
	// as a dangerous scheme. bluemonday will sanitize again downstream,
	// but defense in depth is cheap here.
	if !ghtml.IsDangerousURL(n.Destination) {
		_, _ = w.Write(util.EscapeHTML(util.URLEscape(n.Destination, true)))
	}
	_, _ = w.WriteString(`" alt="`)
	_, _ = w.Write(util.EscapeHTML(collectImageAltText(n, source)))
	_ = w.WriteByte('"')

	if n.Title != nil {
		_, _ = w.WriteString(` title="`)
		_, _ = w.Write(util.EscapeHTML(n.Title))
		_ = w.WriteByte('"')
	}

	// No goldmark extension currently attaches attributes to *ast.Image
	// (goldmark-attributes / parser.WithAttribute are not enabled). We
	// still mirror upstream's RenderAttributes branch here so that adding
	// such an extension later won't silently lose <img> attributes only.
	// ImageAttributeFilter restricts which attribute names are emitted,
	// and bluemonday sanitizes again downstream.
	if node.Attributes() != nil {
		ghtml.RenderAttributes(w, node, ghtml.ImageAttributeFilter)
	}

	_, _ = w.WriteString(` decoding="async"`)

	if !r.seenFirst {
		r.seenFirst = true
	} else {
		_, _ = w.WriteString(` loading="lazy"`)
	}

	// XHTML self-close, consistent with html.WithXHTML() set in createMarkdown.
	_, _ = w.WriteString(" />")
	return ast.WalkSkipChildren, nil
}

// collectImageAltText flattens descendant Text/String nodes into a single
// byte slice for use as the alt attribute. This matches the behavior of
// goldmark's default renderTexts: any inline emphasis markers (**, *, etc.)
// are dropped, leaving only the textual content.
func collectImageAltText(n ast.Node, source []byte) []byte {
	var buf bytes.Buffer
	_ = ast.Walk(n, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch v := node.(type) {
		case *ast.Text:
			buf.Write(v.Segment.Value(source))
		case *ast.String:
			buf.Write(v.Value)
		}
		return ast.WalkContinue, nil
	})
	return buf.Bytes()
}
