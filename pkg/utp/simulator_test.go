package utp

import (
	"net"
	"testing"
	"time"
)

func TestLossyConn_DropRate(t *testing.T) {
	// We will create a mock PacketConn or just test the logic of drop decision.
	// Since we want to test reliability under actual packet loss, let's construct a LossyConn wrapping a real UDP local connection.
	connA, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen UDP: %v", err)
	}
	defer connA.Close()

	connB, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen UDP: %v", err)
	}
	defer connB.Close()

	// Wrap connB with LossyConn with 50% drop rate
	lossyB := NewLossyConn(connB, 0.5, 0, 0)
	defer lossyB.Close()

	// Send 100 packets from connA to connB
	addrB := connB.LocalAddr()
	payload := []byte("test")
	
	sentCount := 100
	for i := 0; i < sentCount; i++ {
		_, err := connA.WriteTo(payload, addrB)
		if err != nil {
			t.Fatalf("failed to write: %v", err)
		}
	}

	// Read on lossyB with short read deadlines.
	receivedCount := 0
	buf := make([]byte, 1024)
	
	for {
		_ = lossyB.SetReadDeadline(time.Now().Add(5 * time.Millisecond))
		_, _, err := lossyB.ReadFrom(buf)
		if err != nil {
			break
		}
		receivedCount++
	}

	// We expect some packets to be dropped (since dropRate is 0.5, we should see roughly 50 received, definitely between 20 and 80)
	t.Logf("Sent: %d, Received: %d", sentCount, receivedCount)
	if receivedCount == sentCount {
		t.Error("expected some packets to be dropped")
	}
	if receivedCount == 0 {
		t.Error("expected at least some packets to be received")
	}
}

func TestLossyConn_Latency(t *testing.T) {
	connA, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen UDP: %v", err)
	}
	defer connA.Close()

	connB, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen UDP: %v", err)
	}
	defer connB.Close()

	// Wrap connB with 20ms to 40ms delay, 0% drop rate
	lossyB := NewLossyConn(connB, 0.0, 20*time.Millisecond, 40*time.Millisecond)
	defer lossyB.Close()

	payload := []byte("latency")
	startTime := time.Now()

	_, err = connA.WriteTo(payload, connB.LocalAddr())
	if err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	buf := make([]byte, 1024)
	_ = lossyB.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	_, _, err = lossyB.ReadFrom(buf)
	if err != nil {
		t.Fatalf("failed to read: %v", err)
	}

	duration := time.Since(startTime)
	t.Logf("Packet delay: %v", duration)

	if duration < 20*time.Millisecond {
		t.Errorf("expected packet delay to be at least 20ms, got %v", duration)
	}
}

func TestLossyConn_RandomGenerator(t *testing.T) {
	// Ensure NewLossyConn sets up the random source correctly
	conn, _ := net.ListenPacket("udp", "127.0.0.1:0")
	defer conn.Close()
	
	lc := NewLossyConn(conn, 0.1, 0, 0)
	if lc.rng == nil {
		t.Error("expected rng (rand.Rand) to be initialized")
	}
	
	// Test drop logic directly
	dropCount := 0
	for i := 0; i < 1000; i++ {
		if lc.shouldDrop() {
			dropCount++
		}
	}
	// Roughly 100 drops expected
	if dropCount < 40 || dropCount > 160 {
		t.Errorf("expected around 100 drops out of 1000 with 10%% rate, got %d", dropCount)
	}
}
