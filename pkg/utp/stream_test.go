package utp

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"
)

func TestUTPConn_StreamReadWrite(t *testing.T) {
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

	// Dial B
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

	defer clientConn.Close()
	defer serverConn.Close()

	// Write data from client to server
	sendData := []byte("stream test data representing a longer payload split across packets")
	writeChan := make(chan error, 1)
	go func() {
		_, err := clientConn.Write(sendData)
		writeChan <- err
	}()

	// Read on server
	readBuf := make([]byte, len(sendData))
	readChan := make(chan error, 1)
	var totalRead int
	go func() {
		var err error
		totalRead, err = io.ReadFull(serverConn, readBuf)
		readChan <- err
	}()

	// Wait and verify
	select {
	case err := <-writeChan:
		if err != nil {
			t.Fatalf("client Write failed: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("client Write timed out")
	}

	select {
	case err := <-readChan:
		if err != nil {
			t.Fatalf("server Read failed: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("server Read timed out")
	}

	if totalRead != len(sendData) {
		t.Errorf("expected read length %d, got %d", len(sendData), totalRead)
	}

	if !bytes.Equal(readBuf, sendData) {
		t.Errorf("expected read data %s, got %s", sendData, readBuf)
	}
}

func TestUTPConn_Deadlines(t *testing.T) {
	connA, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer connA.Close()

	mux := NewSocketMux(connA)
	mux.Start()
	defer mux.Stop()

	// Create dummy conn with state STATE_CONNECTED
	c := &UTPConn{
		state:     STATE_CONNECTED,
		mux:       mux,
		readBuf:   make(chan *Packet, 100),
		closeChan: make(chan struct{}),
	}
	
	// Test Read Deadline
	_ = c.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
	buf := make([]byte, 10)
	startTime := time.Now()
	_, err = c.Read(buf)
	duration := time.Since(startTime)

	if err == nil {
		t.Error("expected timeout error on read deadline, got nil")
	}
	if duration < 10*time.Millisecond {
		t.Errorf("expected deadline delay to be at least 10ms, got %v", duration)
	}
}
