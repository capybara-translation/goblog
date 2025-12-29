package http

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/capybara-translation/goblog/internal/service"
	"github.com/gorilla/mux"
)

// PublicHandlers は公開ページのハンドラーをまとめた構造体です
type PublicHandlers struct {
	postService   service.PostService
	blogTitle     string // ブログのタイトル
	homeTemplate  *template.Template
	postsTemplate *template.Template
	postTemplate  *template.Template
}

// NewPublicHandlers はテンプレートパスを指定してPublicHandlersを作成します
func NewPublicHandlers(postService service.PostService, blogTitle string, templatePattern string) *PublicHandlers {
	dir := filepath.Dir(templatePattern)
	layoutPath := filepath.Join(dir, "layout.html")

	// 各ページごとに独立したテンプレートセットを作成
	homeTemplate := template.Must(template.ParseFiles(layoutPath, filepath.Join(dir, "home.html")))
	postsTemplate := template.Must(template.ParseFiles(layoutPath, filepath.Join(dir, "posts.html")))
	postTemplate := template.Must(template.ParseFiles(layoutPath, filepath.Join(dir, "post.html")))

	return &PublicHandlers{
		postService:   postService,
		blogTitle:     blogTitle,
		homeTemplate:  homeTemplate,
		postsTemplate: postsTemplate,
		postTemplate:  postTemplate,
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
	// クエリパラメータから page を取得
	pageStr := r.URL.Query().Get("page")
	page := 1
	if pageStr != "" {
		if parsed, err := strconv.Atoi(pageStr); err == nil && parsed > 0 {
			page = parsed
		}
	}

	// 1ページあたりの記事数
	const perPage = 20
	offset := (page - 1) * perPage

	// 次のページがあるか判定するため、perPage+1 件取得
	posts, err := h.postService.GetPublishedPosts(perPage+1, offset)
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

// HandleAdmin は管理画面のSPAを配信します（後でReactアプリを配信）
func HandleAdmin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte("<h1>管理画面</h1><p>後でReact SPAをここに配置します。</p>"))
}
