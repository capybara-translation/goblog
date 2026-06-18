package service

import (
	"errors"
	"testing"
	"time"

	"github.com/capybara-translation/goblog/internal/auth"
	"github.com/capybara-translation/goblog/internal/config"
	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/capybara-translation/goblog/internal/repo"
	"golang.org/x/crypto/bcrypt"
)

// mockUserRepository is a mock implementation of UserRepository
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

// mockSessionStore is a mock implementation of SessionStore
type mockSessionStore struct {
	createFunc         func(userID int64, ttl time.Duration, meta auth.SessionMeta) (string, error)
	getFunc            func(sessionID string) (*auth.Session, error)
	deleteFunc         func(sessionID string) error
	cleanupExpiredFunc func()
}

var _ auth.SessionStore = (*mockSessionStore)(nil)

func (m *mockSessionStore) Create(userID int64, ttl time.Duration, meta auth.SessionMeta) (string, error) {
	if m.createFunc != nil {
		return m.createFunc(userID, ttl, meta)
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

func (m *mockSessionStore) Touch(sessionID string, t time.Time)            {}
func (m *mockSessionStore) SetRememberSelector(sessionID, selector string) {}
func (m *mockSessionStore) ListByUser(userID int64) []auth.SessionInfo     { return nil }
func (m *mockSessionStore) DeleteByHandleForUser(userID int64, handle string) (bool, error) {
	return false, nil
}
func (m *mockSessionStore) DeleteBySelector(selector string)                 {}
func (m *mockSessionStore) DeleteByUserExcept(userID int64, exceptID string) {}

// mockRememberTokenStore is a mock implementation of auth.RememberTokenStore.
type mockRememberTokenStore struct {
	createFunc                     func(*domain.RememberToken) error
	findBySelectorFunc             func(string) (*domain.RememberToken, error)
	deleteFunc                     func(string) error
	deleteByUserIDFunc             func(int64) error
	refreshOnUseFunc               func(string, time.Time, time.Time) error
	cleanupExpiredFunc             func() error
	findByUserIDFunc               func(int64) ([]*domain.RememberToken, error)
	deleteByUserExceptSelectorFunc func(int64, string) error
}

func (m *mockRememberTokenStore) Create(t *domain.RememberToken) error {
	if m.createFunc != nil {
		return m.createFunc(t)
	}
	return nil
}

func (m *mockRememberTokenStore) FindBySelector(sel string) (*domain.RememberToken, error) {
	if m.findBySelectorFunc != nil {
		return m.findBySelectorFunc(sel)
	}
	return nil, nil
}

func (m *mockRememberTokenStore) Delete(sel string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(sel)
	}
	return nil
}

func (m *mockRememberTokenStore) DeleteByUserID(uid int64) error {
	if m.deleteByUserIDFunc != nil {
		return m.deleteByUserIDFunc(uid)
	}
	return nil
}

func (m *mockRememberTokenStore) RefreshOnUse(sel string, lastUsed time.Time, newExpiresAt time.Time) error {
	if m.refreshOnUseFunc != nil {
		return m.refreshOnUseFunc(sel, lastUsed, newExpiresAt)
	}
	return nil
}

func (m *mockRememberTokenStore) CleanupExpired() error {
	if m.cleanupExpiredFunc != nil {
		return m.cleanupExpiredFunc()
	}
	return nil
}

func (m *mockRememberTokenStore) FindByUserID(uid int64) ([]*domain.RememberToken, error) {
	if m.findByUserIDFunc != nil {
		return m.findByUserIDFunc(uid)
	}
	return nil, nil
}

func (m *mockRememberTokenStore) DeleteByUserExceptSelector(uid int64, keepSelector string) error {
	if m.deleteByUserExceptSelectorFunc != nil {
		return m.deleteByUserExceptSelectorFunc(uid, keepSelector)
	}
	return nil
}

var _ auth.RememberTokenStore = (*mockRememberTokenStore)(nil)

// newAuthServiceForTest constructs an authService for tests with the given
// dependencies. Hides the ctor signature growth from individual tests.
func newAuthServiceForTest(userRepo repo.UserRepository, sessions auth.SessionStore, remember auth.RememberTokenStore, rememberTTL time.Duration) AuthService {
	return NewAuthService(userRepo, sessions, config.PasswordPolicyNone, 24*time.Hour, remember, rememberTTL)
}

func TestAuthService_Login(t *testing.T) {
	// Generate password hash for testing
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
		createFunc: func(userID int64, ttl time.Duration, meta auth.SessionMeta) (string, error) {
			if userID != 1 {
				t.Errorf("expected userID 1, got %d", userID)
			}
			if ttl != 24*time.Hour {
				t.Errorf("expected TTL 24h, got %v", ttl)
			}
			return "test-session-id", nil
		},
	}

	authService := NewAuthService(mockUserRepo, mockSessionStore, config.PasswordPolicyNone, 24*time.Hour, nil, 24*time.Hour)

	// Login with correct password
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
			// User not found
			return nil, nil
		},
	}

	mockSessionStore := &mockSessionStore{
		createFunc: func(userID int64, ttl time.Duration, meta auth.SessionMeta) (string, error) {
			t.Error("Create should not be called for invalid username")
			return "", nil
		},
	}

	authService := NewAuthService(mockUserRepo, mockSessionStore, config.PasswordPolicyNone, 24*time.Hour, nil, 24*time.Hour)

	// Login with non-existent username
	sessionID, err := authService.Login("nonexistent", "password123", "127.0.0.1")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}

	if sessionID != "" {
		t.Error("expected session ID to be empty for invalid credentials")
	}
}

