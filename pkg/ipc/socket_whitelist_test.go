package ipc

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestHandleClient_Whitelist(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tmpDir, err := os.MkdirTemp("", "socket-whitelist-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	socketPath := filepath.Join(tmpDir, "test.sock")
	listener, err := NewSocketListener(socketPath)
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	defer listener.Close()

	whitelist := []string{"key-alice-123", "key-bob-456"}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go HandleClient(ctx, conn, nil, nil, nil, nil, nil, whitelist)
		}
	}()

	t.Run("Rejected - No Key", func(t *testing.T) {
		conn, err := net.Dial("unix", socketPath)
		if err != nil {
			t.Fatalf("failed to dial: %v", err)
		}
		defer conn.Close()

		decoder := json.NewDecoder(conn)
		encoder := json.NewEncoder(conn)

		// Send command with no peer_key
		req := map[string]interface{}{
			"command": "join_swarm",
			"topic":   "some_topic",
		}
		if err := encoder.Encode(&req); err != nil {
			t.Fatalf("failed to encode: %v", err)
		}

		var resp map[string]interface{}
		if err := decoder.Decode(&resp); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}

		if resp["status"] != "error" {
			t.Errorf("expected error status, got %v", resp["status"])
		}
		if resp["error"] == nil || resp["error"].(string) != "unauthorized client public key or authentication required" {
			t.Errorf("expected unauthorized error message, got %v", resp["error"])
		}
	})

	t.Run("Rejected - Unauthorized Key", func(t *testing.T) {
		conn, err := net.Dial("unix", socketPath)
		if err != nil {
			t.Fatalf("failed to dial: %v", err)
		}
		defer conn.Close()

		decoder := json.NewDecoder(conn)
		encoder := json.NewEncoder(conn)

		// Send command with unauthorized peer_key
		req := map[string]interface{}{
			"command":  "join_swarm",
			"topic":    "some_topic",
			"peer_key": "key-eve-789",
		}
		if err := encoder.Encode(&req); err != nil {
			t.Fatalf("failed to encode: %v", err)
		}

		var resp map[string]interface{}
		if err := decoder.Decode(&resp); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}

		if resp["status"] != "error" {
			t.Errorf("expected error status, got %v", resp["status"])
		}
	})

	t.Run("Accepted - Whitelisted Key", func(t *testing.T) {
		conn, err := net.Dial("unix", socketPath)
		if err != nil {
			t.Fatalf("failed to dial: %v", err)
		}
		defer conn.Close()

		decoder := json.NewDecoder(conn)
		encoder := json.NewEncoder(conn)

		// Send join_swarm with whitelisted peer_key
		req := map[string]interface{}{
			"command":  "join_swarm",
			"topic":    "some_topic",
			"peer_key": "key-alice-123",
		}
		if err := encoder.Encode(&req); err != nil {
			t.Fatalf("failed to encode: %v", err)
		}

		var resp map[string]interface{}
		if err := decoder.Decode(&resp); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}

		// Since other dht / registry objects are nil, join_swarm succeeds but returns success or handles nested nil checks
		if resp["status"] == "error" {
			t.Errorf("expected success or basic status, got error: %v", resp["error"])
		}
	})

	t.Run("Accepted - Explicit Auth Command", func(t *testing.T) {
		conn, err := net.Dial("unix", socketPath)
		if err != nil {
			t.Fatalf("failed to dial: %v", err)
		}
		defer conn.Close()

		decoder := json.NewDecoder(conn)
		encoder := json.NewEncoder(conn)

		// Send explicit auth command
		req := map[string]interface{}{
			"command":  "auth",
			"peer_key": "key-bob-456",
		}
		if err := encoder.Encode(&req); err != nil {
			t.Fatalf("failed to encode: %v", err)
		}

		var resp map[string]interface{}
		if err := decoder.Decode(&resp); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}

		if resp["status"] != "success" || resp["command"] != "auth" {
			t.Errorf("expected success for auth command, got %+v", resp)
		}
	})
}
