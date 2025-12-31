package http

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/capybara-translation/goblog/internal/service"
)

const (
	sessionCookieName = "session_id"
	sessionCookiePath = "/"
)

// AuthHandlers は認証関連のHTTPハンドラーをまとめた構造体です
type AuthHandlers struct {
	authService  service.AuthService
	secureCookie bool // Cookieのsecure属性（HTTPS必須）
}

// NewAuthHandlers は新しいAuthHandlersを作成します
func NewAuthHandlers(authService service.AuthService, secureCookie bool) *AuthHandlers {
	return &AuthHandlers{
		authService:  authService,
		secureCookie: secureCookie,
	}
}

// LoginRequest はログインリクエストのボディです
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// HandleLogin はログイン処理を行います
func (h *AuthHandlers) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	// バリデーション
	if req.Username == "" || req.Password == "" {
		respondJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Username and password are required"})
		return
	}

	// 認証
	sessionID, err := h.authService.Login(req.Username, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			respondJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "Invalid username or password"})
			return
		}
		log.Printf("login error: %v", err)
		respondJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	// セッションからユーザー情報を取得
	user, err := h.authService.GetUserBySession(sessionID)
	if err != nil {
		log.Printf("get user by session error: %v", err)
		respondJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	if user == nil {
		respondJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Failed to get user information"})
		return
	}

	// Cookieにセッション IDを設定
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Path:     sessionCookiePath,
		HttpOnly: true,                 // JavaScriptからアクセス不可
		SameSite: http.SameSiteLaxMode, // CSRF対策
		MaxAge:   24 * 60 * 60,         // 24時間
		Secure:   h.secureCookie,       // HTTPS必須（本番環境ではtrue）
	})

	// CSRFトークンを生成してCookieに設定
	csrfToken, err := generateCSRFToken()
	if err != nil {
		log.Printf("failed to generate CSRF token: %v", err)
		respondJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    csrfToken,
		Path:     sessionCookiePath,
		HttpOnly: false,                // JavaScriptからアクセス可能（Double Submit Cookie方式のため）
		SameSite: http.SameSiteLaxMode, // CSRF対策
		MaxAge:   24 * 60 * 60,         // 24時間
		Secure:   h.secureCookie,       // HTTPS必須（本番環境ではtrue）
	})

	// ユーザー情報を返す（PasswordHashは json:"-" で除外される）
	respondJSON(w, http.StatusOK, user)
}

// HandleLogout はログアウト処理を行います
func (h *AuthHandlers) HandleLogout(w http.ResponseWriter, r *http.Request) {
	// Cookieからセッション IDを取得
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		// セッションCookieがない場合は何もしない
		respondJSON(w, http.StatusOK, map[string]string{"message": "Logout successful"})
		return
	}

	// セッションを削除
	if err := h.authService.Logout(cookie.Value); err != nil {
		log.Printf("logout error: %v", err)
	}

	// セッションCookieを削除（設定時と同じ属性を指定する必要がある）
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     sessionCookiePath,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,             // 即座に削除
		Secure:   h.secureCookie, // 設定時と同じ
	})

	// CSRFトークンCookieも削除
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    "",
		Path:     sessionCookiePath,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,             // 即座に削除
		Secure:   h.secureCookie, // 設定時と同じ
	})

	respondJSON(w, http.StatusOK, map[string]string{"message": "Logout successful"})
}

// HandleMe は現在ログインしているユーザー情報を返します
func (h *AuthHandlers) HandleMe(w http.ResponseWriter, r *http.Request) {
	// Cookieからセッション IDを取得
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		respondJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "Not authenticated"})
		return
	}

	// セッションからユーザー情報を取得
	user, err := h.authService.GetUserBySession(cookie.Value)
	if err != nil {
		log.Printf("get user by session error: %v", err)
		respondJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	if user == nil {
		respondJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "Session expired or invalid"})
		return
	}

	// ユーザー情報を返す（PasswordHashは json:"-" で除外される）
	respondJSON(w, http.StatusOK, user)
}

// respondJSON はJSONレスポンスを返すヘルパー関数です
func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("failed to encode JSON response: %v", err)
	}
}
