package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
)

// This file holds general-purpose, feature-agnostic token primitives:
// generating an opaque random token, hashing it for at-rest storage, and
// comparing hashes in constant time. They are shared by multiple features
// (remember-me login tokens, anonymous reaction visitor keys, ...). Feature-
// specific token logic (e.g. the remember-me selector + cookie codec) lives in
// its own file.

// GenerateRawToken returns 32 cryptographically random bytes, base64-url
// encoded. Use it as an opaque secret token (e.g. a remember-me raw token or an
// anonymous visitor identifier); only its hash should be persisted — see
// HashToken.
func GenerateRawToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand draws from the OS CSPRNG and does not fail in normal
		// operation; a failure means the platform's randomness source is
		// broken, which is unrecoverable for a security token. We panic to keep
		// call sites simple rather than thread an error through this pure helper.
		panic("auth: rand.Read failed for raw token: " + err.Error())
	}
	return base64.URLEncoding.EncodeToString(b)
}

// HashToken hashes an opaque token with SHA-256 and returns its hex encoding.
// SHA-256 is sufficient when the input is high-entropy (e.g. a 32-byte random
// token from GenerateRawToken): a brute-force preimage attack is infeasible
// even without bcrypt's slowness. Do NOT use this for low-entropy secrets such
// as user passwords — use bcrypt for those.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// ConstantTimeEqual reports whether a and b are equal, comparing in constant
// time for equal-length inputs. crypto/subtle.ConstantTimeCompare returns
// immediately when the lengths differ, so the comparison is only constant-time
// when a and b have the same length. That holds for the intended use here: both
// operands are HashToken outputs (fixed-length hex-encoded SHA-256), so the
// length is constant and not secret, and the prefix-match timing channel is
// what we need to close.
func ConstantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
