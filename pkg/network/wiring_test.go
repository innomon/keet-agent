package network

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/innomon/keet-adk-gateway/pkg/crypto"
	"github.com/innomon/keet-adk-gateway/pkg/dht"
	"github.com/innomon/keet-adk-gateway/pkg/hypercore"
	"github.com/innomon/keet-adk-gateway/pkg/ipc"
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

	if addr.Network() != "utp" {
		t.Errorf("expected UTP network, got %s", addr.Network())
	}
}

func TestP2PWiring_DHTDiscoveryAutoDialing(t *testing.T) {
	_, privA, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("keygen A: %v", err)
	}
	_, privB, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("keygen B: %v", err)
	}

	tempDirA, err := os.MkdirTemp("", "wiring-dht-a-*")
	if err != nil {
		t.Fatalf("temp dir A: %v", err)
	}
	defer os.RemoveAll(tempDirA)
	tempDirB, err := os.MkdirTemp("", "wiring-dht-b-*")
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

	// Assert: Verify connection is established automatically via ConnCount() helper
	retries := 50
	connected := false
	for i := 0; i < retries; i++ {
		if pmA.ConnCount() > 0 {
			connected = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if !connected {
		t.Fatal("expected peer manager to automatically dial B and establish Noise connection upon DHT discovery")
	}
}

func TestP2PWiring_End2EndReplicationSocketNotification(t *testing.T) {
	_, privA, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("keygen A: %v", err)
	}
	_, privB, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("keygen B: %v", err)
	}

	tempDirA, err := os.MkdirTemp("", "wiring-e2e-a-*")
	if err != nil {
		t.Fatalf("temp dir A: %v", err)
	}
	defer os.RemoveAll(tempDirA)
	tempDirB, err := os.MkdirTemp("", "wiring-e2e-b-*")
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

	feedKey := "wiring_e2e_sync_feed"

	pmA := NewPeerManager(privA, storageA, nil, feedKey)
	defer pmA.Close()

	pmB := NewPeerManager(privB, storageB, nil, feedKey)
	defer pmB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Socket Path for ADK client listening to B's incoming blocks
	socketPath := "/tmp/keet-adk-e2e-wiring-test.sock"
	_ = os.Remove(socketPath)

	listener, err := ipc.NewSocketListener(socketPath)
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	defer listener.Close()

	// Accept clients at B
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go ipc.HandleClient(ctx, conn, nil, nil, storageB, nil, nil, nil)
		}
	}()

	// Connect ADK client to B
	clientConn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to dial socket: %v", err)
	}
	defer clientConn.Close()

	// Wait for client connection to register
	time.Sleep(100 * time.Millisecond)

	// Wire OnAppendBlock callback on B to trigger IPC socket broadcast
	pmB.OnAppendBlock = func(index uint64, value []byte) {
		ipc.BroadcastChatMessage(feedKey, index, value)
	}

	// Start B listener
	if err := pmB.StartListener(ctx, "127.0.0.1:0"); err != nil {
		t.Fatalf("failed B listener: %v", err)
	}
	bAddr := pmB.Addr().String()

	// Dial from A
	if err := pmA.DialPeer(ctx, bAddr); err != nil {
		t.Fatalf("failed A dial: %v", err)
	}

	// Wait for connection to establish
	time.Sleep(100 * time.Millisecond)

	// Append valid chat message JSON block on A
	chatJSON := `{"sender":"test_wiring_sender","timestamp":1800000000,"content":"e2e wiring test message"}`
	if err := storageA.Append([]byte(chatJSON)); err != nil {
		t.Fatalf("failed A append: %v", err)
	}

	// Trigger A have announcement
	pmA.BroadcastHave(storageA.Len())

	// Read notification from ADK Client connected to B!
	var notification map[string]interface{}
	clientConn.SetReadDeadline(time.Now().Add(3 * time.Second))
	if err := json.NewDecoder(clientConn).Decode(&notification); err != nil {
		t.Fatalf("failed to read notification: %v", err)
	}

	if notification["command"] != "chat_message_received" {
		t.Errorf("expected command 'chat_message_received', got %v", notification["command"])
	}
	if notification["content"] != "e2e wiring test message" {
		t.Errorf("expected content 'e2e wiring test message', got %v", notification["content"])
	}
}
