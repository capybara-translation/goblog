package db

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// Open はSQLiteデータベースへの接続を開きます
func Open(dbPath string) (*sqlx.DB, error) {
	// データベースファイルのディレクトリが存在することを確認
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// データベースを開く
	db, err := sqlx.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 接続確認
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// RunMigrations は埋め込まれたマイグレーションを実行します
func RunMigrations(db *sqlx.DB, migrationsFS embed.FS, migrationPaths ...string) error {
	for _, migrationPath := range migrationPaths {
		// 埋め込まれたマイグレーションファイルを読み込む
		content, err := migrationsFS.ReadFile(migrationPath)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", migrationPath, err)
		}

		// マイグレーションを実行
		if _, err := db.Exec(string(content)); err != nil {
			// SQLiteはALTER TABLE ADD COLUMN IF NOT EXISTSをサポートしていないため、
			// 重複カラムエラーは無視する（マイグレーションが既に適用済みの場合）
			if isDuplicateColumnError(err) {
				continue
			}
			return fmt.Errorf("failed to execute migration %s: %w", migrationPath, err)
		}
	}

	return nil
}

// isDuplicateColumnError は重複カラムエラーかどうかを判定します
func isDuplicateColumnError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "duplicate column name")
}
