package dht

import (
	"os"
	"strings"
	"testing"
)

func TestDHTConfig_Defaults(t *testing.T) {
	node, err := NewDHTNode(nil) // nil config uses defaults
	if err != nil {
		t.Fatalf("failed to create default DHT Node: %v", err)
	}

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

	nodes := node.GetBootstrapNodes()
	if len(nodes) != 2 {
		t.Fatalf("expected 2 bootstrap nodes, got: %d", len(nodes))
	}
	if nodes[0] != "127.0.0.1:4001" || nodes[1] != "127.0.0.1:4002" {
		t.Errorf("unexpected bootstrap nodes: %v", nodes)
	}
}
