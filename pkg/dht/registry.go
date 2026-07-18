package dht

import "sync"

type SwarmRegistry struct {
	mu     sync.RWMutex
	swarms map[[32]byte][]string
}

func NewSwarmRegistry() *SwarmRegistry {
	return &SwarmRegistry{
		swarms: make(map[[32]byte][]string),
	}
}

func (r *SwarmRegistry) RegisterPeer(topic [32]byte, peerAddr string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	peers := r.swarms[topic]
	// Check if already registered
	for _, p := range peers {
		if p == peerAddr {
			return
		}
	}
	r.swarms[topic] = append(peers, peerAddr)
}

func (r *SwarmRegistry) UnregisterPeer(topic [32]byte, peerAddr string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	peers, exists := r.swarms[topic]
	if !exists {
		return
	}

	for i, p := range peers {
		if p == peerAddr {
			// Remove element
			r.swarms[topic] = append(peers[:i], peers[i+1:]...)
			break
		}
	}
}

func (r *SwarmRegistry) GetPeers(topic [32]byte) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	peers, exists := r.swarms[topic]
	if !exists {
		return nil
	}

	// Return a copy to avoid race conditions when read by caller
	copied := make([]string, len(peers))
	copy(copied, peers)
	return copied
}

func (r *SwarmRegistry) ClearSwarm(topic [32]byte) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.swarms, topic)
}
