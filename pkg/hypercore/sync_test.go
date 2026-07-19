package hypercore

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"io"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/innomon/keet-adk-gateway/pkg/crypto"
)

func TestP2PSync_SessionReplication(t *testing.T) {
	// Generate identities for Initiator and Responder
	pubInit, privInit, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	pubResp, privResp, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	// Set up temporary directories for peer flat-file storage
	tempDirA, err := os.MkdirTemp("", "hypercore-sync-a-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDirA)

	tempDirB, err := os.MkdirTemp("", "hypercore-sync-b-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDirB)

	storageA, err := NewStorage(tempDirA)
	if err != nil {
		t.Fatalf("failed to create storage A: %v", err)
	}
	defer storageA.Close()

	storageB, err := NewStorage(tempDirB)
	if err != nil {
		t.Fatalf("failed to create storage B: %v", err)
	}
	defer storageB.Close()

	// Append blocks to A (the sender)
	blocks := [][]byte{
		[]byte("replicated block zero"),
		[]byte("replicated block one"),
		[]byte("replicated block two"),
	}
	for _, block := range blocks {
		if err := storageA.Append(block); err != nil {
			t.Fatalf("failed to append to storage A: %v", err)
		}
	}

	// Create local TCP connection
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer listener.Close()

	var connA, connB net.Conn
	var dialErr error

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		connB, _ = listener.Accept()
	}()

	go func() {
		defer wg.Done()
		connA, dialErr = net.Dial("tcp", listener.Addr().String())
	}()

	wg.Wait()

	if dialErr != nil {
		t.Fatalf("failed to dial: %v", dialErr)
	}
	defer connA.Close()
	defer connB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var secureConnA, secureConnB net.Conn
	var remotePubA, remotePubB ed25519.PublicKey
	var errA, errB error

	wg.Add(2)

	// Initiator handshake (A)
	go func() {
		defer wg.Done()
		secureConnA, remotePubA, errA = crypto.NewSecureConnection(connA, privInit, true)
	}()

	// Responder handshake (B)
	go func() {
		defer wg.Done()
		secureConnB, remotePubB, errB = crypto.NewSecureConnection(connB, privResp, false)
	}()

	wg.Wait()

	if errA != nil {
		t.Fatalf("initiator secure handshake failed: %v", errA)
	}
	if errB != nil {
		t.Fatalf("responder secure handshake failed: %v", errB)
	}

	// Verify remote public key exchange
	if !bytes.Equal(remotePubA, pubResp) {
		t.Errorf("client expected remote public key %x, got %x", pubResp, remotePubA)
	}
	if !bytes.Equal(remotePubB, pubInit) {
		t.Errorf("server expected remote public key %x, got %x", pubInit, remotePubB)
	}

	// Run replication sessions
	feedKey := "sync_test_feed_key"
	sessionA := NewSyncSession(secureConnA, storageA, nil, feedKey, privInit, remotePubA, true)
	sessionB := NewSyncSession(secureConnB, storageB, nil, feedKey, privResp, remotePubB, false)

	errChan := make(chan error, 2)

	go func() {
		errChan <- sessionA.Run(ctx)
	}()

	go func() {
		errChan <- sessionB.Run(ctx)
	}()

	// Wait for storage B to replicate all 3 blocks
	retries := 50
	for i := 0; i < retries; i++ {
		if storageB.Len() >= 3 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if storageB.Len() < 3 {
		t.Errorf("replication timeout: expected B to have 3 blocks, got %d", storageB.Len())
	} else {
		// Verify replicated block values
		for i := uint64(0); i < 3; i++ {
			val, err := storageB.Get(i)
			if err != nil {
				t.Errorf("failed to retrieve replicated block %d: %v", i, err)
			}
			if !bytes.Equal(val, blocks[i]) {
				t.Errorf("block %d mismatch: expected %q, got %q", i, string(blocks[i]), string(val))
			}
		}
	}

	// Stop connection and wait for run routines to exit
	secureConnA.Close()
	secureConnB.Close()

	// Flush error channels
	for i := 0; i < 2; i++ {
		select {
		case err := <-errChan:
			if err != nil && err != io.EOF && !bytes.Contains([]byte(err.Error()), []byte("use of closed network connection")) {
				t.Logf("session exit status: %v", err)
			}
		case <-time.After(100 * time.Millisecond):
		}
	}
}
