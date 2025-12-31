package http

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"

	"github.com/capybara-translation/goblog/internal/service"
)

// contextKey はコンテキストのキーを表す型です
type contextKey string

const (
	// contextKeyUserID はコンテキストに格納するユーザーIDのキーです
	contextKeyUserID contextKey = "user_id"

	// csrfCookieName はCSRFトークンを格納するCookieの名前です
	csrfCookieName = "csrf_token"

	// csrfHeaderName はCSRFトークンを含むHTTPヘッダーの名前です
	csrfHeaderName = "X-CSRF-Token"

	// csrfTokenLength はCSRFトークンのバイト長です
	csrfTokenLength = 32
)

// AuthMiddleware は認証が必要なエンドポイントを保護するミドルウェアです
func AuthMiddleware(authService service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Cookieからセッション IDを取得
			cookie, err := r.Cookie(sessionCookieName)
			if err != nil {
				respondJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "Authentication required"})
				return
			}

			// セッションからユーザー情報を取得
			user, err := authService.GetUserBySession(cookie.Value)
			if err != nil {
				log.Printf("get user by session error: %v", err)
				respondJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
				return
			}

			if user == nil {
				respondJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "Session expired or invalid"})
				return
			}

			// ユーザーIDをコンテキストに設定
			ctx := context.WithValue(r.Context(), contextKeyUserID, user.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserIDFromContext はコンテキストからユーザーIDを取得します
func GetUserIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(contextKeyUserID).(int64)
	return userID, ok
}

// generateCSRFToken はランダムなCSRFトークンを生成します
func generateCSRFToken() (string, error) {
	b := make([]byte, csrfTokenLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// CSRFMiddleware はCSRF攻撃を防ぐミドルウェアです（Double Submit Cookie方式）
// 状態変更を伴うメソッド（POST、PUT、DELETE、PATCH）に対してCSRFトークンを検証します
func CSRFMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// GETとHEADメソッドはCSRFチェックをスキップ（冪等性が保証されている）
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			// CookieからCSRFトークンを取得
			cookie, err := r.Cookie(csrfCookieName)
			if err != nil {
				respondJSON(w, http.StatusForbidden, ErrorResponse{Error: "CSRF token cookie missing"})
				return
			}

			// HTTPヘッダーからCSRFトークンを取得
			headerToken := r.Header.Get(csrfHeaderName)
			if headerToken == "" {
				respondJSON(w, http.StatusForbidden, ErrorResponse{Error: "CSRF token header missing"})
				return
			}

			// Cookieとヘッダーのトークンを比較
			if cookie.Value != headerToken {
				respondJSON(w, http.StatusForbidden, ErrorResponse{Error: "CSRF token mismatch"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
