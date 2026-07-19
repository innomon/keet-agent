package dht

import (
	"sort"
	"sync"
)

// Contact represents a known node on the Kademlia network.
type Contact struct {
	ID   [32]byte
	Addr string
}

// RoutingTable implements K=20 bucket routing space for Kademlia.
type RoutingTable struct {
	localID [32]byte
	buckets [256][]Contact
	mu      sync.RWMutex
}

// NewRoutingTable creates a new routing table for the specified local ID.
func NewRoutingTable(localID [32]byte) *RoutingTable {
	return &RoutingTable{
		localID: localID,
	}
}

func commonPrefixLength(id1, id2 [32]byte) int {
	cpl := 0
	for i := 0; i < 32; i++ {
		xor := id1[i] ^ id2[i]
		if xor == 0 {
			cpl += 8
			continue
		}
		for b := 7; b >= 0; b-- {
			if (xor & (1 << b)) != 0 {
				break
			}
			cpl++
		}
		break
	}
	if cpl > 255 {
		return 255
	}
	return cpl
}

// AddContact inserts or updates a contact in the routing table, enforcing K=20 size limit and LRU eviction.
func (rt *RoutingTable) AddContact(c Contact) {
	if c.ID == rt.localID {
		return // Do not add self
	}

	idx := commonPrefixLength(rt.localID, c.ID)

	rt.mu.Lock()
	defer rt.mu.Unlock()

	bucket := rt.buckets[idx]

	// Check if already exists
	existsIdx := -1
	for i, existing := range bucket {
		if existing.ID == c.ID {
			existsIdx = i
			break
		}
	}

	if existsIdx != -1 {
		// Update address and move to the end of bucket (most recently seen)
		if c.Addr != "" {
			bucket[existsIdx].Addr = c.Addr
		}
		updatedContact := bucket[existsIdx]
		bucket = append(bucket[:existsIdx], bucket[existsIdx+1:]...)
		bucket = append(bucket, updatedContact)
	} else {
		// If bucket is full, evict the first element (least recently seen)
		if len(bucket) >= 20 {
			bucket = bucket[1:]
		}
		bucket = append(bucket, c)
	}

	rt.buckets[idx] = bucket
}

// FindClosestPeers searches for up to count closest contacts to target ID using XOR distance.
func (rt *RoutingTable) FindClosestPeers(target [32]byte, count int) []Contact {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	var allContacts []Contact
	for _, bucket := range rt.buckets {
		allContacts = append(allContacts, bucket...)
	}

	if len(allContacts) == 0 {
		return nil
	}

	// Copy all contacts to sort them
	results := make([]Contact, len(allContacts))
	copy(results, allContacts)

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
