package utp

import (
	"sync"
	"time"
)

const (
	// TARGET is the standard target queuing delay for LEDBAT (100 milliseconds).
	TARGET = 100 * time.Millisecond
	// GAIN determines the scaling of window changes.
	GAIN = 1.0
	// MIN_CWND is the minimum allowed congestion window size (2 packets, 3000 bytes).
	MIN_CWND = 3000
	// MAX_CWND is the maximum allowed congestion window size (1 Megabyte).
	MAX_CWND = 1024 * 1024
)

// LEDBATController implements the Low Extra Delay Background Transport congestion control algorithm.
type LEDBATController struct {
	mss       int
	cwnd      int
	baseDelay time.Duration
	lastDelay time.Duration
	mu        sync.Mutex
}

// NewLEDBATController creates a new LEDBATController instance.
func NewLEDBATController(mss int) *LEDBATController {
	return &LEDBATController{
		mss:  mss,
		cwnd: 4 * mss, // Start with 4 MSS
	}
}

// GetCwnd returns the current congestion window size in bytes.
func (c *LEDBATController) GetCwnd() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cwnd
}

// UpdateDelay updates the measured one-way delay and maintains the base delay running minimum.
func (c *LEDBATController) UpdateDelay(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.lastDelay = d

	if c.baseDelay == 0 || d < c.baseDelay {
		c.baseDelay = d
	}
}

// OnACK updates the congestion window on packet acknowledgment using the LEDBAT linear controller equation.
func (c *LEDBATController) OnACK() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.baseDelay == 0 {
		return
	}

	queuingDelay := c.lastDelay - c.baseDelay
	if queuingDelay < 0 {
		queuingDelay = 0
	}

	offTarget := TARGET - queuingDelay

	// LEDBAT window update: cwnd = cwnd + mss * GAIN * (offTarget/TARGET) * (mss/cwnd)
	scaledFactor := float64(offTarget) / float64(TARGET)
	adjustment := float64(c.mss) * GAIN * scaledFactor * (float64(c.mss) / float64(c.cwnd))

	c.cwnd += int(adjustment)

	if c.cwnd < MIN_CWND {
		c.cwnd = MIN_CWND
	}
	if c.cwnd > MAX_CWND {
		c.cwnd = MAX_CWND
	}
}
