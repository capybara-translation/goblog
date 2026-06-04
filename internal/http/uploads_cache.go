package http

import "net/http"

// uploadsCacheControl wraps a handler serving /uploads/ assets and tags
// responses with a long-lived immutable cache directive. This is safe
// because the upload pipeline always allocates a new UUID per file
// (handlers_image.go) and the OGP cache replaces files by creating a
// new UUID and deleting the old one (ogp_service.go); the bytes behind
// a given /uploads/<uuid>.<ext> URL never change.
func uploadsCacheControl(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		next.ServeHTTP(w, r)
	})
}
