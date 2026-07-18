package hypercore

import (
	"bytes"
	"os"
	"testing"
)

func TestStorage_AppendGet(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hypercore_storage_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := NewStorage(tempDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer storage.Close()

	if storage.Len() != 0 {
		t.Errorf("expected empty storage, got len %d", storage.Len())
	}

	block1 := []byte("first log block payload")
	block2 := []byte("second block with different length")

	if err := storage.Append(block1); err != nil {
		t.Fatalf("failed to append block 1: %v", err)
	}
	if storage.Len() != 1 {
		t.Errorf("expected len 1, got %d", storage.Len())
	}

	if err := storage.Append(block2); err != nil {
		t.Fatalf("failed to append block 2: %v", err)
	}
	if storage.Len() != 2 {
		t.Errorf("expected len 2, got %d", storage.Len())
	}

	// Retrieve blocks
	b1, err := storage.Get(0)
	if err != nil {
		t.Fatalf("failed to get block 0: %v", err)
	}
	if !bytes.Equal(b1, block1) {
		t.Errorf("expected %q, got %q", string(block1), string(b1))
	}

	b2, err := storage.Get(1)
	if err != nil {
		t.Fatalf("failed to get block 1: %v", err)
	}
	if !bytes.Equal(b2, block2) {
		t.Errorf("expected %q, got %q", string(block2), string(b2))
	}

	// Out of bounds check
	_, err = storage.Get(2)
	if err == nil {
		t.Error("expected error getting out of bounds index, got nil")
	}
}

func TestStorage_Persistence(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hypercore_persist_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Write block
	storage1, err := NewStorage(tempDir)
	if err != nil {
		t.Fatalf("failed to create storage1: %v", err)
	}
	block := []byte("persistent payload data")
	if err := storage1.Append(block); err != nil {
		t.Fatalf("append failed: %v", err)
	}
	storage1.Close()

	// Reopen
	storage2, err := NewStorage(tempDir)
	if err != nil {
		t.Fatalf("failed to reopen storage2: %v", err)
	}
	defer storage2.Close()

	if storage2.Len() != 1 {
		t.Errorf("expected len 1 on reopen, got %d", storage2.Len())
	}

	b, err := storage2.Get(0)
	if err != nil {
		t.Fatalf("failed to get block on reopen: %v", err)
	}
	if !bytes.Equal(b, block) {
		t.Errorf("expected %q, got %q", string(block), string(b))
	}
}
