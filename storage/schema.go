package storage

import (
	"context"
	"fmt"
)

func createSchema(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		peer_id TEXT NOT NULL,
		content TEXT NOT NULL,
		timestamp INTEGER NOT NULL,
		is_sent BOOLEAN NOT NULL
	);
	
	CREATE INDEX IF NOT EXISTS idx_messages_peer_timestamp ON messages(peer_id, timestamp DESC);
	`

	_, err := DB.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}
