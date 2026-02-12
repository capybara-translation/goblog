package config

import (
	"os"
	"testing"
)

func TestLoad_DefaultValues(t *testing.T) {
	// Clear environment variables
	clearEnv(t)

	cfg := Load()

	// Verify default values
	if cfg.Port != "8080" {
		t.Errorf("expected default Port to be %q, got %q", "8080", cfg.Port)
	}

	if cfg.SecureCookie != false {
		t.Errorf("expected default SecureCookie to be false, got %v", cfg.SecureCookie)
	}

	if cfg.DatabasePath != "data/goblog.db" {
		t.Errorf("expected default DatabasePath to be %q, got %q", "data/goblog.db", cfg.DatabasePath)
	}

	if cfg.BlogTitle != "goblog" {
		t.Errorf("expected default BlogTitle to be %q, got %q", "goblog", cfg.BlogTitle)
	}

	if cfg.PasswordPolicy != PasswordPolicyNone {
		t.Errorf("expected default PasswordPolicy to be %q, got %q", PasswordPolicyNone, cfg.PasswordPolicy)
	}

	if cfg.UploadDir != "data/uploads" {
		t.Errorf("expected default UploadDir to be %q, got %q", "data/uploads", cfg.UploadDir)
	}

	if cfg.MaxUploadSize != DefaultMaxUploadSize {
		t.Errorf("expected default MaxUploadSize to be %d, got %d", DefaultMaxUploadSize, cfg.MaxUploadSize)
	}
}

func TestLoad_WithEnvironmentVariables(t *testing.T) {
	// Clear environment variables
	clearEnv(t)

	// Set environment variables
	os.Setenv("PORT", "3000")
	os.Setenv("SECURE_COOKIE", "true")
	os.Setenv("PASSWORD_POLICY", "STRONG")
	os.Setenv("DATABASE_PATH", "/var/lib/goblog/production.db")
	os.Setenv("BLOG_TITLE", "My Awesome Blog")
	os.Setenv("UPLOAD_DIR", "/var/lib/goblog/uploads")
	os.Setenv("MAX_UPLOAD_SIZE", "10485760") // 10MB
	t.Cleanup(func() {
		os.Unsetenv("PORT")
		os.Unsetenv("SECURE_COOKIE")
		os.Unsetenv("PASSWORD_POLICY")
		os.Unsetenv("DATABASE_PATH")
		os.Unsetenv("BLOG_TITLE")
		os.Unsetenv("UPLOAD_DIR")
		os.Unsetenv("MAX_UPLOAD_SIZE")
	})

	cfg := Load()

	// Verify environment variable values
	if cfg.Port != "3000" {
		t.Errorf("expected Port to be %q, got %q", "3000", cfg.Port)
	}

	if cfg.SecureCookie != true {
		t.Errorf("expected SecureCookie to be true, got %v", cfg.SecureCookie)
	}

	if cfg.DatabasePath != "/var/lib/goblog/production.db" {
		t.Errorf("expected DatabasePath to be %q, got %q", "/var/lib/goblog/production.db", cfg.DatabasePath)
	}

	if cfg.BlogTitle != "My Awesome Blog" {
		t.Errorf("expected BlogTitle to be %q, got %q", "My Awesome Blog", cfg.BlogTitle)
	}

	if cfg.PasswordPolicy != PasswordPolicyStrong {
		t.Errorf("expected PasswordPolicy to be %q, got %q", PasswordPolicyStrong, cfg.PasswordPolicy)
	}

	if cfg.UploadDir != "/var/lib/goblog/uploads" {
		t.Errorf("expected UploadDir to be %q, got %q", "/var/lib/goblog/uploads", cfg.UploadDir)
	}

	if cfg.MaxUploadSize != 10485760 {
		t.Errorf("expected MaxUploadSize to be %d, got %d", 10485760, cfg.MaxUploadSize)
	}
}

func TestLoad_SecureCookieFalse(t *testing.T) {
	// Clear environment variables
	clearEnv(t)

	// Set SECURE_COOKIE=false
	os.Setenv("SECURE_COOKIE", "false")
	t.Cleanup(func() {
		os.Unsetenv("SECURE_COOKIE")
	})

	cfg := Load()

	if cfg.SecureCookie != false {
		t.Errorf("expected SecureCookie to be false, got %v", cfg.SecureCookie)
	}
}

func TestLoad_SecureCookieInvalidValue(t *testing.T) {
	// Clear environment variables
	clearEnv(t)

	// Set invalid value
	os.Setenv("SECURE_COOKIE", "invalid")
	t.Cleanup(func() {
		os.Unsetenv("SECURE_COOKIE")
	})

	cfg := Load()

	// Verify that invalid value defaults to false
	if cfg.SecureCookie != false {
		t.Errorf("expected SecureCookie to be false (default) for invalid value, got %v", cfg.SecureCookie)
	}
}

