package network

import (
	"bytes"
	"testing"
)

func TestProtobuf_FeedRoundTrip(t *testing.T) {
	dkey := make([]byte, 32)
	for i := range dkey {
		dkey[i] = byte(i)
	}

	feed := Feed{
		DiscoveryKey: dkey,
	}

	data, err := feed.Marshal()
	if err != nil {
		t.Fatalf("failed to marshal Feed: %v", err)
	}

	var parsed Feed
	err = parsed.Unmarshal(data)
	if err != nil {
		t.Fatalf("failed to unmarshal Feed: %v", err)
	}

	if !bytes.Equal(parsed.DiscoveryKey, feed.DiscoveryKey) {
		t.Error("discoveryKey mismatch")
	}
}

func TestProtobuf_HandshakeConstraints(t *testing.T) {
	// 1. Missing protocol should fail validation
	h := Handshake{
		Protocol: "",
		Key:      make([]byte, 32),
	}
	if err := h.Validate(); err == nil {
		t.Error("expected validation error for empty protocol")
	}

	// 2. key must be 32 bytes
	h2 := Handshake{
		Protocol: "hypercore/v1",
		Key:      make([]byte, 16),
	}
	if err := h2.Validate(); err == nil {
		t.Error("expected validation error for short key")
	}
}

func TestProtobuf_HandshakeRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	key[0] = 42

	h := Handshake{
		Protocol:   "hypercore/v1",
		Key:        key,
		Extensions: []string{"extension1", "extension2"},
		Live:       true,
		UserData:   []byte("test-user-data"),
	}

	data, err := h.Marshal()
	if err != nil {
		t.Fatalf("failed to marshal Handshake: %v", err)
	}

	// Validate max frame limit check
	if len(data) > MaxProtobufFrameLength {
		t.Fatalf("serialized handshake exceeds max frame length")
	}

	var parsed Handshake
	err = parsed.Unmarshal(data)
	if err != nil {
		t.Fatalf("failed to unmarshal Handshake: %v", err)
	}

	if parsed.Protocol != h.Protocol {
		t.Errorf("protocol mismatch: %s vs %s", parsed.Protocol, h.Protocol)
	}
	if !bytes.Equal(parsed.Key, h.Key) {
		t.Error("key mismatch")
	}
	if len(parsed.Extensions) != len(h.Extensions) || parsed.Extensions[0] != h.Extensions[0] {
		t.Error("extensions mismatch")
	}
	if parsed.Live != h.Live {
		t.Error("live mismatch")
	}
	if !bytes.Equal(parsed.UserData, h.UserData) {
		t.Error("userData mismatch")
	}
}

func TestProtobuf_RequestRoundTrip(t *testing.T) {
	req := Request{
		Index: 123,
		Bytes: 456,
		Hash:  true,
		Nodes: 3,
	}

	data, err := req.Marshal()
	if err != nil {
		t.Fatalf("failed to marshal Request: %v", err)
	}

	var parsed Request
	err = parsed.Unmarshal(data)
	if err != nil {
		t.Fatalf("failed to unmarshal Request: %v", err)
	}

	if parsed.Index != req.Index || parsed.Bytes != req.Bytes || parsed.Hash != req.Hash || parsed.Nodes != req.Nodes {
		t.Error("Request roundtrip mismatch")
	}
}

func TestProtobuf_DataRoundTrip(t *testing.T) {
	d := Data{
		Index:     789,
		Value:     []byte("hello hypercore data"),
		Signature: []byte("fake-signature-bytes-which-should-be-long"),
	}

	data, err := d.Marshal()
	if err != nil {
		t.Fatalf("failed to marshal Data: %v", err)
	}

	var parsed Data
	err = parsed.Unmarshal(data)
	if err != nil {
		t.Fatalf("failed to unmarshal Data: %v", err)
	}

	if parsed.Index != d.Index || !bytes.Equal(parsed.Value, d.Value) || !bytes.Equal(parsed.Signature, d.Signature) {
		t.Error("Data roundtrip mismatch")
	}
}

func TestProtobuf_CancelRoundTrip(t *testing.T) {
	c := Cancel{
		Index: 12,
		Bytes: 34,
	}

	data, err := c.Marshal()
	if err != nil {
		t.Fatalf("failed to marshal Cancel: %v", err)
	}

	var parsed Cancel
	err = parsed.Unmarshal(data)
	if err != nil {
		t.Fatalf("failed to unmarshal Cancel: %v", err)
	}

	if parsed.Index != c.Index || parsed.Bytes != c.Bytes {
		t.Error("Cancel roundtrip mismatch")
	}
}

func TestProtobuf_ValidationLimits(t *testing.T) {
	// Frame size limit verification (8MB)
	hugeData := make([]byte, MaxProtobufFrameLength+1)
	d := Data{
		Index: 1,
		Value: hugeData,
	}
	_, err := d.Marshal()
	if err == nil {
		t.Error("expected error marshaling frame exceeding MaxProtobufFrameLength")
	}

	// Unmarshaling too large packet should error
	err = d.Unmarshal(hugeData)
	if err == nil {
		t.Error("expected error unmarshaling frame exceeding MaxProtobufFrameLength")
	}
}
