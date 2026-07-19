package network

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"testing"
	"time"

	"github.com/innomon/keet-adk-gateway/pkg/hypercore"
)

func TestTCP_PeerManagerReplication(t *testing.T) {
	// Generate static keys
	_, privA, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key A: %v", err)
	}
	_, privB, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key B: %v", err)
	}

	tempDirA, err := os.MkdirTemp("", "tcp-sync-a-*")
	if err != nil {
		t.Fatalf("temp dir A: %v", err)
	}
	defer os.RemoveAll(tempDirA)

	tempDirB, err := os.MkdirTemp("", "tcp-sync-b-*")
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

	// Append mock block data to storage A
	blockData := []byte("hello from TCP noise replicated feed")
	if err := storageA.Append(blockData); err != nil {
		t.Fatalf("append: %v", err)
	}

	feedKey := "tcp_replication_feed_key"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pmA := NewPeerManager(privA, storageA, nil, feedKey)
	defer pmA.Close()

	pmB := NewPeerManager(privB, storageB, nil, feedKey)
	defer pmB.Close()

	// Start B listener
	if err := pmB.StartListener(ctx, "127.0.0.1:0"); err != nil {
		t.Fatalf("failed to start listener: %v", err)
	}
	bAddr := pmB.listener.Addr().String()

	// Dial from A to B
	if err := pmA.DialPeer(ctx, bAddr); err != nil {
		t.Fatalf("failed to dial peer: %v", err)
	}

	// Poll B until block is replicated
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

	val, err := storageB.Get(0)
	if err != nil {
		t.Fatalf("get block: %v", err)
	}

	if !bytes.Equal(val, blockData) {
		t.Errorf("block mismatch: expected %q, got %q", string(blockData), string(val))
	}
}
