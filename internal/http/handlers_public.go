package http

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

// HandleHome はトップページを表示します
func HandleHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<h1>goblog - トップページ</h1><p>これからブログシステムを作っていきます。</p>")
}

// HandlePosts は記事一覧ページを表示します
func HandlePosts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<h1>記事一覧</h1><p>まだ記事はありません。</p>")
}

// HandlePostDetail は記事詳細ページを表示します
func HandlePostDetail(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slug := vars["slug"]

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<h1>記事詳細</h1><p>スラッグ: %s</p><p>まだ実装されていません。</p>", slug)
}

// HandleAdmin は管理画面のSPAを配信します（後でReactアプリを配信）
func HandleAdmin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<h1>管理画面</h1><p>後でReact SPAをここに配置します。</p>")
}