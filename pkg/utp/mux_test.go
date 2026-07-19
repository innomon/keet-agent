package utp

import (
	"bytes"
	"net"
	"testing"
	"time"
)

type mockPacketConn struct {
	net.PacketConn
	readBuf  chan []byte
	readAddr chan net.Addr
}

func newMockPacketConn() *mockPacketConn {
	return &mockPacketConn{
		readBuf:  make(chan []byte, 100),
		readAddr: make(chan net.Addr, 100),
	}
}

func (m *mockPacketConn) ReadFrom(p []byte) (int, net.Addr, error) {
	select {
	case data := <-m.readBuf:
		addr := <-m.readAddr
		copy(p, data)
		return len(data), addr, nil
	case <-time.After(100 * time.Millisecond):
		return 0, nil, net.ErrClosed
	}
}

func (m *mockPacketConn) Close() error {
	return nil
}

func (m *mockPacketConn) LocalAddr() net.Addr {
	return &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}
}

func TestSocketMux_Demux(t *testing.T) {
	conn := newMockPacketConn()
	mux := NewSocketMux(conn)
	
	// Create a dummy receiver queue for connection ID 100
	packetChan := make(chan *Packet, 10)
	err := mux.RegisterConn(100, packetChan)
	if err != nil {
		t.Fatalf("failed to register connection: %v", err)
	}

	mux.Start()
	defer mux.Stop()

	// Prepare an encoded data packet with ConnID = 100
	pkt := &Packet{
		Header: Header{
			Type:    ST_DATA,
			Version: 1,
			ConnID:  100,
			SeqNum:  5,
		},
		Payload: []byte("demux test"),
	}
	encoded, _ := pkt.Encode()

	// Inject into mock connection
	srcAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 54321}
	conn.readBuf <- encoded
	conn.readAddr <- srcAddr

	// Wait and receive packet on registered queue
	select {
	case received := <-packetChan:
		if received.Header.ConnID != 100 {
			t.Errorf("expected Connection ID 100, got %d", received.Header.ConnID)
		}
		if !bytes.Equal(received.Payload, pkt.Payload) {
			t.Errorf("expected payload %s, got %s", pkt.Payload, received.Payload)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("timeout waiting for packet demultiplexing")
	}
}

func TestSocketMux_DuplicateRegistration(t *testing.T) {
	conn := newMockPacketConn()
	mux := NewSocketMux(conn)
	
	ch1 := make(chan *Packet, 5)
	ch2 := make(chan *Packet, 5)

	if err := mux.RegisterConn(200, ch1); err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	if err := mux.RegisterConn(200, ch2); err == nil {
		t.Error("expected error when registering duplicate Connection ID")
	}
}
