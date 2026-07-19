package dht

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestDHTConfig_Defaults(t *testing.T) {
	node, err := NewDHTNode(nil) // nil config uses defaults
	if err != nil {
		t.Fatalf("failed to create default DHT Node: %v", err)
	}
	defer node.Stop()

	defaults := node.GetBootstrapNodes()
	if len(defaults) == 0 {
		t.Error("expected default bootstrap nodes, got none")
	}

	foundDefault := false
	for _, n := range defaults {
		if strings.Contains(n, "holepunch.to") {
			foundDefault = true
		}
	}
	if !foundDefault {
		t.Errorf("expected default bootstrap nodes to contain 'holepunch.to', got: %v", defaults)
	}
}

func TestDHTConfig_EnvOverride(t *testing.T) {
	os.Setenv("DHT_BOOTSTRAP_NODES", "127.0.0.1:4001,127.0.0.1:4002")
	defer os.Unsetenv("DHT_BOOTSTRAP_NODES")

	node, err := NewDHTNode(nil)
	if err != nil {
		t.Fatalf("failed to create DHT Node with overrides: %v", err)
	}
	defer node.Stop()

	nodes := node.GetBootstrapNodes()
	if len(nodes) != 2 {
		t.Fatalf("expected 2 bootstrap nodes, got: %d", len(nodes))
	}
	if nodes[0] != "127.0.0.1:4001" || nodes[1] != "127.0.0.1:4002" {
		t.Errorf("unexpected bootstrap nodes: %v", nodes)
	}
}

func TestDHTNode_Bootstrap(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 1. Create Bootstrap Node (nodeA)
	tpA, err := NewInProcessTransport("nodeA")
	if err != nil {
		t.Fatalf("failed to create transport A: %v", err)
	}
	idA := [32]byte{1}
	nodeA, err := NewDHTNode(&Config{
		LocalID:        idA,
		Transport:      tpA,
		BootstrapNodes: []string{}, // No bootstrap for itself
	})
	if err != nil {
		t.Fatalf("failed to create nodeA: %v", err)
	}
	if err := nodeA.Start(ctx); err != nil {
		t.Fatalf("failed to start nodeA: %v", err)
	}
	defer nodeA.Stop()

	// 2. Create Node B pointing to nodeA as bootstrap
	tpB, err := NewInProcessTransport("nodeB")
	if err != nil {
		t.Fatalf("failed to create transport B: %v", err)
	}
	idB := [32]byte{2}
	nodeB, err := NewDHTNode(&Config{
		LocalID:        idB,
		Transport:      tpB,
		BootstrapNodes: []string{"nodeA"},
	})
	if err != nil {
		t.Fatalf("failed to create nodeB: %v", err)
	}
	if err := nodeB.Start(ctx); err != nil {
		t.Fatalf("failed to start nodeB: %v", err)
	}
	defer nodeB.Stop()

	// Verify Node B's routing table has nodeA
	closest := nodeB.routing.FindClosestPeers(idA, 1)
	if len(closest) == 0 {
		t.Fatal("expected nodeB routing table to contain nodeA contact, but it is empty")
	}
	if closest[0].ID != idA {
		t.Errorf("expected closest contact to be nodeA (%v), got %v", idA, closest[0].ID)
	}
}

func TestDHTNode_AnnounceAndLookup(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Setup a small 3-node network: B and C bootstrap via A.
	tpA, _ := NewInProcessTransport("nodeA")
	nodeA, _ := NewDHTNode(&Config{LocalID: [32]byte{1}, Transport: tpA, BootstrapNodes: []string{}})
	_ = nodeA.Start(ctx)
	defer nodeA.Stop()

	tpB, _ := NewInProcessTransport("nodeB")
	nodeB, _ := NewDHTNode(&Config{LocalID: [32]byte{2}, Transport: tpB, BootstrapNodes: []string{"nodeA"}})
	_ = nodeB.Start(ctx)
	defer nodeB.Stop()

	tpC, _ := NewInProcessTransport("nodeC")
	nodeC, _ := NewDHTNode(&Config{LocalID: [32]byte{3}, Transport: tpC, BootstrapNodes: []string{"nodeA"}})
	_ = nodeC.Start(ctx)
	defer nodeC.Stop()

	topic := [32]byte{100, 101, 102}

	// Node B announces a topic
	err := nodeB.Announce(ctx, topic, 4002)
	if err != nil {
		t.Fatalf("failed to announce: %v", err)
	}

	// Node C performs lookup on topic
	peers, err := nodeC.Lookup(ctx, topic)
	if err != nil {
		t.Fatalf("lookup failed: %v", err)
	}

	if len(peers) == 0 {
		t.Fatal("expected peers, got none")
	}

	foundB := false
	for _, p := range peers {
		if strings.Contains(p, "nodeB") || strings.Contains(p, "4002") {
			foundB = true
		}
	}

	if !foundB {
		t.Errorf("expected to find nodeB in peer list, got: %v", peers)
	}
}

func TestDHTNode_Leave(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	tpA, _ := NewInProcessTransport("nodeA")
	nodeA, _ := NewDHTNode(&Config{LocalID: [32]byte{1}, Transport: tpA, BootstrapNodes: []string{}})
	_ = nodeA.Start(ctx)
	defer nodeA.Stop()

	tpB, _ := NewInProcessTransport("nodeB")
	nodeB, _ := NewDHTNode(&Config{LocalID: [32]byte{2}, Transport: tpB, BootstrapNodes: []string{"nodeA"}})
	_ = nodeB.Start(ctx)
	defer nodeB.Stop()

	topic := [32]byte{100, 101, 102}

	_ = nodeB.Announce(ctx, topic, 4002)

	// Verify nodeA has the announced peer
	peers, _ := nodeA.Lookup(ctx, topic)
	if len(peers) == 0 {
		t.Fatal("expected peers registered on nodeA")
	}

	// Node B leaves topic
	err := nodeB.Leave(ctx, topic)
	if err != nil {
		t.Fatalf("leave failed: %v", err)
	}

	// Perform lookup on nodeA, it should have been cleared (or won't be returned by B)
	peers, _ = nodeB.Lookup(ctx, topic)
	if len(peers) != 0 {
		t.Errorf("expected 0 peers after B left, got: %v", peers)
	}
}
