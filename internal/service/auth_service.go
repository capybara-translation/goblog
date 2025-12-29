package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/capybara-translation/goblog/internal/auth"
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
	userRepo     repo.UserRepository
	sessionStore auth.SessionStore
	sessionTTL   time.Duration
}

// NewAuthService は新しいAuthServiceを作成します
func NewAuthService(userRepo repo.UserRepository, sessionStore auth.SessionStore) AuthService {
	return &authService{
		userRepo:     userRepo,
		sessionStore: sessionStore,
		sessionTTL:   24 * time.Hour, // セッション有効期限: 24時間
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
