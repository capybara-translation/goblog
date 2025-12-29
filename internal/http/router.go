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
	api.HandleFunc("/health", HandleHealth).Methods("GET")

	// 認証API（認証不要）
	api.HandleFunc("/auth/login", authHandlers.HandleLogin).Methods("POST")
	api.HandleFunc("/auth/logout", authHandlers.HandleLogout).Methods("POST")
	api.HandleFunc("/auth/me", authHandlers.HandleMe).Methods("GET")

	// 記事管理API（現時点では認証不要だが、将来的に認証ミドルウェアを適用する）
	api.HandleFunc("/posts", apiHandlers.HandleGetPosts).Methods("GET")
	api.HandleFunc("/posts", apiHandlers.HandleCreatePost).Methods("POST")
	api.HandleFunc("/posts/{id}", apiHandlers.HandleGetPost).Methods("GET")
	api.HandleFunc("/posts/{id}", apiHandlers.HandleUpdatePost).Methods("PUT")
	api.HandleFunc("/posts/{id}", apiHandlers.HandleDeletePost).Methods("DELETE")
	api.HandleFunc("/posts/{id}/publish", apiHandlers.HandlePublishPost).Methods("POST")
	api.HandleFunc("/posts/{id}/unpublish", apiHandlers.HandleUnpublishPost).Methods("POST")

	return r
}
