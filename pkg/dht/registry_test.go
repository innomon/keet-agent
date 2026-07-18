package dht

import (
	"testing"
)

func TestSwarmRegistry_AddRemove(t *testing.T) {
	reg := NewSwarmRegistry()
	topic := [32]byte{1, 2, 3}

	// Register peers
	reg.RegisterPeer(topic, "127.0.0.1:5001")
	reg.RegisterPeer(topic, "127.0.0.1:5002")

	peers := reg.GetPeers(topic)
	if len(peers) != 2 {
		t.Fatalf("expected 2 peers, got %d", len(peers))
	}

	// Order doesn't strictly matter but check existence
	found1, found2 := false, false
	for _, p := range peers {
		if p == "127.0.0.1:5001" {
			found1 = true
		}
		if p == "127.0.0.1:5002" {
			found2 = true
		}
	}
	if !found1 || !found2 {
		t.Errorf("expected peers not found: %v", peers)
	}

	// Unregister one
	reg.UnregisterPeer(topic, "127.0.0.1:5001")
	peers = reg.GetPeers(topic)
	if len(peers) != 1 {
		t.Fatalf("expected 1 peer, got %d", len(peers))
	}
	if peers[0] != "127.0.0.1:5002" {
		t.Errorf("expected remaining peer '127.0.0.1:5002', got %q", peers[0])
	}

	// Clear swarm
	reg.ClearSwarm(topic)
	peers = reg.GetPeers(topic)
	if len(peers) != 0 {
		t.Errorf("expected 0 peers after clear, got %d", len(peers))
	}
}
