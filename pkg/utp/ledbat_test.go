package utp

import (
	"testing"
	"time"
)

func TestLEDBAT_CwndGrowth(t *testing.T) {
	// Initialize controller with a packet size of 1500 bytes
	c := NewLEDBATController(1500)

	initialCwnd := c.GetCwnd()
	if initialCwnd <= 0 {
		t.Fatalf("expected initial cwnd to be >0, got %d", initialCwnd)
	}

	// Simulate low delay (e.g. 10ms delay, base delay 10ms -> queuing delay 0ms)
	// target is 100ms, so off_target is 100ms - 0ms = 100ms (positive off_target)
	// cwnd should grow!
	c.UpdateDelay(10 * time.Millisecond) // establishes base delay
	c.UpdateDelay(10 * time.Millisecond) // current delay

	for i := 0; i < 10; i++ {
		c.OnACK()
	}

	newCwnd := c.GetCwnd()
	if newCwnd <= initialCwnd {
		t.Errorf("expected cwnd to grow on low delay, went from %d to %d", initialCwnd, newCwnd)
	}
}

func TestLEDBAT_CwndShrink(t *testing.T) {
	c := NewLEDBATController(1500)

	// Simulates low delay to grow cwnd first
	c.UpdateDelay(10 * time.Millisecond)
	for i := 0; i < 50; i++ {
		c.OnACK()
	}
	highCwnd := c.GetCwnd()

	// Simulate high delay (e.g. 160ms delay -> base delay 10ms, queuing delay 150ms)
	// target is 100ms, so off_target is 100ms - 150ms = -50ms (negative off_target)
	// cwnd should shrink!
	c.UpdateDelay(160 * time.Millisecond)

	for i := 0; i < 50; i++ {
		c.OnACK()
	}

	newCwnd := c.GetCwnd()
	if newCwnd >= highCwnd {
		t.Errorf("expected cwnd to shrink on high delay, went from %d to %d", highCwnd, newCwnd)
	}
}

func TestLEDBAT_BaseDelayTracking(t *testing.T) {
	c := NewLEDBATController(1500)

	// Initial updates establishing minimum
	c.UpdateDelay(50 * time.Millisecond)
	c.UpdateDelay(30 * time.Millisecond)
	c.UpdateDelay(40 * time.Millisecond)

	if c.baseDelay != 30*time.Millisecond {
		t.Errorf("expected base delay to be minimum of 30ms, got %v", c.baseDelay)
	}
}
