package ipc

import (
	"net"
	"os"
	"testing"
	"time"
)

func TestSocketListener(t *testing.T) {
	socketPath := "/tmp/keet-adk-test.sock"
	// Ensure cleanup before test
	_ = os.Remove(socketPath)

	// Create a dummy file to test stale socket cleanup
	_, err := os.Create(socketPath)
	if err != nil {
		t.Fatalf("failed to create stale socket dummy: %v", err)
	}

	listener, err := NewSocketListener(socketPath)
	if err != nil {
		t.Fatalf("failed to create socket listener: %v", err)
	}
	defer listener.Close()

	// Verify that the listener is indeed listening on the socketPath
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		t.Error("expected socket file to exist")
	}

	// Try connecting
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
