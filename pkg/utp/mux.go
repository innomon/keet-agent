package utp

import (
	"errors"
	"net"
	"sync"
)

// SocketMux manages a single UDP socket, demultiplexing incoming packets to their respective connection queues.
type SocketMux struct {
	conn      net.PacketConn
	conns     map[uint16]chan *Packet
	mu        sync.RWMutex
	closeChan chan struct{}
	wg        sync.WaitGroup
}

// NewSocketMux creates a new SocketMux instance wrapping the provided PacketConn.
func NewSocketMux(conn net.PacketConn) *SocketMux {
	return &SocketMux{
		conn:      conn,
		conns:     make(map[uint16]chan *Packet),
		closeChan: make(chan struct{}),
	}
}

// Start spawns the background packet read loop.
func (sm *SocketMux) Start() {
	sm.wg.Add(1)
	go sm.readLoop()
}

// Stop shuts down the packet read loop and closes the underlying connection.
func (sm *SocketMux) Stop() {
	close(sm.closeChan)
	sm.conn.Close()
	sm.wg.Wait()
}

// RegisterConn registers a channel to receive incoming packets for a given Connection ID.
func (sm *SocketMux) RegisterConn(connID uint16, packetChan chan *Packet) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.conns[connID]; exists {
		return errors.New("connection ID already registered")
	}

	sm.conns[connID] = packetChan
	return nil
}

// DeregisterConn removes registration for the connection ID.
func (sm *SocketMux) DeregisterConn(connID uint16) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.conns, connID)
}

func (sm *SocketMux) readLoop() {
	defer sm.wg.Done()
	buf := make([]byte, 65535)

	for {
		select {
		case <-sm.closeChan:
			return
		default:
			n, _, err := sm.conn.ReadFrom(buf)
			if err != nil {
				return // connection closed, terminate loop
			}

			pkt, err := DecodePacket(buf[:n])
			if err != nil {
				continue // ignore malformed packet
			}

			sm.mu.RLock()
			ch, exists := sm.conns[pkt.Header.ConnID]
			sm.mu.RUnlock()

			if exists {
				select {
				case ch <- pkt:
				default:
					// queue full, drop packet
				}
			}
		}
	}
}
