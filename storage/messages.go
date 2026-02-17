package storage

import (
	"fmt"
)

type Message struct {
	ID        int64
	PeerID    string
	Content   string
	Timestamp int64
	IsSent    bool
}

// SaveMessage stores a new message in the encrypted database.
func SaveMessage(peerID, content string, timestamp int64, isSent bool) error {
	query := `INSERT INTO messages (peer_id, content, timestamp, is_sent) VALUES (?, ?, ?, ?)`
	_, err := DB.Exec(query, peerID, content, timestamp, isSent)
	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}
	return nil
}

// GetMessages retrieves the last N messages for a specific peer.
func GetMessages(peerID string, limit int) ([]Message, error) {
	query := `
		SELECT id, peer_id, content, timestamp, is_sent 
		FROM messages 
		WHERE peer_id = ? 
		ORDER BY timestamp DESC 
		LIMIT ?`

	rows, err := DB.Query(query, peerID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.PeerID, &m.Content, &m.Timestamp, &m.IsSent); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}

	// Reverse the slice so oldest is first (for chat UI)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// ClearHistory removes all messages from the database.
func ClearHistory() error {
	_, err := DB.Exec("DELETE FROM messages")
	if err != nil {
		return fmt.Errorf("failed to clear history: %w", err)
	}
	return nil
}
