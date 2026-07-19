package crypto

import (
	"bytes"
	"crypto/ed25519"
	"os"
	"path/filepath"
	"testing"
)

func TestNodeKey_Persistence(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "node-key-test-*")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 1. Generate key first time
	priv1, err := LoadOrGenerateNodeKey(tempDir)
	if err != nil {
		t.Fatalf("first generation failed: %v", err)
	}

	if len(priv1) != ed25519.PrivateKeySize {
		t.Errorf("expected key size %d, got %d", ed25519.PrivateKeySize, len(priv1))
	}

	keyFile := filepath.Join(tempDir, "node_key.priv")
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		t.Error("expected node_key.priv file to be created")
	}

	// 2. Load key second time (must match first key)
	priv2, err := LoadOrGenerateNodeKey(tempDir)
	if err != nil {
		t.Fatalf("loading key failed: %v", err)
	}

	if !bytes.Equal(priv1, priv2) {
		t.Error("expected loaded key to be identical to first generated key")
	}
}
