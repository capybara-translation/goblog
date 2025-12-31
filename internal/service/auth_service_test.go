package service

import (
	"errors"
	"testing"
	"time"

	"github.com/capybara-translation/goblog/internal/auth"
	"github.com/capybara-translation/goblog/internal/config"
	"github.com/capybara-translation/goblog/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

// mockUserRepository はUserRepositoryのモック実装です
type mockUserRepository struct {
	findByUsernameFunc func(username string) (*domain.User, error)
	findByIDFunc       func(id int64) (*domain.User, error)
	createFunc         func(user *domain.User) error
	updateFunc         func(user *domain.User) error
	deleteFunc         func(id int64) error
}

func (m *mockUserRepository) FindByUsername(username string) (*domain.User, error) {
	if m.findByUsernameFunc != nil {
		return m.findByUsernameFunc(username)
	}
	return nil, nil
}

func (m *mockUserRepository) FindByID(id int64) (*domain.User, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(id)
	}
	return nil, nil
}

func (m *mockUserRepository) Create(user *domain.User) error {
	if m.createFunc != nil {
		return m.createFunc(user)
	}
	return nil
}

func (m *mockUserRepository) Update(user *domain.User) error {
	if m.updateFunc != nil {
		return m.updateFunc(user)
	}
	return nil
}

func (m *mockUserRepository) Delete(id int64) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(id)
	}
	return nil
}

// mockSessionStore はSessionStoreのモック実装です
type mockSessionStore struct {
	createFunc         func(userID int64, ttl time.Duration) (string, error)
	getFunc            func(sessionID string) (*auth.Session, error)
	deleteFunc         func(sessionID string) error
	cleanupExpiredFunc func()
}

func (m *mockSessionStore) Create(userID int64, ttl time.Duration) (string, error) {
	if m.createFunc != nil {
		return m.createFunc(userID, ttl)
	}
	return "mock-session-id", nil
}

func (m *mockSessionStore) Get(sessionID string) (*auth.Session, error) {
	if m.getFunc != nil {
		return m.getFunc(sessionID)
	}
	return nil, nil
}

func (m *mockSessionStore) Delete(sessionID string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(sessionID)
	}
	return nil
}

func (m *mockSessionStore) CleanupExpired() {
	if m.cleanupExpiredFunc != nil {
		m.cleanupExpiredFunc()
	}
}

func TestAuthService_Login(t *testing.T) {
	// テスト用のパスワードハッシュを生成
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	mockUserRepo := &mockUserRepository{
		findByUsernameFunc: func(username string) (*domain.User, error) {
			if username == "testuser" {
				return &domain.User{
					ID:           1,
					Username:     "testuser",
					PasswordHash: string(hashedPassword),
				}, nil
			}
			return nil, nil
		},
	}

	mockSessionStore := &mockSessionStore{
		createFunc: func(userID int64, ttl time.Duration) (string, error) {
			if userID != 1 {
				t.Errorf("expected userID 1, got %d", userID)
			}
			if ttl != 24*time.Hour {
				t.Errorf("expected TTL 24h, got %v", ttl)
			}
			return "test-session-id", nil
		},
	}

	authService := NewAuthService(mockUserRepo, mockSessionStore, config.PasswordPolicyNone)

	// 正しいパスワードでログイン
	sessionID, err := authService.Login("testuser", "password123", "127.0.0.1")
	if err != nil {
		t.Fatalf("failed to login: %v", err)
	}

	if sessionID != "test-session-id" {
		t.Errorf("expected session ID %q, got %q", "test-session-id", sessionID)
	}
}

func TestAuthService_Login_InvalidUsername(t *testing.T) {
	mockUserRepo := &mockUserRepository{
		findByUsernameFunc: func(username string) (*domain.User, error) {
			// ユーザーが見つからない
			return nil, nil
		},
	}

	mockSessionStore := &mockSessionStore{
		createFunc: func(userID int64, ttl time.Duration) (string, error) {
			t.Error("Create should not be called for invalid username")
			return "", nil
		},
	}

	authService := NewAuthService(mockUserRepo, mockSessionStore, config.PasswordPolicyNone)

	// 存在しないユーザー名でログイン
	sessionID, err := authService.Login("nonexistent", "password123", "127.0.0.1")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}

	if sessionID != "" {
		t.Error("expected session ID to be empty for invalid credentials")
	}
}

