package network

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/innomon/keet-adk-gateway/pkg/crypto"
	"github.com/innomon/keet-adk-gateway/pkg/hypercore"
)

func TestP2PWiring_NodeIdentityAndListener(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "p2p-wiring-test-*")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 1. Verify key is generated and saved
	privKey, err := crypto.LoadOrGenerateNodeKey(tempDir)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	keyFile := filepath.Join(tempDir, "node_key.priv")
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		t.Errorf("expected node identity key file to exist at %s", keyFile)
	}

	// 2. Initialize Hypercore Storage
	storage, err := hypercore.NewStorage(tempDir)
	if err != nil {
		t.Fatalf("storage init: %v", err)
	}
	defer storage.Close()

	// 3. Start PeerManager listener
	pm := NewPeerManager(privKey, storage, nil, "p2p_wiring_test_feed")
	defer pm.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := pm.StartListener(ctx, "127.0.0.1:0"); err != nil {
		t.Fatalf("failed to start listener: %v", err)
	}

	addr := pm.Addr()
	if addr == nil {
		t.Fatal("expected peer manager bound address to be non-nil")
	}

	if addr.Network() != "tcp" {
		t.Errorf("expected TCP network, got %s", addr.Network())
	}
}