func TestAuthService_Login_InvalidPassword(t *testing.T) {
	// Generate password hash for testing
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
		createFunc: func(userID int64, ttl time.Duration, meta auth.SessionMeta) (string, error) {
			t.Error("Create should not be called for invalid password")
			return "", nil
		},
	}

	authService := NewAuthService(mockUserRepo, mockSessionStore, config.PasswordPolicyNone, 24*time.Hour, nil, 24*time.Hour)

	// Login with wrong password
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

	authService := NewAuthService(mockUserRepo, mockSessionStore, config.PasswordPolicyNone, 24*time.Hour, nil, 24*time.Hour)

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

	authService := NewAuthService(mockUserRepo, mockSessionStore, config.PasswordPolicyNone, 24*time.Hour, nil, 24*time.Hour)

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
			// Session not found
			return nil, nil
		},
	}

	authService := NewAuthService(mockUserRepo, mockSessionStore, config.PasswordPolicyNone, 24*time.Hour, nil, 24*time.Hour)

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
			// User not found (session is valid but user was deleted)
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

	authService := NewAuthService(mockUserRepo, mockSessionStore, config.PasswordPolicyNone, 24*time.Hour, nil, 24*time.Hour)

	user, err := authService.GetUserBySession("valid-session")
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound but got: %v", err)
	}

	if user != nil {
		t.Error("expected user to be nil when user not found in database")
	}
}

