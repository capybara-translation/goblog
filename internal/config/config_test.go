package config

import (
	"os"
	"testing"
)

func TestLoad_DefaultValues(t *testing.T) {
	// 環境変数をクリア
	clearEnv(t)

	cfg := Load()

	// デフォルト値を確認
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
}

func TestLoad_WithEnvironmentVariables(t *testing.T) {
	// 環境変数をクリア
	clearEnv(t)

	// 環境変数を設定
	os.Setenv("PORT", "3000")
	os.Setenv("SECURE_COOKIE", "true")
	os.Setenv("DATABASE_PATH", "/var/lib/goblog/production.db")
	os.Setenv("BLOG_TITLE", "My Awesome Blog")
	t.Cleanup(func() {
		os.Unsetenv("PORT")
		os.Unsetenv("SECURE_COOKIE")
		os.Unsetenv("DATABASE_PATH")
		os.Unsetenv("BLOG_TITLE")
	})

	cfg := Load()

	// 環境変数の値を確認
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
}

func TestLoad_SecureCookieFalse(t *testing.T) {
	// 環境変数をクリア
	clearEnv(t)

	// SECURE_COOKIE=false を設定
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
	// 環境変数をクリア
	clearEnv(t)

	// 無効な値を設定
	os.Setenv("SECURE_COOKIE", "invalid")
	t.Cleanup(func() {
		os.Unsetenv("SECURE_COOKIE")
	})

	cfg := Load()

	// 無効な値の場合はデフォルト値（false）になることを確認
	if cfg.SecureCookie != false {
		t.Errorf("expected SecureCookie to be false (default) for invalid value, got %v", cfg.SecureCookie)
	}
}

func TestLoad_PartialEnvironmentVariables(t *testing.T) {
	// 環境変数をクリア
	clearEnv(t)

	// 一部の環境変数のみ設定
	os.Setenv("PORT", "8000")
	t.Cleanup(func() {
		os.Unsetenv("PORT")
	})

	cfg := Load()

	// 設定された環境変数
	if cfg.Port != "8000" {
		t.Errorf("expected Port to be %q, got %q", "8000", cfg.Port)
	}

	// 未設定の環境変数はデフォルト値
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
			name:         "環境変数が設定されている",
			key:          "TEST_KEY",
			defaultValue: "default",
			envValue:     "custom",
			expected:     "custom",
		},
		{
			name:         "環境変数が設定されていない",
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
			name:         "環境変数が設定されていない",
			key:          "TEST_BOOL",
			defaultValue: true,
			envValue:     "",
			expected:     true,
		},
		{
			name:         "無効な値（デフォルトを返す）",
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

// clearEnv はテスト用の環境変数をクリアします
func clearEnv(t *testing.T) {
	t.Helper()
	os.Unsetenv("PORT")
	os.Unsetenv("SECURE_COOKIE")
	os.Unsetenv("DATABASE_PATH")
	os.Unsetenv("BLOG_TITLE")
}
