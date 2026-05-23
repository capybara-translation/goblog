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

	// SessionTTL reports how long a freshly created session remains valid.
	// HTTP handlers use this to drive cookie MaxAge so the cookie and the
	// server-side session expire together.
	SessionTTL() time.Duration

	// IssueRememberToken creates a long-lived remember-me token for the user
	// and returns the cookie value to set.
	IssueRememberToken(userID int64) (cookieValue string, err error)

	// RestoreFromRememberToken validates a remember-me cookie and, on success,
	// issues a new session. Returns (nil, "", nil) for any non-fatal failure
	// (missing, expired, hash mismatch, malformed) so callers can treat it as
	// "no user" without flooding logs.
	RestoreFromRememberToken(cookieValue string) (*domain.User, string, error)

	// RevokeRememberToken deletes the remember-token row identified by the
	// cookie's selector. Malformed cookies are silently ignored.
	RevokeRememberToken(cookieValue string) error

	// RememberTTL reports the TTL applied to newly issued remember tokens.
	RememberTTL() time.Duration
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
	rememberStore  auth.RememberTokenStore
	rememberTTL    time.Duration
	// Brute force protection
	loginAttempts map[string]*loginAttempt // failure information per IP address
	attemptsMutex sync.RWMutex             // access control for loginAttempts
}

// NewAuthService creates a new AuthService. sessionTTL controls how long a
// freshly issued session remains valid; HTTP handlers read it back via
// SessionTTL() to keep the session cookie's MaxAge in sync. rememberStore
// persists "remember me" tokens (may be nil if remember-me is disabled, but
// callers must then avoid invoking the remember-token methods); rememberTTL
// is read back via RememberTTL() to keep the remember cookie's MaxAge in sync.
func NewAuthService(
	userRepo repo.UserRepository,
	sessionStore auth.SessionStore,
	passwordPolicy config.PasswordPolicy,
	sessionTTL time.Duration,
	rememberStore auth.RememberTokenStore,
	rememberTTL time.Duration,
) AuthService {
	s := &authService{
		userRepo:       userRepo,
		sessionStore:   sessionStore,
		sessionTTL:     sessionTTL,
		passwordPolicy: passwordPolicy,
		rememberStore:  rememberStore,
		rememberTTL:    rememberTTL,
		loginAttempts:  make(map[string]*loginAttempt),
	}

	// Periodically clean up old login failure records
	go s.cleanupLoginAttempts()

	return s
}

// SessionTTL reports how long a freshly created session remains valid.
func (s *authService) SessionTTL() time.Duration {
	return s.sessionTTL
}

// RememberTTL reports the TTL applied to newly issued remember tokens.
func (s *authService) RememberTTL() time.Duration {
	return s.rememberTTL
}

// IssueRememberToken creates a long-lived remember-me token for the user and
// returns the cookie value (selector:rawToken) that the caller should set on
// the client. Only the SHA-256 hash of the raw token is persisted.
func (s *authService) IssueRememberToken(userID int64) (string, error) {
	selector := auth.GenerateSelector()
	rawToken := auth.GenerateRawToken()
	token := &domain.RememberToken{
		UserID:    userID,
		Selector:  selector,
		TokenHash: auth.HashToken(rawToken),
		ExpiresAt: time.Now().Add(s.rememberTTL),
	}
	if err := s.rememberStore.Create(token); err != nil {
		return "", fmt.Errorf("create remember token: %w", err)
	}
	return auth.EncodeRememberCookie(selector, rawToken), nil
}

// RestoreFromRememberToken validates a remember-me cookie and, on success,
// creates a new session and returns the user plus the new session ID. Any
// non-fatal failure (malformed cookie, unknown selector, expired token, hash
// mismatch, deleted user) returns (nil, "", nil) so the caller can treat it
// as "no logged-in user" without leaking signal about which failure happened.
// Only genuine infrastructure errors (DB / session store) are surfaced.
func (s *authService) RestoreFromRememberToken(cookieValue string) (*domain.User, string, error) {
	selector, raw, ok := auth.DecodeRememberCookie(cookieValue)
	if !ok {
		return nil, "", nil
	}
	token, err := s.rememberStore.FindBySelector(selector)
	if err != nil {
		return nil, "", fmt.Errorf("find remember token: %w", err)
	}
	if token == nil {
		return nil, "", nil
	}
	if time.Now().After(token.ExpiresAt) {
		// Best-effort cleanup; expired-token path is non-fatal even if delete fails.
		_ = s.rememberStore.Delete(selector)
		return nil, "", nil
	}
	if !auth.ConstantTimeEqual(token.TokenHash, auth.HashToken(raw)) {
		return nil, "", nil
	}

	user, err := s.userRepo.FindByID(token.UserID)
	if err != nil {
		return nil, "", fmt.Errorf("find user: %w", err)
	}
	if user == nil {
		// Orphan token — user was deleted. Clean it up so we don't keep matching.
		_ = s.rememberStore.Delete(selector)
		return nil, "", nil
	}

	sessionID, err := s.sessionStore.Create(user.ID, s.sessionTTL)
	if err != nil {
		return nil, "", fmt.Errorf("create session: %w", err)
	}

	// Best-effort last-used timestamp so admin tooling can see activity.
	_ = s.rememberStore.UpdateLastUsed(selector, time.Now())

	return user, sessionID, nil
}

// RevokeRememberToken deletes the remember-token row identified by the
// cookie's selector. Malformed cookies are silently ignored so that callers
// (typically logout) can pass through any incoming cookie value without
// pre-validation.
func (s *authService) RevokeRememberToken(cookieValue string) error {
	selector, _, ok := auth.DecodeRememberCookie(cookieValue)
	if !ok {
		return nil
	}
	return s.rememberStore.Delete(selector)
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
