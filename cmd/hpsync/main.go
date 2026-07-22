// Command hpsync pulls daily weight / blood-pressure measurements from the
// Tanita Health Planet API into the goblog database.
//
//	hpsync run   — refresh the token, fetch the last 30 days, upsert
//	               (invoked daily by goblog-hpsync.timer; no-op with exit 0
//	               when HEALTHPLANET_ENABLED is false)
//	hpsync auth  — fallback interactive OAuth authorization over SSH using
//	               Tanita's success.html redirect. Normal authorization
//	               happens in the admin panel (/admin/healthplanet).
package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/capybara-translation/goblog"
	"github.com/capybara-translation/goblog/internal/config"
	"github.com/capybara-translation/goblog/internal/db"
	"github.com/capybara-translation/goblog/internal/healthplanet"
	"github.com/capybara-translation/goblog/internal/repo"
	"github.com/capybara-translation/goblog/internal/service"
	"github.com/joho/godotenv"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: hpsync <run|auth>\n\n")
	fmt.Fprintf(os.Stderr, "Sync weight and blood pressure from Health Planet into goblog's DB.\n\n")
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  run    Fetch the last 30 days and upsert into the DB (for the daily timer)\n")
	fmt.Fprintf(os.Stderr, "  auth   Fallback interactive authorization (normally use the admin panel)\n")
	fmt.Fprintf(os.Stderr, "\nEnvironment variables (can also be loaded from .env file):\n")
	fmt.Fprintf(os.Stderr, "  HEALTHPLANET_ENABLED        - Feature flag (default: false; run exits 0 when off)\n")
	fmt.Fprintf(os.Stderr, "  HEALTHPLANET_CLIENT_ID      - OAuth client ID\n")
	fmt.Fprintf(os.Stderr, "  HEALTHPLANET_CLIENT_SECRET  - OAuth client secret\n")
	fmt.Fprintf(os.Stderr, "  DATABASE_PATH               - Path to database file (default: data/goblog.db)\n")
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf(".env file not found or could not be loaded: %v", err)
	}

	if len(os.Args) != 2 {
		usage()
		os.Exit(2)
	}
	cmd := os.Args[1]
	if cmd != "auth" && cmd != "run" {
		usage()
		os.Exit(2)
	}

	cfg := config.Load()
	if !cfg.HealthPlanetEnabled {
		if cmd == "run" {
			// The timer is installed unconditionally; a disabled feature is
			// the normal quiet state, not an error.
			fmt.Println("healthplanet integration is disabled (HEALTHPLANET_ENABLED != true), skipping")
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, "Error: set HEALTHPLANET_ENABLED=true before authorizing")
		os.Exit(1)
	}
	if cfg.HealthPlanetClientID == "" || cfg.HealthPlanetClientSecret == "" {
		fmt.Fprintln(os.Stderr, "Error: HEALTHPLANET_CLIENT_ID and HEALTHPLANET_CLIENT_SECRET must be set")
		os.Exit(1)
	}

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

	tokenRepo := repo.NewHealthPlanetTokenRepository(database)

	switch cmd {
	case "auth":
		// CLI fallback uses Tanita's own success page: the operator copies
		// the code off the address bar. Must NOT use the blog redirect here
		// because the code would go to the SPA instead of this prompt.
		client := healthplanet.NewClient(healthplanet.DefaultBaseURL,
			cfg.HealthPlanetClientID, cfg.HealthPlanetClientSecret, healthplanet.SuccessRedirectURI)
		runAuth(client, tokenRepo)
	case "run":
		// redirectURI is unused by refresh/fetch but keep it consistent.
		client := healthplanet.NewClient(healthplanet.DefaultBaseURL,
			cfg.HealthPlanetClientID, cfg.HealthPlanetClientSecret, healthplanet.SuccessRedirectURI)
		runSync(client, tokenRepo, repo.NewHealthRecordRepository(database))
	}
}

func runAuth(client *healthplanet.Client, tokenRepo repo.HealthPlanetTokenRepository) {
	fmt.Println("Open this URL in a browser on your own machine, log in and approve access:")
	fmt.Println()
	fmt.Println("  " + client.AuthCodeURL())
	fmt.Println()
	fmt.Println("After approving you land on success.html; copy the value of the `code=`")
	fmt.Println("query parameter from the address bar. Codes expire after 10 minutes.")
	fmt.Println()
	fmt.Print("Code: ")

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to read code: %v\n", err)
		os.Exit(1)
	}
	code := strings.TrimSpace(line)
	if code == "" {
		fmt.Fprintln(os.Stderr, "Error: Code is required")
		os.Exit(1)
	}

	tok, err := client.ExchangeCode(code)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Token exchange failed: %v\n", err)
		os.Exit(1)
	}
	expiresAt := time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	if err := tokenRepo.Save(tok.AccessToken, tok.RefreshToken, expiresAt); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to store token: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Authorized. Token stored, expires %s.\n", expiresAt.Format(time.RFC3339))
	fmt.Println("The daily `hpsync run` will keep it refreshed from here on.")
}

func runSync(client *healthplanet.Client, tokenRepo repo.HealthPlanetTokenRepository, recordRepo repo.HealthRecordRepository) {
	svc := service.NewHealthSyncService(client, tokenRepo, recordRepo)
	err := svc.Sync()
	switch {
	case err == nil:
		fmt.Println("Sync complete.")
	case errors.Is(err, service.ErrHealthPlanetNoToken):
		fmt.Fprintln(os.Stderr, "Error: No token stored yet. Authorize from the admin panel (/admin/healthplanet) or run 'hpsync auth'.")
		os.Exit(1)
	default:
		// Includes ErrHealthPlanetReauthRequired, ErrHealthPlanetTokenExpiringSoon
		// and fetch failures: the non-zero exit is what trips the
		// dead-man's-switch monitoring.
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
