package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/capybara-translation/goblog/internal/auth"
	"github.com/capybara-translation/goblog/internal/config"
	"github.com/capybara-translation/goblog/internal/db"
	gobloghttp "github.com/capybara-translation/goblog/internal/http"
	"github.com/capybara-translation/goblog/internal/repo"
	"github.com/capybara-translation/goblog/internal/service"
)

func main() {
	// 設定の読み込み
	cfg := config.Load()

	// データベースの初期化
	database, err := db.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	// マイグレーションの実行
	if err := db.RunMigrations(database, "migrations/001_init.sql", "migrations/002_create_users.sql"); err != nil {
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
	authService := service.NewAuthService(userRepo, sessionStore)

	// ルーターの初期化
	r := gobloghttp.NewRouter(postService, authService, cfg.SecureCookie)

	// サーバー起動
	port := ":" + cfg.Port
	fmt.Printf("Server starting on http://localhost%s\n", port)
	fmt.Printf("Secure Cookie: %v\n", cfg.SecureCookie)
	if err := http.ListenAndServe(port, r); err != nil {
		log.Fatal(err)
	}
}
