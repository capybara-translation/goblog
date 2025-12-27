package http

import (
	"fmt"
	"net/http"
)

// HandleHealth はAPIのヘルスチェックエンドポイントです
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"ok"}`)
}