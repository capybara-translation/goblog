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
	// ErrInvalidCredentials is an error returned when authentication credentials are invalid
	ErrInvalidCredentials = errors.New("invalid username or password")
	// ErrUserNotFound is an error returned when a user is not found
	ErrUserNotFound = errors.New("user not found")
	// ErrUsernameAlreadyExists is an error returned when a username already exists
	ErrUsernameAlreadyExists = errors.New("username already exists")
	// ErrWeakPassword is an error returned when a password is too weak
	ErrWeakPassword = errors.New("password does not meet security requirements")
)

// AuthService is an interface that provides business logic for authentication
type AuthService interface {
	// Login authenticates with username and password, and returns a session ID
	// ipAddress is used for brute force protection
	Login(username, password, ipAddress string) (string, error)

	// Logout deletes a session
	Logout(sessionID string) error

	// GetUserBySession retrieves user information from a session ID
	GetUserBySession(sessionID string) (*domain.User, error)

	// CreateUser creates a new user
	CreateUser(username, password string) (*domain.User, error)
}

// loginAttempt holds information about failed login attempts
type loginAttempt struct {
	failCount  int       // number of failures
	lastFailed time.Time // time of the last failure
}

// authService is an implementation of AuthService
type authService struct {
	userRepo       repo.UserRepository
	sessionStore   auth.SessionStore
	sessionTTL     time.Duration
	passwordPolicy config.PasswordPolicy
	// Brute force protection
	loginAttempts map[string]*loginAttempt // failure information per IP address
	attemptsMutex sync.RWMutex             // access control for loginAttempts
}

// NewAuthService creates a new AuthService
func NewAuthService(userRepo repo.UserRepository, sessionStore auth.SessionStore, passwordPolicy config.PasswordPolicy) AuthService {
	s := &authService{
		userRepo:       userRepo,
		sessionStore:   sessionStore,
		sessionTTL:     24 * time.Hour, // session expiration: 24 hours
		passwordPolicy: passwordPolicy,
		loginAttempts:  make(map[string]*loginAttempt),
	}

	// Periodically clean up old login failure records
	go s.cleanupLoginAttempts()

	return s
}

// Login authenticates with username and password, and returns a session ID
func (s *authService) Login(username, password, ipAddress string) (string, error) {
	// Brute force protection: delay based on failure count
	s.applyLoginDelay(ipAddress)

	// Search for user by username
	user, err := s.userRepo.FindByUsername(username)
	if err != nil {
		return "", fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		s.recordLoginFailure(ipAddress)
		return "", ErrInvalidCredentials
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		s.recordLoginFailure(ipAddress)
		return "", ErrInvalidCredentials
	}

	// Login successful: reset failure count
	s.resetLoginAttempts(ipAddress)

	// Create session
	sessionID, err := s.sessionStore.Create(user.ID, s.sessionTTL)
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	return sessionID, nil
}

// Logout deletes a session
func (s *authService) Logout(sessionID string) error {
	return s.sessionStore.Delete(sessionID)
}

// GetUserBySession retrieves user information from a session ID
func (s *authService) GetUserBySession(sessionID string) (*domain.User, error) {
	// Get session
	session, err := s.sessionStore.Get(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	if session == nil {
		return nil, nil
	}

	// Get user information
	user, err := s.userRepo.FindByID(session.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	return user, nil
}

// CreateUser creates a new user
func (s *authService) CreateUser(username, password string) (*domain.User, error) {
	// Check for username duplication
	existing, err := s.userRepo.FindByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("failed to check username: %w", err)
	}
	if existing != nil {
		return nil, ErrUsernameAlreadyExists
	}

	// Validation based on password policy
	if err := s.validatePassword(password); err != nil {
		return nil, err
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
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

// validatePassword validates a password based on the password policy
func (s *authService) validatePassword(password string) error {
	switch s.passwordPolicy {
	case config.PasswordPolicyNone:
		// No restrictions
		return nil
	case config.PasswordPolicyStrong:
		return s.validateStrongPassword(password)
	default:
		// Treat unknown policies as NONE
		return nil
	}
}

// validateStrongPassword validates a strict password policy
// - Minimum 15 characters
// - At least one uppercase letter
// - At least one lowercase letter
// - At least one digit
// - At least one symbol
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

// applyLoginDelay applies a delay based on the number of login failures
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

	// Delay based on failure count
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

// recordLoginFailure records a login failure
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

// resetLoginAttempts resets the failure count on successful login
func (s *authService) resetLoginAttempts(ipAddress string) {
	if ipAddress == "" {
		return
	}

	s.attemptsMutex.Lock()
	defer s.attemptsMutex.Unlock()

	delete(s.loginAttempts, ipAddress)
}

// cleanupLoginAttempts periodically cleans up old login failure records
func (s *authService) cleanupLoginAttempts() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.attemptsMutex.Lock()
		now := time.Now()
		for ip, attempt := range s.loginAttempts {
			// Delete if 30 minutes have passed since the last failure
			if now.Sub(attempt.lastFailed) > 30*time.Minute {
				delete(s.loginAttempts, ip)
			}
		}
		s.attemptsMutex.Unlock()
	}
}
