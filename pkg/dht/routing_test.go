package dht

import (
	"bytes"
	"testing"
)

func TestRoutingTable_AddAndFind(t *testing.T) {
	localID := [32]byte{0}
	rt := NewRoutingTable(localID)

	// Add some contacts
	c1 := Contact{ID: [32]byte{1}, Addr: "127.0.0.1:4001"}
	c2 := Contact{ID: [32]byte{2}, Addr: "127.0.0.1:4002"}
	c3 := Contact{ID: [32]byte{3}, Addr: "127.0.0.1:4003"}

	rt.AddContact(c1)
	rt.AddContact(c2)
	rt.AddContact(c3)

	// Find closest peers to a target ID {2}
	closest := rt.FindClosestPeers([32]byte{2}, 2)
	if len(closest) != 2 {
		t.Fatalf("expected 2 closest peers, got %d", len(closest))
	}

	// Closest should be c2 (distance 2^2 = 0) and then c3 (distance 3^2 = 1) or c1 (distance 1^2 = 3)
	// Let's verify c2 is the first (closest)
	if !bytes.Equal(closest[0].ID[:], c2.ID[:]) {
		t.Errorf("expected closest peer to be c2, got ID: %v", closest[0].ID)
	}
}
