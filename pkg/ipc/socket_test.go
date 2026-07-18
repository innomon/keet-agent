package ipc

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/innomon/keet-adk-gateway/pkg/hypercore"
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
			go HandleClient(ctx, conn, nil, nil, nil)
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
			go HandleClient(ctx, conn, nil, nil, nil)
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
			go HandleClient(ctx, conn, nil, nil, storage)
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
