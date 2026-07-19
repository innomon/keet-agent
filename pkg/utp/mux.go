package utp

import (
	"encoding/binary"
	"errors"
	"math/rand"
	"net"
	"sync"
)

// SocketMux manages a single UDP socket, demultiplexing incoming packets to their respective connection queues.
type SocketMux struct {
	conn        net.PacketConn
	conns       map[uint16]chan *Packet
	listener    *UTPListener
	mu          sync.RWMutex
	closeChan   chan struct{}
	closeOnce   sync.Once
	wg          sync.WaitGroup
	relayServer string
	stunChan    chan []byte
}

// NewSocketMux creates a new SocketMux instance wrapping the provided PacketConn.
func NewSocketMux(conn net.PacketConn) *SocketMux {
	return &SocketMux{
		conn:      conn,
		conns:     make(map[uint16]chan *Packet),
		closeChan: make(chan struct{}),
		stunChan:  make(chan []byte, 100),
	}
}

// Start spawns the background packet read loop.
func (sm *SocketMux) Start() {
	sm.wg.Add(1)
	go sm.readLoop()
}

// Stop shuts down the packet read loop and closes the underlying connection.
func (sm *SocketMux) Stop() {
	sm.closeOnce.Do(func() {
		close(sm.closeChan)
		sm.conn.Close()
	})
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

// RegisterListener registers a listener to handle inbound ST_SYN connection handshakes.
func (sm *SocketMux) RegisterListener(l *UTPListener) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.listener = l
}

// DeregisterListener removes the registered listener.
func (sm *SocketMux) DeregisterListener() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.listener = nil
}

// SetRelayServer sets the TURN relay server address.
func (sm *SocketMux) SetRelayServer(addr string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.relayServer = addr
}

// GetRelayServer gets the TURN relay server address.
func (sm *SocketMux) GetRelayServer() string {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.relayServer
}

func (sm *SocketMux) readLoop() {
	defer sm.wg.Done()
	buf := make([]byte, 65535)

	for {
		select {
		case <-sm.closeChan:
			return
		default:
			n, src, err := sm.conn.ReadFrom(buf)
			if err != nil {
				return // connection closed, terminate loop
			}

			raw := buf[:n]
			// Intercept STUN/TURN packets by checking STUN magic cookie
			if len(raw) >= 8 && binary.BigEndian.Uint32(raw[4:8]) == stunMagicCookie {
				cpy := make([]byte, len(raw))
				copy(cpy, raw)
				select {
				case sm.stunChan <- cpy:
				default:
				}
				continue
			}

			pkt, err := DecodePacket(raw)
			if err != nil {
				continue // ignore malformed packet
			}

			sm.mu.RLock()
			ch, exists := sm.conns[pkt.Header.ConnID]
			listener := sm.listener
			sm.mu.RUnlock()

			if exists {
				select {
				case ch <- pkt:
				default:
					// queue full, drop packet
				}
			} else if pkt.Header.Type == ST_SYN && listener != nil {
				s := pkt.Header.ConnID
				recvID := s
				sendID := s + 1

				c := &UTPConn{
					state:      STATE_CONNECTED,
					mux:        sm,
					remoteAddr: src,
					recvID:     recvID,
					sendID:     sendID,
					seq:        uint16(rand.Intn(65535)),
					ack:        pkt.Header.SeqNum,
					readBuf:    make(chan *Packet, 100),
					closeChan:  make(chan struct{}),
					finAcked:   make(chan struct{}, 1),
					ackChan:    make(chan uint16, 100),
					readQueue:  make(chan []byte, 100),
				}

				if err := sm.RegisterConn(recvID, c.readBuf); err == nil {
					go c.run()
					// Send SYN-ACK (ST_STATE)
					synAck := &Packet{
						Header: Header{
							Type:    ST_STATE,
							Version: 1,
							ConnID:  sendID,
							AckNum:  pkt.Header.SeqNum,
						},
					}
					data, _ := synAck.Encode()
					_, _ = sm.conn.WriteTo(data, src)

					select {
					case listener.acceptChan <- c:
					default:
					}
				}
			}
		}
	}
}
