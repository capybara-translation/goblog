package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"

	"github.com/capybara-translation/goblog"
	"github.com/capybara-translation/goblog/internal/auth"
	"github.com/capybara-translation/goblog/internal/config"
	"github.com/capybara-translation/goblog/internal/db"
	"github.com/capybara-translation/goblog/internal/repo"
	"github.com/capybara-translation/goblog/internal/service"
	"github.com/joho/godotenv"
	"golang.org/x/term"
)

func main() {
	// Load environment variables from .env file (skip if not found)
	if err := godotenv.Load(); err != nil {
		// Log if .env file doesn't exist, but don't treat it as an error
		log.Printf(".env file not found or could not be loaded: %v", err)
	}

	// Define flags
	username := flag.String("username", "", "Username (required)")
	password := flag.String("password", "", "Password (will prompt if omitted)")
	help := flag.Bool("help", false, "Show help")
	flag.BoolVar(help, "h", false, "Show help (shorthand)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: adduser [options]\n\n")
		fmt.Fprintf(os.Stderr, "Create an admin user for goblog.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  adduser -username admin                    # Enter password at prompt\n")
		fmt.Fprintf(os.Stderr, "  adduser -username admin -password secret   # Specify password (for scripts)\n")
		fmt.Fprintf(os.Stderr, "\nEnvironment variables (can also be loaded from .env file):\n")
		fmt.Fprintf(os.Stderr, "  DATABASE_PATH    - Path to database file (default: data/goblog.db)\n")
		fmt.Fprintf(os.Stderr, "  PASSWORD_POLICY  - Password policy (NONE or STRONG)\n")
	}

	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	// Get username (prompt if flag is empty)
	if *username == "" {
		fmt.Print("Username: ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to read username: %v\n", err)
			os.Exit(1)
		}
		*username = strings.TrimSpace(input)
	}

	if *username == "" {
		fmt.Fprintf(os.Stderr, "Error: Username is required\n")
		os.Exit(1)
	}

	// Get password (prompt if flag is empty)
	if *password == "" {
		fmt.Print("Password: ")
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println() // newline
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to read password: %v\n", err)
			os.Exit(1)
		}
		*password = string(bytePassword)

		// Confirm password
		fmt.Print("Password (confirm): ")
		bytePasswordConfirm, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println() // newline
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to read password: %v\n", err)
			os.Exit(1)
		}

		if *password != string(bytePasswordConfirm) {
			fmt.Fprintf(os.Stderr, "Error: Passwords do not match\n")
			os.Exit(1)
		}
	}

	if *password == "" {
		fmt.Fprintf(os.Stderr, "Error: Password is required\n")
		os.Exit(1)
	}

	// Load configuration
	cfg := config.Load()

	// Initialize database
	database, err := db.Open(cfg.DatabasePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	if err := goblog.InitSchema(database); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to initialize schema: %v\n", err)
		os.Exit(1)
	}

	// Initialize Repository and Service
	userRepo := repo.NewUserRepository(database)
	sessionStore := auth.NewInMemorySessionStore()
	authService := service.NewAuthService(userRepo, sessionStore, cfg.PasswordPolicy, cfg.SessionTTL, nil, cfg.RememberTTL)

	// Create user
	user, err := authService.CreateUser(*username, *password)
	if err != nil {
		if errors.Is(err, service.ErrUsernameAlreadyExists) {
			fmt.Fprintf(os.Stderr, "Error: Username '%s' is already taken\n", *username)
		} else if errors.Is(err, service.ErrWeakPassword) {
			fmt.Fprintf(os.Stderr, "Error: Password is too weak\n")
			fmt.Fprintf(os.Stderr, "Password policy (STRONG) requirements:\n")
			fmt.Fprintf(os.Stderr, "  - At least 15 characters\n")
			fmt.Fprintf(os.Stderr, "  - Contains uppercase letter\n")
			fmt.Fprintf(os.Stderr, "  - Contains lowercase letter\n")
			fmt.Fprintf(os.Stderr, "  - Contains digit\n")
			fmt.Fprintf(os.Stderr, "  - Contains symbol\n")
		} else {
			fmt.Fprintf(os.Stderr, "Error: Failed to create user: %v\n", err)
		}
		os.Exit(1)
	}

	fmt.Printf("User created: username=%s, id=%d\n", user.Username, user.ID)
}
