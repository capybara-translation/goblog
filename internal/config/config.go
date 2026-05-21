package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// PasswordPolicy represents the type of password policy
type PasswordPolicy string

const (
	// PasswordPolicyNone is the unrestricted password policy (for development/test environments)
	PasswordPolicyNone PasswordPolicy = "NONE"
	// PasswordPolicyStrong is the strict password policy (for production environments)
	PasswordPolicyStrong PasswordPolicy = "STRONG"
)

// Config holds application settings
type Config struct {
	// Server settings
	Port string

	// Security settings
	SecureCookie   bool           // true when using HTTPS
	PasswordPolicy PasswordPolicy // Password policy
	TrustedProxies []string       // Trusted proxy IPs/CIDRs for X-Forwarded-For

	// Database settings
	DatabasePath string

	// Site settings
	BlogTitle string // Blog title
	BaseURL   string // Site base URL (e.g., https://example.com)

	// Upload settings
	UploadDir     string // Directory to save uploaded files
	MaxUploadSize int64  // Maximum upload file size (bytes)

	// Pagination settings
	PostsPerPage int // Number of posts per page on the home and tag listings

	// Session settings
	SessionTTL time.Duration // How long an authenticated session remains valid
}

// Default maximum upload size (5MB)
const DefaultMaxUploadSize int64 = 5 * 1024 * 1024

// Pagination defaults and bounds
const (
	DefaultPostsPerPage = 20
	MaxPostsPerPage     = 100
)

// Session defaults and bounds
const (
	DefaultSessionTTL = 24 * time.Hour
	MinSessionTTL     = time.Minute // Below this the value is almost certainly a mistake
)

// Load loads configuration from environment variables
func Load() *Config {
	port := getEnv("PORT", "8080")
	baseURL := getEnv("BASE_URL", "")
	if baseURL == "" {
		baseURL = "http://localhost:" + port
	}
	// Remove trailing slash
	baseURL = strings.TrimSuffix(baseURL, "/")

	return &Config{
		Port:           port,
		SecureCookie:   getEnvAsBool("SECURE_COOKIE", false), // Default: false (for development)
		PasswordPolicy: getPasswordPolicy("PASSWORD_POLICY", PasswordPolicyNone),
		TrustedProxies: getEnvAsStringSlice("TRUSTED_PROXIES"),
		DatabasePath:   getEnv("DATABASE_PATH", "data/goblog.db"),
		BlogTitle:      getEnv("BLOG_TITLE", "goblog"),
		BaseURL:        baseURL,
		UploadDir:      getEnv("UPLOAD_DIR", "data/uploads"),
		MaxUploadSize:  getEnvAsInt64("MAX_UPLOAD_SIZE", DefaultMaxUploadSize),
		PostsPerPage:   getEnvAsIntInRange("POSTS_PER_PAGE", DefaultPostsPerPage, 1, MaxPostsPerPage),
		SessionTTL:     getEnvAsDurationAtLeast("SESSION_TTL", DefaultSessionTTL, MinSessionTTL),
	}
}

// getEnvAsDurationAtLeast retrieves an env var as a time.Duration parsed with
// time.ParseDuration. Falls back to defaultValue when unset, unparseable, or
// below the minimum.
//
// Two distinct fallback paths matter:
//   - time.ParseDuration requires a unit suffix, so unit-less values such as
//     "1" or "24" return an error and take the unparseable path.
//   - Parseable but very small durations (e.g. "500ms", "30s") are caught by
//     the minimum check, which exists as a sanity floor against typos that
//     would otherwise lock the admin out almost immediately.
func getEnvAsDurationAtLeast(key string, defaultValue, minDuration time.Duration) time.Duration {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultValue
	}
	d, err := time.ParseDuration(valStr)
	if err != nil {
		return defaultValue
	}
	if d < minDuration {
		return defaultValue
	}
	return d
}

// getEnvAsIntInRange retrieves an env var as int, falling back to defaultValue
// when unset, unparseable, or outside [lo, hi] (inclusive).
func getEnvAsIntInRange(key string, defaultValue, lo, hi int) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultValue
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultValue
	}
	if val < lo || val > hi {
		return defaultValue
	}
	return val
}

// getEnv retrieves an environment variable or returns the default value if not set
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsBool retrieves an environment variable as a bool
func getEnvAsBool(key string, defaultValue bool) bool {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultValue
	}
	val, err := strconv.ParseBool(valStr)
	if err != nil {
		return defaultValue
	}
	return val
}

// getEnvAsInt64 retrieves an environment variable as an int64
func getEnvAsInt64(key string, defaultValue int64) int64 {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultValue
	}
	val, err := strconv.ParseInt(valStr, 10, 64)
	if err != nil {
		return defaultValue
	}
	return val
}

// getEnvAsStringSlice retrieves an environment variable as a string slice (comma-separated)
func getEnvAsStringSlice(key string) []string {
	valStr := os.Getenv(key)
	if valStr == "" {
		return nil
	}
	parts := strings.Split(valStr, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// getPasswordPolicy retrieves the password policy from environment variable
// Case-insensitive (none/NONE/None, strong/STRONG/Strong are all valid)
func getPasswordPolicy(key string, defaultValue PasswordPolicy) PasswordPolicy {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultValue
	}
	// Convert to uppercase for case-insensitive comparison
	policy := PasswordPolicy(strings.ToUpper(valStr))
	if policy == PasswordPolicyNone || policy == PasswordPolicyStrong {
		return policy
	}
	// Return default value for invalid values
	return defaultValue
}
