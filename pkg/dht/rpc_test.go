package dht

import (
	"testing"
)

func TestRPCCodec_PingPong(t *testing.T) {
	senderID := [32]byte{1, 2, 3}
	txID := [4]byte{9, 9, 9, 9}

	// 1. PING
	ping := &Message{
		TxID:     txID,
		Type:     MsgPing,
		SenderID: senderID,
	}
	data, err := EncodeMessage(ping)
	if err != nil {
		t.Fatalf("failed to encode PING: %v", err)
	}

	decoded, err := DecodeMessage(data)
	if err != nil {
		t.Fatalf("failed to decode PING: %v", err)
	}

	if decoded.Type != MsgPing || decoded.TxID != txID || decoded.SenderID != senderID {
		t.Errorf("decoded message mismatch: %+v", decoded)
	}

	// 2. PONG
	pong := &Message{
		TxID:     txID,
		Type:     MsgPong,
		SenderID: senderID,
	}
	data, err = EncodeMessage(pong)
	if err != nil {
		t.Fatalf("failed to encode PONG: %v", err)
	}

	decoded, err = DecodeMessage(data)
	if err != nil {
		t.Fatalf("failed to decode PONG: %v", err)
	}

	if decoded.Type != MsgPong || decoded.TxID != txID || decoded.SenderID != senderID {
		t.Errorf("decoded message mismatch: %+v", decoded)
	}
}

func TestRPCCodec_FindNode(t *testing.T) {
	senderID := [32]byte{1, 2, 3}
	txID := [4]byte{8, 8, 8, 8}
	targetID := [32]byte{4, 5, 6}

	// 3. FIND_NODE
	req := &Message{
		TxID:     txID,
		Type:     MsgFindNode,
		SenderID: senderID,
		Target:   targetID,
	}
	data, err := EncodeMessage(req)
	if err != nil {
		t.Fatalf("failed to encode FIND_NODE: %v", err)
	}

	decoded, err := DecodeMessage(data)
	if err != nil {
		t.Fatalf("failed to decode FIND_NODE: %v", err)
	}

	if decoded.Type != MsgFindNode || decoded.TxID != txID || decoded.SenderID != senderID || decoded.Target != targetID {
		t.Errorf("decoded message mismatch: %+v", decoded)
	}

	// 4. FIND_NODE_RESP
	contacts := []Contact{
		{ID: [32]byte{10}, Addr: "127.0.0.1:4001"},
		{ID: [32]byte{11}, Addr: "127.0.0.1:4002"},
	}
	resp := &Message{
		TxID:     txID,
		Type:     MsgFindNodeResp,
		SenderID: senderID,
		Contacts: contacts,
	}
	data, err = EncodeMessage(resp)
	if err != nil {
		t.Fatalf("failed to encode FIND_NODE_RESP: %v", err)
	}

	decoded, err = DecodeMessage(data)
	if err != nil {
		t.Fatalf("failed to decode FIND_NODE_RESP: %v", err)
	}

	if decoded.Type != MsgFindNodeResp || decoded.TxID != txID || decoded.SenderID != senderID {
		t.Fatalf("decoded message header mismatch: %+v", decoded)
	}

	if len(decoded.Contacts) != 2 {
		t.Fatalf("expected 2 contacts, got %d", len(decoded.Contacts))
	}

	if decoded.Contacts[0].ID != contacts[0].ID || decoded.Contacts[0].Addr != contacts[0].Addr {
		t.Errorf("contact 0 mismatch: expected %+v, got %+v", contacts[0], decoded.Contacts[0])
	}
}

func TestRPCCodec_AnnounceLookup(t *testing.T) {
	senderID := [32]byte{1, 2, 3}
	txID := [4]byte{7, 7, 7, 7}
	topic := [32]byte{100, 101, 102}

	// 5. ANNOUNCE
	ann := &Message{
		TxID:     txID,
		Type:     MsgAnnounce,
		SenderID: senderID,
		Topic:    topic,
		Port:     12345,
	}
	data, err := EncodeMessage(ann)
	if err != nil {
		t.Fatalf("failed to encode ANNOUNCE: %v", err)
	}

	decoded, err := DecodeMessage(data)
	if err != nil {
		t.Fatalf("failed to decode ANNOUNCE: %v", err)
	}

	if decoded.Type != MsgAnnounce || decoded.TxID != txID || decoded.SenderID != senderID || decoded.Topic != topic || decoded.Port != 12345 {
		t.Errorf("decoded message mismatch: %+v", decoded)
	}

	// 6. LOOKUP
	lkp := &Message{
		TxID:     txID,
		Type:     MsgLookup,
		SenderID: senderID,
		Topic:    topic,
	}
	data, err = EncodeMessage(lkp)
	if err != nil {
		t.Fatalf("failed to encode LOOKUP: %v", err)
	}

	decoded, err = DecodeMessage(data)
	if err != nil {
		t.Fatalf("failed to decode LOOKUP: %v", err)
	}

	if decoded.Type != MsgLookup || decoded.TxID != txID || decoded.SenderID != senderID || decoded.Topic != topic {
		t.Errorf("decoded message mismatch: %+v", decoded)
	}

	// 7. LOOKUP_RESP
	peers := []string{"192.168.1.50:5001", "10.0.0.10:5002"}
	lkpResp := &Message{
		TxID:     txID,
		Type:     MsgLookupResp,
		SenderID: senderID,
		Peers:    peers,
	}
	data, err = EncodeMessage(lkpResp)
	if err != nil {
		t.Fatalf("failed to encode LOOKUP_RESP: %v", err)
	}

	decoded, err = DecodeMessage(data)
	if err != nil {
		t.Fatalf("failed to decode LOOKUP_RESP: %v", err)
	}

	if decoded.Type != MsgLookupResp || decoded.TxID != txID || decoded.SenderID != senderID {
		t.Fatalf("decoded message mismatch: %+v", decoded)
	}

	if len(decoded.Peers) != 2 {
		t.Fatalf("expected 2 peers, got %d", len(decoded.Peers))
	}

	if decoded.Peers[0] != peers[0] || decoded.Peers[1] != peers[1] {
		t.Errorf("peers mismatch: expected %v, got %v", peers, decoded.Peers)
	}
}

func TestRPCCodec_Malformed(t *testing.T) {
	_, err := DecodeMessage(nil)
	if err == nil {
		t.Fatal("expected error decoding nil slice, got nil")
	}

	_, err = DecodeMessage([]byte{1, 2, 3})
	if err == nil {
		t.Fatal("expected error decoding short packet, got nil")
	}
}
