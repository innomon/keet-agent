package chat

import (
	"bytes"
	"testing"
	"time"
)

func TestChatMessage_Serialization(t *testing.T) {
	msg := &ChatMessage{
		Sender:    "92079689c517845283d03b57f84df81d9af2ee26e10956eb5da610ddca6d605f",
		Timestamp: time.Now().Unix(),
		Content:   "hello decentralized world",
	}

	data, err := SerializeMessage(msg)
	if err != nil {
		t.Fatalf("failed to serialize chat message: %v", err)
	}

	decoded, err := DeserializeMessage(data)
	if err != nil {
		t.Fatalf("failed to deserialize chat message: %v", err)
	}

	if decoded.Sender != msg.Sender {
		t.Errorf("expected sender %q, got %q", msg.Sender, decoded.Sender)
	}
	if decoded.Timestamp != msg.Timestamp {
		t.Errorf("expected timestamp %d, got %d", msg.Timestamp, decoded.Timestamp)
	}
	if decoded.Content != msg.Content {
		t.Errorf("expected content %q, got %q", msg.Content, decoded.Content)
	}
}

func TestChatMessage_Validation(t *testing.T) {
	invalidMsg := &ChatMessage{
		Sender:    "",
		Timestamp: 1234567,
		Content:   "bad sender",
	}
	_, err := SerializeMessage(invalidMsg)
	if err == nil {
		t.Error("expected serialization error for empty sender")
	}

	tooLongContent := bytes.Repeat([]byte("A"), 5000)
	invalidMsg = &ChatMessage{
		Sender:    "abc",
		Timestamp: 1234567,
		Content:   string(tooLongContent),
	}
	_, err = SerializeMessage(invalidMsg)
	if err == nil {
		t.Error("expected serialization error for too long content")
	}
}
