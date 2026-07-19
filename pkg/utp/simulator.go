package utp

import (
	"math/rand"
	"net"
	"sync"
	"time"
)

// LossyConn wraps a net.PacketConn to simulate lossy, delayed network environments.
type LossyConn struct {
	net.PacketConn
	dropRate float64
	minDelay time.Duration
	maxDelay time.Duration
	rng      *rand.Rand
	mu       sync.Mutex
}

// NewLossyConn creates a new LossyConn wrapping the provided PacketConn.
func NewLossyConn(conn net.PacketConn, dropRate float64, minDelay, maxDelay time.Duration) *LossyConn {
	source := rand.NewSource(time.Now().UnixNano())
	return &LossyConn{
		PacketConn: conn,
		dropRate:   dropRate,
		minDelay:   minDelay,
		maxDelay:   maxDelay,
		rng:        rand.New(source),
	}
}

func (lc *LossyConn) shouldDrop() bool {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	return lc.rng.Float64() < lc.dropRate
}

// ReadFrom reads a packet from the connection, simulating drop and latency.
func (lc *LossyConn) ReadFrom(p []byte) (int, net.Addr, error) {
	for {
		n, addr, err := lc.PacketConn.ReadFrom(p)
		if err != nil {
			return 0, nil, err
		}

		if lc.shouldDrop() {
			continue // drop the packet, read next
		}

		if lc.minDelay > 0 {
			var delay time.Duration
			lc.mu.Lock()
			if lc.maxDelay > lc.minDelay {
				diff := int64(lc.maxDelay - lc.minDelay)
				delay = lc.minDelay + time.Duration(lc.rng.Int63n(diff))
			} else {
				delay = lc.minDelay
			}
			lc.mu.Unlock()

			time.Sleep(delay)
		}

		return n, addr, nil
	}
}
