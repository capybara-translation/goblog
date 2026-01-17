package http

import (
	"net/http"

	"github.com/capybara-translation/goblog/internal/service"
	"github.com/gorilla/mux"
)

// NewRouter はアプリケーション全体のルーターを作成します
func NewRouter(postService service.PostService, authService service.AuthService, secureCookie bool, blogTitle string, uploadDir string, maxUploadSize int64) *mux.Router {
	return NewRouterWithTemplates(postService, authService, secureCookie, blogTitle, "internal/view/templates/*.html", uploadDir, maxUploadSize)
}

// NewRouterWithTemplates はテンプレートパスを指定してルーターを作成します（テスト用）
func NewRouterWithTemplates(postService service.PostService, authService service.AuthService, secureCookie bool, blogTitle string, templatePattern string, uploadDir string, maxUploadSize int64) *mux.Router {
	r := mux.NewRouter()

	// 公開ページのハンドラーを初期化
	publicHandlers := NewPublicHandlers(postService, blogTitle, templatePattern)

	// 存在しないルートへのアクセス時にカスタム404ページを表示
	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		publicHandlers.renderNotFound(w)
	})

	// 管理画面API用のハンドラーを初期化
	apiHandlers := NewAPIHandlers(postService)

	// 認証用のハンドラーを初期化
	authHandlers := NewAuthHandlers(authService, secureCookie)

	// 画像アップロード用のハンドラーを初期化
	imageHandlers := NewImageHandlers(uploadDir, maxUploadSize)

	// 静的ファイル（favicon等）
	staticFileServer := http.FileServer(http.Dir("internal/view/static"))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", staticFileServer))

	// アップロードされた画像の配信（認証不要）
	uploadsFileServer := http.FileServer(http.Dir(uploadDir))
	r.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", uploadsFileServer))

	// 公開ページ（SSR）
	r.HandleFunc("/", publicHandlers.HandleHome).Methods("GET")
	r.HandleFunc("/posts/{slug}", publicHandlers.HandlePostDetail).Methods("GET")
	r.HandleFunc("/posts", publicHandlers.HandlePosts).Methods("GET")
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
	protectedAPI.HandleFunc("/tags", apiHandlers.HandleGetTags).Methods("GET")

	// Markdownプレビュー
	protectedAPI.HandleFunc("/markdown/preview", HandlePreview).Methods("POST")

	// 画像アップロード
	protectedAPI.HandleFunc("/images", imageHandlers.HandleUploadImage).Methods("POST")

	return r
}
