package http

import (
	"context"
	"log"
	"net/http"

	"github.com/capybara-translation/goblog/internal/service"
)

// contextKey はコンテキストのキーを表す型です
type contextKey string

const (
	// contextKeyUserID はコンテキストに格納するユーザーIDのキーです
	contextKeyUserID contextKey = "user_id"
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
