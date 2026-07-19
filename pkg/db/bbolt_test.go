package db

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestBoltDB_SwarmRepo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bbolt-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	boltDB, err := NewBoltDB(dbPath)
	if err != nil {
		t.Fatalf("failed to open BoltDB: %v", err)
	}
	defer boltDB.Close()

	repo := NewBoltSwarmRepository(boltDB)
	ctx := context.Background()

	topicKey := "test-topic-key-123"
	topicName := "Test Swarm Room"

	// Register
	if err := repo.RegisterSwarm(ctx, topicKey, topicName); err != nil {
		t.Fatalf("failed to register swarm: %v", err)
	}

	// Retrieve
	active, err := repo.GetActiveSwarms(ctx)
	if err != nil {
		t.Fatalf("failed to get active swarms: %v", err)
	}

	found := false
	for _, k := range active {
		if k == topicKey {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected registered swarm topic %q in active swarms list %+v", topicKey, active)
	}

	// Unregister
	if err := repo.UnregisterSwarm(ctx, topicKey); err != nil {
		t.Fatalf("failed to unregister: %v", err)
	}

	// Retrieve again
	active, err = repo.GetActiveSwarms(ctx)
	if err != nil {
		t.Fatalf("failed to get active swarms after unregister: %v", err)
	}

	found = false
	for _, k := range active {
		if k == topicKey {
			found = true
			break
		}
	}
	if found {
		t.Errorf("expected unregistered swarm topic to be removed, but still found in list %+v", active)
	}
}

func TestBoltDB_BlockRepo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bbolt-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	boltDB, err := NewBoltDB(dbPath)
	if err != nil {
		t.Fatalf("failed to open BoltDB: %v", err)
	}
	defer boltDB.Close()

	repo := NewBoltBlockRepository(boltDB)
	ctx := context.Background()

	feedKey := "test_feed_key"
	var blockIndex uint64 = 42
	value := []byte("block data value hello")
	signature := []byte("block signature bytes ed25519")

	// Put
	if err := repo.PutBlock(ctx, feedKey, blockIndex, value, signature); err != nil {
		t.Fatalf("failed to put block: %v", err)
	}

	// Get
	v, s, err := repo.GetBlock(ctx, feedKey, blockIndex)
	if err != nil {
		t.Fatalf("failed to get block: %v", err)
	}

	if !bytes.Equal(v, value) {
		t.Errorf("expected block value %q, got %q", string(value), string(v))
	}
	if !bytes.Equal(s, signature) {
		t.Errorf("expected signature %v, got %v", signature, s)
	}

	// Non-existent get
	_, _, err = repo.GetBlock(ctx, feedKey, 999)
	if err == nil {
		t.Error("expected error retrieving non-existent block, got nil")
	}

	// Check serialization boundaries and corrupted data
	t.Run("Invalid data length", func(t *testing.T) {
		_, _, err := decodeBlock([]byte{1, 2, 3})
		if err == nil {
			t.Error("expected decode error on short data, got nil")
		}
	})
}
