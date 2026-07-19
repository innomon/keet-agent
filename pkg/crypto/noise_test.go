package crypto

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"testing"
)

func TestNoiseHandshake(t *testing.T) {
	// Generate Ed25519 keys
	pubInit, privInit, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	pubResp, privResp, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	// Create in-memory pipe for connection
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	errChan := make(chan error, 2)
	var clientSecConn, serverSecConn net.Conn
	var clientRemotePub, serverRemotePub ed25519.PublicKey

	go func() {
		secConn, remotePub, err := NewSecureConnection(client, privInit, true)
		if err != nil {
			errChan <- err
			return
		}
		clientSecConn = secConn
		clientRemotePub = remotePub
		errChan <- nil
	}()

	go func() {
		secConn, remotePub, err := NewSecureConnection(server, privResp, false)
		if err != nil {
			errChan <- err
			return
		}
		serverSecConn = secConn
		serverRemotePub = remotePub
		errChan <- nil
	}()

	// Wait for handshake
	for i := 0; i < 2; i++ {
		if err := <-errChan; err != nil {
			t.Fatalf("handshake failed: %v", err)
		}
	}

	// Verify remote public key exchange
	if !bytes.Equal(clientRemotePub, pubResp) {
		t.Errorf("client expected remote public key %x, got %x", pubResp, clientRemotePub)
	}
	if !bytes.Equal(serverRemotePub, pubInit) {
		t.Errorf("server expected remote public key %x, got %x", pubInit, serverRemotePub)
	}

	// Test read/write
	msg := []byte("hello noise P2P secure channel")
	go func() {
		_, err := clientSecConn.Write(msg)
		if err != nil {
			errChan <- err
		}
	}()

	buf := make([]byte, 100)
	n, err := serverSecConn.Read(buf)
	if err != nil {
		t.Fatalf("failed to read: %v", err)
	}

	if string(buf[:n]) != string(msg) {
		t.Errorf("expected msg %q, got %q", string(msg), string(buf[:n]))
	}
}
