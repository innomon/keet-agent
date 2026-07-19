package dht

import (
	"errors"
	"net"
	"sync"
)

var (
	inProcessMu       sync.RWMutex
	inProcessRegistry = make(map[string]*InProcessTransport)
)

type inProcessAddr struct {
	addr string
}

func (i inProcessAddr) Network() string { return "inprocess" }
func (i inProcessAddr) String() string  { return i.addr }

type inProcessPacket struct {
	src  net.Addr
	data []byte
}

// InProcessTransport implements the Transport interface using in-memory channels.
// This is used for hermetic, non-flaky testing of Kademlia and peer-to-peer networks.
type InProcessTransport struct {
	addr     inProcessAddr
	incoming chan inProcessPacket
	closed   chan struct{}
	mu       sync.Mutex
	isClosed bool
}

// NewInProcessTransport creates a new in-process transport bound to the specified logical address.
func NewInProcessTransport(addr string) (*InProcessTransport, error) {
	inProcessMu.Lock()
	defer inProcessMu.Unlock()

	if _, exists := inProcessRegistry[addr]; exists {
		return nil, errors.New("address already in use")
	}

	t := &InProcessTransport{
		addr:     inProcessAddr{addr: addr},
		incoming: make(chan inProcessPacket, 1000),
		closed:   make(chan struct{}),
	}
	inProcessRegistry[addr] = t
	return t, nil
}

// ReadFrom reads a message from the incoming in-process packet channel.
func (i *InProcessTransport) ReadFrom(p []byte) (int, net.Addr, error) {
	select {
	case <-i.closed:
		return 0, nil, errors.New("use of closed network connection")
	case pkt := <-i.incoming:
		n := copy(p, pkt.data)
		return n, pkt.src, nil
	}
}

// WriteTo writes a message to another InProcessTransport registered under the given address.
func (i *InProcessTransport) WriteTo(p []byte, addr net.Addr) (int, error) {
	inProcessMu.RLock()
	dest, exists := inProcessRegistry[addr.String()]
	inProcessMu.RUnlock()

	if !exists {
		return 0, errors.New("network unreachable or destination address not bound")
	}

	dest.mu.Lock()
	defer dest.mu.Unlock()

	if dest.isClosed {
		return 0, errors.New("destination connection closed")
	}

	dataCopy := make([]byte, len(p))
	copy(dataCopy, p)

	select {
	case dest.incoming <- inProcessPacket{src: i.addr, data: dataCopy}:
		return len(p), nil
	default:
		return 0, errors.New("destination buffer full")
	}
}

// Addr returns the logical in-process address.
func (i *InProcessTransport) Addr() net.Addr {
	return i.addr
}

// Close unregisters and closes the in-process transport.
func (i *InProcessTransport) Close() error {
	inProcessMu.Lock()
	delete(inProcessRegistry, i.addr.addr)
	inProcessMu.Unlock()

	i.mu.Lock()
	defer i.mu.Unlock()

	if i.isClosed {
		return nil
	}
	i.isClosed = true
	close(i.closed)
	return nil
}
