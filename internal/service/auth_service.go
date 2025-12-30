package service

import (
	"errors"
	"fmt"
	"strings"
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
	Login(username, password string) (string, error)

	// Logout はセッションを削除します
	Logout(sessionID string) error

	// GetUserBySession はセッションIDからユーザー情報を取得します
	GetUserBySession(sessionID string) (*domain.User, error)

	// CreateUser は新しいユーザーを作成します
	CreateUser(username, password string) (*domain.User, error)
}

// authService はAuthServiceの実装です
type authService struct {
	userRepo       repo.UserRepository
	sessionStore   auth.SessionStore
	sessionTTL     time.Duration
	passwordPolicy config.PasswordPolicy
}

// NewAuthService は新しいAuthServiceを作成します
func NewAuthService(userRepo repo.UserRepository, sessionStore auth.SessionStore, passwordPolicy config.PasswordPolicy) AuthService {
	return &authService{
		userRepo:       userRepo,
		sessionStore:   sessionStore,
		sessionTTL:     24 * time.Hour, // セッション有効期限: 24時間
		passwordPolicy: passwordPolicy,
	}
}

// Login はユーザー名とパスワードで認証し、セッションIDを返します
func (s *authService) Login(username, password string) (string, error) {
	// ユーザー名でユーザーを検索
	user, err := s.userRepo.FindByUsername(username)
	if err != nil {
		return "", fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return "", ErrInvalidCredentials
	}

	// パスワードを検証
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

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
