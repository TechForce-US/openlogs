// Package config loads OpenLogs runtime configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all runtime configuration for the OpenLogs server.
type Config struct {
	DBPath        string
	SecretKey     string
	Port          string
	Theme         string
	RetentionDays int
	CustomCSSDir  string

	// AdminEmail and AdminPassword, when both set, cause the server to create
	// this user on startup if it does not already exist. Useful for turnkey
	// Docker deployments. Leave unset to manage users only via `create-user`.
	AdminEmail    string
	AdminPassword string
}

// Load reads configuration from the environment, applies defaults, and validates
// required values. It returns an error rather than exiting so callers can decide
// how to surface the failure.
func Load() (*Config, error) {
	c := &Config{
		DBPath:        getEnv("OPENLOGS_DB_PATH", "openlogs.db"),
		SecretKey:     os.Getenv("OPENLOGS_SECRET_KEY"),
		Port:          getEnv("OPENLOGS_PORT", "8080"),
		Theme:         getEnv("OPENLOGS_THEME", "modern"),
		RetentionDays: 30,
		CustomCSSDir:  getEnv("OPENLOGS_CUSTOM_CSS_DIR", "web/static/custom"),
		AdminEmail:    os.Getenv("OPENLOGS_ADMIN_EMAIL"),
		AdminPassword: os.Getenv("OPENLOGS_ADMIN_PASSWORD"),
	}

	if c.SecretKey == "" {
		return nil, fmt.Errorf("OPENLOGS_SECRET_KEY is required but not set")
	}

	if c.Theme != "modern" && c.Theme != "terminal" {
		return nil, fmt.Errorf("OPENLOGS_THEME must be \"modern\" or \"terminal\", got %q", c.Theme)
	}

	if v := os.Getenv("OPENLOGS_RETENTION_DAYS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("OPENLOGS_RETENTION_DAYS must be a positive integer, got %q", v)
		}
		c.RetentionDays = n
	}

	return c, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
