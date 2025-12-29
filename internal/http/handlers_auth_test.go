package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/capybara-translation/goblog/internal/service"
)

// mockAuthService は AuthService のモック実装です
type mockAuthService struct {
	loginFunc            func(username, password string) (string, error)
	logoutFunc           func(sessionID string) error
	getUserBySessionFunc func(sessionID string) (*domain.User, error)
	createUserFunc       func(username, password string) (*domain.User, error)
}

func (m *mockAuthService) Login(username, password string) (string, error) {
	if m.loginFunc != nil {
		return m.loginFunc(username, password)
	}
	return "", nil
}

func (m *mockAuthService) Logout(sessionID string) error {
	if m.logoutFunc != nil {
		return m.logoutFunc(sessionID)
	}
	return nil
}

func (m *mockAuthService) GetUserBySession(sessionID string) (*domain.User, error) {
	if m.getUserBySessionFunc != nil {
		return m.getUserBySessionFunc(sessionID)
	}
	return nil, nil
}

func (m *mockAuthService) CreateUser(username, password string) (*domain.User, error) {
	if m.createUserFunc != nil {
		return m.createUserFunc(username, password)
	}
	return nil, nil
}

var _ service.AuthService = (*mockAuthService)(nil)

func TestHandleLogin_Success(t *testing.T) {
	now := time.Now()
	mockService := &mockAuthService{
		loginFunc: func(username, password string) (string, error) {
			if username == "testuser" && password == "password123" {
				return "test-session-id", nil
			}
			return "", service.ErrInvalidCredentials
		},
		getUserBySessionFunc: func(sessionID string) (*domain.User, error) {
			if sessionID == "test-session-id" {
				return &domain.User{
					ID:        1,
					Username:  "testuser",
					CreatedAt: now,
					UpdatedAt: now,
				}, nil
			}
			return nil, nil
		},
	}

	handlers := NewAuthHandlers(mockService, false)

	// ログインリクエストを作成
	reqBody := LoginRequest{
		Username: "testuser",
		Password: "password123",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// ハンドラーを実行
	handlers.HandleLogin(rec, req)

	// ステータスコードを確認
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// レスポンスボディを確認（ユーザー情報が返される）
	var resp domain.User
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Username != "testuser" {
		t.Errorf("expected username %q, got %q", "testuser", resp.Username)
	}

	if resp.ID == 0 {
		t.Error("expected user ID to be set")
	}

	// Cookieが設定されていることを確認
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session cookie to be set")
	}

	var sessionCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == sessionCookieName {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("expected session_id cookie to be set")
	}

	if sessionCookie.Value != "test-session-id" {
		t.Errorf("expected session cookie value %q, got %q", "test-session-id", sessionCookie.Value)
	}

	if !sessionCookie.HttpOnly {
		t.Error("expected session cookie to be HttpOnly")
	}

	if sessionCookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("expected SameSite=Lax, got %v", sessionCookie.SameSite)
	}
}

func TestHandleLogin_InvalidUsername(t *testing.T) {
	mockService := &mockAuthService{
		loginFunc: func(username, password string) (string, error) {
			return "", service.ErrInvalidCredentials
		},
	}

	handlers := NewAuthHandlers(mockService, false)

	// 存在しないユーザー名でログイン
	reqBody := LoginRequest{
		Username: "nonexistent",
		Password: "password123",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handlers.HandleLogin(rec, req)

	// ステータスコードを確認
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	// エラーメッセージを確認
	var resp ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error != "Invalid username or password" {
		t.Errorf("expected error %q, got %q", "Invalid username or password", resp.Error)
	}
}

func TestHandleLogin_InvalidPassword(t *testing.T) {
	mockService := &mockAuthService{
		loginFunc: func(username, password string) (string, error) {
			return "", service.ErrInvalidCredentials
		},
	}

	handlers := NewAuthHandlers(mockService, false)

	// 間違ったパスワードでログイン
	reqBody := LoginRequest{
		Username: "testuser",
		Password: "wrongpassword",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handlers.HandleLogin(rec, req)

	// ステータスコードを確認
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestHandleLogin_MissingCredentials(t *testing.T) {
	mockService := &mockAuthService{
		loginFunc: func(username, password string) (string, error) {
			t.Error("Login should not be called for missing credentials")
			return "", nil
		},
	}

	handlers := NewAuthHandlers(mockService, false)

	tests := []struct {
		name     string
		username string
		password string
	}{
		{"empty username", "", "password123"},
		{"empty password", "testuser", ""},
		{"both empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := LoginRequest{
				Username: tt.username,
				Password: tt.password,
			}
			body, _ := json.Marshal(reqBody)

			req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handlers.HandleLogin(rec, req)

			// ステータスコードを確認
			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}

			// エラーメッセージを確認
			var resp ErrorResponse
			if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if resp.Error != "Username and password are required" {
				t.Errorf("expected error %q, got %q", "Username and password are required", resp.Error)
			}
		})
	}
}

