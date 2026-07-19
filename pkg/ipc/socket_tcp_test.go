package ipc

import (
	"context"
	"encoding/json"
	"net"
	"testing"
)

func TestNewSocketListener_TCP(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Bind to local ephemeral TCP port
	listener, err := NewSocketListener("127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to open TCP socket listener: %v", err)
	}
	defer listener.Close()

	if listener.network != "tcp" {
		t.Errorf("expected network tcp, got %q", listener.network)
	}

	addrStr := listener.listener.Addr().String()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		go HandleClient(ctx, conn, nil, nil, nil, nil, nil, nil)
	}()

	conn, err := net.Dial("tcp", addrStr)
	if err != nil {
		t.Fatalf("failed to dial TCP listener: %v", err)
	}
	defer conn.Close()

	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)

	req := map[string]interface{}{
		"command": "auth",
	}
	if err := encoder.Encode(&req); err != nil {
		t.Fatalf("failed to encode: %v", err)
	}

	var resp map[string]interface{}
	if err := decoder.Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if resp["status"] != "success" {
		t.Errorf("expected success response, got %+v", resp)
	}
}

func TestNewSocketListener_TCP_Prefix(t *testing.T) {
	listener, err := NewSocketListener("tcp://127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to open prefixed TCP socket listener: %v", err)
	}
	defer listener.Close()

	if listener.network != "tcp" {
		t.Errorf("expected network tcp, got %q", listener.network)
	}
}

func TestNewSocketListener_Error(t *testing.T) {
	_, err := NewSocketListener("invalid-port-format:abc")
	if err == nil {
		t.Error("expected error binding to invalid TCP port, got nil")
	}
}


