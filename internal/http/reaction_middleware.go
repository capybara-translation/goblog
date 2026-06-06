package http

import "net/http"

// RequireXRequestedWith is a lightweight CSRF defense for the public reaction
// API: state-changing requests must carry an X-Requested-With header. Custom
// headers cannot be set by cross-site <form> submits and trigger a CORS
// preflight for cross-origin fetch, so only same-origin JS we control can call
// these endpoints. GET/HEAD/OPTIONS bypass the check.
func RequireXRequestedWith() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}
			if r.Header.Get("X-Requested-With") == "" {
				respondJSON(w, http.StatusForbidden, ErrorResponse{Error: "missing X-Requested-With header"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