func TestHandleLogin_InvalidJSON(t *testing.T) {
	mockService := &mockAuthService{
		loginFunc: func(username, password string) (string, error) {
			t.Error("Login should not be called for invalid JSON")
			return "", nil
		},
	}

	handlers := NewAuthHandlers(mockService, false)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handlers.HandleLogin(rec, req)

	// ステータスコードを確認
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleLogout_Success(t *testing.T) {
	var loggedOutSessionID string

	mockService := &mockAuthService{
		logoutFunc: func(sessionID string) error {
			loggedOutSessionID = sessionID
			return nil
		},
	}

	handlers := NewAuthHandlers(mockService, false)

	// ログアウトリクエストを作成
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  sessionCookieName,
		Value: "test-session-id",
	})
	rec := httptest.NewRecorder()

	// ハンドラーを実行
	handlers.HandleLogout(rec, req)

	// ステータスコードを確認
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// レスポンスボディを確認
	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["message"] != "Logout successful" {
		t.Errorf("expected message %q, got %q", "Logout successful", resp["message"])
	}

	// Logout が正しいセッションIDで呼ばれたことを確認
	if loggedOutSessionID != "test-session-id" {
		t.Errorf("expected logout session ID %q, got %q", "test-session-id", loggedOutSessionID)
	}

	// Cookieが削除されていることを確認（MaxAge = -1）
	cookies := rec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == sessionCookieName {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("expected session cookie to be set for deletion")
	}

	if sessionCookie.MaxAge != -1 {
		t.Errorf("expected MaxAge = -1 for cookie deletion, got %d", sessionCookie.MaxAge)
	}

	// Cookie属性が設定時と一致することを確認
	if !sessionCookie.HttpOnly {
		t.Error("expected session cookie to be HttpOnly")
	}

	if sessionCookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("expected SameSite=Lax, got %v", sessionCookie.SameSite)
	}

	if sessionCookie.Secure {
		t.Error("expected Secure to be false (secureCookie=false)")
	}
}

func TestHandleLogout_NoCookie(t *testing.T) {
	mockService := &mockAuthService{
		logoutFunc: func(sessionID string) error {
			// Cookieがない場合、Logoutは呼ばれないはず
			t.Error("Logout should not be called when no cookie is present")
			return nil
		},
	}

	handlers := NewAuthHandlers(mockService, false)

	// Cookieなしでログアウト
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	rec := httptest.NewRecorder()

	handlers.HandleLogout(rec, req)

	// ステータスコードを確認（エラーにならない）
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// レスポンスを確認
	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["message"] != "Logout successful" {
		t.Errorf("expected message %q, got %q", "Logout successful", resp["message"])
	}
}

