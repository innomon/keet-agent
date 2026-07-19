package ipc

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/innomon/keet-adk-gateway/pkg/dht"
	"github.com/innomon/keet-adk-gateway/pkg/hypercore"
	"github.com/innomon/keet-adk-gateway/pkg/network"
)

func TestSocketListener(t *testing.T) {
	socketPath := "/tmp/keet-adk-test.sock"
	_ = os.Remove(socketPath)

	_, err := os.Create(socketPath)
	if err != nil {
		t.Fatalf("failed to create stale socket dummy: %v", err)
	}

	listener, err := NewSocketListener(socketPath)
	if err != nil {
		t.Fatalf("failed to create socket listener: %v", err)
	}
	defer listener.Close()

	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		t.Error("expected socket file to exist")
	}

	connChan := make(chan net.Conn, 1)
	errChan := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			errChan <- err
			return
		}
		connChan <- conn
	}()

	clientConn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to dial socket: %v", err)
	}
	defer clientConn.Close()

	select {
	case conn := <-connChan:
		conn.Close()
	case err := <-errChan:
		t.Fatalf("accept error: %v", err)
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for connection accept")
	}
}

func TestSocket_ConcurrentClients(t *testing.T) {
	socketPath := "/tmp/keet-adk-concurrent-test.sock"
	_ = os.Remove(socketPath)

	listener, err := NewSocketListener(socketPath)
	if err != nil {
		t.Fatalf("failed to create socket listener: %v", err)
	}
	defer listener.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start accepting loop in background
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					return
				}
			}
			go HandleClient(ctx, conn, nil, nil, nil, nil, nil, nil)
		}
	}()

	const clientCount = 5
	var wg sync.WaitGroup
	wg.Add(clientCount)

	for i := 0; i < clientCount; i++ {
		go func(clientId int) {
			defer wg.Done()

			conn, err := net.Dial("unix", socketPath)
			if err != nil {
				t.Errorf("client %d: failed to dial: %v", clientId, err)
				return
			}
			defer conn.Close()

			// Send command
			req := map[string]interface{}{"command": "test", "id": clientId}
			if err := json.NewEncoder(conn).Encode(&req); err != nil {
				t.Errorf("client %d: failed to send: %v", clientId, err)
				return
			}

			// Read response
			var resp map[string]interface{}
			if err := json.NewDecoder(conn).Decode(&resp); err != nil {
				t.Errorf("client %d: failed to decode response: %v", clientId, err)
				return
			}

			if resp["status"] != "acknowledged" {
				t.Errorf("client %d: expected status 'acknowledged', got: %v", clientId, resp["status"])
			}
			if resp["origin"] != "keet_peer" {
				t.Errorf("client %d: expected origin 'keet_peer', got: %v", clientId, resp["origin"])
			}
		}(i)
	}

	wg.Wait()
}

func TestSocket_SwarmCommands(t *testing.T) {
	socketPath := "/tmp/keet-adk-swarm-test.sock"
	_ = os.Remove(socketPath)

	listener, err := NewSocketListener(socketPath)
	if err != nil {
		t.Fatalf("failed to create socket listener: %v", err)
	}
	defer listener.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go HandleClient(ctx, conn, nil, nil, nil, nil, nil, nil)
		}
	}()

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer conn.Close()

	// 1. Test join_swarm
	reqJoin := map[string]interface{}{
		"command":  "join_swarm",
		"topic":    "test-room",
		"peer_key": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}
	if err := json.NewEncoder(conn).Encode(&reqJoin); err != nil {
		t.Fatalf("failed to send join_swarm: %v", err)
	}

	var respJoin map[string]interface{}
	if err := json.NewDecoder(conn).Decode(&respJoin); err != nil {
		t.Fatalf("failed to read join_swarm response: %v", err)
	}
	if respJoin["status"] != "success" || respJoin["command"] != "join_swarm" {
		t.Errorf("unexpected join_swarm response: %v", respJoin)
	}

	// 2. Test leave_swarm
	reqLeave := map[string]interface{}{
		"command": "leave_swarm",
		"topic":   "test-room",
	}
	if err := json.NewEncoder(conn).Encode(&reqLeave); err != nil {
		t.Fatalf("failed to send leave_swarm: %v", err)
	}

	var respLeave map[string]interface{}
	if err := json.NewDecoder(conn).Decode(&respLeave); err != nil {
		t.Fatalf("failed to read leave_swarm response: %v", err)
	}
	if respLeave["status"] != "success" || respLeave["command"] != "leave_swarm" {
		t.Errorf("unexpected leave_swarm response: %v", respLeave)
	}
}

