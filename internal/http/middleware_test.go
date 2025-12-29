package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/capybara-translation/goblog/internal/domain"
)

func TestAuthMiddleware_Success(t *testing.T) {
	now := time.Now()
	mockService := &mockAuthService{
		getUserBySessionFunc: func(sessionID string) (*domain.User, error) {
			if sessionID == "valid-session-id" {
				return &domain.User{
					ID:        123,
					Username:  "testuser",
					CreatedAt: now,
					UpdatedAt: now,
				}, nil
			}
			return nil, nil
		},
	}

	// ミドルウェアを適用したハンドラーを作成
	var capturedUserID int64
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// コンテキストからユーザーIDを取得
		userID, ok := GetUserIDFromContext(r.Context())
		if !ok {
			t.Error("expected user ID in context")
			return
		}
		capturedUserID = userID
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware := AuthMiddleware(mockService)
	protectedHandler := middleware(handler)

	// 有効なセッションIDでリクエスト
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  sessionCookieName,
		Value: "valid-session-id",
	})
	rec := httptest.NewRecorder()

	// ハンドラーを実行
	protectedHandler.ServeHTTP(rec, req)

	// ステータスコードを確認
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// コンテキストに正しいユーザーIDが設定されていることを確認
	if capturedUserID != 123 {
		t.Errorf("expected user ID %d in context, got %d", 123, capturedUserID)
	}
}

func TestAuthMiddleware_NoCookie(t *testing.T) {
	mockService := &mockAuthService{
		getUserBySessionFunc: func(sessionID string) (*domain.User, error) {
			t.Error("GetUserBySession should not be called when no cookie is present")
			return nil, nil
		},
	}

	// ミドルウェアを適用したハンドラーを作成
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called when auth fails")
		w.WriteHeader(http.StatusOK)
	})

	middleware := AuthMiddleware(mockService)
	protectedHandler := middleware(handler)

	// Cookieなしでリクエスト
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()

	protectedHandler.ServeHTTP(rec, req)

	// ステータスコードを確認
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	// エラーメッセージを確認
	expectedError := "Authentication required"
	if !strings.Contains(rec.Body.String(), expectedError) {
		t.Errorf("expected error message to contain %q, got %q", expectedError, rec.Body.String())
	}
}

func TestAuthMiddleware_InvalidSession(t *testing.T) {
	mockService := &mockAuthService{
		getUserBySessionFunc: func(sessionID string) (*domain.User, error) {
			// セッションが無効（ユーザーが見つからない）
			return nil, nil
		},
	}

	// ミドルウェアを適用したハンドラーを作成
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called when auth fails")
		w.WriteHeader(http.StatusOK)
	})

	middleware := AuthMiddleware(mockService)
	protectedHandler := middleware(handler)

	// 無効なセッションIDでリクエスト
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  sessionCookieName,
		Value: "invalid-session-id",
	})
	rec := httptest.NewRecorder()

	protectedHandler.ServeHTTP(rec, req)

	// ステータスコードを確認
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	// エラーメッセージを確認
	expectedError := "Session expired or invalid"
	if !strings.Contains(rec.Body.String(), expectedError) {
		t.Errorf("expected error message to contain %q, got %q", expectedError, rec.Body.String())
	}
}

func TestAuthMiddleware_DatabaseError(t *testing.T) {
	mockService := &mockAuthService{
		getUserBySessionFunc: func(sessionID string) (*domain.User, error) {
			// データベースエラーをシミュレート
			return nil, errors.New("database connection error")
		},
	}

	// ミドルウェアを適用したハンドラーを作成
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called when database error occurs")
		w.WriteHeader(http.StatusOK)
	})

	middleware := AuthMiddleware(mockService)
	protectedHandler := middleware(handler)

	// リクエストを作成
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  sessionCookieName,
		Value: "valid-session-id",
	})
	rec := httptest.NewRecorder()

	protectedHandler.ServeHTTP(rec, req)

	// DBエラー時は500エラーを返す
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	// エラーメッセージを確認
	expectedError := "Internal server error"
	if !strings.Contains(rec.Body.String(), expectedError) {
		t.Errorf("expected error message to contain %q, got %q", expectedError, rec.Body.String())
	}
}

func TestGetUserIDFromContext_Success(t *testing.T) {
	now := time.Now()
	mockService := &mockAuthService{
		getUserBySessionFunc: func(sessionID string) (*domain.User, error) {
			if sessionID == "valid-session-id" {
				return &domain.User{
					ID:        456,
					Username:  "testuser",
					CreatedAt: now,
					UpdatedAt: now,
				}, nil
			}
			return nil, nil
		},
	}

	// ミドルウェア経由でリクエストを処理
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := GetUserIDFromContext(r.Context())
		if !ok {
			t.Error("expected user ID in context")
			return
		}

		if userID != 456 {
			t.Errorf("expected user ID %d, got %d", 456, userID)
		}
		w.WriteHeader(http.StatusOK)
	})

	middleware := AuthMiddleware(mockService)
	protectedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  sessionCookieName,
		Value: "valid-session-id",
	})
	rec := httptest.NewRecorder()

	protectedHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestGetUserIDFromContext_NotFound(t *testing.T) {
	// コンテキストにユーザーIDがない場合
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	userID, ok := GetUserIDFromContext(req.Context())
	if ok {
		t.Error("expected user ID not to be found in context")
	}

	if userID != 0 {
		t.Errorf("expected user ID to be 0 when not found, got %d", userID)
	}
}
