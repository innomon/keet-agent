package utp

import (
	"sync"
	"time"
)

// RTOCalculator implements the Jacob-Karels algorithm for estimating RTO.
type RTOCalculator struct {
	srtt        time.Duration
	rttvar      time.Duration
	rto         time.Duration
	initialized bool
	mu          sync.Mutex
}

// NewRTOCalculator creates a new RTOCalculator with default RTO.
func NewRTOCalculator() *RTOCalculator {
	return &RTOCalculator{
		rto: 500 * time.Millisecond,
	}
}

// Update recalculates the smoothed RTT and variance based on a new measurement.
func (c *RTOCalculator) Update(rtt time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.initialized {
		c.srtt = rtt
		c.rttvar = rtt / 2
		c.initialized = true
	} else {
		diff := c.srtt - rtt
		if diff < 0 {
			diff = -diff
		}
		c.rttvar = time.Duration(0.75*float64(c.rttvar) + 0.25*float64(diff))
		c.srtt = time.Duration(0.875*float64(c.srtt) + 0.125*float64(rtt))
	}
	c.rto = c.srtt + 4*c.rttvar
	if c.rto < 200*time.Millisecond {
		c.rto = 200 * time.Millisecond
	}
	if c.rto > 60*time.Second {
		c.rto = 60 * time.Second
	}
}

// GetRTO returns the current estimate of RTO.
func (c *RTOCalculator) GetRTO() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.rto
}

// seqLessOrEqual checks if sequence number s1 <= s2, handling wrapping.
func seqLessOrEqual(s1, s2 uint16) bool {
	return int16(s1-s2) <= 0
}

// RetransmitQueue tracks unacknowledged outbound packets.
type RetransmitQueue struct {
	packets     []*Packet
	dupAckCount map[uint16]int
	mu          sync.Mutex
}

// NewRetransmitQueue creates a new RetransmitQueue instance.
func NewRetransmitQueue() *RetransmitQueue {
	return &RetransmitQueue{
		dupAckCount: make(map[uint16]int),
	}
}

// Add appends a packet to the queue.
func (q *RetransmitQueue) Add(p *Packet) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.packets = append(q.packets, p)
}

// Len returns the number of unacknowledged packets.
func (q *RetransmitQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.packets)
}

// Ack removes all packets in the queue with SeqNum <= ackNum, returning the acknowledged packets.
func (q *RetransmitQueue) Ack(ackNum uint16) []*Packet {
	q.mu.Lock()
	defer q.mu.Unlock()

	var acked []*Packet
	var remaining []*Packet

	for _, p := range q.packets {
		if seqLessOrEqual(p.Header.SeqNum, ackNum) {
			acked = append(acked, p)
		} else {
			remaining = append(remaining, p)
		}
	}

	q.packets = remaining
	// Clear duplicate ACK counts for any sequence number <= ackNum
	for seq := range q.dupAckCount {
		if seqLessOrEqual(seq, ackNum) {
			delete(q.dupAckCount, seq)
		}
	}

	return acked
}

// HandleDupAck tracks duplicate ACKs and returns true if fast retransmit should trigger.
func (q *RetransmitQueue) HandleDupAck(ackNum uint16) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.dupAckCount[ackNum]++
	return q.dupAckCount[ackNum] == 3
}
