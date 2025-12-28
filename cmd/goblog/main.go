package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/capybara-translation/goblog/internal/db"
	gobloghttp "github.com/capybara-translation/goblog/internal/http"
	"github.com/capybara-translation/goblog/internal/repo"
	"github.com/capybara-translation/goblog/internal/service"
)

func main() {
	// データベースの初期化
	database, err := db.Open("data/goblog.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	// マイグレーションの実行
	if err := db.RunMigrations(database, "migrations/001_init.sql"); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	fmt.Println("Database initialized successfully")

	// Repository層の初期化
	postRepo := repo.NewPostRepository(database)

	// Service層の初期化
	postService := service.NewPostService(postRepo)

	// ルーターの初期化
	r := gobloghttp.NewRouter(postService)

	// サーバー起動
	port := ":8080"
	fmt.Printf("Server starting on http://localhost%s\n", port)
	if err := http.ListenAndServe(port, r); err != nil {
		log.Fatal(err)
	}
}