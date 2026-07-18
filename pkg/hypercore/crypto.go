package hypercore

import "crypto/ed25519"

func SignRootHash(privKey ed25519.PrivateKey, rootHash [32]byte) []byte {
	return ed25519.Sign(privKey, rootHash[:])
}

func VerifySignature(pubKey ed25519.PublicKey, rootHash [32]byte, signature []byte) bool {
	return ed25519.Verify(pubKey, rootHash[:], signature)
}
