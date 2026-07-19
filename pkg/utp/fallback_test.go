package utp

import (
	"encoding/binary"
	"net"
	"testing"
	"time"
)

func TestTURN_FallbackOnTimeout(t *testing.T) {
	// Start a mock TURN server on a local UDP address
	turnConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen TURN server: %v", err)
	}
	defer turnConn.Close()

	// Mock TURN server response loop
	go func() {
		buf := make([]byte, 1024)
		for {
			_, addr, err := turnConn.ReadFrom(buf)
			if err != nil {
				return
			}
			msgType := uint16(buf[0])<<8 | uint16(buf[1])
			if msgType == 0x0003 { // Allocate Request
				txID := buf[8:20]
				// Respond with success and relayed address = turnConn.LocalAddr()
				header := []byte{
					0x01, 0x03, // Success Response
					0x00, 0x0c, // Length
					0x21, 0x12, 0xa4, 0x42, // Cookie
				}
				header = append(header, txID...)

				// Attribute: XOR-RELAYED-ADDRESS (0x0016), Length (8)
				// Port = turnConn port. Address = 127.0.0.1
				udpAddr := turnConn.LocalAddr().(*net.UDPAddr)
				portBytes := make([]byte, 2)
				binary.BigEndian.PutUint16(portBytes, uint16(udpAddr.Port)^(stunMagicCookie>>16))
				ipBytes := make([]byte, 4)
				binary.BigEndian.PutUint32(ipBytes, 0x7f000001^stunMagicCookie)

				attr := []byte{
					0x00, 0x16,
					0x00, 0x08,
					0x00, 0x01, // Reserved + Family
				}
				attr = append(attr, portBytes...)
				attr = append(attr, ipBytes...)

				resp := append(header, attr...)
				_, _ = turnConn.WriteTo(resp, addr)
			}
		}
	}()

	// Try dialing with a very short handshake timeout and fallback server configured
	localConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer localConn.Close()

	mux := NewSocketMux(localConn)
	mux.Start()
	defer mux.Stop()

	// Configure relay on multiplexer
	mux.SetRelayServer(turnConn.LocalAddr().String())

	// Try dialing a non-existent peer address (which will timeout and trigger fallback)
	nonExistentAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:9999")
	startTime := time.Now()
	conn, err := DialUTPWithTimeoutAndRelay(mux, nonExistentAddr, 100*time.Millisecond)
	duration := time.Since(startTime)

	if err == nil {
		conn.Close()
		t.Error("expected dial to fail on non-existent peer, but succeeded")
	}

	if duration < 100*time.Millisecond {
		t.Errorf("expected timeout delay to take at least 100ms, got %v", duration)
	}
}
