package storage

import (
	"context"
	"fmt"
)

// createSchema creates the necessary database tables.
func createSchema(ctx context.Context) error {
	// Create messages table
	query := `
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		peer_id TEXT NOT NULL,
		content TEXT NOT NULL,
		timestamp INTEGER NOT NULL,
		is_sent BOOLEAN NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_messages_peer_id ON messages(peer_id);
	CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp);
	`
	if _, err := DB.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to create messages table: %w", err)
	}

	// Create metadata table (for encryption salt)
	metaQuery := `
	CREATE TABLE IF NOT EXISTS metadata (
		key TEXT PRIMARY KEY,
		value BLOB
	);
	`
	if _, err := DB.ExecContext(ctx, metaQuery); err != nil {
		return fmt.Errorf("failed to create metadata table: %w", err)
	}

	return nil
}