func TestAuthService_Login_InvalidPassword(t *testing.T) {
	// テスト用のパスワードハッシュを生成
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	mockUserRepo := &mockUserRepository{
		findByUsernameFunc: func(username string) (*domain.User, error) {
			return &domain.User{
				ID:           1,
				Username:     "testuser",
				PasswordHash: string(hashedPassword),
			}, nil
		},
	}

	mockSessionStore := &mockSessionStore{
		createFunc: func(userID int64, ttl time.Duration) (string, error) {
			t.Error("Create should not be called for invalid password")
			return "", nil
		},
	}

	authService := NewAuthService(mockUserRepo, mockSessionStore, config.PasswordPolicyNone)

	// 間違ったパスワードでログイン
	sessionID, err := authService.Login("testuser", "wrongpassword", "127.0.0.1")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}

	if sessionID != "" {
		t.Error("expected session ID to be empty for invalid password")
	}
}

func TestAuthService_Logout(t *testing.T) {
	var deletedSessionID string

	mockUserRepo := &mockUserRepository{}
	mockSessionStore := &mockSessionStore{
		deleteFunc: func(sessionID string) error {
			deletedSessionID = sessionID
			return nil
		},
	}

	authService := NewAuthService(mockUserRepo, mockSessionStore, config.PasswordPolicyNone)

	err := authService.Logout("test-session-id")
	if err != nil {
		t.Fatalf("failed to logout: %v", err)
	}

	if deletedSessionID != "test-session-id" {
		t.Errorf("expected to delete session %q, got %q", "test-session-id", deletedSessionID)
	}
}

func TestAuthService_GetUserBySession_Success(t *testing.T) {
	mockUserRepo := &mockUserRepository{
		findByIDFunc: func(id int64) (*domain.User, error) {
			if id == 123 {
				return &domain.User{
					ID:       123,
					Username: "testuser",
				}, nil
			}
			return nil, nil
		},
	}

	mockSessionStore := &mockSessionStore{
		getFunc: func(sessionID string) (*auth.Session, error) {
			if sessionID == "valid-session" {
				return &auth.Session{
					UserID:    123,
					ExpiresAt: time.Now().Add(1 * time.Hour),
				}, nil
			}
			return nil, nil
		},
	}

	authService := NewAuthService(mockUserRepo, mockSessionStore, config.PasswordPolicyNone)

	user, err := authService.GetUserBySession("valid-session")
	if err != nil {
		t.Fatalf("failed to get user by session: %v", err)
	}

	if user == nil {
		t.Fatal("expected user to exist")
	}

	if user.ID != 123 {
		t.Errorf("expected user ID 123, got %d", user.ID)
	}

	if user.Username != "testuser" {
		t.Errorf("expected username %q, got %q", "testuser", user.Username)
	}
}

func TestAuthService_GetUserBySession_NotFound(t *testing.T) {
	mockUserRepo := &mockUserRepository{}
	mockSessionStore := &mockSessionStore{
		getFunc: func(sessionID string) (*auth.Session, error) {
			// セッションが見つからない
			return nil, nil
		},
	}

	authService := NewAuthService(mockUserRepo, mockSessionStore, config.PasswordPolicyNone)

	user, err := authService.GetUserBySession("nonexistent-session-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user != nil {
		t.Error("expected user to be nil for nonexistent session")
	}
}

func TestAuthService_GetUserBySession_UserNotFoundInDB(t *testing.T) {
	mockUserRepo := &mockUserRepository{
		findByIDFunc: func(id int64) (*domain.User, error) {
			// ユーザーが見つからない（セッションは有効だがユーザーが削除された場合）
			return nil, nil
		},
	}

	mockSessionStore := &mockSessionStore{
		getFunc: func(sessionID string) (*auth.Session, error) {
			return &auth.Session{
				UserID:    999,
				ExpiresAt: time.Now().Add(1 * time.Hour),
			}, nil
		},
	}

	authService := NewAuthService(mockUserRepo, mockSessionStore, config.PasswordPolicyNone)

	user, err := authService.GetUserBySession("valid-session")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user != nil {
		t.Error("expected user to be nil when user not found in database")
	}
}

