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

func TestRoutingTable_KBucketEnforcementAndLRU(t *testing.T) {
	localID := [32]byte{0}
	rt := NewRoutingTable(localID)

	// Create 21 contacts that will fall into the same bucket.
	// Since localID is {0}, any ID with a specific common prefix length (e.g., first byte != 0) will fall in a bucket.
	// Let's make sure they all have the same prefix length to localID.
	// For example, if we use ID where first byte is 128 (binary 10000000), they all have common prefix length of 0.
	// So let's vary the second byte so they are different contacts, but all fall in bucket 0.
	var contacts [22]Contact
	for i := 0; i < 22; i++ {
		id := [32]byte{128}
		id[1] = byte(i)
		contacts[i] = Contact{ID: id, Addr: "127.0.0.1:0"}
	}

	// Add the first 20 contacts
	for i := 0; i < 20; i++ {
		rt.AddContact(contacts[i])
	}

	// Verify the bucket size is 20
	closest := rt.FindClosestPeers([32]byte{128}, 100)
	if len(closest) != 20 {
		t.Fatalf("expected 20 contacts, got %d", len(closest))
	}

	// Update LRU of the least-recently-seen (contacts[0]) by adding it again
	rt.AddContact(contacts[0])

	// Now add the 21st contact (contacts[20]). This should evict the least-recently-seen,
	// which was contacts[1], because contacts[0] was just updated to most-recently-seen.
	rt.AddContact(contacts[20])

	closest = rt.FindClosestPeers([32]byte{128}, 100)
	if len(closest) != 20 {
		t.Fatalf("expected 20 contacts, got %d", len(closest))
	}

	// Verify contacts[1] was evicted (is not in the closest list)
	foundC1 := false
	foundC0 := false
	foundC20 := false
	for _, c := range closest {
		if c.ID == contacts[1].ID {
			foundC1 = true
		}
		if c.ID == contacts[0].ID {
			foundC0 = true
		}
		if c.ID == contacts[20].ID {
			foundC20 = true
		}
	}

	if foundC1 {
		t.Error("expected contacts[1] to be evicted, but it was found")
	}
	if !foundC0 {
		t.Error("expected contacts[0] to be preserved, but it was missing")
	}
	if !foundC20 {
		t.Error("expected contacts[20] to be added, but it was missing")
	}
}