func TestAuthService_CreateUser(t *testing.T) {
	var createdUser *domain.User

	mockUserRepo := &mockUserRepository{
		findByUsernameFunc: func(username string) (*domain.User, error) {
			// Username is not taken
			return nil, nil
		},
		createFunc: func(user *domain.User) error {
			// Simulate Create by setting ID
			user.ID = 1
			createdUser = user
			return nil
		},
	}

	mockSessionStore := &mockSessionStore{}

	authService := NewAuthService(mockUserRepo, mockSessionStore, config.PasswordPolicyNone, 24*time.Hour, nil, 24*time.Hour)

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

	// Verify password hash is set
	if user.PasswordHash == "" {
		t.Error("expected password hash to be set")
	}

	// Verify password hash is different from original password
	if user.PasswordHash == "password123" {
		t.Error("expected password to be hashed, not stored in plain text")
	}

	// Verify it is hashed with bcrypt
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("password123"))
	if err != nil {
		t.Errorf("password hash verification failed: %v", err)
	}

	// Verify user passed to Repository
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
			// Username already exists
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

	authService := NewAuthService(mockUserRepo, mockSessionStore, config.PasswordPolicyNone, 24*time.Hour, nil, 24*time.Hour)

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

	authService := NewAuthService(mockUserRepo, mockSessionStore, config.PasswordPolicyNone, 24*time.Hour, nil, 24*time.Hour)

	// Create two users with the same password
	_, err := authService.CreateUser("user1", "samepassword")
	if err != nil {
		t.Fatalf("failed to create user1: %v", err)
	}

	_, err = authService.CreateUser("user2", "samepassword")
	if err != nil {
		t.Fatalf("failed to create user2: %v", err)
	}

	// Verify different hashes are generated for same password (bcrypt salt)
	if user1Hash == user2Hash {
		t.Error("expected different password hashes for same password (bcrypt should use different salts)")
	}

	// Verify both hashes can validate correct password
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

	authService := NewAuthService(mockUserRepo, mockSessionStore, config.PasswordPolicyNone, 24*time.Hour, nil, 24*time.Hour)

	// With NONE policy, short passwords are OK
	tests := []struct {
		name     string
		password string
	}{
		{"Short password", "pass"},
		{"Numbers only", "12345"},
		{"No symbols", "password"},
		{"Lowercase only", "abcdefgh"},
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

	authService := NewAuthService(mockUserRepo, mockSessionStore, config.PasswordPolicyStrong, 24*time.Hour, nil, 24*time.Hour)

	// Passwords that meet STRONG policy requirements
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

	authService := NewAuthService(mockUserRepo, mockSessionStore, config.PasswordPolicyStrong, 24*time.Hour, nil, 24*time.Hour)

	tests := []struct {
		name     string
		password string
	}{
		{"Too short", "Pass1!"},                       // 6 characters
		{"No uppercase", "password123!"},              // No uppercase
		{"No lowercase", "PASSWORD123!"},              // No lowercase
		{"No numbers", "Password!@#$"},                // No numbers
		{"No symbols", "Password1234"},                // No symbols
		{"14 characters", "Passw0rd!1234"},            // 14 characters (less than 15)
		{"12 characters", "Passw0rd!12"},              // 12 characters (less than 15)
		{"Multiple requirements missing", "password"}, // No uppercase, numbers, or symbols
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
	// Generate password hash for testing
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
		createFunc: func(userID int64, ttl time.Duration, meta auth.SessionMeta) (string, error) {
			return "test-session-id", nil
		},
	}

	authService := NewAuthService(mockUserRepo, mockSessionStore, config.PasswordPolicyNone, 24*time.Hour, nil, 24*time.Hour)

	ipAddress := "192.168.1.100"

	// First 3 failures have no delay
	start := time.Now()
	for i := 0; i < 3; i++ {
		_, err := authService.Login("testuser", "wrongpassword", ipAddress)
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Errorf("attempt %d: expected ErrInvalidCredentials, got %v", i+1, err)
		}
	}
	elapsed := time.Since(start)

	// First 3 attempts have no delay, so should complete in less than 1 second
	if elapsed > 1*time.Second {
		t.Errorf("expected first 3 failures to complete quickly, took %v", elapsed)
	}

	// 4th failure onwards has delay (2 seconds)
	start = time.Now()
	_, err = authService.Login("testuser", "wrongpassword", ipAddress)
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
	elapsed = time.Since(start)

	// 4th attempt should have 2 second delay
	if elapsed < 2*time.Second {
		t.Errorf("expected 4th failure to have 2s delay, took only %v", elapsed)
	}
	if elapsed > 3*time.Second {
		t.Errorf("expected 4th failure to complete in ~2s, took %v", elapsed)
	}
}

func TestAuthService_BruteForce_SuccessResetsCounter(t *testing.T) {
	// Generate password hash for testing
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
		createFunc: func(userID int64, ttl time.Duration, meta auth.SessionMeta) (string, error) {
			return "test-session-id", nil
		},
	}

	authService := NewAuthService(mockUserRepo, mockSessionStore, config.PasswordPolicyNone, 24*time.Hour, nil, 24*time.Hour)

	ipAddress := "192.168.1.101"

	// Fail 2 times
	for i := 0; i < 2; i++ {
		_, err := authService.Login("testuser", "wrongpassword", ipAddress)
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Errorf("attempt %d: expected ErrInvalidCredentials, got %v", i+1, err)
		}
	}

	// Success
	_, err = authService.Login("testuser", "password123", ipAddress)
	if err != nil {
		t.Fatalf("expected successful login, got %v", err)
	}

	// Counter should be reset
	// Subsequent failure should complete immediately (no delay)
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
	// Generate password hash for testing
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
		createFunc: func(userID int64, ttl time.Duration, meta auth.SessionMeta) (string, error) {
			return "test-session-id", nil
		},
	}

	authService := NewAuthService(mockUserRepo, mockSessionStore, config.PasswordPolicyNone, 24*time.Hour, nil, 24*time.Hour)

	// Fail 3 times from IP1 (triggers delay state)
	ip1 := "192.168.1.100"
	for i := 0; i < 3; i++ {
		_, err := authService.Login("testuser", "wrongpassword", ip1)
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Errorf("IP1 attempt %d: expected ErrInvalidCredentials, got %v", i+1, err)
		}
	}

	// First failure from IP2 has no delay (independent counter)
	ip2 := "192.168.1.200"
	start := time.Now()
	_, err = authService.Login("testuser", "wrongpassword", ip2)
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
	elapsed := time.Since(start)

	// IP2 has new counter so no delay
	if elapsed > 1*time.Second {
		t.Errorf("expected IP2 first failure to complete quickly (independent counter), took %v", elapsed)
	}
}

