package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
)

// LoadOrGenerateNodeKey loads an Ed25519 private key from node_key.priv in storageDir,
// or generates and saves one if it does not exist.
func LoadOrGenerateNodeKey(storageDir string) (ed25519.PrivateKey, error) {
	keyPath := filepath.Join(storageDir, "node_key.priv")

	// Check if already exists
	data, err := os.ReadFile(keyPath)
	if err == nil {
		if len(data) == ed25519.PrivateKeySize {
			return ed25519.PrivateKey(data), nil
		}
	}

	// Generate a new one
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	// Ensure directory exists
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return nil, err
	}

	// Save key to file
	if err := os.WriteFile(keyPath, priv, 0600); err != nil {
		return nil, err
	}

	return priv, nil
}
