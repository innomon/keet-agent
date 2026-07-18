package hypercore

import (
	"errors"

	"golang.org/x/crypto/blake2b"
)

func HashLeaf(data []byte) [32]byte {
	buf := make([]byte, 1+len(data))
	buf[0] = 0x00
	copy(buf[1:], data)
	return blake2b.Sum256(buf)
}

func HashParent(left, right [32]byte) [32]byte {
	buf := make([]byte, 1+64)
	buf[0] = 0x01
	copy(buf[1:33], left[:])
	copy(buf[33:65], right[:])
	return blake2b.Sum256(buf)
}

func ComputeRootHash(leaves [][]byte) ([32]byte, error) {
	if len(leaves) == 0 {
		return [32]byte{}, errors.New("empty leaves array")
	}

	nodes := make([][32]byte, len(leaves))
	for i, leaf := range leaves {
		nodes[i] = HashLeaf(leaf)
	}

	for len(nodes) > 1 {
		var nextLevel [][32]byte
		for i := 0; i < len(nodes); i += 2 {
			if i+1 < len(nodes) {
				nextLevel = append(nextLevel, HashParent(nodes[i], nodes[i+1]))
			} else {
				// Promote the single odd node
				nextLevel = append(nextLevel, nodes[i])
			}
		}
		nodes = nextLevel
	}

	return nodes[0], nil
}
