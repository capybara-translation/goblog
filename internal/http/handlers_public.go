package http

import (
	"html"
	"html/template"
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

// PublicHandlers は公開ページのハンドラーをまとめた構造体です
type PublicHandlers struct {
	postService      service.PostService
	blogTitle        string // ブログのタイトル
	homeTemplate     *template.Template
	postsTemplate    *template.Template
	postTemplate     *template.Template
	tagsTemplate     *template.Template
	tagPostsTemplate *template.Template
}

// truncateRunes は文字列をルーン（文字）単位で切り詰めます
// バイト単位ではなくルーン単位で切り詰めることで、日本語などのマルチバイト文字を安全に扱えます
func truncateRunes(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes])
}

// htmlTagPattern はHTMLタグにマッチする正規表現パターン
var htmlTagPattern = regexp.MustCompile(`<[^>]*>`)

// stripHTMLTags はHTML文字列からタグを除去してプレーンテキストを返します
func stripHTMLTags(htmlStr string) string {
	// HTMLタグを除去
	text := htmlTagPattern.ReplaceAllString(htmlStr, "")
	// HTMLエンティティをデコード
	text = html.UnescapeString(text)
	// 連続する空白を1つにまとめ、前後の空白を削除
	text = strings.Join(strings.Fields(text), " ")
	return text
}

// splitTags はカンマ区切りのタグ文字列をスライスに変換します
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

// getTimezoneLocation はTZ環境変数からタイムゾーンを取得します
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

// formatDateWithTZ は日付をISO 8601形式 + タイムゾーン略称で表示します
// TZ環境変数で設定されたタイムゾーンに変換してからフォーマットします
func formatDateWithTZ(t time.Time) string {
	return t.In(getTimezoneLocation()).Format("2006-01-02 (MST)")
}

// formatDateDetailWithTZ は日付を詳細形式（時刻+タイムゾーン略称）で表示します
// マウスオーバー時のツールチップ用
func formatDateDetailWithTZ(t time.Time) string {
	return t.In(getTimezoneLocation()).Format("2006-01-02 15:04 (MST)")
}

// mdConverter はMarkdown変換器のシングルトンインスタンス
var mdConverter = markdown.NewConverter()

// renderMarkdown はMarkdownをHTMLに変換します
func renderMarkdown(content string) template.HTML {
	htmlContent, err := mdConverter.Convert(content)
	if err != nil {
		// エラー時はHTMLエスケープしたテキストを返す
		return template.HTML(template.HTMLEscapeString(content))
	}
	return template.HTML(htmlContent)
}

// markdownExcerpt はMarkdownをプレーンテキストに変換して切り詰めます
func markdownExcerpt(content string, maxLen int) string {
	// Markdown → HTML → プレーンテキスト → 切り詰め
	htmlContent, err := mdConverter.Convert(content)
	if err != nil {
		// エラー時は元のテキストを切り詰めて返す
		return truncateRunes(content, maxLen)
	}
	plainText := stripHTMLTags(htmlContent)
	return truncateRunes(plainText, maxLen)
}

// highlightQuery は検索クエリに一致する文字列を<mark>タグでハイライトします
// XSS対策のため、テキストはHTMLエスケープしてから処理します
func highlightQuery(text string, query string) template.HTML {
	// テキストをHTMLエスケープ
	escapedText := html.EscapeString(text)

	if query == "" {
		return template.HTML(escapedText)
	}

	// クエリもHTMLエスケープ（エスケープ後のテキストとマッチさせるため）
	escapedQuery := html.EscapeString(query)

	// 正規表現の特殊文字をエスケープ
	quotedQuery := regexp.QuoteMeta(escapedQuery)

	// 大文字小文字を区別しないパターン
	pattern := regexp.MustCompile("(?i)(" + quotedQuery + ")")

	// マッチ部分を<mark>でラップ
	highlighted := pattern.ReplaceAllString(escapedText, "<mark>$1</mark>")

	return template.HTML(highlighted)
}

// NewPublicHandlers はテンプレートパスを指定してPublicHandlersを作成します
func NewPublicHandlers(postService service.PostService, blogTitle string, templatePattern string) *PublicHandlers {
	dir := filepath.Dir(templatePattern)
	layoutPath := filepath.Join(dir, "layout.html")

	// カスタムテンプレート関数を定義
	funcMap := template.FuncMap{
		"truncate":               truncateRunes,
		"splitTags":              splitTags,
		"formatDateWithTZ":       formatDateWithTZ,
		"formatDateDetailWithTZ": formatDateDetailWithTZ,
		"highlightQuery":         highlightQuery,
		"renderMarkdown":         renderMarkdown,
		"markdownExcerpt":        markdownExcerpt,
	}

	// 各ページごとに独立したテンプレートセットを作成
	homeTemplate := template.Must(template.New("").Funcs(funcMap).ParseFiles(layoutPath, filepath.Join(dir, "home.html")))
	postsTemplate := template.Must(template.New("").Funcs(funcMap).ParseFiles(layoutPath, filepath.Join(dir, "posts.html")))
	postTemplate := template.Must(template.New("").Funcs(funcMap).ParseFiles(layoutPath, filepath.Join(dir, "post.html")))
	tagsTemplate := template.Must(template.New("").Funcs(funcMap).ParseFiles(layoutPath, filepath.Join(dir, "tags.html")))
	tagPostsTemplate := template.Must(template.New("").Funcs(funcMap).ParseFiles(layoutPath, filepath.Join(dir, "tag_posts.html")))

	return &PublicHandlers{
		postService:      postService,
		blogTitle:        blogTitle,
		homeTemplate:     homeTemplate,
		postsTemplate:    postsTemplate,
		postTemplate:     postTemplate,
		tagsTemplate:     tagsTemplate,
		tagPostsTemplate: tagPostsTemplate,
	}
}

