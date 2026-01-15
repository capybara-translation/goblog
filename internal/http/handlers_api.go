package http

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/capybara-translation/goblog/internal/markdown"
	"github.com/capybara-translation/goblog/internal/service"
	"github.com/gorilla/mux"
)

// APIHandlers は管理画面API用のハンドラーをまとめた構造体です
type APIHandlers struct {
	postService service.PostService
}

// NewAPIHandlers は新しいAPIHandlersを作成します
func NewAPIHandlers(postService service.PostService) *APIHandlers {
	return &APIHandlers{
		postService: postService,
	}
}

// HandleHealth はAPIのヘルスチェックエンドポイントです
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"ok"}`)
}

// リクエスト/レスポンス用の構造体

// CreatePostRequest は記事作成リクエストの構造体です
type CreatePostRequest struct {
	Title   string `json:"title"`
	Slug    string `json:"slug"`
	Content string `json:"content"`
	Tags    string `json:"tags"`
}

// UpdatePostRequest は記事更新リクエストの構造体です
type UpdatePostRequest struct {
	Title   string `json:"title"`
	Slug    string `json:"slug"`
	Content string `json:"content"`
	Tags    string `json:"tags"`
}

// ErrorResponse はエラーレスポンスの構造体です
type ErrorResponse struct {
	Error string `json:"error"`
}

// PostsResponse は記事一覧APIのレスポンス構造体です
type PostsResponse struct {
	Posts []*domain.Post `json:"posts"`
	Total int            `json:"total"`
}

// ヘルパー関数

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("failed to encode JSON: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message})
}

// slugPattern はURLセーフなスラグのパターン（英小文字、数字、ハイフンのみ）
var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// validatePostRequest は記事作成・更新リクエストのバリデーションを行います
func validatePostRequest(title, slug string) error {
	// タイトル必須チェック
	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("title is required")
	}

	// スラグ必須チェック
	if strings.TrimSpace(slug) == "" {
		return fmt.Errorf("slug is required")
	}

	// スラグに空白が含まれていないかチェック
	if strings.ContainsAny(slug, " \t\n\r") {
		return fmt.Errorf("slug must not contain whitespace")
	}

	// スラグがURLセーフかチェック（英小文字、数字、ハイフンのみ）
	if !slugPattern.MatchString(slug) {
		return fmt.Errorf("slug must contain only lowercase letters, numbers, and hyphens")
	}

	return nil
}

// HandleGetPosts は記事一覧を取得します
// GET /api/v1/posts?status=draft|published&tag=Go&q=検索語&limit=20&offset=0
func (h *APIHandlers) HandleGetPosts(w http.ResponseWriter, r *http.Request) {
	// クエリパラメータを取得
	statusStr := r.URL.Query().Get("status")
	tagStr := r.URL.Query().Get("tag")
	queryStr := r.URL.Query().Get("q")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	// デフォルト値
	const maxLimit = 200
	limit := 20
	offset := 0

	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
			if limit > maxLimit {
				limit = maxLimit
			}
		}
	}
	if offsetStr != "" {
		if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	var status *domain.PostStatus
	if statusStr != "" {
		s, err := domain.ParsePostStatus(statusStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		status = &s
	}

	var posts []*domain.Post
	var total int
	var err error

	// 検索モード
	if queryStr != "" {
		posts, err = h.postService.SearchPosts(queryStr, status, limit, offset)
		if err == nil {
			total, err = h.postService.CountSearchPosts(queryStr, status)
		}
	} else if tagStr != "" {
		// タグフィルタリング
		posts, err = h.postService.GetAllPostsByTag(tagStr, status, limit, offset)
		if err == nil {
			total, err = h.postService.CountPostsByTag(tagStr, status)
		}
	} else {
		posts, err = h.postService.GetAllPosts(status, limit, offset)
		if err == nil {
			total, err = h.postService.CountPosts(status)
		}
	}

	if err != nil {
		log.Printf("failed to get posts: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to get posts")
		return
	}

	// nilスライスはJSONで"null"になるため、空スライスを保証
	if posts == nil {
		posts = []*domain.Post{}
	}

	writeJSON(w, http.StatusOK, PostsResponse{
		Posts: posts,
		Total: total,
	})
}

// HandleGetPost は記事詳細を取得します
// GET /api/v1/posts/{id}
func (h *APIHandlers) HandleGetPost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid post ID")
		return
	}

	post, err := h.postService.GetPostByID(id)
	if err != nil {
		log.Printf("failed to get post: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to get post")
		return
	}

	if post == nil {
		writeError(w, http.StatusNotFound, "Post not found")
		return
	}

	writeJSON(w, http.StatusOK, post)
}

// HandleCreatePost は記事を作成します
// POST /api/v1/posts
func (h *APIHandlers) HandleCreatePost(w http.ResponseWriter, r *http.Request) {
	var req CreatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// バリデーション
	if err := validatePostRequest(req.Title, req.Slug); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	post, err := h.postService.CreatePost(req.Title, req.Slug, req.Content, req.Tags)
	if err != nil {
		log.Printf("failed to create post: %v", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, post)
}

// HandleUpdatePost は記事を更新します
// PUT /api/v1/posts/{id}
func (h *APIHandlers) HandleUpdatePost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid post ID")
		return
	}

	var req UpdatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// バリデーション
	if err := validatePostRequest(req.Title, req.Slug); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	post, err := h.postService.UpdatePost(id, req.Title, req.Slug, req.Content, req.Tags)
	if err != nil {
		log.Printf("failed to update post: %v", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, post)
}

// HandleDeletePost は記事を削除します
// DELETE /api/v1/posts/{id}
func (h *APIHandlers) HandleDeletePost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid post ID")
		return
	}

	if err := h.postService.DeletePost(id); err != nil {
		log.Printf("failed to delete post: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to delete post")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandlePublishPost は記事を公開します
// POST /api/v1/posts/{id}/publish
func (h *APIHandlers) HandlePublishPost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid post ID")
		return
	}

	post, err := h.postService.PublishPost(id)
	if err != nil {
		log.Printf("failed to publish post: %v", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, post)
}

// HandleUnpublishPost は記事を非公開にします
// POST /api/v1/posts/{id}/unpublish
func (h *APIHandlers) HandleUnpublishPost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid post ID")
		return
	}

	post, err := h.postService.UnpublishPost(id)
	if err != nil {
		log.Printf("failed to unpublish post: %v", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, post)
}

// HandleGetTags はタグ一覧を取得します
// GET /api/v1/tags?status=published|draft
func (h *APIHandlers) HandleGetTags(w http.ResponseWriter, r *http.Request) {
	statusStr := r.URL.Query().Get("status")

	var status *domain.PostStatus
	if statusStr != "" {
		s, err := domain.ParsePostStatus(statusStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		status = &s
	}

	tagCounts, err := h.postService.GetAllTags(status)
	if err != nil {
		log.Printf("failed to get tags: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to get tags")
		return
	}

	// レスポンス用の構造体
	type TagResponse struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	tags := make([]TagResponse, 0, len(tagCounts))
	for name, count := range tagCounts {
		tags = append(tags, TagResponse{Name: name, Count: count})
	}

	// 記事数の降順でソート、同数の場合はタグ名の昇順
	slices.SortFunc(tags, func(a, b TagResponse) int {
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

	writeJSON(w, http.StatusOK, tags)
}

// PreviewRequest はMarkdownプレビューリクエストの構造体です
type PreviewRequest struct {
	Content string `json:"content"`
}

// PreviewResponse はMarkdownプレビューレスポンスの構造体です
type PreviewResponse struct {
	HTML string `json:"html"`
}

// HandlePreview はMarkdownをHTMLに変換して返します
// POST /api/v1/preview
func HandlePreview(w http.ResponseWriter, r *http.Request) {
	var req PreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	converter := markdown.NewConverter()
	html, err := converter.Convert(req.Content)
	if err != nil {
		log.Printf("failed to convert markdown: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to convert markdown")
		return
	}

	writeJSON(w, http.StatusOK, PreviewResponse{HTML: html})
}
