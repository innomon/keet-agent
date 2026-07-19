package network

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"net"
	"os"
	"testing"
	"time"

	"github.com/innomon/keet-adk-gateway/pkg/hypercore"
)

func TestE2E_TraversalAndReplication(t *testing.T) {
	// 1. Start mock TURN server
	turnConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen TURN: %v", err)
	}
	defer turnConn.Close()

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
					0x00, 0x01,
				}
				attr = append(attr, portBytes...)
				attr = append(attr, ipBytes...)

				resp := append(header, attr...)
				_, _ = turnConn.WriteTo(resp, addr)
			}
		}
	}()

	// 2. Set up flat-file storages for Node A and Node B
	dirA, _ := os.MkdirTemp("", "e2e_storage_a_*")
	dirB, _ := os.MkdirTemp("", "e2e_storage_b_*")
	defer os.RemoveAll(dirA)
	defer os.RemoveAll(dirB)

	storageA, _ := hypercore.NewStorage(dirA)
	storageB, _ := hypercore.NewStorage(dirB)
	defer storageA.Close()
	defer storageB.Close()

	// Append initial block to A
	_ = storageA.Append([]byte("e2e-payload-data"))

	// 3. Initialize PeerManagers
	pubA, privA, _ := ed25519.GenerateKey(rand.Reader)
	pubB, privB, _ := ed25519.GenerateKey(rand.Reader)
	_ = pubA
	_ = pubB

	feedKey := "e2e_traversal_feed_key_which_must_be_32_bytes_long!!"[:32]
	pmB := NewPeerManager(privB, storageB, nil, feedKey)
	pmB.SetRelayServer(turnConn.LocalAddr().String())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start B listener on random port
	err = pmB.StartListener(ctx, "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start listener B: %v", err)
	}
	defer pmB.listener.Close()

	pmA := NewPeerManager(privA, storageA, nil, feedKey)
	pmA.SetRelayServer(turnConn.LocalAddr().String())

	// 4. Dial B from A
	err = pmA.DialPeer(ctx, pmB.listener.Addr().String())
	if err != nil {
		t.Fatalf("failed to dial peer: %v", err)
	}

	// 5. Wait and verify replication success
	retries := 20
	for i := 0; i < retries; i++ {
		if storageB.Len() >= 1 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if storageB.Len() < 1 {
		t.Errorf("E2E replication failed: B did not receive block")
	} else {
		val, _ := storageB.Get(0)
		if string(val) != "e2e-payload-data" {
			t.Errorf("unexpected replicated block content: %s", string(val))
		}
	}
}
