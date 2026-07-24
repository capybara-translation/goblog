package http

import (
	"embed"
	"fmt"
	"html"
	"html/template"
	"io/fs"
	"log"
	"net"
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
	postService       service.PostService
	postViewService   service.PostViewService
	ogpService        service.OGPService
	reactionService   service.ReactionService       // Nil disables the SSR reaction block.
	currentUserHelper *CurrentUserHelper            // Nil disables admin-only UI (edit links, etc.); resolves session-or-remember-token.
	trustedProxies    []*net.IPNet                  // Trusted proxies for resolving the client IP when recording views.
	blogTitle         string                        // Blog title
	baseURL           string                        // Site base URL (for sitemap)
	postsPerPage      int                           // Posts per page on listing views
	healthDisplay     *service.HealthDisplayService // Nil disables /health and the nav link/sitemap entry (Health Planet integration off).
	homeTemplate      *template.Template
	postTemplate      *template.Template
	tagsTemplate      *template.Template
	tagPostsTemplate  *template.Template
	notFoundTemplate  *template.Template
	healthTemplate    *template.Template
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

// mdConverter is a singleton used by the package-level markdownExcerpt
// (called from ogp_meta.go to build OGP description tags). The per-request
// converter used in NewPublicHandlers* is configured separately with OGP
// and DimensionsProvider; this singleton is intentionally bare because
// excerpting strips HTML anyway, making link cards / width-height
// irrelevant to its output.
var mdConverter = markdown.NewConverter()

// markdownExcerpt converts Markdown to plain text and truncates it.
// Called from ogp_meta.go for OGP description tags.
func markdownExcerpt(content string, maxLen int) string {
	htmlContent, err := mdConverter.Convert(content)
	if err != nil {
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

// resolvedTrustedProxies reuses the CurrentUserHelper's already-parsed proxy
// nets when a helper exists (avoiding a second parse of the same config and the
// duplicate invalid-value warnings that would produce), and parses the raw list
// only when there is no helper (authService == nil).
func resolvedTrustedProxies(helper *CurrentUserHelper, raw []string) []*net.IPNet {
	if helper != nil {
		return helper.trustedProxies
	}
	return parseTrustedProxies(raw)
}

// NewPublicHandlers creates PublicHandlers from embedded templates.
// dimensions and variants may be nil; when non-nil, the rendered <img>
// tags carry width/height and srcset/sizes attributes resolved from the
// upload directory on disk.
func NewPublicHandlers(postService service.PostService, postViewService service.PostViewService, ogpService service.OGPService, reactionService service.ReactionService, authService service.AuthService, secureCookie bool, trustedProxies []string, blogTitle, baseURL string, postsPerPage int, healthDisplay *service.HealthDisplayService, templatesFS embed.FS, dimensions markdown.DimensionsProvider, variants markdown.VariantsProvider) *PublicHandlers {
	converter := markdown.NewConverterFor(ogpService, dimensions, variants)

	// Define custom template functions with closure-based markdown functions
	funcMap := template.FuncMap{
		"truncate":               truncateRunes,
		"splitTags":              splitTags,
		"formatDateWithTZ":       formatDateWithTZ,
		"formatDateDetailWithTZ": formatDateDetailWithTZ,
		"highlightQuery":         highlightQuery,
		"healthValue1": func(v *float64) string {
			if v == nil {
				return ""
			}
			return strconv.FormatFloat(*v, 'f', 1, 64)
		},
		"healthValue0": func(v *float64) string {
			if v == nil {
				return ""
			}
			return strconv.FormatFloat(*v, 'f', 0, 64)
		},
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
	healthTemplate := parseTemplates("layout.html", "health.html")

	var helper *CurrentUserHelper
	if authService != nil {
		// public-page remember-me restores resolve the client IP via clientIP with trustedProxies (see current_user.go)
		helper = NewCurrentUserHelper(authService, secureCookie, trustedProxies)
	}

	return &PublicHandlers{
		postService:       postService,
		postViewService:   postViewService,
		ogpService:        ogpService,
		reactionService:   reactionService,
		currentUserHelper: helper,
		trustedProxies:    resolvedTrustedProxies(helper, trustedProxies),
		blogTitle:         blogTitle,
		baseURL:           baseURL,
		postsPerPage:      postsPerPage,
		healthDisplay:     healthDisplay,
		homeTemplate:      homeTemplate,
		postTemplate:      postTemplate,
		tagsTemplate:      tagsTemplate,
		tagPostsTemplate:  tagPostsTemplate,
		notFoundTemplate:  notFoundTemplate,
		healthTemplate:    healthTemplate,
	}
}

// NewPublicHandlersFromPath creates PublicHandlers by loading templates from the filesystem (for testing).
// dimensions and variants may be nil.
func NewPublicHandlersFromPath(postService service.PostService, postViewService service.PostViewService, reactionService service.ReactionService, authService service.AuthService, secureCookie bool, trustedProxies []string, blogTitle, baseURL, templatePattern string, postsPerPage int, healthDisplay *service.HealthDisplayService, dimensions markdown.DimensionsProvider, variants markdown.VariantsProvider) *PublicHandlers {
	dir := filepath.Dir(templatePattern)
	layoutPath := filepath.Join(dir, "layout.html")

	// Match NewPublicHandlers' wiring so dimensions/variants/OGP-driven
	// attributes flow through the test build too. ogpService is
	// intentionally nil here; tests that exercise OGP go through NewRouter
	// (embedded path).
	converter := markdown.NewConverterFor(nil, dimensions, variants)

	funcMap := template.FuncMap{
		"truncate":               truncateRunes,
		"splitTags":              splitTags,
		"formatDateWithTZ":       formatDateWithTZ,
		"formatDateDetailWithTZ": formatDateDetailWithTZ,
		"highlightQuery":         highlightQuery,
		"healthValue1": func(v *float64) string {
			if v == nil {
				return ""
			}
			return strconv.FormatFloat(*v, 'f', 1, 64)
		},
		"healthValue0": func(v *float64) string {
			if v == nil {
				return ""
			}
			return strconv.FormatFloat(*v, 'f', 0, 64)
		},
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
			return template.HTML(highlightHTMLContent(htmlContent, query))
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

	// Create independent template sets for each page
	homeTemplate := template.Must(template.New("").Funcs(funcMap).ParseFiles(layoutPath, filepath.Join(dir, "home.html")))
	postTemplate := template.Must(template.New("").Funcs(funcMap).ParseFiles(layoutPath, filepath.Join(dir, "post.html")))
	tagsTemplate := template.Must(template.New("").Funcs(funcMap).ParseFiles(layoutPath, filepath.Join(dir, "tags.html")))
	tagPostsTemplate := template.Must(template.New("").Funcs(funcMap).ParseFiles(layoutPath, filepath.Join(dir, "tag_posts.html")))
	notFoundTemplate := template.Must(template.New("").Funcs(funcMap).ParseFiles(layoutPath, filepath.Join(dir, "notfound.html")))
	healthTemplate := template.Must(template.New("").Funcs(funcMap).ParseFiles(layoutPath, filepath.Join(dir, "health.html")))

	var helper *CurrentUserHelper
	if authService != nil {
		// public-page remember-me restores resolve the client IP via clientIP with trustedProxies (see current_user.go)
		helper = NewCurrentUserHelper(authService, secureCookie, trustedProxies)
	}

	return &PublicHandlers{
		postService:       postService,
		postViewService:   postViewService,
		reactionService:   reactionService,
		currentUserHelper: helper,
		trustedProxies:    resolvedTrustedProxies(helper, trustedProxies),
		blogTitle:         blogTitle,
		baseURL:           baseURL,
		postsPerPage:      postsPerPage,
		healthDisplay:     healthDisplay,
		homeTemplate:      homeTemplate,
		postTemplate:      postTemplate,
		tagsTemplate:      tagsTemplate,
		tagPostsTemplate:  tagPostsTemplate,
		notFoundTemplate:  notFoundTemplate,
		healthTemplate:    healthTemplate,
	}
}

// attachHealthSummaries fills Post.HealthSummary for posts that have a
// HealthDate with data. One batched query per page (mirrors AttachReactions).
// No-op when the Health Planet integration is disabled.
func (h *PublicHandlers) attachHealthSummaries(posts []*domain.Post) {
	if h.healthDisplay == nil || len(posts) == 0 {
		return
	}
	dates := make([]string, 0, len(posts))
	for _, p := range posts {
		if p.HealthDate != nil {
			dates = append(dates, *p.HealthDate)
		}
	}
	if len(dates) == 0 {
		return
	}
	summaries, err := h.healthDisplay.SummariesForDates(dates)
	if err != nil {
		log.Printf("warning: failed to attach health summaries: %v", err)
		return // バッジは装飾。ページ全体は落とさない
	}
	for _, p := range posts {
		if p.HealthDate != nil {
			p.HealthSummary = summaries[*p.HealthDate] // 無い日付は nil のまま
		}
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

// isAdminRequest reports whether the current request belongs to an
// authenticated admin. It defers to CurrentUserHelper so that an expired
// session_id is automatically restored from the remember_token cookie —
// which is why this method takes a ResponseWriter: a successful restore
// emits fresh session+CSRF cookies on the response.
//
// Performance: when no cookie is present, CurrentUserHelper short-circuits
// without touching the session store or the DB.
func (h *PublicHandlers) isAdminRequest(w http.ResponseWriter, r *http.Request) bool {
	if h.currentUserHelper == nil {
		return false
	}
	user, err := h.currentUserHelper.Optional(w, r)
	if err != nil {
		return false
	}
	return user != nil
}

// renderNotFound renders the 404 page
func (h *PublicHandlers) renderNotFound(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{
		"SiteTitle":     h.blogTitle,
		"PinnedPosts":   h.getPinnedPosts(),
		"OGP":           h.defaultOGP("Not Found - "+h.blogTitle, r.URL.Path, h.blogTitle),
		"Query":         "",
		"HealthEnabled": h.healthDisplay != nil,
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

	perPage := h.postsPerPage
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

	if h.reactionService != nil {
		if err := h.reactionService.AttachReactions(posts); err != nil {
			log.Printf("failed to attach reactions: %v", err)
		}
	}
	h.attachHealthSummaries(posts)

	data := map[string]any{
		"SiteTitle":     h.blogTitle,
		"Posts":         posts,
		"CurrentPage":   page,
		"HasPrev":       page > 1,
		"HasNext":       hasNext,
		"PrevPage":      page - 1,
		"NextPage":      page + 1,
		"Query":         queryStr,
		"NoIndex":       queryStr != "" || page > 1,
		"PinnedPosts":   h.getPinnedPosts(),
		"OGP":           h.defaultOGP(h.blogTitle, r.URL.Path, h.blogTitle),
		"IsAdmin":       h.isAdminRequest(w, r),
		"HealthEnabled": h.healthDisplay != nil,
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
		ip := clientIP(r, h.trustedProxies)
		ua := r.UserAgent()
		postID := post.ID
		go func() {
			if err := h.postViewService.RecordView(postID, ip, ua); err != nil {
				log.Printf("failed to record view for post %d: %v", postID, err)
			}
		}()
	}

	// Attach reaction summaries for SSR. Counts are visitor-independent (empty
	// visitor key => reacted=false everywhere), so they don't add a per-visitor
	// dimension to the rendered HTML; the per-visitor reacted state is layered
	// on client-side by reactions.js.
	if h.reactionService != nil {
		if err := h.reactionService.AttachReactions([]*domain.Post{post}); err != nil {
			log.Printf("failed to attach reactions for post %d: %v", post.ID, err)
		}
	}
	h.attachHealthSummaries([]*domain.Post{post})

	data := map[string]any{
		"SiteTitle":     h.blogTitle,
		"Post":          post,
		"PinnedPosts":   h.getPinnedPosts(),
		"OGP":           h.postOGP(post, r.URL.Path),
		"Query":         "",
		"IsAdmin":       h.isAdminRequest(w, r),
		"HealthEnabled": h.healthDisplay != nil,
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
		"SiteTitle":     h.blogTitle,
		"Tags":          tags,
		"PinnedPosts":   h.getPinnedPosts(),
		"OGP":           h.defaultOGP("Tags - "+h.blogTitle, r.URL.Path, "Tags from "+h.blogTitle),
		"Query":         "",
		"HealthEnabled": h.healthDisplay != nil,
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

	perPage := h.postsPerPage
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

	if h.reactionService != nil {
		if err := h.reactionService.AttachReactions(posts); err != nil {
			log.Printf("failed to attach reactions: %v", err)
		}
	}
	h.attachHealthSummaries(posts)

	data := map[string]any{
		"SiteTitle":     h.blogTitle,
		"Tag":           tag,
		"Posts":         posts,
		"CurrentPage":   page,
		"HasPrev":       page > 1,
		"HasNext":       hasNext,
		"PrevPage":      page - 1,
		"NextPage":      page + 1,
		"PinnedPosts":   h.getPinnedPosts(),
		"OGP":           h.defaultOGP(tag+" - "+h.blogTitle, r.URL.Path, "Posts tagged with "+tag),
		"Query":         "",
		"IsAdmin":       h.isAdminRequest(w, r),
		"HealthEnabled": h.healthDisplay != nil,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tagPostsTemplate.ExecuteTemplate(w, "tag_posts", data); err != nil {
		log.Printf("failed to execute template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
