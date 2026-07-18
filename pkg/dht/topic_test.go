package dht

import (
	"encoding/hex"
	"testing"
)

func TestResolveTopicKey_String(t *testing.T) {
	topic := "my-awesome-chat-room"

	// Resolve the topic key
	key, err := ResolveTopicKey(topic)
	if err != nil {
		t.Fatalf("failed to resolve topic key: %v", err)
	}

	if len(key) != 32 {
		t.Errorf("expected 32-byte key, got %d bytes", len(key))
	}
}

func TestResolveTopicKey_Hex(t *testing.T) {
	// A valid 32-byte hex string (64 characters)
	hexString := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	expectedBytes, _ := hex.DecodeString(hexString)

	key, err := ResolveTopicKey(hexString)
	if err != nil {
		t.Fatalf("failed to resolve hex topic key: %v", err)
	}

	for i := range key {
		if key[i] != expectedBytes[i] {
			t.Fatalf("expected key byte %d to be %d, got %d", i, expectedBytes[i], key[i])
		}
	}
}

func TestResolveTopicKey_Empty(t *testing.T) {
	_, err := ResolveTopicKey("")
	if err == nil {
		t.Error("expected error for empty topic, got nil")
	}
}
