package hypercore

import (
	"bytes"
	"testing"
)

func TestExtension_Negotiate(t *testing.T) {
	local := []string{"deflate", "manifest"}
	remote := []string{"deflate", "other"}

	shared := NegotiateExtensions(local, remote)
	if len(shared) != 1 || shared[0] != "deflate" {
		t.Errorf("expected shared extensions [deflate], got %v", shared)
	}
}

func TestExtension_CompressionRoundTrip(t *testing.T) {
	payload := []byte("hello hypercore block replication. this should be compressed and decompressed successfully!")

	compressed, err := CompressBlock(payload)
	if err != nil {
		t.Fatalf("failed to compress block: %v", err)
	}

	if len(compressed) == 0 {
		t.Fatal("compressed block is empty")
	}

	decompressed, err := DecompressBlock(compressed)
	if err != nil {
		t.Fatalf("failed to decompress block: %v", err)
	}

	if !bytes.Equal(decompressed, payload) {
		t.Errorf("decompressed payload mismatch. expected %q, got %q", string(payload), string(decompressed))
	}
}

func TestExtension_DecompressInvalid(t *testing.T) {
	_, err := DecompressBlock([]byte("invalid-compressed-data"))
	if err == nil {
		t.Error("expected error decompressing invalid data")
	}
}
