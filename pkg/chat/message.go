package chat

import (
	"encoding/json"
	"errors"
)

type ChatMessage struct {
	Sender    string `json:"sender"`    // Hex-encoded sender public key
	Timestamp int64  `json:"timestamp"` // Unix timestamp in seconds
	Content   string `json:"content"`   // Message text content
}

func (m *ChatMessage) Validate() error {
	if m.Sender == "" {
		return errors.New("sender public key cannot be empty")
	}
	if m.Timestamp <= 0 {
		return errors.New("invalid timestamp")
	}
	if m.Content == "" {
		return errors.New("message content cannot be empty")
	}
	if len(m.Content) > 4096 {
		return errors.New("message content exceeds maximum limit of 4096 characters")
	}
	return nil
}

func SerializeMessage(m *ChatMessage) ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(m)
}

func DeserializeMessage(data []byte) (*ChatMessage, error) {
	var m ChatMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return &m, nil
}
