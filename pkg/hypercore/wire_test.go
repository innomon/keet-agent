package hypercore

import (
	"bytes"
	"testing"
)

func TestWire_Handshake(t *testing.T) {
	msg := &Handshake{
		Protocol: "hypercore/v10",
		Key:      []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
	}

	data, err := EncodeHandshake(msg)
	if err != nil {
		t.Fatalf("failed to encode handshake: %v", err)
	}

	decoded, err := DecodeHandshake(data)
	if err != nil {
		t.Fatalf("failed to decode handshake: %v", err)
	}

	if decoded.Protocol != msg.Protocol {
		t.Errorf("expected protocol %q, got %q", msg.Protocol, decoded.Protocol)
	}

	if !bytes.Equal(decoded.Key, msg.Key) {
		t.Errorf("expected key %v, got %v", msg.Key, decoded.Key)
	}
}

func TestWire_Have(t *testing.T) {
	msg := &Have{
		Start: 10,
		Len:   5,
	}

	data, err := EncodeHave(msg)
	if err != nil {
		t.Fatalf("failed to encode Have: %v", err)
	}

	decoded, err := DecodeHave(data)
	if err != nil {
		t.Fatalf("failed to decode Have: %v", err)
	}

	if decoded.Start != msg.Start || decoded.Len != msg.Len {
		t.Errorf("expected Have %+v, got %+v", msg, decoded)
	}
}

func TestWire_Want(t *testing.T) {
	msg := &Want{
		Start: 20,
		Len:   10,
	}

	data, err := EncodeWant(msg)
	if err != nil {
		t.Fatalf("failed to encode Want: %v", err)
	}

	decoded, err := DecodeWant(data)
	if err != nil {
		t.Fatalf("failed to decode Want: %v", err)
	}

	if decoded.Start != msg.Start || decoded.Len != msg.Len {
		t.Errorf("expected Want %+v, got %+v", msg, decoded)
	}
}

func TestWire_Request(t *testing.T) {
	msg := &Request{
		Index: 42,
	}

	data, err := EncodeRequest(msg)
	if err != nil {
		t.Fatalf("failed to encode Request: %v", err)
	}

	decoded, err := DecodeRequest(data)
	if err != nil {
		t.Fatalf("failed to decode Request: %v", err)
	}

	if decoded.Index != msg.Index {
		t.Errorf("expected Request index %d, got %d", msg.Index, decoded.Index)
	}
}

func TestWire_Data(t *testing.T) {
	msg := &Data{
		Index:     100,
		Value:     []byte("hello block data"),
		Signature: []byte("sign"),
	}

	data, err := EncodeData(msg)
	if err != nil {
		t.Fatalf("failed to encode Data: %v", err)
	}

	decoded, err := DecodeData(data)
	if err != nil {
		t.Fatalf("failed to decode Data: %v", err)
	}

	if decoded.Index != msg.Index {
		t.Errorf("expected index %d, got %d", msg.Index, decoded.Index)
	}

	if !bytes.Equal(decoded.Value, msg.Value) {
		t.Errorf("expected value %s, got %s", string(msg.Value), string(decoded.Value))
	}

	if !bytes.Equal(decoded.Signature, msg.Signature) {
		t.Errorf("expected signature %v, got %v", msg.Signature, decoded.Signature)
	}
}

func TestWire_DecodeErrors(t *testing.T) {
	// 1. Invalid message type on DecodeHandshake
	if _, err := DecodeHandshake([]byte{99}); err == nil {
		t.Error("expected error for wrong type in handshake")
	}

	// 2. Invalid Have type
	if _, err := DecodeHave([]byte{99}); err == nil {
		t.Error("expected error for wrong type in Have")
	}

	// 3. Invalid Want type
	if _, err := DecodeWant([]byte{99}); err == nil {
		t.Error("expected error for wrong type in Want")
	}

	// 4. Invalid Request type
	if _, err := DecodeRequest([]byte{99}); err == nil {
		t.Error("expected error for wrong type in Request")
	}

	// 5. Invalid Data type
	if _, err := DecodeData([]byte{99}); err == nil {
		t.Error("expected error for wrong type in Data")
	}
}

func TestWire_ReadStringBytesSafetyLimits(t *testing.T) {
	// Handshake header: type 0 (1 byte)
	// Protocol string size: 15MB (0x00F00000) (4 bytes)
	largeLength := []byte{0, 0x00, 0xf0, 0x00, 0x00}
	if _, err := DecodeHandshake(largeLength); err == nil {
		t.Error("expected safety limit error for large protocol string")
	}
}

