package hypercore

import (
	"bytes"
	"testing"
)

func TestMerkle_LeafHash(t *testing.T) {
	data := []byte("hello block leaf")
	h1 := HashLeaf(data)

	// Modify data, hash should change
	data2 := []byte("hello block leaf modified")
	h2 := HashLeaf(data2)

	if bytes.Equal(h1[:], h2[:]) {
		t.Error("expected different hashes for different data")
	}
}

func TestMerkle_ParentHash(t *testing.T) {
	h1 := HashLeaf([]byte("block 1"))
	h2 := HashLeaf([]byte("block 2"))

	p1 := HashParent(h1, h2)
	p2 := HashParent(h2, h1)

	if bytes.Equal(p1[:], p2[:]) {
		t.Error("expected parent hash to be non-commutative (dependent on order of child nodes)")
	}
}

func TestMerkle_RootHash(t *testing.T) {
	leaves := [][]byte{
		[]byte("leaf 1"),
		[]byte("leaf 2"),
		[]byte("leaf 3"),
		[]byte("leaf 4"),
	}

	root, err := ComputeRootHash(leaves)
	if err != nil {
		t.Fatalf("failed to compute root hash: %v", err)
	}

	if len(root) != 32 {
		t.Errorf("expected 32-byte root hash, got %d", len(root))
	}

	// Computing again with same leaves should yield same root
	root2, _ := ComputeRootHash(leaves)
	if !bytes.Equal(root[:], root2[:]) {
		t.Error("expected consistent root hashes")
	}

	// Empty leaves check
	_, err = ComputeRootHash(nil)
	if err == nil {
		t.Error("expected error for empty leaves, got nil")
	}
}
