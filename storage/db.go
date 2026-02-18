package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

// InitDB initializes the SQLite database.
// NOTE: Encryption is currently DISABLED for cross-platform compatibility.
// The password argument is ignored in this version.
func InitDB(password string) error {
	// Determine the database path
	configDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get user config directory: %w", err)
	}

	appDir := filepath.Join(configDir, "shellchat")
	if err := os.MkdirAll(appDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	dbPath := filepath.Join(appDir, "shellchat.db")

	// Open the database using modernc.org/sqlite (pure Go)
	// Build DSN
	dsn := dbPath

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	DB = db

	// Initialize Schema
	if err := createSchema(context.Background()); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// CloseDB closes the database connection.
func CloseDB() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}
