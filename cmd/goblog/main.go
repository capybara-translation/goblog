package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/capybara-translation/goblog"
	"github.com/capybara-translation/goblog/internal/auth"
	"github.com/capybara-translation/goblog/internal/config"
	"github.com/capybara-translation/goblog/internal/db"
	gobloghttp "github.com/capybara-translation/goblog/internal/http"
	"github.com/capybara-translation/goblog/internal/repo"
	"github.com/capybara-translation/goblog/internal/service"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file (skip if not found)
	if err := godotenv.Load(); err != nil {
		// Log if .env file doesn't exist, but don't treat it as an error
		log.Printf(".env file not found or could not be loaded: %v", err)
	}

	// Load configuration
	cfg := config.Load()

	// Initialize database
	database, err := db.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	// Run migrations (from embedded files)
	if err := db.RunMigrations(database, goblog.Migrations, "migrations/001_create_posts.sql", "migrations/002_create_users.sql", "migrations/003_add_is_pinned.sql"); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	fmt.Println("Database initialized successfully")

	// Initialize repository layer
	postRepo := repo.NewPostRepository(database)
	userRepo := repo.NewUserRepository(database)

	// Initialize SessionStore
	sessionStore := auth.NewInMemorySessionStore()

	// Initialize service layer
	postService := service.NewPostService(postRepo)
	authService := service.NewAuthService(userRepo, sessionStore, cfg.PasswordPolicy)

	// Create upload directory
	if err := os.MkdirAll(cfg.UploadDir, 0755); err != nil {
		log.Fatalf("Failed to create upload directory: %v", err)
	}

	// Initialize router (using embedded resources)
	r := gobloghttp.NewRouter(postService, authService, cfg.SecureCookie, cfg.BlogTitle, cfg.BaseURL, cfg.UploadDir, cfg.MaxUploadSize, goblog.Templates, goblog.StaticFiles)

	// Start server
	port := ":" + cfg.Port
	fmt.Printf("Server starting on http://localhost%s\n", port)
	fmt.Printf("Secure Cookie: %v\n", cfg.SecureCookie)
	if err := http.ListenAndServe(port, r); err != nil {
		log.Fatal(err)
	}
}
