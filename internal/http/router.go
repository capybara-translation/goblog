package http

import (
	"github.com/capybara-translation/goblog/internal/service"
	"github.com/gorilla/mux"
)

// NewRouter はアプリケーション全体のルーターを作成します
func NewRouter(postService service.PostService, authService service.AuthService, secureCookie bool, blogTitle string) *mux.Router {
	return NewRouterWithTemplates(postService, authService, secureCookie, blogTitle, "internal/view/templates/*.html")
}

// NewRouterWithTemplates はテンプレートパスを指定してルーターを作成します（テスト用）
func NewRouterWithTemplates(postService service.PostService, authService service.AuthService, secureCookie bool, blogTitle string, templatePattern string) *mux.Router {
	r := mux.NewRouter()

	// 公開ページのハンドラーを初期化
	publicHandlers := NewPublicHandlers(postService, blogTitle, templatePattern)

	// 管理画面API用のハンドラーを初期化
	apiHandlers := NewAPIHandlers(postService)

	// 認証用のハンドラーを初期化
	authHandlers := NewAuthHandlers(authService, secureCookie)

	// 公開ページ（SSR）
	r.HandleFunc("/", publicHandlers.HandleHome).Methods("GET")
	r.HandleFunc("/posts/{slug}", publicHandlers.HandlePostDetail).Methods("GET")
	r.HandleFunc("/posts", publicHandlers.HandlePosts).Methods("GET")

	// 管理画面（SPA）
	r.HandleFunc("/admin", HandleAdmin).Methods("GET")
	r.PathPrefix("/admin/").HandlerFunc(HandleAdmin)

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

	return r
}
