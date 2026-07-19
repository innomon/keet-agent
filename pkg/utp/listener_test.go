package utp

import (
	"net"
	"testing"
)

func TestUTPListener_AddrAndClose(t *testing.T) {
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer conn.Close()

	mux := NewSocketMux(conn)
	listener := NewUTPListener(mux)

	// Test Addr
	if listener.Addr().String() != conn.LocalAddr().String() {
		t.Errorf("expected address %s, got %s", conn.LocalAddr(), listener.Addr())
	}

	// Test Close
	if err := listener.Close(); err != nil {
		t.Errorf("expected nil error on listener Close, got %v", err)
	}

	// Accept should return error now
	_, err = listener.Accept()
	if err == nil {
		t.Error("expected error when accepting on a closed listener")
	}
}
