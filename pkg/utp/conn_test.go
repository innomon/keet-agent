package utp

import (
	"net"
	"testing"
	"time"
)

func TestConnection_Handshake(t *testing.T) {
	// Setup client UDP socket and server UDP socket
	connA, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen A: %v", err)
	}
	defer connA.Close()

	connB, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen B: %v", err)
	}
	defer connB.Close()

	// Instantiate SocketMux on both sides
	muxA := NewSocketMux(connA)
	muxB := NewSocketMux(connB)

	muxA.Start()
	defer muxA.Stop()

	muxB.Start()
	defer muxB.Stop()

	// Create listener on B
	listenerB := NewUTPListener(muxB)
	defer listenerB.Close()

	// Client dials from A to B
	errChan := make(chan error, 1)
	var clientConn *UTPConn

	go func() {
		var err error
		clientConn, err = DialUTP(muxA, connB.LocalAddr())
		errChan <- err
	}()

	// Server accepts connection on B
	serverConnChan := make(chan net.Conn, 1)
	go func() {
		conn, err := listenerB.Accept()
		if err == nil {
			serverConnChan <- conn
		}
	}()

	// Wait for client dial and server accept to complete
	select {
	case err := <-errChan:
		if err != nil {
			t.Fatalf("DialUTP failed: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("DialUTP timed out")
	}

	var serverConn net.Conn
	select {
	case serverConn = <-serverConnChan:
		// success
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Accept timed out")
	}

	// Verify both connections are in connected state
	if clientConn.state != STATE_CONNECTED {
		t.Errorf("expected client state to be STATE_CONNECTED, got %v", clientConn.state)
	}
	if serverConn.(*UTPConn).state != STATE_CONNECTED {
		t.Errorf("expected server state to be STATE_CONNECTED, got %v", serverConn.(*UTPConn).state)
	}
}

func TestConnection_Teardown(t *testing.T) {
	connA, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen A: %v", err)
	}
	defer connA.Close()

	connB, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen B: %v", err)
	}
	defer connB.Close()

	muxA := NewSocketMux(connA)
	muxB := NewSocketMux(connB)

	muxA.Start()
	defer muxA.Stop()

	muxB.Start()
	defer muxB.Stop()

	listenerB := NewUTPListener(muxB)
	defer listenerB.Close()

	// Client dials B
	errChan := make(chan error, 1)
	var clientConn *UTPConn
	go func() {
		var err error
		clientConn, err = DialUTP(muxA, connB.LocalAddr())
		errChan <- err
	}()

	var serverConn net.Conn
	serverConnChan := make(chan net.Conn, 1)
	go func() {
		conn, err := listenerB.Accept()
		if err == nil {
			serverConnChan <- conn
		}
	}()

	select {
	case err := <-errChan:
		if err != nil {
			t.Fatalf("DialUTP failed: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("DialUTP timed out")
	}

	select {
	case serverConn = <-serverConnChan:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Accept timed out")
	}

	// Now start teardown process
	// Client closes connection
	go func() {
		_ = clientConn.Close()
	}()

	// Wait and verify both connections transition to STATE_CLOSED
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if clientConn.state == STATE_CLOSED && serverConn.(*UTPConn).state == STATE_CLOSED {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if clientConn.state != STATE_CLOSED {
		t.Errorf("expected client state to be STATE_CLOSED, got %v", clientConn.state)
	}
	if serverConn.(*UTPConn).state != STATE_CLOSED {
		t.Errorf("expected server state to be STATE_CLOSED, got %v", serverConn.(*UTPConn).state)
	}
}

