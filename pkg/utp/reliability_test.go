package utp

import (
	"testing"
	"time"
)

func TestRetransmitQueue_AddAndAck(t *testing.T) {
	q := NewRetransmitQueue()

	p1 := &Packet{Header: Header{SeqNum: 10}}
	p2 := &Packet{Header: Header{SeqNum: 11}}

	q.Add(p1)
	q.Add(p2)

	if q.Len() != 2 {
		t.Errorf("expected queue length 2, got %d", q.Len())
	}

	// Ack up to 10
	acked := q.Ack(10)
	if len(acked) != 1 || acked[0].Header.SeqNum != 10 {
		t.Errorf("expected acked list to contain seq 10, got %v", acked)
	}

	if q.Len() != 1 {
		t.Errorf("expected queue length 1 after ack, got %d", q.Len())
	}
}

func TestRetransmitQueue_FastRetransmit(t *testing.T) {
	q := NewRetransmitQueue()

	p1 := &Packet{Header: Header{SeqNum: 15}}
	q.Add(p1)

	// Simulate receiving 3 duplicate ACKs for sequence 14 (meaning 15 is missing)
	// We call HandleDupAck(14)
	shouldRetransmit := false
	for i := 0; i < 3; i++ {
		if q.HandleDupAck(14) {
			shouldRetransmit = true
		}
	}

	if !shouldRetransmit {
		t.Error("expected fast retransmit to trigger after 3 duplicate ACKs")
	}

	// A fourth dup ACK should not trigger it again immediately
	if q.HandleDupAck(14) {
		t.Error("expected fast retransmit to only trigger once on 3rd dup ACK")
	}
}

func TestRTOCalculator(t *testing.T) {
	calc := NewRTOCalculator()

	// Initial RTO should be standard (e.g. 500ms or 1s)
	initialRTO := calc.GetRTO()
	if initialRTO < 100*time.Millisecond {
		t.Errorf("expected initial RTO to be substantial, got %v", initialRTO)
	}

	// Update with a round trip time of 100ms
	calc.Update(100 * time.Millisecond)
	rto1 := calc.GetRTO()

	// Update with RTT of 110ms
	calc.Update(110 * time.Millisecond)
	rto2 := calc.GetRTO()

	if rto1 == initialRTO {
		t.Error("expected RTO to adjust after update")
	}
	_ = rto2
}
