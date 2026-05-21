package http

import (
	"embed"
	"fmt"
	"html"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/capybara-translation/goblog/internal/markdown"
	"github.com/capybara-translation/goblog/internal/service"
	"github.com/gorilla/mux"
)

// PublicHandlers is a struct that groups handlers for public pages
type PublicHandlers struct {
	postService      service.PostService
	postViewService  service.PostViewService
	ogpService       service.OGPService
	mdConverter      markdown.Converter
	blogTitle        string // Blog title
	baseURL          string // Site base URL (for sitemap)
	homeTemplate     *template.Template
	postTemplate     *template.Template
	tagsTemplate     *template.Template
	tagPostsTemplate *template.Template
	notFoundTemplate *template.Template
}

// truncateRunes truncates a string by rune (character) count
// By truncating by rune count instead of byte count, it safely handles multibyte characters like Japanese
func truncateRunes(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes])
}

// htmlTagPattern is a regular expression pattern that matches HTML tags
var htmlTagPattern = regexp.MustCompile(`<[^>]*>`)

// stripHTMLTags removes tags from an HTML string and returns plain text
func stripHTMLTags(htmlStr string) string {
	// Remove HTML tags
	text := htmlTagPattern.ReplaceAllString(htmlStr, "")
	// Decode HTML entities
	text = html.UnescapeString(text)
	// Collapse consecutive whitespace to single space and trim leading/trailing whitespace
	text = strings.Join(strings.Fields(text), " ")
	return text
}

