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

// mockAuthService is a mock implementation of AuthService
type mockAuthService struct {
	loginFunc                    func(username, password, ipAddress string) (string, error)
	logoutFunc                   func(sessionID string) error
	getUserBySessionFunc         func(sessionID string) (*domain.User, error)
	createUserFunc               func(username, password string) (*domain.User, error)
	issueRememberTokenFunc       func(int64) (string, error)
	restoreFromRememberTokenFunc func(string) (*domain.User, string, error)
	revokeRememberTokenFunc      func(string) error
	sessionTTL                   time.Duration
	rememberTTL                  time.Duration
}

func (m *mockAuthService) Login(username, password, ipAddress string) (string, error) {
	if m.loginFunc != nil {
		return m.loginFunc(username, password, ipAddress)
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

func (m *mockAuthService) SessionTTL() time.Duration {
	if m.sessionTTL > 0 {
		return m.sessionTTL
	}
	return 24 * time.Hour
}

func (m *mockAuthService) IssueRememberToken(uid int64) (string, error) {
	if m.issueRememberTokenFunc != nil {
		return m.issueRememberTokenFunc(uid)
	}
	return "", nil
}

func (m *mockAuthService) RestoreFromRememberToken(c string) (*domain.User, string, error) {
	if m.restoreFromRememberTokenFunc != nil {
		return m.restoreFromRememberTokenFunc(c)
	}
	return nil, "", nil
}

func (m *mockAuthService) RevokeRememberToken(c string) error {
	if m.revokeRememberTokenFunc != nil {
		return m.revokeRememberTokenFunc(c)
	}
	return nil
}

func (m *mockAuthService) RememberTTL() time.Duration {
	if m.rememberTTL > 0 {
		return m.rememberTTL
	}
	return 30 * 24 * time.Hour
}

var _ service.AuthService = (*mockAuthService)(nil)

func TestHandleLogin_Success(t *testing.T) {
	now := time.Now()
	mockService := &mockAuthService{
		loginFunc: func(username, password, ipAddress string) (string, error) {
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

	handlers := NewAuthHandlers(mockService, false, nil)

	// Create login request
	reqBody := LoginRequest{
		Username: "testuser",
		Password: "password123",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// Execute handler
	handlers.HandleLogin(rec, req)

	// Verify status code
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Verify response body (user information is returned)
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

	// Verify that cookies are set
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

func TestHandleLogin_CookieMaxAgeMatchesSessionTTL(t *testing.T) {
	tests := []struct {
		name       string
		ttl        time.Duration
		wantMaxAge int
	}{
		{"default 24h", 24 * time.Hour, 24 * 60 * 60},
		{"short 30 minutes", 30 * time.Minute, 30 * 60},
		{"long 7 days", 7 * 24 * time.Hour, 7 * 24 * 60 * 60},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockAuthService{
				loginFunc: func(username, password, ipAddress string) (string, error) {
					return "test-session-id", nil
				},
				getUserBySessionFunc: func(sessionID string) (*domain.User, error) {
					return &domain.User{ID: 1, Username: "u"}, nil
				},
				sessionTTL: tt.ttl,
			}
			handlers := NewAuthHandlers(mockService, false, nil)

			reqBody := LoginRequest{Username: "u", Password: "p"}
			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handlers.HandleLogin(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d", rec.Code)
			}

			cookies := rec.Result().Cookies()
			var sawSession, sawCSRF bool
			for _, c := range cookies {
				switch c.Name {
				case sessionCookieName:
					sawSession = true
				case csrfCookieName:
					sawCSRF = true
				default:
					continue
				}
				if c.MaxAge != tt.wantMaxAge {
					t.Errorf("%s cookie MaxAge = %d, want %d (SessionTTL=%v)",
						c.Name, c.MaxAge, tt.wantMaxAge, tt.ttl)
				}
			}
			if !sawSession || !sawCSRF {
				t.Fatalf("missing required cookies: session=%v csrf=%v", sawSession, sawCSRF)
			}
		})
	}
}

func TestHandleLogin_InvalidUsername(t *testing.T) {
	mockService := &mockAuthService{
		loginFunc: func(username, password, ipAddress string) (string, error) {
			return "", service.ErrInvalidCredentials
		},
	}

	handlers := NewAuthHandlers(mockService, false, nil)

	// Login with non-existent username
	reqBody := LoginRequest{
		Username: "nonexistent",
		Password: "password123",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handlers.HandleLogin(rec, req)

	// Verify status code
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	// Verify error message
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
		loginFunc: func(username, password, ipAddress string) (string, error) {
			return "", service.ErrInvalidCredentials
		},
	}

	handlers := NewAuthHandlers(mockService, false, nil)

	// Login with wrong password
	reqBody := LoginRequest{
		Username: "testuser",
		Password: "wrongpassword",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handlers.HandleLogin(rec, req)

	// Verify status code
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestHandleLogin_MissingCredentials(t *testing.T) {
	mockService := &mockAuthService{
		loginFunc: func(username, password, ipAddress string) (string, error) {
			t.Error("Login should not be called for missing credentials")
			return "", nil
		},
	}

	handlers := NewAuthHandlers(mockService, false, nil)

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

			// Verify status code
			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}

			// Verify error message
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
		loginFunc: func(username, password, ipAddress string) (string, error) {
			t.Error("Login should not be called for invalid JSON")
			return "", nil
		},
	}

	handlers := NewAuthHandlers(mockService, false, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handlers.HandleLogin(rec, req)

	// Verify status code
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

	handlers := NewAuthHandlers(mockService, false, nil)

	// Create logout request
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  sessionCookieName,
		Value: "test-session-id",
	})
	rec := httptest.NewRecorder()

	// Execute handler
	handlers.HandleLogout(rec, req)

	// Verify status code
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Verify response body
	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["message"] != "Logout successful" {
		t.Errorf("expected message %q, got %q", "Logout successful", resp["message"])
	}

	// Verify that Logout was called with the correct session ID
	if loggedOutSessionID != "test-session-id" {
		t.Errorf("expected logout session ID %q, got %q", "test-session-id", loggedOutSessionID)
	}

	// Verify that the cookie is deleted (MaxAge = -1)
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

	// Verify that cookie attributes match the settings
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
			// Logout should not be called when there is no cookie
			t.Error("Logout should not be called when no cookie is present")
			return nil
		},
	}

	handlers := NewAuthHandlers(mockService, false, nil)

	// Logout without cookie
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	rec := httptest.NewRecorder()

	handlers.HandleLogout(rec, req)

	// Verify status code (should not error)
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Verify response
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

	// Assuming production environment (secureCookie=true)
	handlers := NewAuthHandlers(mockService, true, nil)

	// Create logout request
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  sessionCookieName,
		Value: "test-session-id",
	})
	rec := httptest.NewRecorder()

	// Execute handler
	handlers.HandleLogout(rec, req)

	// Verify status code
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Verify that Logout was called with the correct session ID
	if loggedOutSessionID != "test-session-id" {
		t.Errorf("expected logout session ID %q, got %q", "test-session-id", loggedOutSessionID)
	}

	// Get the cookie for deletion
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

	// Verify that cookie attributes match production environment settings
	if sessionCookie.MaxAge != -1 {
		t.Errorf("expected MaxAge = -1 for cookie deletion, got %d", sessionCookie.MaxAge)
	}

	if !sessionCookie.HttpOnly {
		t.Error("expected session cookie to be HttpOnly")
	}

	if sessionCookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("expected SameSite=Lax, got %v", sessionCookie.SameSite)
	}

	// Verify that Secure=true in production environment
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

	handlers := NewAuthHandlers(mockService, false, nil)

	// Create /me request
	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.AddCookie(&http.Cookie{
		Name:  sessionCookieName,
		Value: "valid-session",
	})
	rec := httptest.NewRecorder()

	// Execute handler
	handlers.HandleMe(rec, req)

	// Verify status code
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Verify response body (user information is returned)
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

	// Verify that CreatedAt and UpdatedAt are also returned
	if resp.CreatedAt.IsZero() {
		t.Error("expected created_at to be set")
	}

	if resp.UpdatedAt.IsZero() {
		t.Error("expected updated_at to be set")
	}

	// Verify that PasswordHash is not included in JSON (remains zero value)
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

	handlers := NewAuthHandlers(mockService, false, nil)

	// Request without cookie
	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	rec := httptest.NewRecorder()

	handlers.HandleMe(rec, req)

	// Verify status code
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	// Verify error message
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
			// Session is invalid (user not found)
			return nil, nil
		},
	}

	handlers := NewAuthHandlers(mockService, false, nil)

	// Request with invalid session ID
	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.AddCookie(&http.Cookie{
		Name:  sessionCookieName,
		Value: "invalid-session-id",
	})
	rec := httptest.NewRecorder()

	handlers.HandleMe(rec, req)

	// Verify status code
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	// Verify error message
	var resp ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error != "Session expired or invalid" {
		t.Errorf("expected error %q, got %q", "Session expired or invalid", resp.Error)
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name           string
		trustedProxies []string
		remoteAddr     string
		xForwardedFor  string
		xRealIP        string
		expectedIP     string
	}{
		{
			name:           "no trusted proxies - ignores X-Forwarded-For",
			trustedProxies: nil,
			remoteAddr:     "192.168.1.100:12345",
			xForwardedFor:  "1.2.3.4",
			expectedIP:     "192.168.1.100",
		},
		{
			name:           "no trusted proxies - ignores X-Real-IP",
			trustedProxies: nil,
			remoteAddr:     "192.168.1.100:12345",
			xRealIP:        "5.6.7.8",
			expectedIP:     "192.168.1.100",
		},
		{
			name:           "no trusted proxies - returns RemoteAddr",
			trustedProxies: nil,
			remoteAddr:     "10.0.0.1:54321",
			expectedIP:     "10.0.0.1",
		},
		{
			name:           "trusted proxy - uses X-Forwarded-For",
			trustedProxies: []string{"127.0.0.1"},
			remoteAddr:     "127.0.0.1:12345",
			xForwardedFor:  "203.0.113.50",
			expectedIP:     "203.0.113.50",
		},
		{
			name:           "trusted proxy - uses X-Forwarded-For first IP",
			trustedProxies: []string{"127.0.0.1"},
			remoteAddr:     "127.0.0.1:12345",
			xForwardedFor:  "203.0.113.50, 10.0.0.1",
			expectedIP:     "203.0.113.50",
		},
		{
			name:           "trusted proxy - uses X-Real-IP when no X-Forwarded-For",
			trustedProxies: []string{"127.0.0.1"},
			remoteAddr:     "127.0.0.1:12345",
			xRealIP:        "203.0.113.50",
			expectedIP:     "203.0.113.50",
		},
		{
			name:           "trusted proxy - falls back to RemoteAddr when no headers",
			trustedProxies: []string{"127.0.0.1"},
			remoteAddr:     "127.0.0.1:12345",
			expectedIP:     "127.0.0.1",
		},
		{
			name:           "untrusted source - ignores X-Forwarded-For even with trusted proxies configured",
			trustedProxies: []string{"127.0.0.1"},
			remoteAddr:     "192.168.1.100:12345",
			xForwardedFor:  "1.2.3.4",
			expectedIP:     "192.168.1.100",
		},
		{
			name:           "CIDR range - trusted",
			trustedProxies: []string{"10.0.0.0/8"},
			remoteAddr:     "10.0.0.5:12345",
			xForwardedFor:  "203.0.113.50",
			expectedIP:     "203.0.113.50",
		},
		{
			name:           "CIDR range - not trusted",
			trustedProxies: []string{"10.0.0.0/8"},
			remoteAddr:     "192.168.1.1:12345",
			xForwardedFor:  "1.2.3.4",
			expectedIP:     "192.168.1.1",
		},
		{
			name:           "multiple trusted proxies",
			trustedProxies: []string{"127.0.0.1", "10.0.0.0/8"},
			remoteAddr:     "10.0.0.5:12345",
			xForwardedFor:  "203.0.113.50",
			expectedIP:     "203.0.113.50",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := NewAuthHandlers(nil, false, tt.trustedProxies)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}

			got := handlers.getClientIP(req)
			if got != tt.expectedIP {
				t.Errorf("getClientIP() = %q, want %q", got, tt.expectedIP)
			}
		})
	}
}
