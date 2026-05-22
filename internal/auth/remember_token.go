package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"time"

	"github.com/capybara-translation/goblog/internal/domain"
)

// RememberTokenStore manages long-lived remember-me tokens.
type RememberTokenStore interface {
	Create(token *domain.RememberToken) error
	FindBySelector(selector string) (*domain.RememberToken, error)
	Delete(selector string) error
	DeleteByUserID(userID int64) error
	UpdateLastUsed(selector string, t time.Time) error
	CleanupExpired() error
}

// rememberCookieSeparator separates selector and raw token in the cookie value.
const rememberCookieSeparator = ":"

// GenerateSelector returns 16 cryptographically random bytes, base64-url
// encoded. The selector is the public lookup key stored in plain text.
func GenerateSelector() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// rand.Read on /dev/urandom never fails on supported platforms; panic
		// here matches the existing generateSessionID style.
		panic("auth: rand.Read failed for selector: " + err.Error())
	}
	return base64.URLEncoding.EncodeToString(b)
}

// GenerateRawToken returns 32 cryptographically random bytes, base64-url
// encoded. The raw token is the secret; only its hash is stored.
func GenerateRawToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("auth: rand.Read failed for raw token: " + err.Error())
	}
	return base64.URLEncoding.EncodeToString(b)
}

// HashToken hashes a raw token with SHA-256 and returns its hex encoding.
// SHA-256 is sufficient here: the raw token has 256 bits of entropy so a
// brute-force preimage attack is infeasible even without bcrypt's slowness.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// ConstantTimeEqual compares two strings in constant time. Use this to
// compare token hashes so a timing side channel cannot leak whether a
// prefix matched.
func ConstantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// EncodeRememberCookie builds the cookie value: "<selector>:<raw_token>".
func EncodeRememberCookie(selector, rawToken string) string {
	return selector + rememberCookieSeparator + rawToken
}

// DecodeRememberCookie splits the cookie value. Returns ok=false on any
// malformed input (no separator, empty halves, more than one separator).
func DecodeRememberCookie(value string) (selector, rawToken string, ok bool) {
	parts := strings.Split(value, rememberCookieSeparator)
	if len(parts) != 2 {
		return "", "", false
	}
	if parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}
