package http

import (
	"github.com/gorilla/mux"
)

// NewRouter はアプリケーション全体のルーターを作成します
func NewRouter() *mux.Router {
	r := mux.NewRouter()

	// 公開ページ（SSR）
	r.HandleFunc("/", HandleHome).Methods("GET")
	r.HandleFunc("/posts/{slug}", HandlePostDetail).Methods("GET")
	r.HandleFunc("/posts", HandlePosts).Methods("GET")

	// 管理画面（SPA）
	r.HandleFunc("/admin", HandleAdmin).Methods("GET")
	r.PathPrefix("/admin/").HandlerFunc(HandleAdmin)

	// API（管理画面が叩く）
	api := r.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/health", HandleHealth).Methods("GET")

	return r
}