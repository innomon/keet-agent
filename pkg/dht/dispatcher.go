package dht

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"
)

// RequestHandler handles incoming Kademlia request messages.
type RequestHandler func(ctx context.Context, req *Message, src net.Addr) (*Message, error)

type pendingRequest struct {
	responseChan chan *Message
	expiry       time.Time
}

// Dispatcher coordinates sending Kademlia RPC requests and routing incoming messages to callbacks.
type Dispatcher struct {
	transport Transport
	handler   RequestHandler
	pending   sync.Map // Map of [4]byte (TxID) -> *pendingRequest
	closeChan chan struct{}
	wg        sync.WaitGroup
}

// NewDispatcher creates a new dispatcher operating on the given transport.
func NewDispatcher(t Transport, h RequestHandler) *Dispatcher {
	return &Dispatcher{
		transport: t,
		handler:   h,
		closeChan: make(chan struct{}),
	}
}

// Start spawns the background worker loops.
func (d *Dispatcher) Start() {
	d.wg.Add(1)
	go d.readLoop()

	d.wg.Add(1)
	go d.cleanupLoop()
}

// Stop halts all dispatcher routines and closes the underlying transport.
func (d *Dispatcher) Stop() {
	close(d.closeChan)
	d.transport.Close()
	d.wg.Wait()
}

func (d *Dispatcher) readLoop() {
	defer d.wg.Done()
	buf := make([]byte, 65535)
	for {
		select {
		case <-d.closeChan:
			return
		default:
			n, src, err := d.transport.ReadFrom(buf)
			if err != nil {
				return // transport closed, exit loop
			}

			m, err := DecodeMessage(buf[:n])
			if err != nil {
				continue // ignore malformed packet
			}

			if isResponse(m.Type) {
				if val, ok := d.pending.Load(m.TxID); ok {
					pr := val.(*pendingRequest)
					select {
					case pr.responseChan <- m:
					default:
					}
				}
				continue
			}

			// If request message, process asynchronously
			if d.handler != nil {
				go func(msg *Message, addr net.Addr) {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()

					resp, err := d.handler(ctx, msg, addr)
					if err != nil || resp == nil {
						return
					}

					resp.TxID = msg.TxID
					respData, err := EncodeMessage(resp)
					if err != nil {
						return
					}

					_, _ = d.transport.WriteTo(respData, addr)
				}(m, src)
			}
		}
	}
}

func isResponse(t MsgType) bool {
	return t == MsgPong || t == MsgFindNodeResp || t == MsgLookupResp
}

// SendRequest transmits an RPC request to target address and awaits a matching response with 5s timeout.
func (d *Dispatcher) SendRequest(ctx context.Context, target net.Addr, req *Message) (*Message, error) {
	pr := &pendingRequest{
		responseChan: make(chan *Message, 1),
		expiry:       time.Now().Add(5 * time.Second),
	}
	d.pending.Store(req.TxID, pr)
	defer d.pending.Delete(req.TxID)

	data, err := EncodeMessage(req)
	if err != nil {
		return nil, err
	}

	if _, err := d.transport.WriteTo(data, target); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-d.closeChan:
		return nil, errors.New("dispatcher closed")
	case resp := <-pr.responseChan:
		return resp, nil
	case <-time.After(5 * time.Second):
		return nil, errors.New("request timeout")
	}
}

func (d *Dispatcher) cleanupLoop() {
	defer d.wg.Done()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-d.closeChan:
			return
		case <-ticker.C:
			now := time.Now()
			d.pending.Range(func(key, value interface{}) bool {
				pr := value.(*pendingRequest)
				if now.After(pr.expiry) {
					d.pending.Delete(key)
				}
				return true
			})
		}
	}
}
