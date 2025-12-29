package config

import (
	"os"
	"strconv"
)

// Config はアプリケーション設定を保持します
type Config struct {
	// Server settings
	Port string

	// Security settings
	SecureCookie bool // HTTPSを使用する場合にtrue

	// Database settings
	DatabasePath string

	// Site settings
	BlogTitle string // ブログのタイトル
}

// Load は環境変数から設定を読み込みます
func Load() *Config {
	return &Config{
		Port:         getEnv("PORT", "8080"),
		SecureCookie: getEnvAsBool("SECURE_COOKIE", false), // デフォルト: false（開発環境用）
		DatabasePath: getEnv("DATABASE_PATH", "data/goblog.db"),
		BlogTitle:    getEnv("BLOG_TITLE", "goblog"),
	}
}

// getEnv は環境変数を取得し、存在しない場合はデフォルト値を返します
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsBool は環境変数をboolとして取得します
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
