package db

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/innomon/keet-adk-gateway/pkg/config"
)

func TestBlockRepo_PutGet(t *testing.T) {
	cfg := config.LoadConfig()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := Connect(ctx, cfg)
	if err != nil {
		if os.Getenv("DB_HOST") == "" {
			t.Skipf("PostgreSQL is not running, skipping block repo test: %v", err)
		} else {
			t.Fatalf("failed to connect: %v", err)
		}
	}
	defer db.Close()

	if err := RunMigrations(ctx, db); err != nil {
		t.Fatalf("migrations failed: %v", err)
	}

	repo := NewBlockRepository(db)

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
}
