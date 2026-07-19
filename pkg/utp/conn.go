package utp

import (
	"errors"
	"math/rand"
	"net"
	"sync"
	"time"
)

// ConnState represents the current state of a UTP connection.
type ConnState int

const (
	// STATE_NONE means connection is not yet initialized.
	STATE_NONE ConnState = iota
	// STATE_SYN_SENT means client ST_SYN has been sent, waiting for SYN-ACK.
	STATE_SYN_SENT
	// STATE_SYN_RECEIVED means server ST_SYN has been received.
	STATE_SYN_RECEIVED
	// STATE_CONNECTED means handshake completed and the connection is active.
	STATE_CONNECTED
	// STATE_FIN_SENT means connection is shutting down.
	STATE_FIN_SENT
	// STATE_CLOSED means connection is closed.
	STATE_CLOSED
)

// UTPConn implements net.Conn using the µTP reliable UDP stream transport.
type UTPConn struct {
	state      ConnState
	mux        *SocketMux
	remoteAddr net.Addr
	recvID     uint16
	sendID     uint16
	seq        uint16
	ack        uint16
	readBuf    chan *Packet
	closeChan  chan struct{}
	mu         sync.Mutex
}

// DialUTP initiates a new µTP connection to the target remote address using the provided SocketMux.
func DialUTP(mux *SocketMux, addr net.Addr) (*UTPConn, error) {
	// Generate random client connection ID
	s := uint16(rand.Intn(60000) + 1000)
	recvID := s + 1
	sendID := s

	c := &UTPConn{
		state:      STATE_SYN_SENT,
		mux:        mux,
		remoteAddr: addr,
		recvID:     recvID,
		sendID:     sendID,
		seq:        1,
		readBuf:    make(chan *Packet, 100),
		closeChan:  make(chan struct{}),
	}

	err := mux.RegisterConn(recvID, c.readBuf)
	if err != nil {
		return nil, err
	}

	// Send ST_SYN
	syn := &Packet{
		Header: Header{
			Type:    ST_SYN,
			Version: 1,
			ConnID:  sendID,
			SeqNum:  c.seq,
		},
	}
	data, err := syn.Encode()
	if err != nil {
		mux.DeregisterConn(recvID)
		return nil, err
	}

	_, err = mux.conn.WriteTo(data, addr)
	if err != nil {
		mux.DeregisterConn(recvID)
		return nil, err
	}

	// Wait for ST_STATE (SYN-ACK) response
	select {
	case pkt := <-c.readBuf:
		if pkt.Header.Type == ST_STATE && pkt.Header.AckNum == c.seq {
			c.state = STATE_CONNECTED
			c.ack = pkt.Header.SeqNum
			return c, nil
		}
	case <-time.After(1 * time.Second):
	}

	mux.DeregisterConn(recvID)
	return nil, errors.New("µTP connection handshake timed out")
}

// Read reads data from the connection (stub for handshake phase).
func (c *UTPConn) Read(b []byte) (n int, err error) {
	return 0, nil
}

// Write writes data to the connection (stub for handshake phase).
func (c *UTPConn) Write(b []byte) (n int, err error) {
	return 0, nil
}

// Close closes the connection.
func (c *UTPConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = STATE_CLOSED
	c.mux.DeregisterConn(c.recvID)
	return nil
}

// LocalAddr returns the local network address.
func (c *UTPConn) LocalAddr() net.Addr {
	return c.mux.conn.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (c *UTPConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

// SetDeadline sets the read and write deadlines.
func (c *UTPConn) SetDeadline(t time.Time) error {
	return nil
}

// SetReadDeadline sets the read deadline.
func (c *UTPConn) SetReadDeadline(t time.Time) error {
	return nil
}

// SetWriteDeadline sets the write deadline.
func (c *UTPConn) SetWriteDeadline(t time.Time) error {
	return nil
}