func TestAuthService_BruteForce_EmptyIPAddress(t *testing.T) {
	// Generate password hash for testing
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
		createFunc: func(userID int64, ttl time.Duration, meta auth.SessionMeta) (string, error) {
			return "test-session-id", nil
		},
	}

	authService := NewAuthService(mockUserRepo, mockSessionStore, config.PasswordPolicyNone, 24*time.Hour, nil, 24*time.Hour)

	// When IP address is empty, brute force protection is disabled
	// No delay no matter how many failures
	start := time.Now()
	for i := 0; i < 5; i++ {
		_, err := authService.Login("testuser", "wrongpassword", "")
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Errorf("attempt %d: expected ErrInvalidCredentials, got %v", i+1, err)
		}
	}
	elapsed := time.Since(start)

	// No delay so should complete in less than 1 second
	if elapsed > 1*time.Second {
		t.Errorf("expected failures with empty IP to complete quickly (no brute force protection), took %v", elapsed)
	}
}

func TestAuthService_SessionTTL_PropagatesConstructorArg(t *testing.T) {
	tests := []struct {
		name string
		ttl  time.Duration
	}{
		{"default", 24 * time.Hour},
		{"short", 30 * time.Minute},
		{"long", 7 * 24 * time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewAuthService(&mockUserRepository{}, &mockSessionStore{}, config.PasswordPolicyNone, tt.ttl, nil, 24*time.Hour)
			if got := svc.SessionTTL(); got != tt.ttl {
				t.Errorf("SessionTTL() = %v, want %v", got, tt.ttl)
			}
		})
	}
}

func TestAuthService_Login_PassesSessionTTLToStore(t *testing.T) {
	const wantTTL = 90 * time.Minute

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("bcrypt.GenerateFromPassword: %v", err)
	}

	mockUserRepo := &mockUserRepository{
		findByUsernameFunc: func(username string) (*domain.User, error) {
			return &domain.User{ID: 1, Username: username, PasswordHash: string(hashedPassword)}, nil
		},
	}
	var capturedTTL time.Duration
	mockSessionStore := &mockSessionStore{
		createFunc: func(userID int64, ttl time.Duration, meta auth.SessionMeta) (string, error) {
			capturedTTL = ttl
			return "session-id", nil
		},
	}

	svc := NewAuthService(mockUserRepo, mockSessionStore, config.PasswordPolicyNone, wantTTL, nil, 24*time.Hour)
	if _, err := svc.Login("user", "password123", "1.2.3.4"); err != nil {
		t.Fatalf("Login: %v", err)
	}

	if capturedTTL != wantTTL {
		t.Errorf("sessionStore.Create received ttl=%v, want %v", capturedTTL, wantTTL)
	}
}

func TestAuthService_IssueRememberToken_PersistsAndReturnsCookie(t *testing.T) {
	var created *domain.RememberToken
	store := &mockRememberTokenStore{
		createFunc: func(t *domain.RememberToken) error {
			created = t
			return nil
		},
	}
	svc := newAuthServiceForTest(&mockUserRepository{}, &mockSessionStore{}, store, 30*24*time.Hour)

	cookie, err := svc.IssueRememberToken(42)
	if err != nil {
		t.Fatalf("IssueRememberToken: %v", err)
	}
	if cookie == "" {
		t.Fatal("expected non-empty cookie value")
	}
	if created == nil {
		t.Fatal("expected Create call")
	}
	if created.UserID != 42 {
		t.Errorf("UserID = %d, want 42", created.UserID)
	}
	if created.Selector == "" || created.TokenHash == "" {
		t.Errorf("selector/hash must be set: %+v", created)
	}

	// Cookie format: selector:raw
	sel, raw, ok := auth.DecodeRememberCookie(cookie)
	if !ok {
		t.Fatalf("cookie %q does not decode", cookie)
	}
	if sel != created.Selector {
		t.Errorf("cookie selector = %q, db selector = %q", sel, created.Selector)
	}
	if auth.HashToken(raw) != created.TokenHash {
		t.Error("cookie raw token does not hash to stored hash")
	}
}

