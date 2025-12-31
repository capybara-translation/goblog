package service

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/capybara-translation/goblog/internal/auth"
	"github.com/capybara-translation/goblog/internal/config"
	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/capybara-translation/goblog/internal/repo"
	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrInvalidCredentials は認証情報が不正な場合のエラーです
	ErrInvalidCredentials = errors.New("invalid username or password")
	// ErrUserNotFound はユーザーが見つからない場合のエラーです
	ErrUserNotFound = errors.New("user not found")
	// ErrUsernameAlreadyExists はユーザー名が既に存在する場合のエラーです
	ErrUsernameAlreadyExists = errors.New("username already exists")
	// ErrWeakPassword はパスワードが弱い場合のエラーです
	ErrWeakPassword = errors.New("password does not meet security requirements")
)

// AuthService は認証に関するビジネスロジックを提供するインターフェースです
type AuthService interface {
	// Login はユーザー名とパスワードで認証し、セッションIDを返します
	// ipAddress はブルートフォース対策のために使用されます
	Login(username, password, ipAddress string) (string, error)

	// Logout はセッションを削除します
	Logout(sessionID string) error

	// GetUserBySession はセッションIDからユーザー情報を取得します
	GetUserBySession(sessionID string) (*domain.User, error)

	// CreateUser は新しいユーザーを作成します
	CreateUser(username, password string) (*domain.User, error)
}

// loginAttempt はログイン失敗の試行情報を保持します
type loginAttempt struct {
	failCount  int       // 失敗回数
	lastFailed time.Time // 最後の失敗時刻
}

// authService はAuthServiceの実装です
type authService struct {
	userRepo       repo.UserRepository
	sessionStore   auth.SessionStore
	sessionTTL     time.Duration
	passwordPolicy config.PasswordPolicy
	// ブルートフォース対策
	loginAttempts map[string]*loginAttempt // IPアドレスごとの失敗情報
	attemptsMutex sync.RWMutex             // loginAttemptsへのアクセス制御
}

// NewAuthService は新しいAuthServiceを作成します
func NewAuthService(userRepo repo.UserRepository, sessionStore auth.SessionStore, passwordPolicy config.PasswordPolicy) AuthService {
	s := &authService{
		userRepo:       userRepo,
		sessionStore:   sessionStore,
		sessionTTL:     24 * time.Hour, // セッション有効期限: 24時間
		passwordPolicy: passwordPolicy,
		loginAttempts:  make(map[string]*loginAttempt),
	}

	// 古いログイン失敗記録を定期的にクリーンアップ
	go s.cleanupLoginAttempts()

	return s
}

// Login はユーザー名とパスワードで認証し、セッションIDを返します
func (s *authService) Login(username, password, ipAddress string) (string, error) {
	// ブルートフォース対策: 失敗回数に基づく遅延
	s.applyLoginDelay(ipAddress)

	// ユーザー名でユーザーを検索
	user, err := s.userRepo.FindByUsername(username)
	if err != nil {
		return "", fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		s.recordLoginFailure(ipAddress)
		return "", ErrInvalidCredentials
	}

	// パスワードを検証
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		s.recordLoginFailure(ipAddress)
		return "", ErrInvalidCredentials
	}

	// ログイン成功: 失敗カウントをリセット
	s.resetLoginAttempts(ipAddress)

	// セッションを作成
	sessionID, err := s.sessionStore.Create(user.ID, s.sessionTTL)
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	return sessionID, nil
}

// Logout はセッションを削除します
func (s *authService) Logout(sessionID string) error {
	return s.sessionStore.Delete(sessionID)
}

