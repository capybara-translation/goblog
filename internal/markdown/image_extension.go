package markdown

import (
	"bytes"
	"strconv"

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
	seenFirst  bool
	dimensions DimensionsProvider // may be nil
	variants   VariantsProvider   // may be nil
}

func newImageRenderer(dimensions DimensionsProvider, variants VariantsProvider) *imageRenderer {
	return &imageRenderer{dimensions: dimensions, variants: variants}
}

// articleSizes is the responsive sizes attribute used for body images.
// "672px" matches Tailwind's max-w-2xl (the article container in
// layout.html); below 768px the image takes the full viewport width.
// If the article width ever changes, update this together.
const articleSizes = "(max-width: 768px) 100vw, 672px"

func (r *imageRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindImage, r.renderImage)
}

func (r *imageRenderer) renderImage(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.Image)

	// Pick the URL for the src attribute: when WebP variants exist we use
	// the middle width as the legacy-client default; otherwise the
	// original. Variants are also written out as a srcset further down.
	var vs []Variant
	if r.variants != nil {
		vs = r.variants.Variants(string(n.Destination))
	}
	srcURL := n.Destination
	if len(vs) > 0 {
		srcURL = []byte(vs[len(vs)/2].URL)
	}

	_, _ = w.WriteString(`<img src="`)
	// Mirror goldmark's safety: only emit the URL when it's not classified
	// as a dangerous scheme. bluemonday will sanitize again downstream,
	// but defense in depth is cheap here.
	if !ghtml.IsDangerousURL(srcURL) {
		_, _ = w.Write(util.EscapeHTML(util.URLEscape(srcURL, true)))
	}
	_, _ = w.WriteString(`" alt="`)
	_, _ = w.Write(util.EscapeHTML(collectImageAltText(n, source)))
	_ = w.WriteByte('"')

	if n.Title != nil {
		_, _ = w.WriteString(` title="`)
		_, _ = w.Write(util.EscapeHTML(n.Title))
		_ = w.WriteByte('"')
	}

	if len(vs) > 0 {
		_, _ = w.WriteString(` srcset="`)
		for i, v := range vs {
			if i > 0 {
				_, _ = w.WriteString(", ")
			}
			_, _ = w.Write(util.EscapeHTML(util.URLEscape([]byte(v.URL), true)))
			_, _ = w.WriteString(" ")
			_, _ = w.WriteString(strconv.Itoa(v.Width))
			_, _ = w.WriteString("w")
		}
		_, _ = w.WriteString(`" sizes="`)
		_, _ = w.WriteString(articleSizes)
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

	// Intrinsic dimensions help the browser reserve layout space before
	// the bytes arrive. The provider is responsible for refusing any
	// URL it cannot trust (external URLs, paths outside /uploads/) by
	// returning ok=false; the renderer treats absence as "skip silently".
	if r.dimensions != nil {
		if width, height, ok := r.dimensions.Get(string(n.Destination)); ok && width > 0 && height > 0 {
			_, _ = w.WriteString(` width="`)
			_, _ = w.WriteString(strconv.Itoa(width))
			_, _ = w.WriteString(`" height="`)
			_, _ = w.WriteString(strconv.Itoa(height))
			_ = w.WriteByte('"')
		}
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
