package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/capybara-translation/goblog/internal/auth"
	"github.com/capybara-translation/goblog/internal/config"
	"github.com/capybara-translation/goblog/internal/db"
	gobloghttp "github.com/capybara-translation/goblog/internal/http"
	"github.com/capybara-translation/goblog/internal/repo"
	"github.com/capybara-translation/goblog/internal/service"
	"github.com/joho/godotenv"
)

func main() {
	// .envファイルから環境変数を読み込む（存在しない場合はスキップ）
	if err := godotenv.Load(); err != nil {
		// .envファイルが存在しない場合はログに記録するが、エラーにはしない
		log.Printf(".env file not found or could not be loaded: %v", err)
	}

	// 設定の読み込み
	cfg := config.Load()

	// データベースの初期化
	database, err := db.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	// マイグレーションの実行
	if err := db.RunMigrations(database, "migrations/001_create_posts.sql", "migrations/002_create_users.sql", "migrations/003_add_is_pinned.sql"); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	fmt.Println("Database initialized successfully")

	// Repository層の初期化
	postRepo := repo.NewPostRepository(database)
	userRepo := repo.NewUserRepository(database)

	// SessionStoreの初期化
	sessionStore := auth.NewInMemorySessionStore()

	// Service層の初期化
	postService := service.NewPostService(postRepo)
	authService := service.NewAuthService(userRepo, sessionStore, cfg.PasswordPolicy)

	// アップロードディレクトリの作成
	if err := os.MkdirAll(cfg.UploadDir, 0755); err != nil {
		log.Fatalf("Failed to create upload directory: %v", err)
	}

	// ルーターの初期化
	r := gobloghttp.NewRouter(postService, authService, cfg.SecureCookie, cfg.BlogTitle, cfg.BaseURL, cfg.UploadDir, cfg.MaxUploadSize)

	// サーバー起動
	port := ":" + cfg.Port
	fmt.Printf("Server starting on http://localhost%s\n", port)
	fmt.Printf("Secure Cookie: %v\n", cfg.SecureCookie)
	if err := http.ListenAndServe(port, r); err != nil {
		log.Fatal(err)
	}
}