func TestLoad_PartialEnvironmentVariables(t *testing.T) {
	// Clear environment variables
	clearEnv(t)

	// Set only some environment variables
	os.Setenv("PORT", "8000")
	t.Cleanup(func() {
		os.Unsetenv("PORT")
	})

	cfg := Load()

	// Configured environment variable
	if cfg.Port != "8000" {
		t.Errorf("expected Port to be %q, got %q", "8000", cfg.Port)
	}

	// Unset environment variables use default values
	if cfg.SecureCookie != false {
		t.Errorf("expected default SecureCookie to be false, got %v", cfg.SecureCookie)
	}

	if cfg.DatabasePath != "data/goblog.db" {
		t.Errorf("expected default DatabasePath to be %q, got %q", "data/goblog.db", cfg.DatabasePath)
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{
			name:         "Environment variable is set",
			key:          "TEST_KEY",
			defaultValue: "default",
			envValue:     "custom",
			expected:     "custom",
		},
		{
			name:         "Environment variable is not set",
			key:          "TEST_KEY",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				t.Cleanup(func() {
					os.Unsetenv(tt.key)
				})
			}

			result := getEnv(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetEnvAsBool(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue bool
		envValue     string
		expected     bool
	}{
		{
			name:         "true",
			key:          "TEST_BOOL",
			defaultValue: false,
			envValue:     "true",
			expected:     true,
		},
		{
			name:         "false",
			key:          "TEST_BOOL",
			defaultValue: true,
			envValue:     "false",
			expected:     false,
		},
		{
			name:         "1 (true)",
			key:          "TEST_BOOL",
			defaultValue: false,
			envValue:     "1",
			expected:     true,
		},
		{
			name:         "0 (false)",
			key:          "TEST_BOOL",
			defaultValue: true,
			envValue:     "0",
			expected:     false,
		},
		{
			name:         "Environment variable is not set",
			key:          "TEST_BOOL",
			defaultValue: true,
			envValue:     "",
			expected:     true,
		},
		{
			name:         "Invalid value (returns default)",
			key:          "TEST_BOOL",
			defaultValue: false,
			envValue:     "invalid",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				t.Cleanup(func() {
					os.Unsetenv(tt.key)
				})
			} else {
				os.Unsetenv(tt.key)
			}

			result := getEnvAsBool(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGetPasswordPolicy(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue PasswordPolicy
		envValue     string
		expected     PasswordPolicy
	}{
		{
			name:         "NONE",
			key:          "TEST_PASSWORD_POLICY",
			defaultValue: PasswordPolicyStrong,
			envValue:     "NONE",
			expected:     PasswordPolicyNone,
		},
		{
			name:         "STRONG",
			key:          "TEST_PASSWORD_POLICY",
			defaultValue: PasswordPolicyNone,
			envValue:     "STRONG",
			expected:     PasswordPolicyStrong,
		},
		{
			name:         "Environment variable is not set",
			key:          "TEST_PASSWORD_POLICY",
			defaultValue: PasswordPolicyNone,
			envValue:     "",
			expected:     PasswordPolicyNone,
		},
		{
			name:         "Invalid value (returns default)",
			key:          "TEST_PASSWORD_POLICY",
			defaultValue: PasswordPolicyNone,
			envValue:     "INVALID",
			expected:     PasswordPolicyNone,
		},
		{
			name:         "Lowercase none",
			key:          "TEST_PASSWORD_POLICY",
			defaultValue: PasswordPolicyStrong,
			envValue:     "none",
			expected:     PasswordPolicyNone,
		},
		{
			name:         "Lowercase strong",
			key:          "TEST_PASSWORD_POLICY",
			defaultValue: PasswordPolicyNone,
			envValue:     "strong",
			expected:     PasswordPolicyStrong,
		},
		{
			name:         "Mixed case: None",
			key:          "TEST_PASSWORD_POLICY",
			defaultValue: PasswordPolicyStrong,
			envValue:     "None",
			expected:     PasswordPolicyNone,
		},
		{
			name:         "Mixed case: Strong",
			key:          "TEST_PASSWORD_POLICY",
			defaultValue: PasswordPolicyNone,
			envValue:     "Strong",
			expected:     PasswordPolicyStrong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				t.Cleanup(func() {
					os.Unsetenv(tt.key)
				})
			} else {
				os.Unsetenv(tt.key)
			}

			result := getPasswordPolicy(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetEnvAsInt64(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue int64
		envValue     string
		expected     int64
	}{
		{
			name:         "Valid number",
			key:          "TEST_INT64",
			defaultValue: 100,
			envValue:     "500",
			expected:     500,
		},
		{
			name:         "Environment variable is not set",
			key:          "TEST_INT64",
			defaultValue: 100,
			envValue:     "",
			expected:     100,
		},
		{
			name:         "Invalid value (returns default)",
			key:          "TEST_INT64",
			defaultValue: 100,
			envValue:     "invalid",
			expected:     100,
		},
		{
			name:         "Large number",
			key:          "TEST_INT64",
			defaultValue: 0,
			envValue:     "10485760", // 10MB
			expected:     10485760,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				t.Cleanup(func() {
					os.Unsetenv(tt.key)
				})
			} else {
				os.Unsetenv(tt.key)
			}

			result := getEnvAsInt64(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

// clearEnv clears environment variables for testing
func clearEnv(t *testing.T) {
	t.Helper()
	os.Unsetenv("PORT")
	os.Unsetenv("SECURE_COOKIE")
	os.Unsetenv("PASSWORD_POLICY")
	os.Unsetenv("DATABASE_PATH")
	os.Unsetenv("BLOG_TITLE")
	os.Unsetenv("UPLOAD_DIR")
	os.Unsetenv("MAX_UPLOAD_SIZE")
}
