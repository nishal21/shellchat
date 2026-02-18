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
func InitDB(storageDir, password string) error {
	if password == "" {
		return fmt.Errorf("password cannot be empty")
	}

	appDir := filepath.Join(storageDir, "shellchat")
	if err := os.MkdirAll(appDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	dbPath := filepath.Join(appDir, "shellchat.db")

	// Open the database using modernc.org/sqlite (pure Go)
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

	// Initialize Schema (messages table)
	if err := createSchema(context.Background()); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Initialize Encryption (Salt & Key Derivation)
	if err := initEncryption(password); err != nil {
		return fmt.Errorf("failed to initialize encryption: %w", err)
	}

	return nil
}

// initEncryption handles the salt and key derivation
func initEncryption(password string) error {
	// Create metadata table if not exists (for salt)
	_, err := DB.Exec(`CREATE TABLE IF NOT EXISTS metadata (key TEXT PRIMARY KEY, value BLOB)`)
	if err != nil {
		return fmt.Errorf("failed to create metadata table: %w", err)
	}

	// Check if salt exists
	var salt []byte
	err = DB.QueryRow("SELECT value FROM metadata WHERE key = 'salt'").Scan(&salt)
	if err == sql.ErrNoRows {
		// New database: Generate new salt
		salt, err = GenerateSalt()
		if err != nil {
			return fmt.Errorf("failed to generate salt: %w", err)
		}
		// Store salt
		_, err = DB.Exec("INSERT INTO metadata (key, value) VALUES ('salt', ?)", salt)
		if err != nil {
			return fmt.Errorf("failed to store salt: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to query salt: %w", err)
	}

	// Derive session key
	SessionKey = DeriveKey(password, salt)
	return nil
}

// CloseDB closes the database connection.
func CloseDB() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}
