package network

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/innomon/keet-adk-gateway/pkg/crypto"
	"github.com/innomon/keet-adk-gateway/pkg/dht"
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

func TestP2PWiring_DHTDiscoveryAutoDialing(t *testing.T) {
	_, privA, _ := ed25519.GenerateKey(rand.Reader)
	_, privB, _ := ed25519.GenerateKey(rand.Reader)

	tempDirA, _ := os.MkdirTemp("", "wiring-dht-a-*")
	defer os.RemoveAll(tempDirA)
	tempDirB, _ := os.MkdirTemp("", "wiring-dht-b-*")
	defer os.RemoveAll(tempDirB)

	storageA, _ := hypercore.NewStorage(tempDirA)
	defer storageA.Close()
	storageB, _ := hypercore.NewStorage(tempDirB)
	defer storageB.Close()

	feedKey := "wiring_dht_auto_dial_feed"

	pmA := NewPeerManager(privA, storageA, nil, feedKey)
	defer pmA.Close()

	pmB := NewPeerManager(privB, storageB, nil, feedKey)
	defer pmB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start listener on B
	if err := pmB.StartListener(ctx, "127.0.0.1:0"); err != nil {
		t.Fatalf("failed B listener: %v", err)
	}
	bAddr := pmB.Addr().String()

	// DHT registry setup on A
	swarmRegistry := dht.NewSwarmRegistry()
	
	// Wire OnRegisterPeer callback to DialPeer!
	swarmRegistry.OnRegisterPeer = func(topic [32]byte, peerAddr string) {
		_ = pmA.DialPeer(ctx, peerAddr)
	}

	// Start A listener so it exists
	if err := pmA.StartListener(ctx, "127.0.0.1:0"); err != nil {
		t.Fatalf("failed A listener: %v", err)
	}

	// Act: Trigger DHT Registration on A, which registers B's address
	var mockTopic [32]byte
	copy(mockTopic[:], []byte("mock_topic_hash_bytes"))
	swarmRegistry.RegisterPeer(mockTopic, bAddr)

	// Assert: Verify connection is established automatically
	retries := 50
	connected := false
	for i := 0; i < retries; i++ {
		pmA.mu.Lock()
		if len(pmA.conns) > 0 {
			connected = true
			pmA.mu.Unlock()
			break
		}
		pmA.mu.Unlock()
		time.Sleep(50 * time.Millisecond)
	}

	if !connected {
		t.Fatal("expected peer manager to automatically dial B and establish Noise connection upon DHT discovery")
	}
}