// GetUserBySession はセッションIDからユーザー情報を取得します
func (s *authService) GetUserBySession(sessionID string) (*domain.User, error) {
	// セッションを取得
	session, err := s.sessionStore.Get(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	if session == nil {
		return nil, nil
	}

	// ユーザー情報を取得
	user, err := s.userRepo.FindByID(session.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return user, nil
}

// CreateUser は新しいユーザーを作成します
func (s *authService) CreateUser(username, password string) (*domain.User, error) {
	// ユーザー名の重複チェック
	existing, err := s.userRepo.FindByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("failed to check username: %w", err)
	}
	if existing != nil {
		return nil, ErrUsernameAlreadyExists
	}

	// パスワードポリシーに基づくバリデーション
	if err := s.validatePassword(password); err != nil {
		return nil, err
	}

	// パスワードをハッシュ化
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// ユーザーを作成
	now := time.Now()
	user := &domain.User{
		Username:     username,
		PasswordHash: string(hashedPassword),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// validatePassword はパスワードポリシーに基づいてパスワードを検証します
func (s *authService) validatePassword(password string) error {
	switch s.passwordPolicy {
	case config.PasswordPolicyNone:
		// 制限なし
		return nil
	case config.PasswordPolicyStrong:
		return s.validateStrongPassword(password)
	default:
		// 不明なポリシーの場合はNONEとして扱う
		return nil
	}
}

// validateStrongPassword は厳格なパスワードポリシーを検証します
// - 最小15文字
// - 大文字を1文字以上含む
// - 小文字を1文字以上含む
// - 数字を1文字以上含む
// - 記号を1文字以上含む
func (s *authService) validateStrongPassword(password string) error {
	if len(password) < 15 {
		return fmt.Errorf("%w: password must be at least 15 characters", ErrWeakPassword)
	}

	var (
		hasUpper   bool
		hasLower   bool
		hasDigit   bool
		hasSpecial bool
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	var missing []string
	if !hasUpper {
		missing = append(missing, "uppercase letter")
	}
	if !hasLower {
		missing = append(missing, "lowercase letter")
	}
	if !hasDigit {
		missing = append(missing, "digit")
	}
	if !hasSpecial {
		missing = append(missing, "special character")
	}

	if len(missing) > 0 {
		return fmt.Errorf("%w: password must contain at least one %s", ErrWeakPassword, strings.Join(missing, ", "))
	}

	return nil
}

// applyLoginDelay はログイン失敗回数に基づいて遅延を適用します
func (s *authService) applyLoginDelay(ipAddress string) {
	if ipAddress == "" {
		return
	}

	s.attemptsMutex.RLock()
	attempt, exists := s.loginAttempts[ipAddress]
	s.attemptsMutex.RUnlock()

	if !exists {
		return
	}

	// 失敗回数に応じた遅延
	var delay time.Duration
	switch {
	case attempt.failCount >= 10:
		delay = 30 * time.Second
	case attempt.failCount >= 5:
		delay = 5 * time.Second
	case attempt.failCount >= 3:
		delay = 2 * time.Second
	}

	if delay > 0 {
		time.Sleep(delay)
	}
}

// recordLoginFailure はログイン失敗を記録します
func (s *authService) recordLoginFailure(ipAddress string) {
	if ipAddress == "" {
		return
	}

	s.attemptsMutex.Lock()
	defer s.attemptsMutex.Unlock()

	attempt, exists := s.loginAttempts[ipAddress]
	if !exists {
		s.loginAttempts[ipAddress] = &loginAttempt{
			failCount:  1,
			lastFailed: time.Now(),
		}
	} else {
		attempt.failCount++
		attempt.lastFailed = time.Now()
	}
}

// resetLoginAttempts はログイン成功時に失敗カウントをリセットします
func (s *authService) resetLoginAttempts(ipAddress string) {
	if ipAddress == "" {
		return
	}

	s.attemptsMutex.Lock()
	defer s.attemptsMutex.Unlock()

	delete(s.loginAttempts, ipAddress)
}

// cleanupLoginAttempts は古いログイン失敗記録を定期的にクリーンアップします
func (s *authService) cleanupLoginAttempts() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.attemptsMutex.Lock()
		now := time.Now()
		for ip, attempt := range s.loginAttempts {
			// 最後の失敗から30分経過したら削除
			if now.Sub(attempt.lastFailed) > 30*time.Minute {
				delete(s.loginAttempts, ip)
			}
		}
		s.attemptsMutex.Unlock()
	}
}
