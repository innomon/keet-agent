package network

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"net"
	"testing"
	"time"
)

func TestPeerManager_RelayFallback(t *testing.T) {
	// Start mock TURN server
	turnConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen mock TURN server: %v", err)
	}
	defer turnConn.Close()

	// Handle mock TURN allocate requests
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
				header := []byte{
					0x01, 0x03, // Success Response
					0x00, 0x0c, // Length
					0x21, 0x12, 0xa4, 0x42, // Cookie
				}
				header = append(header, txID...)

				udpAddr := turnConn.LocalAddr().(*net.UDPAddr)
				portBytes := make([]byte, 2)
				binary.BigEndian.PutUint16(portBytes, uint16(udpAddr.Port)^0x2112)
				ipBytes := make([]byte, 4)
				binary.BigEndian.PutUint32(ipBytes, 0x7f000001^0x2112A442)

				attr := []byte{
					0x00, 0x16,
					0x00, 0x08,
					0x00, 0x01, // IPv4
				}
				attr = append(attr, portBytes...)
				attr = append(attr, ipBytes...)

				resp := append(header, attr...)
				_, _ = turnConn.WriteTo(resp, addr)
			}
		}
	}()

	// Instantiate PeerManager with the relay server configured
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	pm := NewPeerManager(priv, nil, nil, "feedKey")
	pm.SetRelayServer(turnConn.LocalAddr().String())

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Try dialing a non-existent peer
	err = pm.DialPeer(ctx, "127.0.0.1:9999")
	if err == nil {
		t.Error("expected dial to fail on non-existent peer, but succeeded")
	}
}
