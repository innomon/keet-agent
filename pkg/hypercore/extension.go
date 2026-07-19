package hypercore

import (
	"bytes"
	"compress/zlib"
	"io"
)

// NegotiateExtensions returns the intersection of extensions supported by both sides.
func NegotiateExtensions(local, remote []string) []string {
	var shared []string
	remoteSet := make(map[string]bool)
	for _, ext := range remote {
		remoteSet[ext] = true
	}
	for _, ext := range local {
		if remoteSet[ext] {
			shared = append(shared, ext)
		}
	}
	return shared
}

// CompressBlock compresses block data using zlib.
func CompressBlock(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	_, err := w.Write(data)
	if err != nil {
		return nil, err
	}
	err = w.Close()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// DecompressBlock decompresses block data using zlib.
func DecompressBlock(data []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}
