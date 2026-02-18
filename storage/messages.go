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
	// Encrypt content
	encryptedContent, err := Encrypt(content)
	if err != nil {
		return fmt.Errorf("failed to encrypt message: %w", err)
	}

	query := `INSERT INTO messages (peer_id, content, timestamp, is_sent) VALUES (?, ?, ?, ?)`
	_, err = DB.Exec(query, peerID, encryptedContent, timestamp, isSent)
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

		// Decrypt content
		decryptedContent, err := Decrypt(m.Content)
		if err != nil {
			// If decryption fails (e.g., wrong password or corrupted data),
			// we might want to return the raw content or an error indicator.
			// For now, let's return a placeholder so the UI doesn't crash.
			m.Content = fmt.Sprintf("[Decryption Failed: %v]", err)
		} else {
			m.Content = decryptedContent
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
