package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/capybara-translation/goblog"
	"github.com/capybara-translation/goblog/internal/auth"
	"github.com/capybara-translation/goblog/internal/config"
	"github.com/capybara-translation/goblog/internal/db"
	gobloghttp "github.com/capybara-translation/goblog/internal/http"
	"github.com/capybara-translation/goblog/internal/ogp"
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

	if err := goblog.InitSchema(database); err != nil {
		log.Fatalf("Failed to initialize schema: %v", err)
	}

	fmt.Println("Database initialized successfully")

	// Initialize repository layer
	postRepo := repo.NewPostRepository(database)
	postViewRepo := repo.NewPostViewRepository(database)
	reactionRepo := repo.NewReactionRepository(database)
	userRepo := repo.NewUserRepository(database)
	ogpRepo := repo.NewOGPRepository(database)

	// Initialize SessionStore
	sessionStore := auth.NewInMemorySessionStore()

	// Remember-me tokens live in SQLite so they survive restarts (unlike
	// sessions). Sweep expired rows hourly.
	rememberStore := repo.NewSQLiteRememberTokenStore(database)
	auth.StartRememberTokenCleanupLoop(rememberStore, time.Hour)

	// Initialize service layer
	postService := service.NewPostService(postRepo)
	postViewService := service.NewPostViewService(postViewRepo)
	reactionService := service.NewReactionService(postService, reactionRepo)
	authService := service.NewAuthService(userRepo, sessionStore, cfg.PasswordPolicy, cfg.SessionTTL, rememberStore, cfg.RememberTTL)

	// Initialize OGP service for link cards
	ogpFetcher := ogp.NewFetcher(ogp.FetchTimeout)
	ogpService := service.NewOGPService(ogpRepo, ogpFetcher, cfg.UploadDir)

	// Create upload directory
	if err := os.MkdirAll(cfg.UploadDir, 0755); err != nil {
		log.Fatalf("Failed to create upload directory: %v", err)
	}

	// Initialize router (using embedded resources)
	r := gobloghttp.NewRouter(postService, postViewService, authService, ogpService, reactionService, cfg.SecureCookie, cfg.TrustedProxies, cfg.BlogTitle, cfg.BaseURL, cfg.UploadDir, cfg.MaxUploadSize, cfg.PostsPerPage, goblog.Templates, goblog.StaticFiles)

	// Start server
	port := ":" + cfg.Port
	fmt.Printf("Server starting on http://localhost%s\n", port)
	fmt.Printf("Secure Cookie: %v\n", cfg.SecureCookie)
	if err := http.ListenAndServe(port, r); err != nil {
		log.Fatal(err)
	}
}
