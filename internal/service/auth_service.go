package service

import (
	"errors"
	"fmt"
	"log"
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
	// Login authenticates with username and password, and returns a session ID.
	// ipAddress is used for brute force protection; userAgent/ipAddress are also
	// recorded on the session for the device-management feature.
	Login(username, password, ipAddress, userAgent string) (string, error)

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

	// IssueRememberToken creates a long-lived remember-me token for the user,
	// recording the device's user agent and IP, and returns the cookie value.
	IssueRememberToken(userID int64, userAgent, ipAddress string) (cookieValue string, err error)

	// RestoreFromRememberToken validates a remember-me cookie and, on success,
	// issues a new session carrying the given userAgent/ipAddress and linked to
	// the token's selector. Returns (nil, "", nil) for any non-fatal failure
	// (missing, expired, hash mismatch, malformed) so callers can treat it as
	// "no user" without flooding logs.
	RestoreFromRememberToken(cookieValue, userAgent, ipAddress string) (*domain.User, string, error)

	// RevokeRememberToken deletes the remember-token row identified by the
	// cookie's selector. Malformed cookies are silently ignored.
	RevokeRememberToken(cookieValue string) error

	// RememberTTL reports the TTL applied to newly issued remember tokens.
	RememberTTL() time.Duration

	// AttachRememberSelector links an already-created session to a remember
	// token's selector (used right after a remember-me login).
	AttachRememberSelector(sessionID, selector string)

	// TouchSession bumps the session's last-used timestamp.
	TouchSession(sessionID string, t time.Time)
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
// persists "remember me" tokens and may be nil to disable remember-me: the
// remember-token methods degrade safely in that case (IssueRememberToken
// returns an error, RestoreFromRememberToken is a no-op miss, and
// RevokeRememberToken is a no-op), so callers do not need to guard against a
// nil store. rememberTTL is read back via RememberTTL() to keep the remember
// cookie's MaxAge in sync.
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

// AttachRememberSelector links an already-created session to a remember
// token's selector (used right after a remember-me login).
func (s *authService) AttachRememberSelector(sessionID, selector string) {
	s.sessionStore.SetRememberSelector(sessionID, selector)
}

// TouchSession bumps the session's last-used timestamp.
func (s *authService) TouchSession(sessionID string, t time.Time) {
	s.sessionStore.Touch(sessionID, t)
}

// IssueRememberToken creates a long-lived remember-me token for the user and
// returns the cookie value (selector:rawToken) that the caller should set on
// the client. Only the SHA-256 hash of the raw token is persisted.
//
// Returns an error if remember-me is not configured (rememberStore == nil) so
// the caller can degrade gracefully rather than panicking.
func (s *authService) IssueRememberToken(userID int64, userAgent, ipAddress string) (string, error) {
	if s.rememberStore == nil {
		return "", errors.New("remember-me is not configured")
	}
	selector := auth.GenerateSelector()
	rawToken := auth.GenerateRawToken()
	token := &domain.RememberToken{
		UserID:    userID,
		Selector:  selector,
		TokenHash: auth.HashToken(rawToken),
		ExpiresAt: time.Now().Add(s.rememberTTL),
		UserAgent: userAgent,
		IPAddress: ipAddress,
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
func (s *authService) RestoreFromRememberToken(cookieValue, userAgent, ipAddress string) (*domain.User, string, error) {
	if s.rememberStore == nil {
		// Remember-me disabled: treat any presented token as a miss.
		return nil, "", nil
	}
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
		// Hash mismatch with a valid selector signals either token theft + a
		// brute-force attempt or token-tampering. Treat it as a theft event
		// and revoke every remember token this user owns so the attacker
		// cannot fall back to another stolen pair. The user will need to
		// re-authenticate on every device, which is the safe outcome.
		log.Printf("RestoreFromRememberToken: hash mismatch for selector=%s user_id=%d; revoking all tokens for this user", selector, token.UserID)
		if revokeErr := s.rememberStore.DeleteByUserID(token.UserID); revokeErr != nil {
			log.Printf("RestoreFromRememberToken: DeleteByUserID failed after hash mismatch: %v", revokeErr)
		}
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

	sessionID, err := s.sessionStore.Create(user.ID, s.sessionTTL, auth.SessionMeta{
		UserAgent:        userAgent,
		IP:               ipAddress,
		RememberSelector: selector,
	})
	if err != nil {
		return nil, "", fmt.Errorf("create session: %w", err)
	}

	// Refresh on use: bump last_used_at AND extend expires_at by another full
	// rememberTTL window. Without sliding expiration, an active daily user
	// would still be force-logged-out every TTL — which defeats the purpose
	// of "remember me". Best-effort; if it fails the user is still restored.
	now := time.Now()
	_ = s.rememberStore.RefreshOnUse(selector, now, now.Add(s.rememberTTL))

	// Self-heal device metadata: tokens issued before this feature existed have
	// empty user_agent/ip_address (migration default ''), which the device list
	// renders as "不明な端末". Backfill from the current request only when the
	// stored UA is empty, so we heal legacy rows without overwriting an existing
	// token's issue-time metadata. Best-effort; failure does not affect restore.
	if token.UserAgent == "" && userAgent != "" {
		_ = s.rememberStore.UpdateDeviceInfo(selector, userAgent, ipAddress)
	}

	return user, sessionID, nil
}

// RevokeRememberToken deletes the remember-token row identified by the
// cookie's selector. Malformed cookies are silently ignored so that callers
// (typically logout) can pass through any incoming cookie value without
// pre-validation.
func (s *authService) RevokeRememberToken(cookieValue string) error {
	if s.rememberStore == nil {
		// Remember-me disabled: nothing to revoke.
		return nil
	}
	selector, _, ok := auth.DecodeRememberCookie(cookieValue)
	if !ok {
		return nil
	}
	return s.rememberStore.Delete(selector)
}

// Login authenticates with username and password, and returns a session ID
func (s *authService) Login(username, password, ipAddress, userAgent string) (string, error) {
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
	sessionID, err := s.sessionStore.Create(user.ID, s.sessionTTL, auth.SessionMeta{
		UserAgent: userAgent,
		IP:        ipAddress,
	})
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