func TestAuthService_CreateUser(t *testing.T) {
	var createdUser *domain.User

	mockUserRepo := &mockUserRepository{
		findByUsernameFunc: func(username string) (*domain.User, error) {
			// ユーザー名が使用されていない
			return nil, nil
		},
		createFunc: func(user *domain.User) error {
			// IDを設定してCreateをシミュレート
			user.ID = 1
			createdUser = user
			return nil
		},
	}

	mockSessionStore := &mockSessionStore{}

	authService := NewAuthService(mockUserRepo, mockSessionStore, config.PasswordPolicyNone)

	user, err := authService.CreateUser("newuser", "password123")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	if user == nil {
		t.Fatal("expected user to be non-nil")
	}

	if user.ID != 1 {
		t.Errorf("expected user ID 1, got %d", user.ID)
	}

	if user.Username != "newuser" {
		t.Errorf("expected username %q, got %q", "newuser", user.Username)
	}

	// パスワードハッシュが設定されていることを確認
	if user.PasswordHash == "" {
		t.Error("expected password hash to be set")
	}

	// パスワードハッシュが元のパスワードと異なることを確認
	if user.PasswordHash == "password123" {
		t.Error("expected password to be hashed, not stored in plain text")
	}

	// bcryptでハッシュ化されていることを確認
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("password123"))
	if err != nil {
		t.Errorf("password hash verification failed: %v", err)
	}

	// Repositoryに渡されたユーザーを確認
	if createdUser == nil {
		t.Fatal("expected create to be called")
	}

	if createdUser.Username != "newuser" {
		t.Errorf("expected created user username %q, got %q", "newuser", createdUser.Username)
	}
}

func TestAuthService_CreateUser_DuplicateUsername(t *testing.T) {
	mockUserRepo := &mockUserRepository{
		findByUsernameFunc: func(username string) (*domain.User, error) {
			// ユーザー名が既に存在
			return &domain.User{
				ID:       99,
				Username: username,
			}, nil
		},
		createFunc: func(user *domain.User) error {
			t.Error("Create should not be called for duplicate username")
			return nil
		},
	}

	mockSessionStore := &mockSessionStore{}

	authService := NewAuthService(mockUserRepo, mockSessionStore, config.PasswordPolicyNone)

	_, err := authService.CreateUser("duplicate", "password123")
	if !errors.Is(err, ErrUsernameAlreadyExists) {
		t.Errorf("expected ErrUsernameAlreadyExists, got %v", err)
	}
}

func TestAuthService_CreateUser_PasswordHashing(t *testing.T) {
	var user1Hash, user2Hash string

	mockUserRepo := &mockUserRepository{
		findByUsernameFunc: func(username string) (*domain.User, error) {
			return nil, nil
		},
		createFunc: func(user *domain.User) error {
			user.ID = 1
			if user.Username == "user1" {
				user1Hash = user.PasswordHash
			} else if user.Username == "user2" {
				user2Hash = user.PasswordHash
			}
			return nil
		},
	}

	mockSessionStore := &mockSessionStore{}

	authService := NewAuthService(mockUserRepo, mockSessionStore, config.PasswordPolicyNone)

	// 同じパスワードで2人のユーザーを作成
	_, err := authService.CreateUser("user1", "samepassword")
	if err != nil {
		t.Fatalf("failed to create user1: %v", err)
	}

	_, err = authService.CreateUser("user2", "samepassword")
	if err != nil {
		t.Fatalf("failed to create user2: %v", err)
	}

	// 同じパスワードでも異なるハッシュが生成されることを確認（bcryptのsalt）
	if user1Hash == user2Hash {
		t.Error("expected different password hashes for same password (bcrypt should use different salts)")
	}

	// 両方のハッシュが正しいパスワードを検証できることを確認
	err = bcrypt.CompareHashAndPassword([]byte(user1Hash), []byte("samepassword"))
	if err != nil {
		t.Error("user1 password hash verification failed")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user2Hash), []byte("samepassword"))
	if err != nil {
		t.Error("user2 password hash verification failed")
	}
}

func TestAuthService_PasswordPolicy_None(t *testing.T) {
	mockUserRepo := &mockUserRepository{
		findByUsernameFunc: func(username string) (*domain.User, error) {
			return nil, nil
		},
		createFunc: func(user *domain.User) error {
			user.ID = 1
			return nil
		},
	}

	mockSessionStore := &mockSessionStore{}

	authService := NewAuthService(mockUserRepo, mockSessionStore, config.PasswordPolicyNone)

	// NONEポリシーでは短いパスワードでもOK
	tests := []struct {
		name     string
		password string
	}{
		{"短いパスワード", "pass"},
		{"数字のみ", "12345"},
		{"記号なし", "password"},
		{"小文字のみ", "abcdefgh"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := authService.CreateUser("testuser"+tt.name, tt.password)
			if err != nil {
				t.Errorf("expected no error for password %q with NONE policy, got %v", tt.password, err)
			}
		})
	}
}