// splitTags converts a comma-separated tag string to a slice
func splitTags(tags string) []string {
	if tags == "" {
		return []string{}
	}
	result := []string{}
	for _, tag := range strings.Split(tags, ",") {
		trimmed := strings.TrimSpace(tag)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// getTimezoneLocation gets the timezone from the TZ environment variable
func getTimezoneLocation() *time.Location {
	var loc *time.Location
	var err error

	if tz := os.Getenv("TZ"); tz != "" {
		loc, err = time.LoadLocation(tz)
	}

	if err != nil || loc == nil {
		loc = time.Local
	}

	return loc
}

// getUTCOffsetString returns the UTC offset of the timezone as a string
// Example: "UTC+9", "UTC-5", "UTC+0"
func getUTCOffsetString(t time.Time) string {
	_, offset := t.Zone()
	hours := offset / 3600
	if hours >= 0 {
		return fmt.Sprintf("UTC+%d", hours)
	}
	return fmt.Sprintf("UTC%d", hours)
}

// formatDateWithTZ displays a date in ISO 8601 format + timezone abbreviation + UTC offset
// Converts to the timezone set in the TZ environment variable before formatting
// Example: "2026-01-17 (JST, UTC+9)"
func formatDateWithTZ(t time.Time) string {
	localTime := t.In(getTimezoneLocation())
	return localTime.Format("2006-01-02") + " (" + localTime.Format("MST") + ", " + getUTCOffsetString(localTime) + ")"
}

// formatDateDetailWithTZ displays a date in detailed format (time + timezone abbreviation + UTC offset)
// For mouse-over tooltip
// Example: "2026-01-17 22:00 (JST, UTC+9)"
func formatDateDetailWithTZ(t time.Time) string {
	localTime := t.In(getTimezoneLocation())
	return localTime.Format("2006-01-02 15:04") + " (" + localTime.Format("MST") + ", " + getUTCOffsetString(localTime) + ")"
}

// mdConverter is a singleton instance of the Markdown converter
var mdConverter = markdown.NewConverter()

// renderMarkdown converts Markdown to HTML
func renderMarkdown(content string) template.HTML {
	htmlContent, err := mdConverter.Convert(content)
	if err != nil {
		// On error, return HTML-escaped text
		return template.HTML(template.HTMLEscapeString(content))
	}
	return template.HTML(htmlContent)
}

// markdownExcerpt converts Markdown to plain text and truncates it
func markdownExcerpt(content string, maxLen int) string {
	// Markdown -> HTML -> plain text -> truncate
	htmlContent, err := mdConverter.Convert(content)
	if err != nil {
		// On error, return truncated original text
		return truncateRunes(content, maxLen)
	}
	plainText := stripHTMLTags(htmlContent)
	return truncateRunes(plainText, maxLen)
}

// highlightQuery highlights strings matching the search query with <mark> tags
// For XSS protection, text is HTML-escaped before processing
func highlightQuery(text string, query string) template.HTML {
	// HTML-escape the text
	escapedText := html.EscapeString(text)

	if query == "" {
		return template.HTML(escapedText)
	}

	// HTML-escape the query too (to match against escaped text)
	escapedQuery := html.EscapeString(query)

	// Escape regex special characters
	quotedQuery := regexp.QuoteMeta(escapedQuery)

	// Case-insensitive pattern
	pattern := regexp.MustCompile("(?i)(" + quotedQuery + ")")

	// Wrap matched parts with <mark>
	highlighted := pattern.ReplaceAllString(escapedText, "<mark>$1</mark>")

	return template.HTML(highlighted)
}

// highlightHTMLContent highlights only the text parts within HTML
// Text inside HTML tags is not modified
func highlightHTMLContent(htmlContent string, query string) string {
	if query == "" {
		return htmlContent
	}

	// HTML-escape the query (since text in HTML is already escaped)
	escapedQuery := html.EscapeString(query)
	quotedQuery := regexp.QuoteMeta(escapedQuery)
	searchPattern := regexp.MustCompile("(?i)(" + quotedQuery + ")")

	// Parse HTML and highlight only text parts
	// Find and highlight text outside of tags
	var result strings.Builder
	remaining := htmlContent

	for len(remaining) > 0 {
		// Find the next tag
		tagStart := strings.Index(remaining, "<")

		if tagStart == -1 {
			// No tag found, all remaining content is text
			highlighted := searchPattern.ReplaceAllString(remaining, "<mark>$1</mark>")
			result.WriteString(highlighted)
			break
		}

		if tagStart > 0 {
			// Highlight text before the tag
			textPart := remaining[:tagStart]
			highlighted := searchPattern.ReplaceAllString(textPart, "<mark>$1</mark>")
			result.WriteString(highlighted)
		}

		// Find the end of the tag
		tagEnd := strings.Index(remaining[tagStart:], ">")
		if tagEnd == -1 {
			// No closing bracket found, add remaining content as-is
			result.WriteString(remaining[tagStart:])
			break
		}

		// Add the tag as-is (without highlighting)
		tagEnd += tagStart + 1
		result.WriteString(remaining[tagStart:tagEnd])
		remaining = remaining[tagEnd:]
	}

	return result.String()
}

// renderMarkdownWithHighlight converts Markdown to HTML and highlights the search query
func renderMarkdownWithHighlight(content string, query string) template.HTML {
	htmlContent, err := mdConverter.Convert(content)
	if err != nil {
		// On error, return HTML-escaped text
		return template.HTML(template.HTMLEscapeString(content))
	}

	if query == "" {
		return template.HTML(htmlContent)
	}

	highlighted := highlightHTMLContent(htmlContent, query)
	return template.HTML(highlighted)
}

// NewPublicHandlers creates PublicHandlers from embedded templates
func NewPublicHandlers(postService service.PostService, postViewService service.PostViewService, ogpService service.OGPService, blogTitle, baseURL string, templatesFS embed.FS) *PublicHandlers {
	// Create markdown converter with OGP support
	var converter markdown.Converter
	if ogpService != nil {
		converter = markdown.NewConverterWithOGP(ogpService)
	} else {
		converter = markdown.NewConverter()
	}

	// Define custom template functions with closure-based markdown functions
	funcMap := template.FuncMap{
		"truncate":               truncateRunes,
		"splitTags":              splitTags,
		"formatDateWithTZ":       formatDateWithTZ,
		"formatDateDetailWithTZ": formatDateDetailWithTZ,
		"highlightQuery":         highlightQuery,
		// Closure-based markdown functions that use the OGP-enabled converter
		"renderMarkdown": func(content string) template.HTML {
			htmlContent, err := converter.Convert(content)
			if err != nil {
				return template.HTML(template.HTMLEscapeString(content))
			}
			return template.HTML(htmlContent)
		},
		"renderMarkdownWithHighlight": func(content string, query string) template.HTML {
			htmlContent, err := converter.Convert(content)
			if err != nil {
				return template.HTML(template.HTMLEscapeString(content))
			}
			if query == "" {
				return template.HTML(htmlContent)
			}
			highlighted := highlightHTMLContent(htmlContent, query)
			return template.HTML(highlighted)
		},
		"markdownExcerpt": func(content string, maxLen int) string {
			htmlContent, err := converter.Convert(content)
			if err != nil {
				return truncateRunes(content, maxLen)
			}
			plainText := stripHTMLTags(htmlContent)
			return truncateRunes(plainText, maxLen)
		},
	}

	// Create sub FS from embedded templates
	templateFS, err := fs.Sub(templatesFS, "internal/view/templates")
	if err != nil {
		panic(fmt.Sprintf("failed to get templates sub FS: %v", err))
	}

	// Helper function: parse embedded templates
	parseTemplates := func(names ...string) *template.Template {
		t := template.New("").Funcs(funcMap)
		for _, name := range names {
			content, err := fs.ReadFile(templateFS, name)
			if err != nil {
				panic(fmt.Sprintf("failed to read template %s: %v", name, err))
			}
			_, err = t.New(name).Parse(string(content))
			if err != nil {
				panic(fmt.Sprintf("failed to parse template %s: %v", name, err))
			}
		}
		return t
	}

	// Create independent template sets for each page
	homeTemplate := parseTemplates("layout.html", "home.html")
	postTemplate := parseTemplates("layout.html", "post.html")
	tagsTemplate := parseTemplates("layout.html", "tags.html")
	tagPostsTemplate := parseTemplates("layout.html", "tag_posts.html")
	notFoundTemplate := parseTemplates("layout.html", "notfound.html")

	return &PublicHandlers{
		postService:      postService,
		postViewService:  postViewService,
		ogpService:       ogpService,
		mdConverter:      converter,
		blogTitle:        blogTitle,
		baseURL:          baseURL,
		homeTemplate:     homeTemplate,
		postTemplate:     postTemplate,
		tagsTemplate:     tagsTemplate,
		tagPostsTemplate: tagPostsTemplate,
		notFoundTemplate: notFoundTemplate,
	}
}

// NewPublicHandlersFromPath creates PublicHandlers by loading templates from the filesystem (for testing)
func NewPublicHandlersFromPath(postService service.PostService, postViewService service.PostViewService, blogTitle, baseURL, templatePattern string) *PublicHandlers {
	dir := filepath.Dir(templatePattern)
	layoutPath := filepath.Join(dir, "layout.html")

	// Define custom template functions
	funcMap := template.FuncMap{
		"truncate":                    truncateRunes,
		"splitTags":                   splitTags,
		"formatDateWithTZ":            formatDateWithTZ,
		"formatDateDetailWithTZ":      formatDateDetailWithTZ,
		"highlightQuery":              highlightQuery,
		"renderMarkdown":              renderMarkdown,
		"renderMarkdownWithHighlight": renderMarkdownWithHighlight,
		"markdownExcerpt":             markdownExcerpt,
	}

	// Create independent template sets for each page
	homeTemplate := template.Must(template.New("").Funcs(funcMap).ParseFiles(layoutPath, filepath.Join(dir, "home.html")))
	postTemplate := template.Must(template.New("").Funcs(funcMap).ParseFiles(layoutPath, filepath.Join(dir, "post.html")))
	tagsTemplate := template.Must(template.New("").Funcs(funcMap).ParseFiles(layoutPath, filepath.Join(dir, "tags.html")))
	tagPostsTemplate := template.Must(template.New("").Funcs(funcMap).ParseFiles(layoutPath, filepath.Join(dir, "tag_posts.html")))
	notFoundTemplate := template.Must(template.New("").Funcs(funcMap).ParseFiles(layoutPath, filepath.Join(dir, "notfound.html")))

	return &PublicHandlers{
		postService:      postService,
		postViewService:  postViewService,
		blogTitle:        blogTitle,
		baseURL:          baseURL,
		homeTemplate:     homeTemplate,
		postTemplate:     postTemplate,
		tagsTemplate:     tagsTemplate,
		tagPostsTemplate: tagPostsTemplate,
		notFoundTemplate: notFoundTemplate,
	}
}

// getPinnedPosts is a helper method that retrieves pinned published posts
func (h *PublicHandlers) getPinnedPosts() []*domain.Post {
	pinnedPosts, err := h.postService.GetPinnedPosts()
	if err != nil {
		log.Printf("failed to get pinned posts: %v", err)
		return []*domain.Post{}
	}
	return pinnedPosts
}

// renderNotFound renders the 404 page
func (h *PublicHandlers) renderNotFound(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{
		"SiteTitle":   h.blogTitle,
		"PinnedPosts": h.getPinnedPosts(),
		"OGP":         h.defaultOGP("Not Found - "+h.blogTitle, r.URL.Path, h.blogTitle),
		"Query":       "",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	if err := h.notFoundTemplate.ExecuteTemplate(w, "notfound", data); err != nil {
		log.Printf("failed to execute notfound template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// HandleHome displays the home page (posts list, page 1 by default)
func (h *PublicHandlers) HandleHome(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	queryStr := r.URL.Query().Get("q")
	page := 1
	if pageStr != "" {
		if parsed, err := strconv.Atoi(pageStr); err == nil && parsed > 0 {
			page = parsed
		}
	}

	const maxQueryLength = 200
	if len(queryStr) > maxQueryLength {
		http.Error(w, "Search query too long", http.StatusBadRequest)
		return
	}

	const perPage = 20
	offset := (page - 1) * perPage

	var posts []*domain.Post
	var err error
	if queryStr != "" {
		posts, err = h.postService.SearchPublishedPosts(queryStr, perPage+1, offset)
	} else {
		posts, err = h.postService.GetPublishedPosts(perPage+1, offset)
	}
	if err != nil {
		log.Printf("failed to get published posts: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	hasNext := len(posts) > perPage
	if hasNext {
		posts = posts[:perPage]
	}

	data := map[string]any{
		"SiteTitle":   h.blogTitle,
		"Posts":       posts,
		"CurrentPage": page,
		"HasPrev":     page > 1,
		"HasNext":     hasNext,
		"PrevPage":    page - 1,
		"NextPage":    page + 1,
		"Query":       queryStr,
		"NoIndex":     queryStr != "" || page > 1,
		"PinnedPosts": h.getPinnedPosts(),
		"OGP":         h.defaultOGP(h.blogTitle, r.URL.Path, h.blogTitle),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.homeTemplate.ExecuteTemplate(w, "home", data); err != nil {
		log.Printf("failed to execute template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// HandlePosts redirects /posts (with any query) to / (page 1 of the home posts list)
func (h *PublicHandlers) HandlePosts(w http.ResponseWriter, r *http.Request) {
	target := "/"
	if raw := r.URL.RawQuery; raw != "" {
		target = "/?" + raw
	}
	http.Redirect(w, r, target, http.StatusMovedPermanently)
}

// HandlePostDetail displays the post detail page
func (h *PublicHandlers) HandlePostDetail(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slug := vars["slug"]

	// Get post by slug
	post, err := h.postService.GetPostBySlug(slug)
	if err != nil {
		log.Printf("failed to get post by slug: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if post == nil {
		h.renderNotFound(w, r)
		return
	}

	// Record view asynchronously
	if h.postViewService != nil {
		ip := extractIP(r.RemoteAddr)
		ua := r.UserAgent()
		postID := post.ID
		go func() {
			if err := h.postViewService.RecordView(postID, ip, ua); err != nil {
				log.Printf("failed to record view for post %d: %v", postID, err)
			}
		}()
	}

	data := map[string]any{
		"SiteTitle":   h.blogTitle,
		"Post":        post,
		"PinnedPosts": h.getPinnedPosts(),
		"OGP":         h.postOGP(post, r.URL.Path),
		"Query":       "",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.postTemplate.ExecuteTemplate(w, "post", data); err != nil {
		log.Printf("failed to execute template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// HandleTags displays the tags list page
func (h *PublicHandlers) HandleTags(w http.ResponseWriter, r *http.Request) {
	// Get tags from published posts
	tagCounts, err := h.postService.GetPublishedTags()
	if err != nil {
		log.Printf("failed to get published tags: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Sort tags by post count in descending order
	type TagInfo struct {
		Name  string
		Count int
	}

	tags := make([]TagInfo, 0, len(tagCounts))
	for name, count := range tagCounts {
		tags = append(tags, TagInfo{Name: name, Count: count})
	}

	// Sort by post count descending, then by tag name ascending for ties
	slices.SortFunc(tags, func(a, b TagInfo) int {
		// Compare by post count (descending)
		if a.Count != b.Count {
			return b.Count - a.Count
		}
		// Compare by tag name (ascending)
		if a.Name < b.Name {
			return -1
		} else if a.Name > b.Name {
			return 1
		}
		return 0
	})

	data := map[string]any{
		"SiteTitle":   h.blogTitle,
		"Tags":        tags,
		"PinnedPosts": h.getPinnedPosts(),
		"OGP":         h.defaultOGP("Tags - "+h.blogTitle, r.URL.Path, "Tags from "+h.blogTitle),
		"Query":       "",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tagsTemplate.ExecuteTemplate(w, "tags", data); err != nil {
		log.Printf("failed to execute template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// HandleTagPosts displays the posts list page for a specific tag
func (h *PublicHandlers) HandleTagPosts(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tag := vars["tag"]

	if tag == "" {
		h.renderNotFound(w, r)
		return
	}

	// URL decode
	decodedTag, err := url.QueryUnescape(tag)
	if err != nil {
		h.renderNotFound(w, r)
		return
	}
	tag = decodedTag

	// Pagination
	pageStr := r.URL.Query().Get("page")
	page := 1
	if pageStr != "" {
		if parsed, err := strconv.Atoi(pageStr); err == nil && parsed > 0 {
			page = parsed
		}
	}

	const perPage = 20
	offset := (page - 1) * perPage

	// Get posts
	posts, err := h.postService.GetPublishedPostsByTag(tag, perPage+1, offset)
	if err != nil {
		log.Printf("failed to get posts by tag: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Determine if there is a next page
	hasNext := len(posts) > perPage
	if hasNext {
		posts = posts[:perPage]
	}

	data := map[string]any{
		"SiteTitle":   h.blogTitle,
		"Tag":         tag,
		"Posts":       posts,
		"CurrentPage": page,
		"HasPrev":     page > 1,
		"HasNext":     hasNext,
		"PrevPage":    page - 1,
		"NextPage":    page + 1,
		"PinnedPosts": h.getPinnedPosts(),
		"OGP":         h.defaultOGP(tag+" - "+h.blogTitle, r.URL.Path, "Posts tagged with "+tag),
		"Query":       "",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tagPostsTemplate.ExecuteTemplate(w, "tag_posts", data); err != nil {
		log.Printf("failed to execute template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
