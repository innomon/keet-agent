package dht

import (
	"bytes"
	"testing"
)

func TestInProcessTransport_Exchange(t *testing.T) {
	// Create node A
	t1, err := NewInProcessTransport("nodeA")
	if err != nil {
		t.Fatalf("failed to create nodeA: %v", err)
	}
	defer t1.Close()

	// Create node B
	t2, err := NewInProcessTransport("nodeB")
	if err != nil {
		t.Fatalf("failed to create nodeB: %v", err)
	}
	defer t2.Close()

	msg := []byte("hello stub")

	// Send from t1 to t2
	_, err = t1.WriteTo(msg, t2.Addr())
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	// Read on t2
	buf := make([]byte, 1024)
	n, raddr, err := t2.ReadFrom(buf)
	if err != nil {
		t.Fatalf("ReadFrom failed: %v", err)
	}

	if n != len(msg) {
		t.Errorf("expected read len %d, got %d", len(msg), n)
	}

	if !bytes.Equal(buf[:n], msg) {
		t.Errorf("expected msg %s, got %s", msg, buf[:n])
	}

	if raddr.String() != "nodeA" {
		t.Errorf("expected sender address nodeA, got %s", raddr.String())
	}
}

func TestInProcessTransport_DuplicateBind(t *testing.T) {
	t1, err := NewInProcessTransport("nodeX")
	if err != nil {
		t.Fatalf("first bind failed: %v", err)
	}
	defer t1.Close()

	_, err = NewInProcessTransport("nodeX")
	if err == nil {
		t.Fatal("expected error on duplicate bind, got nil")
	}
}

func TestInProcessTransport_Close(t *testing.T) {
	t1, err := NewInProcessTransport("nodeY")
	if err != nil {
		t.Fatalf("bind failed: %v", err)
	}
	t1.Close()

	buf := make([]byte, 1024)
	_, _, err = t1.ReadFrom(buf)
	if err == nil {
		t.Fatal("expected error reading from closed transport, got nil")
	}

	// Send to closed
	t2, err := NewInProcessTransport("nodeZ")
	if err != nil {
		t.Fatalf("bind failed: %v", err)
	}
	defer t2.Close()

	_, err = t2.WriteTo([]byte("ping"), inProcessAddr{addr: "nodeY"})
	if err == nil {
		t.Fatal("expected error sending to closed address, got nil")
	}
}
