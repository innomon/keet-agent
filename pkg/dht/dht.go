package dht

import (
	"os"
	"strings"
)

type Config struct {
	BootstrapNodes []string
}

type DHTNode struct {
	bootstrapNodes []string
}

func NewDHTNode(cfg *Config) (*DHTNode, error) {
	var bootstrapNodes []string

	// Check env override
	if envNodes := os.Getenv("DHT_BOOTSTRAP_NODES"); envNodes != "" {
		parts := strings.Split(envNodes, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				bootstrapNodes = append(bootstrapNodes, part)
			}
		}
	} else if cfg != nil && len(cfg.BootstrapNodes) > 0 {
		bootstrapNodes = cfg.BootstrapNodes
	} else {
		// Public Holepunch defaults
		bootstrapNodes = []string{
			"dht1.holepunch.to:49737",
			"dht2.holepunch.to:49737",
			"dht3.holepunch.to:49737",
		}
	}

	return &DHTNode{
		bootstrapNodes: bootstrapNodes,
	}, nil
}

func (n *DHTNode) GetBootstrapNodes() []string {
	return n.bootstrapNodes
}
