package utp

import (
	"bytes"
	"testing"
)

func TestPacket_SerializationRoundTrip(t *testing.T) {
	// Create a sample packet
	p := &Packet{
		Header: Header{
			Type:    ST_DATA,
			Version: 1,
			Ext:     0,
			ConnID:  1234,
			TMicro:  56789,
			TDiff:   9876,
			WNDSize: 65536,
			SeqNum:  10,
			AckNum:  9,
		},
		Payload: []byte("hello utp!"),
	}

	// Encode packet
	encoded, err := p.Encode()
	if err != nil {
		t.Fatalf("failed to encode packet: %v", err)
	}

	// Total size should be 20 bytes header + 10 bytes payload = 30 bytes
	expectedLen := 20 + len(p.Payload)
	if len(encoded) != expectedLen {
		t.Errorf("expected encoded length %d, got %d", expectedLen, len(encoded))
	}

	// Decode packet
	decoded, err := DecodePacket(encoded)
	if err != nil {
		t.Fatalf("failed to decode packet: %v", err)
	}

	// Verify header fields
	if decoded.Header.Type != p.Header.Type {
		t.Errorf("expected type %d, got %d", p.Header.Type, decoded.Header.Type)
	}
	if decoded.Header.Version != p.Header.Version {
		t.Errorf("expected version %d, got %d", p.Header.Version, decoded.Header.Version)
	}
	if decoded.Header.Ext != p.Header.Ext {
		t.Errorf("expected ext %d, got %d", p.Header.Ext, decoded.Header.Ext)
	}
	if decoded.Header.ConnID != p.Header.ConnID {
		t.Errorf("expected connID %d, got %d", p.Header.ConnID, decoded.Header.ConnID)
	}
	if decoded.Header.TMicro != p.Header.TMicro {
		t.Errorf("expected tMicro %d, got %d", p.Header.TMicro, decoded.Header.TMicro)
	}
	if decoded.Header.TDiff != p.Header.TDiff {
		t.Errorf("expected tDiff %d, got %d", p.Header.TDiff, decoded.Header.TDiff)
	}
	if decoded.Header.WNDSize != p.Header.WNDSize {
		t.Errorf("expected wndSize %d, got %d", p.Header.WNDSize, decoded.Header.WNDSize)
	}
	if decoded.Header.SeqNum != p.Header.SeqNum {
		t.Errorf("expected seqNum %d, got %d", p.Header.SeqNum, decoded.Header.SeqNum)
	}
	if decoded.Header.AckNum != p.Header.AckNum {
		t.Errorf("expected ackNum %d, got %d", p.Header.AckNum, decoded.Header.AckNum)
	}

	// Verify payload
	if !bytes.Equal(decoded.Payload, p.Payload) {
		t.Errorf("expected payload %s, got %s", p.Payload, decoded.Payload)
	}
}

func TestDecodePacket_TooShort(t *testing.T) {
	// Header must be at least 20 bytes
	_, err := DecodePacket(make([]byte, 19))
	if err == nil {
		t.Error("expected error when decoding packet shorter than 20 bytes")
	}
}

func TestDecodePacket_InvalidVersion(t *testing.T) {
	// Create packet with invalid version
	header := make([]byte, 20)
	// Version is lower 4 bits of first byte. Let's make it version 2.
	header[0] = (byte(ST_DATA) << 4) | byte(2)

	_, err := DecodePacket(header)
	if err == nil {
		t.Error("expected error when decoding packet with unsupported version")
	}
}
