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

	// Shared image-dimensions cache. The Markdown renderer asks it for
	// width/height to emit on <img>; the upload handler primes it after
	// each save so the first public-page render avoids a disk hit.
	dimensions := service.NewDiskDimensionsService(uploadDir)

	// Shared WebP variants cache. The renderer asks it for the set of
	// "-Nw.webp" siblings of each /uploads/<file> URL to compose srcset.
	// The upload handler generates the variants in a background goroutine.
	variants := service.NewDiskVariantsService(uploadDir)

	// Initialize public page handlers (using embedded templates)
	publicHandlers := NewPublicHandlers(postService, postViewService, ogpService, authService, secureCookie, blogTitle, baseURL, postsPerPage, templatesFS, dimensions, variants)

	// Display custom 404 page for non-existent routes
	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		publicHandlers.renderNotFound(w, req)
	})

	// Initialize admin API handlers
	apiHandlers := NewAPIHandlers(postService, postViewService)

	// Initialize auth handlers
	authHandlers := NewAuthHandlers(authService, secureCookie, trustedProxies)

	// Shared helper for resolving the current user (session or remember-me).
	// Used by AuthMiddleware so protected endpoints honor remember tokens.
	currentUserHelper := NewCurrentUserHelper(authService, secureCookie)

	// Initialize image upload handlers
	imageHandlers := NewImageHandlers(uploadDir, maxUploadSize, dimensions)

	// Static files (favicon, etc.) - served from embedded files
	staticSubFS, _ := fs.Sub(staticFS, "internal/view/static")
	staticFileServer := http.FileServer(http.FS(staticSubFS))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", staticFileServer))

	// アップロードされた画像の配信（認証不要、ディレクトリリスティング無効）
	// UUID ファイル名でアップロードされ、同 URL の中身は差し替えないため
	// long-lived immutable キャッシュを付与している。read-only エンドポイント
	// なので GET/HEAD のみ受け付ける（POST 等は 405）。
	uploadsFileServer := http.FileServer(noDirListingFS{http.Dir(uploadDir)})
	r.PathPrefix("/uploads/").Handler(uploadsCacheControl(http.StripPrefix("/uploads/", uploadsFileServer))).Methods("GET", "HEAD")

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
	protectedAPI.Use(AuthMiddleware(currentUserHelper))
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

	// Markdown preview (with OGP link card support + width/height + srcset)
	previewHandler := NewPreviewHandler(ogpService, dimensions, variants)
	protectedAPI.HandleFunc("/markdown/preview", previewHandler.HandlePreview).Methods("POST")

	// Image upload
	protectedAPI.HandleFunc("/images", imageHandlers.HandleUploadImage).Methods("POST")

	return r
}

// NewRouterWithTemplates creates a router with specified template path (for testing)
func NewRouterWithTemplates(postService service.PostService, postViewService service.PostViewService, authService service.AuthService, secureCookie bool, trustedProxies []string, blogTitle, baseURL, templatePattern string, uploadDir string, maxUploadSize int64, postsPerPage int) *mux.Router {
	r := mux.NewRouter()

	// Shared image-dimensions / variants caches (see NewRouter for rationale).
	dimensions := service.NewDiskDimensionsService(uploadDir)
	variants := service.NewDiskVariantsService(uploadDir)

	// Initialize public page handlers (loading templates from filesystem)
	publicHandlers := NewPublicHandlersFromPath(postService, postViewService, authService, secureCookie, blogTitle, baseURL, templatePattern, postsPerPage, dimensions, variants)

	// Display custom 404 page for non-existent routes
	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		publicHandlers.renderNotFound(w, req)
	})

	// Initialize admin API handlers
	apiHandlers := NewAPIHandlers(postService, postViewService)

	// Initialize auth handlers
	authHandlers := NewAuthHandlers(authService, secureCookie, trustedProxies)

	// Shared helper for resolving the current user (session or remember-me).
	currentUserHelper := NewCurrentUserHelper(authService, secureCookie)

	// Initialize image upload handlers
	imageHandlers := NewImageHandlers(uploadDir, maxUploadSize, dimensions)

	// Static files (served from filesystem)
	staticFileServer := http.FileServer(http.Dir("internal/view/static"))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", staticFileServer))

	// Serve uploaded images (no authentication required, directory listing disabled)
	// Filenames are UUIDs and contents are never updated in place, so we tag
	// the responses with a long-lived immutable Cache-Control directive.
	// Read-only endpoint: only GET/HEAD are accepted (other methods 405).
	uploadsFileServer := http.FileServer(noDirListingFS{http.Dir(uploadDir)})
	r.PathPrefix("/uploads/").Handler(uploadsCacheControl(http.StripPrefix("/uploads/", uploadsFileServer))).Methods("GET", "HEAD")

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
	protectedAPI.Use(AuthMiddleware(currentUserHelper))
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

	// Markdown preview (without OGP support in test mode; dimensions + variants still flow)
	testPreviewHandler := NewPreviewHandler(nil, dimensions, variants)
	protectedAPI.HandleFunc("/markdown/preview", testPreviewHandler.HandlePreview).Methods("POST")

	// Image upload
	protectedAPI.HandleFunc("/images", imageHandlers.HandleUploadImage).Methods("POST")

	return r
}