func TestAuthService_RestoreFromRememberToken_Success(t *testing.T) {
	selector := "sel"
	raw := "raw"
	hash := auth.HashToken(raw)
	store := &mockRememberTokenStore{
		findBySelectorFunc: func(s string) (*domain.RememberToken, error) {
			if s != selector {
				return nil, nil
			}
			return &domain.RememberToken{
				UserID: 1, Selector: selector, TokenHash: hash,
				ExpiresAt: time.Now().Add(time.Hour),
			}, nil
		},
	}
	userRepo := &mockUserRepository{
		findByIDFunc: func(id int64) (*domain.User, error) {
			return &domain.User{ID: 1, Username: "u"}, nil
		},
	}
	sessions := &mockSessionStore{
		createFunc: func(uid int64, ttl time.Duration, meta auth.SessionMeta) (string, error) {
			return "new-session", nil
		},
	}
	svc := newAuthServiceForTest(userRepo, sessions, store, time.Hour)

	user, newSession, err := svc.RestoreFromRememberToken(auth.EncodeRememberCookie(selector, raw))
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if user == nil || user.ID != 1 {
		t.Errorf("user = %+v, want id=1", user)
	}
	if newSession != "new-session" {
		t.Errorf("session = %q", newSession)
	}
}

func TestAuthService_RestoreFromRememberToken_FailurePaths(t *testing.T) {
	validHash := auth.HashToken("raw")

	tests := []struct {
		name   string
		cookie string
		store  *mockRememberTokenStore
	}{
		{
			name:   "malformed cookie",
			cookie: "not-a-valid-cookie",
			store:  &mockRememberTokenStore{},
		},
		{
			name:   "selector not found",
			cookie: auth.EncodeRememberCookie("missing", "raw"),
			store: &mockRememberTokenStore{
				findBySelectorFunc: func(string) (*domain.RememberToken, error) {
					return nil, nil
				},
			},
		},
		{
			name:   "expired token",
			cookie: auth.EncodeRememberCookie("sel", "raw"),
			store: &mockRememberTokenStore{
				findBySelectorFunc: func(string) (*domain.RememberToken, error) {
					return &domain.RememberToken{
						UserID: 1, Selector: "sel", TokenHash: validHash,
						ExpiresAt: time.Now().Add(-time.Minute),
					}, nil
				},
			},
		},
		{
			name:   "hash mismatch",
			cookie: auth.EncodeRememberCookie("sel", "wrong"),
			store: &mockRememberTokenStore{
				findBySelectorFunc: func(string) (*domain.RememberToken, error) {
					return &domain.RememberToken{
						UserID: 1, Selector: "sel", TokenHash: validHash,
						ExpiresAt: time.Now().Add(time.Hour),
					}, nil
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newAuthServiceForTest(&mockUserRepository{}, &mockSessionStore{}, tt.store, time.Hour)
			user, session, err := svc.RestoreFromRememberToken(tt.cookie)
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if user != nil || session != "" {
				t.Errorf("expected no restoration, got user=%+v session=%q", user, session)
			}
		})
	}
}

func TestAuthService_RestoreFromRememberToken_HashMismatch_RevokesAllUserTokens(t *testing.T) {
	const userID int64 = 42
	validHash := auth.HashToken("the-real-raw-token")

	var deleteByUserIDCalls int
	store := &mockRememberTokenStore{
		findBySelectorFunc: func(string) (*domain.RememberToken, error) {
			return &domain.RememberToken{
				UserID: userID, Selector: "sel", TokenHash: validHash,
				ExpiresAt: time.Now().Add(time.Hour),
			}, nil
		},
		deleteByUserIDFunc: func(uid int64) error {
			if uid != userID {
				t.Errorf("DeleteByUserID called for uid %d, want %d", uid, userID)
			}
			deleteByUserIDCalls++
			return nil
		},
	}
	svc := newAuthServiceForTest(&mockUserRepository{}, &mockSessionStore{}, store, time.Hour)

	user, session, err := svc.RestoreFromRememberToken(auth.EncodeRememberCookie("sel", "WRONG-raw-token"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user != nil || session != "" {
		t.Errorf("expected no restoration on hash mismatch, got user=%+v session=%q", user, session)
	}
	if deleteByUserIDCalls != 1 {
		t.Errorf("DeleteByUserID call count = %d, want 1 (theft detection)", deleteByUserIDCalls)
	}
}

func TestAuthService_RestoreFromRememberToken_RefreshesOnUse(t *testing.T) {
	const rememberTTL = 30 * 24 * time.Hour
	raw := "good-raw"
	hash := auth.HashToken(raw)

	var refreshCall struct {
		selector     string
		lastUsed     time.Time
		newExpiresAt time.Time
	}
	store := &mockRememberTokenStore{
		findBySelectorFunc: func(string) (*domain.RememberToken, error) {
			return &domain.RememberToken{
				UserID: 1, Selector: "sel", TokenHash: hash,
				ExpiresAt: time.Now().Add(24 * time.Hour),
			}, nil
		},
		refreshOnUseFunc: func(sel string, lastUsed time.Time, newExpiresAt time.Time) error {
			refreshCall.selector = sel
			refreshCall.lastUsed = lastUsed
			refreshCall.newExpiresAt = newExpiresAt
			return nil
		},
	}
	userRepo := &mockUserRepository{
		findByIDFunc: func(int64) (*domain.User, error) {
			return &domain.User{ID: 1, Username: "u"}, nil
		},
	}
	svc := newAuthServiceForTest(userRepo, &mockSessionStore{}, store, rememberTTL)

	before := time.Now()
	_, _, err := svc.RestoreFromRememberToken(auth.EncodeRememberCookie("sel", raw))
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	after := time.Now()

	if refreshCall.selector != "sel" {
		t.Errorf("refresh selector = %q", refreshCall.selector)
	}
	if refreshCall.lastUsed.Before(before) || refreshCall.lastUsed.After(after) {
		t.Errorf("lastUsed = %v, want within [%v, %v]", refreshCall.lastUsed, before, after)
	}
	expectedExpiry := refreshCall.lastUsed.Add(rememberTTL)
	if !refreshCall.newExpiresAt.Equal(expectedExpiry) {
		t.Errorf("newExpiresAt = %v, want lastUsed + rememberTTL = %v", refreshCall.newExpiresAt, expectedExpiry)
	}
}

func TestAuthService_RevokeRememberToken(t *testing.T) {
	var deletedSel string
	store := &mockRememberTokenStore{
		deleteFunc: func(s string) error {
			deletedSel = s
			return nil
		},
	}
	svc := newAuthServiceForTest(&mockUserRepository{}, &mockSessionStore{}, store, time.Hour)

	if err := svc.RevokeRememberToken(auth.EncodeRememberCookie("sel-z", "raw")); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if deletedSel != "sel-z" {
		t.Errorf("deleted selector = %q, want sel-z", deletedSel)
	}
}

func TestAuthService_RevokeRememberToken_MalformedCookieIsNoop(t *testing.T) {
	called := false
	store := &mockRememberTokenStore{
		deleteFunc: func(string) error {
			called = true
			return nil
		},
	}
	svc := newAuthServiceForTest(&mockUserRepository{}, &mockSessionStore{}, store, time.Hour)

	if err := svc.RevokeRememberToken("nonsense"); err != nil {
		t.Errorf("expected nil error on malformed cookie, got %v", err)
	}
	if called {
		t.Error("malformed cookie must not trigger Delete")
	}
}

func TestAuthService_RememberMethods_NilStoreDoNotPanic(t *testing.T) {
	// NewAuthService documents that rememberStore may be nil (remember-me
	// disabled). The three remember methods must degrade gracefully instead
	// of dereferencing a nil store and panicking.
	svc := NewAuthService(&mockUserRepository{}, &mockSessionStore{}, config.PasswordPolicyNone, 24*time.Hour, nil, 24*time.Hour)

	if _, err := svc.IssueRememberToken(1); err == nil {
		t.Error("IssueRememberToken with nil store should return an error")
	}

	user, session, err := svc.RestoreFromRememberToken(auth.EncodeRememberCookie("sel", "raw"))
	if err != nil || user != nil || session != "" {
		t.Errorf("RestoreFromRememberToken with nil store should be a no-op miss, got user=%+v session=%q err=%v", user, session, err)
	}

	if err := svc.RevokeRememberToken(auth.EncodeRememberCookie("sel", "raw")); err != nil {
		t.Errorf("RevokeRememberToken with nil store should be a no-op, got %v", err)
	}
}

func TestAuthService_RememberTTL_PropagatesConstructorArg(t *testing.T) {
	svc := newAuthServiceForTest(&mockUserRepository{}, &mockSessionStore{}, &mockRememberTokenStore{}, 14*24*time.Hour)
	if got := svc.RememberTTL(); got != 14*24*time.Hour {
		t.Errorf("RememberTTL() = %v, want 14d", got)
	}
}