func TestAuthService_PasswordPolicy_Strong_Valid(t *testing.T) {
	mockUserRepo := &mockUserRepository{
		findByUsernameFunc: func(username string) (*domain.User, error) {
			return nil, nil
		},
		createFunc: func(user *domain.User) error {
			user.ID = 1
			return nil
		},
	}

	mockSessionStore := &mockSessionStore{}

	authService := NewAuthService(mockUserRepo, mockSessionStore, config.PasswordPolicyStrong)

	// STRONGポリシーの要件を満たすパスワード
	validPasswords := []string{
		"MyStr0ng#Passw0rd",  // 17文字、すべての要件を満たす
		"Abcd1234!@#$567",    // 16文字、すべての要件を満たす
		"VerySecure1!Pass",   // 16文字、すべての要件を満たす
		"Test1234@Password!", // 18文字、すべての要件を満たす
		"Str0ng&SecurePass",  // 17文字、すべての要件を満たす
	}

	for _, password := range validPasswords {
		t.Run(password, func(t *testing.T) {
			_, err := authService.CreateUser("testuser"+password, password)
			if err != nil {
				t.Errorf("expected no error for valid strong password %q, got %v", password, err)
			}
		})
	}
}

func TestAuthService_PasswordPolicy_Strong_Invalid(t *testing.T) {
	mockUserRepo := &mockUserRepository{
		findByUsernameFunc: func(username string) (*domain.User, error) {
			return nil, nil
		},
		createFunc: func(user *domain.User) error {
			t.Error("Create should not be called for invalid password")
			return nil
		},
	}

	mockSessionStore := &mockSessionStore{}

	authService := NewAuthService(mockUserRepo, mockSessionStore, config.PasswordPolicyStrong)

	tests := []struct {
		name     string
		password string
	}{
		{"短すぎる", "Pass1!"},        // 6文字
		{"大文字なし", "password123!"}, // 大文字なし
		{"小文字なし", "PASSWORD123!"}, // 小文字なし
		{"数字なし", "Password!@#$"},  // 数字なし
		{"記号なし", "Password1234"},  // 記号なし
		{"14文字", "Passw0rd!1234"}, // 14文字（15文字未満）
		{"12文字", "Passw0rd!12"},   // 12文字（15文字未満）
		{"複数要件不足", "password"},    // 大文字、数字、記号なし
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := authService.CreateUser("testuser", tt.password)
			if err == nil {
				t.Errorf("expected error for invalid strong password %q, got nil", tt.password)
			}
			if !errors.Is(err, ErrWeakPassword) {
				t.Errorf("expected ErrWeakPassword, got %v", err)
			}
		})
	}
}

func TestAuthService_BruteForce_MultipleFailures(t *testing.T) {
	// テスト用のパスワードハッシュを生成
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	mockUserRepo := &mockUserRepository{
		findByUsernameFunc: func(username string) (*domain.User, error) {
			if username == "testuser" {
				return &domain.User{
					ID:           1,
					Username:     "testuser",
					PasswordHash: string(hashedPassword),
				}, nil
			}
			return nil, nil
		},
	}

	mockSessionStore := &mockSessionStore{
		createFunc: func(userID int64, ttl time.Duration) (string, error) {
			return "test-session-id", nil
		},
	}

	authService := NewAuthService(mockUserRepo, mockSessionStore, config.PasswordPolicyNone)

	ipAddress := "192.168.1.100"

	// 最初の3回の失敗は遅延なし
	start := time.Now()
	for i := 0; i < 3; i++ {
		_, err := authService.Login("testuser", "wrongpassword", ipAddress)
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Errorf("attempt %d: expected ErrInvalidCredentials, got %v", i+1, err)
		}
	}
	elapsed := time.Since(start)

	// 最初の3回は遅延がないので、1秒未満で完了するはず
	if elapsed > 1*time.Second {
		t.Errorf("expected first 3 failures to complete quickly, took %v", elapsed)
	}

	// 4回目の失敗から遅延が発生する（2秒）
	start = time.Now()
	_, err = authService.Login("testuser", "wrongpassword", ipAddress)
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
	elapsed = time.Since(start)

	// 4回目は2秒の遅延があるはず
	if elapsed < 2*time.Second {
		t.Errorf("expected 4th failure to have 2s delay, took only %v", elapsed)
	}
	if elapsed > 3*time.Second {
		t.Errorf("expected 4th failure to complete in ~2s, took %v", elapsed)
	}
}

