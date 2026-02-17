package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/xeodou/go-sqlcipher"
)

var DB *sql.DB

// InitDB initializes the encrypted SQLite database with the given password.
// The password is used to derive the encryption key.
func InitDB(password string) error {
	if password == "" {
		return fmt.Errorf("password cannot be empty")
	}

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

	// Open the database with the encryption key pragma
	// Syntax for sqlcipher usually involves passing PRAGMAs or options in DSN
	// For go-sqlcipher, key is usually passed via PRAGMA key after open, or via DSN parameters if supported.
	// xeodou/go-sqlcipher supports standard database/sql.

	dsn := fmt.Sprintf("%s?_pragma_key=%s&_pragma_cipher_page_size=4096", dbPath, password)

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database (possibly wrong password): %w", err)
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
