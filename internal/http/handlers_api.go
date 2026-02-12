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

// APIHandlers is a struct that groups handlers for admin panel API
type APIHandlers struct {
	postService service.PostService
}

// NewAPIHandlers creates a new APIHandlers
func NewAPIHandlers(postService service.PostService) *APIHandlers {
	return &APIHandlers{
		postService: postService,
	}
}

// HandleHealth is the API health check endpoint
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"ok"}`)
}

// Structs for requests/responses

// CreatePostRequest is the struct for post creation requests
type CreatePostRequest struct {
	Title    string `json:"title"`
	Slug     string `json:"slug"`
	Content  string `json:"content"`
	Tags     string `json:"tags"`
	IsPinned bool   `json:"is_pinned"`
}

// UpdatePostRequest is the struct for post update requests
type UpdatePostRequest struct {
	Title    string `json:"title"`
	Slug     string `json:"slug"`
	Content  string `json:"content"`
	Tags     string `json:"tags"`
	IsPinned bool   `json:"is_pinned"`
}

// ErrorResponse is the struct for error responses
type ErrorResponse struct {
	Error string `json:"error"`
}

// PostsResponse is the response struct for posts list API
type PostsResponse struct {
	Posts []*domain.Post `json:"posts"`
	Total int            `json:"total"`
}

// Helper functions

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

// slugPattern is the pattern for URL-safe slugs (lowercase letters, numbers, and hyphens only)
var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// validatePostRequest validates post creation and update requests
func validatePostRequest(title, slug string) error {
	// Title required check
	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("title is required")
	}

	// Slug required check
	if strings.TrimSpace(slug) == "" {
		return fmt.Errorf("slug is required")
	}

	// Check that slug does not contain whitespace
	if strings.ContainsAny(slug, " \t\n\r") {
		return fmt.Errorf("slug must not contain whitespace")
	}

	// Check if slug is URL-safe (lowercase letters, numbers, and hyphens only)
	if !slugPattern.MatchString(slug) {
		return fmt.Errorf("slug must contain only lowercase letters, numbers, and hyphens")
	}

	return nil
}

// HandleGetPosts retrieves the list of posts
// GET /api/v1/posts?status=draft|published&tag=Go&q=searchterm&limit=20&offset=0
func (h *APIHandlers) HandleGetPosts(w http.ResponseWriter, r *http.Request) {
	// Get query parameters
	statusStr := r.URL.Query().Get("status")
	tagStr := r.URL.Query().Get("tag")
	queryStr := r.URL.Query().Get("q")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	// Search query length limit
	const maxQueryLength = 200
	if len(queryStr) > maxQueryLength {
		writeError(w, http.StatusBadRequest, "search query too long (max 200 characters)")
		return
	}

	// Default values
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

	// Search mode
	if queryStr != "" {
		posts, err = h.postService.SearchPosts(queryStr, status, limit, offset)
		if err == nil {
			total, err = h.postService.CountSearchPosts(queryStr, status)
		}
	} else if tagStr != "" {
		// Tag filtering
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

	// Ensure empty slice instead of nil (nil becomes "null" in JSON)
	if posts == nil {
		posts = []*domain.Post{}
	}

	writeJSON(w, http.StatusOK, PostsResponse{
		Posts: posts,
		Total: total,
	})
}

// HandleGetPost retrieves post details
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

// HandleCreatePost creates a post
// POST /api/v1/posts
func (h *APIHandlers) HandleCreatePost(w http.ResponseWriter, r *http.Request) {
	var req CreatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validation
	if err := validatePostRequest(req.Title, req.Slug); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	post, err := h.postService.CreatePost(req.Title, req.Slug, req.Content, req.Tags, req.IsPinned)
	if err != nil {
		log.Printf("failed to create post: %v", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, post)
}

// HandleUpdatePost updates a post
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

	// Validation
	if err := validatePostRequest(req.Title, req.Slug); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	post, err := h.postService.UpdatePost(id, req.Title, req.Slug, req.Content, req.Tags, req.IsPinned)
	if err != nil {
		log.Printf("failed to update post: %v", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, post)
}

// HandleDeletePost deletes a post
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

// HandlePublishPost publishes a post
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

// HandleUnpublishPost unpublishes a post
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

// HandleGetTags retrieves the list of tags
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

	// Struct for response
	type TagResponse struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	tags := make([]TagResponse, 0, len(tagCounts))
	for name, count := range tagCounts {
		tags = append(tags, TagResponse{Name: name, Count: count})
	}

	// Sort by post count descending, then by tag name ascending if counts are equal
	slices.SortFunc(tags, func(a, b TagResponse) int {
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

	writeJSON(w, http.StatusOK, tags)
}

// PreviewRequest is the struct for Markdown preview requests
type PreviewRequest struct {
	Content string `json:"content"`
}

// PreviewResponse is the struct for Markdown preview responses
type PreviewResponse struct {
	HTML string `json:"html"`
}

// HandlePreview converts Markdown to HTML and returns it
// If the content contains @@CURSOR@@, it is replaced with a scroll marker
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

// HandlePinPost pins a post
// POST /api/v1/posts/{id}/pin
func (h *APIHandlers) HandlePinPost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid post ID")
		return
	}

	post, err := h.postService.SetPinned(id, true)
	if err != nil {
		log.Printf("failed to pin post: %v", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, post)
}

// HandleUnpinPost unpins a post
// POST /api/v1/posts/{id}/unpin
func (h *APIHandlers) HandleUnpinPost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid post ID")
		return
	}

	post, err := h.postService.SetPinned(id, false)
	if err != nil {
		log.Printf("failed to unpin post: %v", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, post)
}