func TestAuthService_BruteForce_SuccessResetsCounter(t *testing.T) {
	// テスト用のパスワードハッシュを生成
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	mockUserRepo := &mockUserRepository{
		findByUsernameFunc: func(username string) (*domain.User, error) {
			if username == "testuser" {
				return &domain.User{
					ID:           1,
					Username:     "testuser",
					PasswordHash: string(hashedPassword),
				}, nil
			}
			return nil, nil
		},
	}

	mockSessionStore := &mockSessionStore{
		createFunc: func(userID int64, ttl time.Duration) (string, error) {
			return "test-session-id", nil
		},
	}

	authService := NewAuthService(mockUserRepo, mockSessionStore, config.PasswordPolicyNone)

	ipAddress := "192.168.1.101"

	// 2回失敗
	for i := 0; i < 2; i++ {
		_, err := authService.Login("testuser", "wrongpassword", ipAddress)
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Errorf("attempt %d: expected ErrInvalidCredentials, got %v", i+1, err)
		}
	}

	// 成功
	_, err = authService.Login("testuser", "password123", ipAddress)
	if err != nil {
		t.Fatalf("expected successful login, got %v", err)
	}

	// カウンターがリセットされているはず
	// 再度失敗してもすぐに完了するはず（遅延なし）
	start := time.Now()
	_, err = authService.Login("testuser", "wrongpassword", ipAddress)
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
	elapsed := time.Since(start)

	// 遅延がないので1秒未満で完了するはず
	if elapsed > 1*time.Second {
		t.Errorf("expected failure after success to complete quickly (counter reset), took %v", elapsed)
	}
}

func TestAuthService_BruteForce_DifferentIPsIndependent(t *testing.T) {
	// テスト用のパスワードハッシュを生成
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	mockUserRepo := &mockUserRepository{
		findByUsernameFunc: func(username string) (*domain.User, error) {
			if username == "testuser" {
				return &domain.User{
					ID:           1,
					Username:     "testuser",
					PasswordHash: string(hashedPassword),
				}, nil
			}
			return nil, nil
		},
	}

	mockSessionStore := &mockSessionStore{
		createFunc: func(userID int64, ttl time.Duration) (string, error) {
			return "test-session-id", nil
		},
	}

	authService := NewAuthService(mockUserRepo, mockSessionStore, config.PasswordPolicyNone)

	// IP1から3回失敗（遅延が発生する状態にする）
	ip1 := "192.168.1.100"
	for i := 0; i < 3; i++ {
		_, err := authService.Login("testuser", "wrongpassword", ip1)
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Errorf("IP1 attempt %d: expected ErrInvalidCredentials, got %v", i+1, err)
		}
	}

	// IP2からの最初の失敗は遅延なし（独立したカウンター）
	ip2 := "192.168.1.200"
	start := time.Now()
	_, err = authService.Login("testuser", "wrongpassword", ip2)
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
	elapsed := time.Since(start)

	// IP2は新しいカウンターなので遅延なし
	if elapsed > 1*time.Second {
		t.Errorf("expected IP2 first failure to complete quickly (independent counter), took %v", elapsed)
	}
}

func TestAuthService_BruteForce_EmptyIPAddress(t *testing.T) {
	// テスト用のパスワードハッシュを生成
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	mockUserRepo := &mockUserRepository{
		findByUsernameFunc: func(username string) (*domain.User, error) {
			if username == "testuser" {
				return &domain.User{
					ID:           1,
					Username:     "testuser",
					PasswordHash: string(hashedPassword),
				}, nil
			}
			return nil, nil
		},
	}

	mockSessionStore := &mockSessionStore{
		createFunc: func(userID int64, ttl time.Duration) (string, error) {
			return "test-session-id", nil
		},
	}

	authService := NewAuthService(mockUserRepo, mockSessionStore, config.PasswordPolicyNone)

	// IPアドレスが空の場合、ブルートフォース対策は無効
	// 何回失敗しても遅延なし
	start := time.Now()
	for i := 0; i < 5; i++ {
		_, err := authService.Login("testuser", "wrongpassword", "")
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Errorf("attempt %d: expected ErrInvalidCredentials, got %v", i+1, err)
		}
	}
	elapsed := time.Since(start)

	// 遅延がないので1秒未満で完了するはず
	if elapsed > 1*time.Second {
		t.Errorf("expected failures with empty IP to complete quickly (no brute force protection), took %v", elapsed)
	}
}
