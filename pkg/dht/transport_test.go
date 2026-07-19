package dht

import (
	"bytes"
	"net"
	"testing"
	"time"
)

func TestUDPTransport_LifecycleAndExchange(t *testing.T) {
	// Bind transport A as Transport interface
	var t1 Transport
	var err error
	t1, err = NewUDPTransport("127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create UDP transport 1: %v", err)
	}
	defer t1.Close()

	// Bind transport B as Transport interface
	var t2 Transport
	t2, err = NewUDPTransport("127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create UDP transport 2: %v", err)
	}
	defer t2.Close()

	var addr1 net.Addr = t1.Addr()
	var addr2 net.Addr = t2.Addr()

	if addr1 == nil || addr2 == nil {
		t.Fatal("expected non-nil addresses")
	}

	msg := []byte("hello kademlia")

	// Send from t1 to t2
	_, err = t1.WriteTo(msg, addr2)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	// Read on t2 with timeout
	buf := make([]byte, 1024)
	err = t2.(*UDPTransport).conn.SetReadDeadline(time.Now().Add(time.Second))
	if err != nil {
		t.Fatalf("SetReadDeadline failed: %v", err)
	}

	n, raddr, err := t2.ReadFrom(buf)
	if err != nil {
		t.Fatalf("ReadFrom failed: %v", err)
	}

	if n != len(msg) {
		t.Errorf("expected read len %d, got %d", len(msg), n)
	}

	if !bytes.Equal(buf[:n], msg) {
		t.Errorf("expected received message %s, got %s", msg, buf[:n])
	}

	if raddr.String() != addr1.String() {
		t.Errorf("expected sender address %s, got %s", addr1, raddr)
	}
}