func TestSocket_HypercoreCommands(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hypercore_socket_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := hypercore.NewStorage(tempDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer storage.Close()

	socketPath := "/tmp/keet-adk-hypercore-test.sock"
	_ = os.Remove(socketPath)

	listener, err := NewSocketListener(socketPath)
	if err != nil {
		t.Fatalf("failed to create socket listener: %v", err)
	}
	defer listener.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go HandleClient(ctx, conn, nil, nil, storage, nil, nil, nil)
		}
	}()

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer conn.Close()

	// 1. Test append_block
	reqAppend := map[string]interface{}{
		"command": "append_block",
		"data":    "aGVsbG8gYmxvY2s=", // "hello block" in base64
	}
	if err := json.NewEncoder(conn).Encode(&reqAppend); err != nil {
		t.Fatalf("failed to send append_block: %v", err)
	}

	var respAppend map[string]interface{}
	if err := json.NewDecoder(conn).Decode(&respAppend); err != nil {
		t.Fatalf("failed to read append_block response: %v", err)
	}
	if respAppend["status"] != "success" || respAppend["command"] != "append_block" {
		t.Errorf("unexpected append_block response: %v", respAppend)
	}
	if respAppend["index"] != float64(0) {
		t.Errorf("expected index 0, got %v", respAppend["index"])
	}

	// 2. Test get_block
	reqGet := map[string]interface{}{
		"command": "get_block",
		"index":   0,
	}
	if err := json.NewEncoder(conn).Encode(&reqGet); err != nil {
		t.Fatalf("failed to send get_block: %v", err)
	}

	var respGet map[string]interface{}
	if err := json.NewDecoder(conn).Decode(&respGet); err != nil {
		t.Fatalf("failed to read get_block response: %v", err)
	}
	if respGet["status"] != "success" || respGet["command"] != "get_block" {
		t.Errorf("unexpected get_block response: %v", respGet)
	}
	if respGet["data"] != "aGVsbG8gYmxvY2s=" {
		t.Errorf("expected 'aGVsbG8gYmxvY2s=', got %v", respGet["data"])
	}
}

func TestSocket_NotificationBroadcast(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hypercore_socket_notify_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := hypercore.NewStorage(tempDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer storage.Close()

	socketPath := "/tmp/keet-adk-notify-test.sock"
	_ = os.Remove(socketPath)

	listener, err := NewSocketListener(socketPath)
	if err != nil {
		t.Fatalf("failed to create socket listener: %v", err)
	}
	defer listener.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go HandleClient(ctx, conn, nil, nil, storage, nil, nil, nil)
		}
	}()

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer conn.Close()

	// Connect a second client to check broadcast functionality
	conn2, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to dial client 2: %v", err)
	}
	defer conn2.Close()

	// Wait for clients to bind
	time.Sleep(100 * time.Millisecond)

	// Append a valid ChatMessage JSON block
	chatJSON := `{"sender":"test_sender_key","timestamp":1700000000,"content":"broadcast test message"}`
	
	reqAppend := map[string]interface{}{
		"command": "append_block",
		"data":    base64.StdEncoding.EncodeToString([]byte(chatJSON)),
	}
	if err := json.NewEncoder(conn).Encode(&reqAppend); err != nil {
		t.Fatalf("failed to send append_block: %v", err)
	}

	// Read append ack response from conn
	var respAppend map[string]interface{}
	if err := json.NewDecoder(conn).Decode(&respAppend); err != nil {
		t.Fatalf("failed to read response: %v", err)
	}

	// Read notification from conn2 (which is passive but should receive the broadcast!)
	var notification map[string]interface{}
	
	// Set read deadline so it doesn't block indefinitely
	conn2.SetReadDeadline(time.Now().Add(1 * time.Second))
	if err := json.NewDecoder(conn2).Decode(&notification); err != nil {
		t.Fatalf("failed to read notification: %v", err)
	}

	if notification["command"] != "chat_message_received" {
		t.Errorf("expected command 'chat_message_received', got %v", notification["command"])
	}
	if notification["content"] != "broadcast test message" {
		t.Errorf("expected content 'broadcast test message', got %v", notification["content"])
	}
}

