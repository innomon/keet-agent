package hypercore

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
)

func TestCrypto_SignatureVerification(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate keys: %v", err)
	}

	rootHash := [32]byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
		0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
		0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20,
	}

	signature := SignRootHash(priv, rootHash)
	if len(signature) != ed25519.SignatureSize {
		t.Errorf("expected signature size %d, got %d", ed25519.SignatureSize, len(signature))
	}

	// Verify valid signature
	if !VerifySignature(pub, rootHash, signature) {
		t.Error("failed to verify valid signature")
	}

	// Modify root hash, signature verification should fail
	badHash := rootHash
	badHash[0] = 0xff
	if VerifySignature(pub, badHash, signature) {
		t.Error("expected verification to fail for invalid root hash")
	}

	// Modify signature, verification should fail
	badSignature := make([]byte, len(signature))
	copy(badSignature, signature)
	badSignature[0] ^= 0xff
	if VerifySignature(pub, rootHash, badSignature) {
		t.Error("expected verification to fail for modified signature")
	}

	// Wrong public key
	_, otherPriv, _ := ed25519.GenerateKey(rand.Reader)
	otherSignature := SignRootHash(otherPriv, rootHash)
	if VerifySignature(pub, rootHash, otherSignature) {
		t.Error("expected verification to fail for wrong public key/signature")
	}
}
