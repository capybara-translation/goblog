package http

import (
	"embed"
	"io/fs"
	"net/http"
	"os"

	"github.com/capybara-translation/goblog/internal/service"
	"github.com/gorilla/mux"
)

// noDirListingFS wraps http.FileSystem to disable directory listing.
// Returns 404 for directory access instead of listing files.
type noDirListingFS struct {
	fs http.FileSystem
}

func (n noDirListingFS) Open(name string) (http.File, error) {
	f, err := n.fs.Open(name)
	if err != nil {
		return nil, err
	}
	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if stat.IsDir() {
		f.Close()
		return nil, os.ErrNotExist
	}
	return f, nil
}

// NewRouter creates the application router using embedded resources
func NewRouter(postService service.PostService, postViewService service.PostViewService, authService service.AuthService, ogpService service.OGPService, secureCookie bool, trustedProxies []string, blogTitle, baseURL, uploadDir string, maxUploadSize int64, postsPerPage int, templatesFS, staticFS embed.FS) *mux.Router {
	r := mux.NewRouter()

	// Initialize public page handlers (using embedded templates)
	publicHandlers := NewPublicHandlers(postService, postViewService, ogpService, authService, blogTitle, baseURL, postsPerPage, templatesFS)

	// Display custom 404 page for non-existent routes
	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		publicHandlers.renderNotFound(w, req)
	})

	// Initialize admin API handlers
	apiHandlers := NewAPIHandlers(postService, postViewService)

	// Initialize auth handlers
	authHandlers := NewAuthHandlers(authService, secureCookie, trustedProxies)

	// Initialize image upload handlers
	imageHandlers := NewImageHandlers(uploadDir, maxUploadSize)

	// Static files (favicon, etc.) - served from embedded files
	staticSubFS, _ := fs.Sub(staticFS, "internal/view/static")
	staticFileServer := http.FileServer(http.FS(staticSubFS))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", staticFileServer))

	// アップロードされた画像の配信（認証不要、ディレクトリリスティング無効）
	uploadsFileServer := http.FileServer(noDirListingFS{http.Dir(uploadDir)})
	r.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", uploadsFileServer))

	// 公開ページ（SSR）
	r.HandleFunc("/", publicHandlers.HandleHome).Methods("GET")
	r.HandleFunc("/sitemap.xml", publicHandlers.HandleSitemap).Methods("GET")
	r.HandleFunc("/posts/{slug}", publicHandlers.HandlePostDetail).Methods("GET")
	r.HandleFunc("/posts", publicHandlers.HandlePosts).Methods("GET")
	r.HandleFunc("/posts/", publicHandlers.HandlePosts).Methods("GET")
	r.HandleFunc("/tags", publicHandlers.HandleTags).Methods("GET")
	r.HandleFunc("/tags/{tag}", publicHandlers.HandleTagPosts).Methods("GET")

	// 管理画面（SPA）
	r.HandleFunc("/admin", HandleAdminSPA).Methods("GET")
	r.PathPrefix("/admin/").HandlerFunc(HandleAdminSPA)

	// API（管理画面が叩く）
	api := r.PathPrefix("/api/v1").Subrouter()

	// 認証不要なエンドポイント
	api.HandleFunc("/health", HandleHealth).Methods("GET")
	api.HandleFunc("/auth/login", authHandlers.HandleLogin).Methods("POST")

	// 認証が必要なエンドポイント
	protectedAPI := api.PathPrefix("").Subrouter()
	protectedAPI.Use(AuthMiddleware(authService))
	protectedAPI.Use(CSRFMiddleware())

	// 認証API
	protectedAPI.HandleFunc("/auth/logout", authHandlers.HandleLogout).Methods("POST")
	protectedAPI.HandleFunc("/auth/me", authHandlers.HandleMe).Methods("GET")

	// 記事管理API
	protectedAPI.HandleFunc("/posts", apiHandlers.HandleGetPosts).Methods("GET")
	protectedAPI.HandleFunc("/posts", apiHandlers.HandleCreatePost).Methods("POST")
	protectedAPI.HandleFunc("/posts/{id}", apiHandlers.HandleGetPost).Methods("GET")
	protectedAPI.HandleFunc("/posts/{id}", apiHandlers.HandleUpdatePost).Methods("PUT")
	protectedAPI.HandleFunc("/posts/{id}", apiHandlers.HandleDeletePost).Methods("DELETE")
	protectedAPI.HandleFunc("/posts/{id}/publish", apiHandlers.HandlePublishPost).Methods("POST")
	protectedAPI.HandleFunc("/posts/{id}/unpublish", apiHandlers.HandleUnpublishPost).Methods("POST")
	protectedAPI.HandleFunc("/posts/{id}/pin", apiHandlers.HandlePinPost).Methods("POST")
	protectedAPI.HandleFunc("/posts/{id}/unpin", apiHandlers.HandleUnpinPost).Methods("POST")
	protectedAPI.HandleFunc("/tags", apiHandlers.HandleGetTags).Methods("GET")

	// Markdown preview (with OGP link card support)
	previewHandler := NewPreviewHandler(ogpService)
	protectedAPI.HandleFunc("/markdown/preview", previewHandler.HandlePreview).Methods("POST")

	// Image upload
	protectedAPI.HandleFunc("/images", imageHandlers.HandleUploadImage).Methods("POST")

	return r
}