func TestSocket_P2PReplicationBroadcastNotification(t *testing.T) {
	_, privA, _ := ed25519.GenerateKey(rand.Reader)
	_, privB, _ := ed25519.GenerateKey(rand.Reader)

	tempDirA, _ := os.MkdirTemp("", "ipc-p2p-sync-a-*")
	defer os.RemoveAll(tempDirA)
	tempDirB, _ := os.MkdirTemp("", "ipc-p2p-sync-b-*")
	defer os.RemoveAll(tempDirB)

	storageA, _ := hypercore.NewStorage(tempDirA)
	defer storageA.Close()
	storageB, _ := hypercore.NewStorage(tempDirB)
	defer storageB.Close()

	feedKey := "p2p_socket_broadcast_feed"

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Socket Path for ADK client listening to B's incoming blocks
	socketPath := "/tmp/keet-adk-p2p-broadcast-test.sock"
	_ = os.Remove(socketPath)

	listener, err := NewSocketListener(socketPath)
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	defer listener.Close()

	// Accept clients at B and run HandleClient
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go HandleClient(ctx, conn, nil, nil, storageB, nil, nil, nil)
		}
	}()

	// Connect ADK client to B
	clientConn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to dial socket: %v", err)
	}
	defer clientConn.Close()

	// Wait for ADK client to bind
	time.Sleep(100 * time.Millisecond)

	pmA := network.NewPeerManager(privA, storageA, nil, feedKey)
	defer pmA.Close()

	pmB := network.NewPeerManager(privB, storageB, nil, feedKey)
	defer pmB.Close()

	// Broadcast callback on B to trigger Socket notification
	pmB.OnAppendBlock = func(index uint64, value []byte) {
		BroadcastChatMessage(feedKey, index, value)
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

	// Wait for handshakes
	time.Sleep(100 * time.Millisecond)

	// Append a valid chat message JSON block on A
	chatJSON := `{"sender":"test_p2p_sender","timestamp":1750000000,"content":"p2p live message broadcast"}`
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
	if notification["content"] != "p2p live message broadcast" {
		t.Errorf("expected content 'p2p live message broadcast', got %v", notification["content"])
	}
}

func TestSocket_DHTIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 1. Setup 2-node DHT network
	tpA, _ := dht.NewInProcessTransport("nodeA")
	idA := [32]byte{1}
	nodeA, _ := dht.NewDHTNode(&dht.Config{LocalID: idA, Transport: tpA, BootstrapNodes: []string{}})
	_ = nodeA.Start(ctx, nil)
	defer nodeA.Stop()

	tpB, _ := dht.NewInProcessTransport("nodeB")
	idB := [32]byte{2}
	nodeB, _ := dht.NewDHTNode(&dht.Config{LocalID: idB, Transport: tpB, BootstrapNodes: []string{"nodeA"}})
	_ = nodeB.Start(ctx, nil)
	defer nodeB.Stop()

	// Pre-announce nodeA on the topic key
	topic := "chat-room-123"
	resolvedKey, _ := dht.ResolveTopicKey(topic)
	_ = nodeA.Announce(ctx, resolvedKey, 6001)

	// 2. Setup socket listener
	socketPath := "/tmp/keet-adk-dht-integration.sock"
	_ = os.Remove(socketPath)

	listener, err := NewSocketListener(socketPath)
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	defer listener.Close()

	regB := dht.NewSwarmRegistry()
	regB.P2PPort = 6002

	// Run HandleClient with nodeB and regB
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go HandleClient(ctx, conn, nodeB, regB, nil, nil, nil, nil)
		}
	}()

	// 3. Dial as client and send join_swarm
	clientConn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to dial socket: %v", err)
	}
	defer clientConn.Close()

	req := map[string]interface{}{
		"command":  "join_swarm",
		"topic":    topic,
		"peer_key": "nodeB_identity_key",
	}

	if err := json.NewEncoder(clientConn).Encode(&req); err != nil {
		t.Fatalf("failed to send join_swarm: %v", err)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(clientConn).Decode(&resp); err != nil {
		t.Fatalf("failed to read response: %v", err)
	}

	if resp["status"] != "success" {
		t.Fatalf("expected status 'success', got: %v", resp)
	}

	// Verify nodeA's peer was discovered and registered in regB!
	// Give it a brief moment for the lookup to complete
	time.Sleep(100 * time.Millisecond)

	peers := regB.GetPeers(resolvedKey)
	foundA := false
	for _, p := range peers {
		if p == "nodeA:6001" || p == "nodeA" {
			foundA = true
		}
	}

	if !foundA {
		t.Errorf("expected to discover nodeA on the swarm registry of B, got peers: %v", peers)
	}

	// Send leave_swarm and verify
	reqLeave := map[string]interface{}{
		"command": "leave_swarm",
		"topic":   topic,
	}
	if err := json.NewEncoder(clientConn).Encode(&reqLeave); err != nil {
		t.Fatalf("failed to send leave_swarm: %v", err)
	}

	var respLeave map[string]interface{}
	if err := json.NewDecoder(clientConn).Decode(&respLeave); err != nil {
		t.Fatalf("failed to read leave response: %v", err)
	}
	if respLeave["status"] != "success" {
		t.Fatalf("expected leave status 'success', got: %v", respLeave)
	}
}
