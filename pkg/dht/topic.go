package dht

import (
	"encoding/hex"
	"errors"

	"golang.org/x/crypto/blake2b"
)

func ResolveTopicKey(topic string) ([32]byte, error) {
	if topic == "" {
		return [32]byte{}, errors.New("empty topic is invalid")
	}

	// If the topic is a 64-character hex string, parse it directly
	if len(topic) == 64 {
		decoded, err := hex.DecodeString(topic)
		if err == nil && len(decoded) == 32 {
			var key [32]byte
			copy(key[:], decoded)
			return key, nil
		}
	}

	// Otherwise, generate key using Blake2b-256
	hash := blake2b.Sum256([]byte(topic))
	return hash, nil
}
