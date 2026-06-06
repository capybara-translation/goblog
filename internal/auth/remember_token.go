package auth

import (
	"crypto/rand"
	"encoding/base64"
	"log"
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
	// RefreshOnUse atomically updates both the last_used_at timestamp and the
	// expires_at deadline. Called whenever a remember-me token is used to
	// successfully restore a session; the deadline extension gives the
	// listing "sliding expiration" semantics (an active user is not forced
	// to re-authenticate every TTL window).
	RefreshOnUse(selector string, lastUsed time.Time, newExpiresAt time.Time) error
	CleanupExpired() error
}

// rememberCookieSeparator separates selector and raw token in the cookie value.
const rememberCookieSeparator = ":"

// GenerateSelector returns 16 cryptographically random bytes, base64-url
// encoded. The selector is the public lookup key stored in plain text.
func GenerateSelector() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand draws from the OS CSPRNG and does not fail in normal
		// operation; a failure means the platform's randomness source is
		// broken, which is unrecoverable for a security token. We panic to
		// keep the cookie-codec call sites simple rather than thread an error
		// through these pure helpers. (Note: generateSessionID instead returns
		// an error — these remember-token helpers deliberately differ.)
		panic("auth: rand.Read failed for selector: " + err.Error())
	}
	return base64.URLEncoding.EncodeToString(b)
}

// GenerateRawToken, HashToken, and ConstantTimeEqual — the general token
// primitives the remember-me scheme also relies on — live in token.go so they
// can be shared without implying remember-me specificity.

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

// StartRememberTokenCleanupLoop runs CleanupExpired() on the given store
// periodically. The returned channel can be closed to stop the loop (useful
// in tests; production callers may simply leak it on shutdown).
//
// The function takes the interface, so it lives in the auth package alongside
// the interface definition rather than next to any specific implementation.
func StartRememberTokenCleanupLoop(store RememberTokenStore, interval time.Duration) chan<- struct{} {
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := store.CleanupExpired(); err != nil {
					// Log so a persistently failing sweep (e.g. DB
					// unavailable) leaves an operational signal instead of
					// silently letting expired rows accumulate. Matches the
					// OGP cache cleanup loop's behavior.
					log.Printf("remember-token cleanup failed: %v", err)
				}
			case <-stop:
				return
			}
		}
	}()
	return stop
}