func TestHandleLogout_WithSecureCookie(t *testing.T) {
	var loggedOutSessionID string

	mockService := &mockAuthService{
		logoutFunc: func(sessionID string) error {
			loggedOutSessionID = sessionID
			return nil
		},
	}

	// 本番環境を想定（secureCookie=true）
	handlers := NewAuthHandlers(mockService, true)

	// ログアウトリクエストを作成
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  sessionCookieName,
		Value: "test-session-id",
	})
	rec := httptest.NewRecorder()

	// ハンドラーを実行
	handlers.HandleLogout(rec, req)

	// ステータスコードを確認
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Logout が正しいセッションIDで呼ばれたことを確認
	if loggedOutSessionID != "test-session-id" {
		t.Errorf("expected logout session ID %q, got %q", "test-session-id", loggedOutSessionID)
	}

	// 削除用Cookieを取得
	cookies := rec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == sessionCookieName {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("expected session cookie to be set for deletion")
	}

	// Cookie属性が本番環境の設定と一致することを確認
	if sessionCookie.MaxAge != -1 {
		t.Errorf("expected MaxAge = -1 for cookie deletion, got %d", sessionCookie.MaxAge)
	}

	if !sessionCookie.HttpOnly {
		t.Error("expected session cookie to be HttpOnly")
	}

	if sessionCookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("expected SameSite=Lax, got %v", sessionCookie.SameSite)
	}

	// 本番環境ではSecure=trueであることを確認
	if !sessionCookie.Secure {
		t.Error("expected Secure to be true (secureCookie=true)")
	}
}

func TestHandleMe_Success(t *testing.T) {
	now := time.Now()
	mockService := &mockAuthService{
		getUserBySessionFunc: func(sessionID string) (*domain.User, error) {
			if sessionID == "valid-session" {
				return &domain.User{
					ID:        1,
					Username:  "testuser",
					CreatedAt: now,
					UpdatedAt: now,
				}, nil
			}
			return nil, nil
		},
	}

	handlers := NewAuthHandlers(mockService, false)

	// /me リクエストを作成
	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.AddCookie(&http.Cookie{
		Name:  sessionCookieName,
		Value: "valid-session",
	})
	rec := httptest.NewRecorder()

	// ハンドラーを実行
	handlers.HandleMe(rec, req)

	// ステータスコードを確認
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// レスポンスボディを確認（ユーザー情報が返される）
	var resp domain.User
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.ID != 1 {
		t.Errorf("expected user ID 1, got %d", resp.ID)
	}

	if resp.Username != "testuser" {
		t.Errorf("expected username %q, got %q", "testuser", resp.Username)
	}

	// CreatedAt, UpdatedAt も返されることを確認
	if resp.CreatedAt.IsZero() {
		t.Error("expected created_at to be set")
	}

	if resp.UpdatedAt.IsZero() {
		t.Error("expected updated_at to be set")
	}

	// PasswordHash は JSON に含まれないことを確認（ゼロ値のまま）
	if resp.PasswordHash != "" {
		t.Error("expected password_hash to not be included in JSON response")
	}
}

func TestHandleMe_NotAuthenticated(t *testing.T) {
	mockService := &mockAuthService{
		getUserBySessionFunc: func(sessionID string) (*domain.User, error) {
			t.Error("GetUserBySession should not be called without cookie")
			return nil, nil
		},
	}

	handlers := NewAuthHandlers(mockService, false)

	// Cookieなしでリクエスト
	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	rec := httptest.NewRecorder()

	handlers.HandleMe(rec, req)

	// ステータスコードを確認
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	// エラーメッセージを確認
	var resp ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error != "Not authenticated" {
		t.Errorf("expected error %q, got %q", "Not authenticated", resp.Error)
	}
}

func TestHandleMe_InvalidSession(t *testing.T) {
	mockService := &mockAuthService{
		getUserBySessionFunc: func(sessionID string) (*domain.User, error) {
			// セッションが無効（ユーザーが見つからない）
			return nil, nil
		},
	}

	handlers := NewAuthHandlers(mockService, false)

	// 無効なセッションIDでリクエスト
	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.AddCookie(&http.Cookie{
		Name:  sessionCookieName,
		Value: "invalid-session-id",
	})
	rec := httptest.NewRecorder()

	handlers.HandleMe(rec, req)

	// ステータスコードを確認
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	// エラーメッセージを確認
	var resp ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error != "Session expired or invalid" {
		t.Errorf("expected error %q, got %q", "Session expired or invalid", resp.Error)
	}
}
