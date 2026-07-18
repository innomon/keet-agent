package dht

import (
	"sort"
	"sync"
)

type Contact struct {
	ID   [32]byte
	Addr string
}

type RoutingTable struct {
	localID  [32]byte
	contacts []Contact
	mu       sync.RWMutex
}

func NewRoutingTable(localID [32]byte) *RoutingTable {
	return &RoutingTable{
		localID: localID,
	}
}

func (rt *RoutingTable) AddContact(c Contact) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	// Check if already exists, update address if so
	for i, existing := range rt.contacts {
		if existing.ID == c.ID {
			rt.contacts[i].Addr = c.Addr
			return
		}
	}

	rt.contacts = append(rt.contacts, c)
}

func (rt *RoutingTable) FindClosestPeers(target [32]byte, count int) []Contact {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	if len(rt.contacts) == 0 {
		return nil
	}

	// Copy contacts to sort them
	results := make([]Contact, len(rt.contacts))
	copy(results, rt.contacts)

	sort.Slice(results, func(i, j int) bool {
		d1 := xorDistance(results[i].ID, target)
		d2 := xorDistance(results[j].ID, target)
		return compareDistance(d1, d2) < 0
	})

	if len(results) > count {
		return results[:count]
	}
	return results
}

func xorDistance(id1, id2 [32]byte) [32]byte {
	var dist [32]byte
	for i := 0; i < 32; i++ {
		dist[i] = id1[i] ^ id2[i]
	}
	return dist
}

func compareDistance(dist1, dist2 [32]byte) int {
	for i := 0; i < 32; i++ {
		if dist1[i] < dist2[i] {
			return -1
		}
		if dist1[i] > dist2[i] {
			return 1
		}
	}
	return 0
}