// NewRouterWithTemplates creates a router with specified template path (for testing)
func NewRouterWithTemplates(postService service.PostService, postViewService service.PostViewService, authService service.AuthService, secureCookie bool, trustedProxies []string, blogTitle, baseURL, templatePattern string, uploadDir string, maxUploadSize int64, postsPerPage int) *mux.Router {
	r := mux.NewRouter()

	// Initialize public page handlers (loading templates from filesystem)
	publicHandlers := NewPublicHandlersFromPath(postService, postViewService, authService, blogTitle, baseURL, templatePattern, postsPerPage)

	// Display custom 404 page for non-existent routes
	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		publicHandlers.renderNotFound(w, req)
	})

	// Initialize admin API handlers
	apiHandlers := NewAPIHandlers(postService, postViewService)

	// Initialize auth handlers
	authHandlers := NewAuthHandlers(authService, secureCookie, trustedProxies)

	// Initialize image upload handlers
	imageHandlers := NewImageHandlers(uploadDir, maxUploadSize)

	// Static files (served from filesystem)
	staticFileServer := http.FileServer(http.Dir("internal/view/static"))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", staticFileServer))

	// Serve uploaded images (no authentication required, directory listing disabled)
	uploadsFileServer := http.FileServer(noDirListingFS{http.Dir(uploadDir)})
	r.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", uploadsFileServer))

	// Public pages (SSR)
	r.HandleFunc("/", publicHandlers.HandleHome).Methods("GET")
	r.HandleFunc("/sitemap.xml", publicHandlers.HandleSitemap).Methods("GET")
	r.HandleFunc("/posts/{slug}", publicHandlers.HandlePostDetail).Methods("GET")
	r.HandleFunc("/posts", publicHandlers.HandlePosts).Methods("GET")
	r.HandleFunc("/posts/", publicHandlers.HandlePosts).Methods("GET")
	r.HandleFunc("/tags", publicHandlers.HandleTags).Methods("GET")
	r.HandleFunc("/tags/{tag}", publicHandlers.HandleTagPosts).Methods("GET")

	// Admin panel (SPA)
	r.HandleFunc("/admin", HandleAdminSPA).Methods("GET")
	r.PathPrefix("/admin/").HandlerFunc(HandleAdminSPA)

	// API (used by admin panel)
	api := r.PathPrefix("/api/v1").Subrouter()

	// Endpoints that don't require authentication
	api.HandleFunc("/health", HandleHealth).Methods("GET")
	api.HandleFunc("/auth/login", authHandlers.HandleLogin).Methods("POST")

	// Endpoints that require authentication
	protectedAPI := api.PathPrefix("").Subrouter()
	protectedAPI.Use(AuthMiddleware(authService))
	protectedAPI.Use(CSRFMiddleware())

	// Auth API
	protectedAPI.HandleFunc("/auth/logout", authHandlers.HandleLogout).Methods("POST")
	protectedAPI.HandleFunc("/auth/me", authHandlers.HandleMe).Methods("GET")

	// Post management API
	protectedAPI.HandleFunc("/posts", apiHandlers.HandleGetPosts).Methods("GET")
	protectedAPI.HandleFunc("/posts", apiHandlers.HandleCreatePost).Methods("POST")
	protectedAPI.HandleFunc("/posts/{id}", apiHandlers.HandleGetPost).Methods("GET")
	protectedAPI.HandleFunc("/posts/{id}", apiHandlers.HandleUpdatePost).Methods("PUT")
	protectedAPI.HandleFunc("/posts/{id}", apiHandlers.HandleDeletePost).Methods("DELETE")
	protectedAPI.HandleFunc("/posts/{id}/publish", apiHandlers.HandlePublishPost).Methods("POST")
	protectedAPI.HandleFunc("/posts/{id}/unpublish", apiHandlers.HandleUnpublishPost).Methods("POST")
	protectedAPI.HandleFunc("/posts/{id}/pin", apiHandlers.HandlePinPost).Methods("POST")
	protectedAPI.HandleFunc("/posts/{id}/unpin", apiHandlers.HandleUnpinPost).Methods("POST")
	protectedAPI.HandleFunc("/tags", apiHandlers.HandleGetTags).Methods("GET")

	// Markdown preview (without OGP support in test mode)
	testPreviewHandler := NewPreviewHandler(nil)
	protectedAPI.HandleFunc("/markdown/preview", testPreviewHandler.HandlePreview).Methods("POST")

	// Image upload
	protectedAPI.HandleFunc("/images", imageHandlers.HandleUploadImage).Methods("POST")

	return r
}