// HandleHome はトップページを表示します
func (h *PublicHandlers) HandleHome(w http.ResponseWriter, r *http.Request) {
	// 最近の記事を5件取得
	posts, err := h.postService.GetPublishedPosts(5, 0)
	if err != nil {
		log.Printf("failed to get published posts: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"SiteTitle": h.blogTitle,
		"Posts":     posts,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.homeTemplate.ExecuteTemplate(w, "home", data); err != nil {
		log.Printf("failed to execute template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// HandlePosts は記事一覧ページを表示します
func (h *PublicHandlers) HandlePosts(w http.ResponseWriter, r *http.Request) {
	// クエリパラメータから page と q を取得
	pageStr := r.URL.Query().Get("page")
	queryStr := r.URL.Query().Get("q")
	page := 1
	if pageStr != "" {
		if parsed, err := strconv.Atoi(pageStr); err == nil && parsed > 0 {
			page = parsed
		}
	}

	// 1ページあたりの記事数
	const perPage = 20
	offset := (page - 1) * perPage

	var posts []*domain.Post
	var err error

	// 次のページがあるか判定するため、perPage+1 件取得
	if queryStr != "" {
		// 検索モード
		posts, err = h.postService.SearchPublishedPosts(queryStr, perPage+1, offset)
	} else {
		posts, err = h.postService.GetPublishedPosts(perPage+1, offset)
	}
	if err != nil {
		log.Printf("failed to get published posts: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 次のページがあるか判定
	hasNext := len(posts) > perPage
	if hasNext {
		posts = posts[:perPage] // 表示用に perPage 件に切り詰める
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
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.postsTemplate.ExecuteTemplate(w, "posts", data); err != nil {
		log.Printf("failed to execute template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// HandlePostDetail は記事詳細ページを表示します
func (h *PublicHandlers) HandlePostDetail(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slug := vars["slug"]

	// スラッグで記事を取得
	post, err := h.postService.GetPostBySlug(slug)
	if err != nil {
		log.Printf("failed to get post by slug: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if post == nil {
		http.NotFound(w, r)
		return
	}

	data := map[string]any{
		"SiteTitle": h.blogTitle,
		"Post":      post,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.postTemplate.ExecuteTemplate(w, "post", data); err != nil {
		log.Printf("failed to execute template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// HandleTags はタグ一覧ページを表示します
func (h *PublicHandlers) HandleTags(w http.ResponseWriter, r *http.Request) {
	// 公開済み記事のタグを取得
	tagCounts, err := h.postService.GetPublishedTags()
	if err != nil {
		log.Printf("failed to get published tags: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// タグを記事数の降順でソート
	type TagInfo struct {
		Name  string
		Count int
	}

	tags := make([]TagInfo, 0, len(tagCounts))
	for name, count := range tagCounts {
		tags = append(tags, TagInfo{Name: name, Count: count})
	}

	// 記事数の降順、同数の場合はタグ名の昇順でソート
	slices.SortFunc(tags, func(a, b TagInfo) int {
		// 記事数で比較（降順）
		if a.Count != b.Count {
			return b.Count - a.Count
		}
		// タグ名で比較（昇順）
		if a.Name < b.Name {
			return -1
		} else if a.Name > b.Name {
			return 1
		}
		return 0
	})

	data := map[string]any{
		"SiteTitle": h.blogTitle,
		"Tags":      tags,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tagsTemplate.ExecuteTemplate(w, "tags", data); err != nil {
		log.Printf("failed to execute template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// HandleTagPosts は特定のタグの記事一覧ページを表示します
func (h *PublicHandlers) HandleTagPosts(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tag := vars["tag"]

	if tag == "" {
		http.NotFound(w, r)
		return
	}

	// URLデコード
	decodedTag, err := url.QueryUnescape(tag)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	tag = decodedTag

	// ページネーション
	pageStr := r.URL.Query().Get("page")
	page := 1
	if pageStr != "" {
		if parsed, err := strconv.Atoi(pageStr); err == nil && parsed > 0 {
			page = parsed
		}
	}

	const perPage = 20
	offset := (page - 1) * perPage

	// 記事を取得
	posts, err := h.postService.GetPublishedPostsByTag(tag, perPage+1, offset)
	if err != nil {
		log.Printf("failed to get posts by tag: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 次のページがあるか判定
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
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tagPostsTemplate.ExecuteTemplate(w, "tag_posts", data); err != nil {
		log.Printf("failed to execute template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}