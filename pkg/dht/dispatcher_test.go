package dht

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"
)

func TestDispatcher_PingPongRoundTrip(t *testing.T) {
	// Create node A (receiver/server)
	tpA, err := NewInProcessTransport("nodeA")
	if err != nil {
		t.Fatalf("failed to create transport A: %v", err)
	}
	defer tpA.Close()

	// Handler on node A to respond to PING with PONG
	handler := func(ctx context.Context, req *Message, src net.Addr) (*Message, error) {
		if req.Type == MsgPing {
			return &Message{
				Type:     MsgPong,
				SenderID: [32]byte{10},
			}, nil
		}
		return nil, errors.New("unsupported message type")
	}

	dispA := NewDispatcher(tpA, handler)
	dispA.Start()
	defer dispA.Stop()

	// Create node B (sender/client)
	tpB, err := NewInProcessTransport("nodeB")
	if err != nil {
		t.Fatalf("failed to create transport B: %v", err)
	}
	defer tpB.Close()

	dispB := NewDispatcher(tpB, nil)
	dispB.Start()
	defer dispB.Stop()

	// Send PING from B to A
	pingMsg := &Message{
		TxID:     [4]byte{1, 2, 3, 4},
		Type:     MsgPing,
		SenderID: [32]byte{20},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	resp, err := dispB.SendRequest(ctx, tpA.Addr(), pingMsg)
	if err != nil {
		t.Fatalf("SendRequest failed: %v", err)
	}

	if resp.Type != MsgPong {
		t.Errorf("expected PONG response, got type: %d", resp.Type)
	}

	if resp.TxID != pingMsg.TxID {
		t.Errorf("expected tx ID matching %v, got %v", pingMsg.TxID, resp.TxID)
	}
}

func TestDispatcher_Timeout(t *testing.T) {
	tpB, err := NewInProcessTransport("nodeClient")
	if err != nil {
		t.Fatalf("failed to create client transport: %v", err)
	}
	defer tpB.Close()

	dispB := NewDispatcher(tpB, nil)
	dispB.Start()
	defer dispB.Stop()

	// Send request to an address with no active dispatcher
	pingMsg := &Message{
		TxID:     [4]byte{5, 6, 7, 8},
		Type:     MsgPing,
		SenderID: [32]byte{20},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Dest is "nodeA", which is not registered anymore because previous test closed it
	dest := inProcessAddr{addr: "nonexistent"}
	_, err = dispB.SendRequest(ctx, dest, pingMsg)
	if err == nil {
		t.Fatal("expected request to fail or timeout, got success")
	}
}

func TestDispatcher_ConcurrentRequests(t *testing.T) {
	tpA, err := NewInProcessTransport("nodeServer")
	if err != nil {
		t.Fatalf("failed to create transport: %v", err)
	}
	defer tpA.Close()

	handler := func(ctx context.Context, req *Message, src net.Addr) (*Message, error) {
		return &Message{
			Type:     MsgPong,
			SenderID: [32]byte{10},
		}, nil
	}

	dispA := NewDispatcher(tpA, handler)
	dispA.Start()
	defer dispA.Stop()

	// Client
	tpB, err := NewInProcessTransport("nodeConcurrent")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer tpB.Close()

	dispB := NewDispatcher(tpB, nil)
	dispB.Start()
	defer dispB.Stop()

	const count = 20
	var wg sync.WaitGroup
	wg.Add(count)

	for i := 0; i < count; i++ {
		go func(idx int) {
			defer wg.Done()
			req := &Message{
				TxID:     [4]byte{byte(idx), 0, 0, 0},
				Type:     MsgPing,
				SenderID: [32]byte{20},
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			resp, err := dispB.SendRequest(ctx, tpA.Addr(), req)
			if err != nil {
				t.Errorf("request %d failed: %v", idx, err)
				return
			}
			if resp.TxID[0] != byte(idx) {
				t.Errorf("expected response tx ID matching request tx ID %d, got %d", idx, resp.TxID[0])
			}
		}(i)
	}

	wg.Wait()
}
