// Command openlogs is a self-hosted, framework-agnostic log management server.
//
// Usage:
//
//	openlogs serve                  Run the HTTP server (default).
//	openlogs create-user <email>    Create a web UI user interactively.
//
// The web assets are embedded at compile time via the sibling web package, so the
// binary is fully self-contained.
package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/jmstewart1127/openlogs/internal/broker"
	"github.com/jmstewart1127/openlogs/internal/config"
	"github.com/jmstewart1127/openlogs/internal/db"
	"github.com/jmstewart1127/openlogs/internal/handler"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
)

func main() {
	cmd := "serve"
	args := os.Args[1:]
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd = args[0]
		args = args[1:]
	}

	switch cmd {
	case "serve":
		if err := runServe(); err != nil {
			log.Fatalf("openlogs: %v", err)
		}
	case "create-user":
		if err := runCreateUser(args); err != nil {
			log.Fatalf("openlogs: %v", err)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\nUsage:\n  openlogs serve\n  openlogs create-user <email>\n", cmd)
		os.Exit(2)
	}
}

// runServe loads configuration, opens the database, and runs the HTTP server.
func runServe() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	database, err := db.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer database.Close()

	if err := bootstrapAdmin(database, cfg.AdminEmail, cfg.AdminPassword); err != nil {
		return err
	}

	b := broker.New()
	app, err := handler.New(database, b, cfg)
	if err != nil {
		return err
	}

	startRetention(database, cfg.RetentionDays)

	mux, err := app.Router()
	if err != nil {
		return fmt.Errorf("build router: %w", err)
	}

	addr := ":" + cfg.Port
	log.Printf("openlogs listening on %s (theme=%s, retention=%dd, db=%s)", addr, cfg.Theme, cfg.RetentionDays, cfg.DBPath)
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	return server.ListenAndServe()
}

// bootstrapAdmin creates the configured admin user on startup if both email and
// password are provided and the user does not already exist. It is idempotent, so
// it is safe across container restarts. A no-op when either value is empty.
func bootstrapAdmin(database *db.DB, email, password string) error {
	email = strings.TrimSpace(email)
	if email == "" || password == "" {
		return nil
	}

	existing, err := database.GetUserByEmail(email)
	if err != nil {
		return fmt.Errorf("bootstrap admin: %w", err)
	}
	if existing != nil {
		log.Printf("admin bootstrap: user %s already exists, skipping", email)
		return nil
	}

	if len(password) < 8 {
		return fmt.Errorf("OPENLOGS_ADMIN_PASSWORD must be at least 8 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("bootstrap admin: hash password: %w", err)
	}
	if _, err := database.CreateUser(email, string(hash)); err != nil {
		return fmt.Errorf("bootstrap admin: %w", err)
	}
	log.Printf("admin bootstrap: created user %s", email)
	return nil
}

// startRetention runs the retention job immediately and then every 24 hours.
func startRetention(database *db.DB, retentionDays int) {
	run := func() {
		n, err := database.DeleteOldLogs(retentionDays)
		if err != nil {
			log.Printf("retention: error deleting old logs: %v", err)
			return
		}
		log.Printf("retention: deleted %d log entries older than %d days", n, retentionDays)
		if err := database.DeleteExpiredSessions(); err != nil {
			log.Printf("retention: error deleting expired sessions: %v", err)
		}
	}

	run()
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			run()
		}
	}()
}

// runCreateUser implements the interactive create-user subcommand. It reads the
// database path from OPENLOGS_DB_PATH (default openlogs.db) so it can run without
// the full server configuration.
func runCreateUser(args []string) error {
	if len(args) != 1 || strings.TrimSpace(args[0]) == "" {
		return errors.New("usage: openlogs create-user <email>")
	}
	email := strings.TrimSpace(args[0])

	dbPath := os.Getenv("OPENLOGS_DB_PATH")
	if dbPath == "" {
		dbPath = "openlogs.db"
	}

	database, err := db.Open(dbPath)
	if err != nil {
		return err
	}
	defer database.Close()

	password, err := promptPassword()
	if err != nil {
		return err
	}
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	if _, err := database.CreateUser(email, string(hash)); err != nil {
		if errors.Is(err, db.ErrDuplicateEmail) {
			return fmt.Errorf("a user with email %q already exists", email)
		}
		return err
	}

	fmt.Printf("Created user %s\n", email)
	return nil
}

// promptPassword reads a password twice from the terminal without echoing it.
func promptPassword() (string, error) {
	fmt.Print("Password: ")
	p1, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}
	fmt.Print("Confirm password: ")
	p2, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}
	if string(p1) != string(p2) {
		return "", errors.New("passwords do not match")
	}
	return string(p1), nil
}
