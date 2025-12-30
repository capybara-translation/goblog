package config

import (
	"os"
	"strconv"
	"strings"
)

// PasswordPolicy はパスワードポリシーの種類を表します
type PasswordPolicy string

const (
	// PasswordPolicyNone は制限なしのパスワードポリシーです（開発/テスト環境用）
	PasswordPolicyNone PasswordPolicy = "NONE"
	// PasswordPolicyStrong は厳格なパスワードポリシーです（本番環境用）
	PasswordPolicyStrong PasswordPolicy = "STRONG"
)

// Config はアプリケーション設定を保持します
type Config struct {
	// Server settings
	Port string

	// Security settings
	SecureCookie   bool           // HTTPSを使用する場合にtrue
	PasswordPolicy PasswordPolicy // パスワードポリシー

	// Database settings
	DatabasePath string

	// Site settings
	BlogTitle string // ブログのタイトル
}

// Load は環境変数から設定を読み込みます
func Load() *Config {
	return &Config{
		Port:           getEnv("PORT", "8080"),
		SecureCookie:   getEnvAsBool("SECURE_COOKIE", false), // デフォルト: false（開発環境用）
		PasswordPolicy: getPasswordPolicy("PASSWORD_POLICY", PasswordPolicyNone),
		DatabasePath:   getEnv("DATABASE_PATH", "data/goblog.db"),
		BlogTitle:      getEnv("BLOG_TITLE", "goblog"),
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

// getPasswordPolicy は環境変数からパスワードポリシーを取得します
// 大文字小文字を区別しません（none/NONE/None、strong/STRONG/Strong すべて有効）
func getPasswordPolicy(key string, defaultValue PasswordPolicy) PasswordPolicy {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultValue
	}
	// 大文字小文字を区別しないように大文字に変換
	policy := PasswordPolicy(strings.ToUpper(valStr))
	if policy == PasswordPolicyNone || policy == PasswordPolicyStrong {
		return policy
	}
	// 不正な値の場合はデフォルト値を返す
	return defaultValue
}
