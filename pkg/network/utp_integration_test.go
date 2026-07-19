package network

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"testing"
	"time"

	"github.com/innomon/keet-adk-gateway/pkg/chat"
	"github.com/innomon/keet-adk-gateway/pkg/hypercore"
)

func TestUTP_EndToEndChatSync(t *testing.T) {
	// Generate keys for Peer A and Peer B
	_, privA, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key A: %v", err)
	}
	_, privB, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key B: %v", err)
	}

	// Create temp storage directories
	tempDirA, err := os.MkdirTemp("", "utp-e2e-sync-a-*")
	if err != nil {
		t.Fatalf("temp dir A: %v", err)
	}
	defer os.RemoveAll(tempDirA)

	tempDirB, err := os.MkdirTemp("", "utp-e2e-sync-b-*")
	if err != nil {
		t.Fatalf("temp dir B: %v", err)
	}
	defer os.RemoveAll(tempDirB)

	storageA, err := hypercore.NewStorage(tempDirA)
	if err != nil {
		t.Fatalf("storage A: %v", err)
	}
	defer storageA.Close()

	storageB, err := hypercore.NewStorage(tempDirB)
	if err != nil {
		t.Fatalf("storage B: %v", err)
	}
	defer storageB.Close()

	// 1. Create a ChatMessage and serialize it
	msg := &chat.ChatMessage{
		Sender:    "92079689c517845283d03b57f84df81d9af2ee26e10956eb5da610ddca6d605f",
		Timestamp: time.Now().Unix(),
		Content:   "P2P decentralized chat over uTP UDP transport",
	}
	msgData, err := chat.SerializeMessage(msg)
	if err != nil {
		t.Fatalf("failed to serialize chat message: %v", err)
	}

	// Append serialized chat message to Peer A's storage
	if err := storageA.Append(msgData); err != nil {
		t.Fatalf("append message to storage A: %v", err)
	}

	feedKey := "utp_e2e_replication_feed_key"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pmA := NewPeerManager(privA, storageA, nil, feedKey)
	defer pmA.Close()

	pmB := NewPeerManager(privB, storageB, nil, feedKey)
	defer pmB.Close()

	// 2. Start Peer B listener on UTP
	if err := pmB.StartListener(ctx, "127.0.0.1:0"); err != nil {
		t.Fatalf("failed to start listener B: %v", err)
	}
	bAddr := pmB.listener.Addr().String()

	// 3. Dial Peer B from Peer A
	if err := pmA.DialPeer(ctx, bAddr); err != nil {
		t.Fatalf("failed to dial from A to B: %v", err)
	}

	// 4. Poll Peer B's storage until block is replicated
	retries := 50
	for i := 0; i < retries; i++ {
		if storageB.Len() >= 1 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if storageB.Len() < 1 {
		t.Fatalf("replicated block failed to arrive within timeout")
	}

	// 5. Retrieve block and deserialize ChatMessage
	replicatedData, err := storageB.Get(0)
	if err != nil {
		t.Fatalf("get block: %v", err)
	}

	decodedMsg, err := chat.DeserializeMessage(replicatedData)
	if err != nil {
		t.Fatalf("failed to deserialize replicated chat message: %v", err)
	}

	// Verify fields match exactly
	if decodedMsg.Sender != msg.Sender {
		t.Errorf("sender mismatch: expected %q, got %q", msg.Sender, decodedMsg.Sender)
	}
	if decodedMsg.Timestamp != msg.Timestamp {
		t.Errorf("timestamp mismatch: expected %d, got %d", msg.Timestamp, decodedMsg.Timestamp)
	}
	if decodedMsg.Content != msg.Content {
		t.Errorf("content mismatch: expected %q, got %q", msg.Content, decodedMsg.Content)
	}
}
