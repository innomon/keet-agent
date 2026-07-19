package utp

import (
	"errors"
	"io"
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
	// STATE_FIN_SENT means client/server ST_FIN has been sent.
	STATE_FIN_SENT
	// STATE_CLOSED means connection is closed.
	STATE_CLOSED
)

// UTPConn implements net.Conn using the µTP reliable UDP stream transport.
type UTPConn struct {
	state         ConnState
	mux           *SocketMux
	remoteAddr    net.Addr
	recvID        uint16
	sendID        uint16
	seq           uint16
	ack           uint16
	readBuf       chan *Packet
	closeChan     chan struct{}
	finAcked      chan struct{}
	ackChan       chan uint16
	readQueue     chan []byte
	unread        []byte
	readDeadline  time.Time
	writeDeadline time.Time
	mu            sync.Mutex
}

// DialUTP initiates a new µTP connection to the target remote address using the provided SocketMux.
func DialUTP(mux *SocketMux, addr net.Addr) (*UTPConn, error) {
	s := uint16(rand.Intn(60000) + 1000)
	recvID := s + 1
	sendID := s

	c := &UTPConn{
		state:     STATE_SYN_SENT,
		mux:       mux,
		remoteAddr: addr,
		recvID:    recvID,
		sendID:    sendID,
		seq:       1,
		readBuf:   make(chan *Packet, 100),
		closeChan: make(chan struct{}),
		finAcked:  make(chan struct{}, 1),
		ackChan:   make(chan uint16, 100),
		readQueue: make(chan []byte, 100),
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
			go c.run()
			return c, nil
		}
	case <-time.After(1 * time.Second):
	}

	mux.DeregisterConn(recvID)
	return nil, errors.New("µTP connection handshake timed out")
}

func (c *UTPConn) run() {
	for {
		select {
		case pkt := <-c.readBuf:
			c.handlePacket(pkt)
		case <-c.closeChan:
			return
		}
	}
}

func (c *UTPConn) handlePacket(pkt *Packet) {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch pkt.Header.Type {
	case ST_FIN:
		c.ack = pkt.Header.SeqNum
		// Send ACK (ST_STATE)
		ackPkt := &Packet{
			Header: Header{
				Type:    ST_STATE,
				Version: 1,
				ConnID:  c.sendID,
				AckNum:  pkt.Header.SeqNum,
			},
		}
		data, _ := ackPkt.Encode()
		_, _ = c.mux.conn.WriteTo(data, c.remoteAddr)

		c.state = STATE_CLOSED
		c.mux.DeregisterConn(c.recvID)
		close(c.closeChan)

	case ST_STATE:
		if c.state == STATE_FIN_SENT && pkt.Header.AckNum == c.seq {
			c.state = STATE_CLOSED
			c.mux.DeregisterConn(c.recvID)
			select {
			case c.finAcked <- struct{}{}:
			default:
			}
			close(c.closeChan)
		} else {
			select {
			case c.ackChan <- pkt.Header.AckNum:
			default:
			}
		}

	case ST_DATA:
		c.ack = pkt.Header.SeqNum
		// Send ACK (ST_STATE)
		ackPkt := &Packet{
			Header: Header{
				Type:    ST_STATE,
				Version: 1,
				ConnID:  c.sendID,
				AckNum:  pkt.Header.SeqNum,
			},
		}
		data, _ := ackPkt.Encode()
		_, _ = c.mux.conn.WriteTo(data, c.remoteAddr)

		select {
		case c.readQueue <- pkt.Payload:
		default:
		}
	}
}

// Read reads data from the connection.
func (c *UTPConn) Read(b []byte) (n int, err error) {
	c.mu.Lock()
	if len(c.unread) > 0 {
		n = copy(b, c.unread)
		c.unread = c.unread[n:]
		c.mu.Unlock()
		return n, nil
	}

	rd := c.readDeadline
	c.mu.Unlock()

	var timeoutChan <-chan time.Time
	if !rd.IsZero() {
		d := time.Until(rd)
		if d <= 0 {
			return 0, errors.New("i/o timeout")
		}
		timeoutChan = time.After(d)
	}

	select {
	case payload, ok := <-c.readQueue:
		if !ok {
			return 0, io.EOF
		}
		c.mu.Lock()
		n = copy(b, payload)
		if n < len(payload) {
			c.unread = payload[n:]
		}
		c.mu.Unlock()
		return n, nil
	case <-c.closeChan:
		return 0, io.EOF
	case <-timeoutChan:
		return 0, errors.New("i/o timeout")
	}
}

// Write writes data to the connection.
func (c *UTPConn) Write(b []byte) (n int, err error) {
	c.mu.Lock()
	if c.state != STATE_CONNECTED {
		c.mu.Unlock()
		return 0, errors.New("connection not open")
	}
	c.mu.Unlock()

	totalSent := 0
	chunkSize := 1400

	for totalSent < len(b) {
		end := totalSent + chunkSize
		if end > len(b) {
			end = len(b)
		}
		chunk := b[totalSent:end]

		c.mu.Lock()
		c.seq++
		seq := c.seq
		sendID := c.sendID
		ackNum := c.ack
		remoteAddr := c.remoteAddr
		mux := c.mux
		c.mu.Unlock()

		pkt := &Packet{
			Header: Header{
				Type:    ST_DATA,
				Version: 1,
				ConnID:  sendID,
				SeqNum:  seq,
				AckNum:  ackNum,
			},
			Payload: chunk,
		}
		data, err := pkt.Encode()
		if err != nil {
			return totalSent, err
		}

		// Retransmit loop
		for {
			_, err = mux.conn.WriteTo(data, remoteAddr)
			if err != nil {
				return totalSent, err
			}

			// Wait for ACK
			timeout := 100 * time.Millisecond
			c.mu.Lock()
			wd := c.writeDeadline
			c.mu.Unlock()

			var timeoutChan <-chan time.Time
			if !wd.IsZero() {
				d := time.Until(wd)
				if d <= 0 {
					return totalSent, errors.New("i/o timeout")
				}
				if d < timeout {
					timeoutChan = time.After(d)
				} else {
					timeoutChan = time.After(timeout)
				}
			} else {
				timeoutChan = time.After(timeout)
			}

			select {
			case ack := <-c.ackChan:
				if ack == seq {
					break
				}
				continue
			case <-timeoutChan:
				c.mu.Lock()
				isClosed := c.state == STATE_CLOSED
				c.mu.Unlock()
				if isClosed {
					return totalSent, errors.New("connection closed during write")
				}
				continue
			}
			break
		}

		totalSent = end
	}

	return totalSent, nil
}

// Close closes the connection.
func (c *UTPConn) Close() error {
	c.mu.Lock()
	if c.state == STATE_CLOSED {
		c.mu.Unlock()
		return nil
	}

	c.state = STATE_FIN_SENT
	c.seq++
	finSeq := c.seq
	sendID := c.sendID
	remoteAddr := c.remoteAddr
	mux := c.mux
	recvID := c.recvID
	finAckedChan := c.finAcked
	c.mu.Unlock()

	// Send ST_FIN
	fin := &Packet{
		Header: Header{
			Type:    ST_FIN,
			Version: 1,
			ConnID:  sendID,
			SeqNum:  finSeq,
		},
	}
	data, err := fin.Encode()
	if err != nil {
		return err
	}

	_, err = mux.conn.WriteTo(data, remoteAddr)
	if err != nil {
		return err
	}

	// Wait for FIN-ACK
	select {
	case <-finAckedChan:
		// Graceful close complete
	case <-time.After(1 * time.Second):
		// Timeout, force close
		c.mu.Lock()
		if c.state != STATE_CLOSED {
			c.state = STATE_CLOSED
			mux.DeregisterConn(recvID)
			close(c.closeChan)
		}
		c.mu.Unlock()
	}

	return nil
}

// LocalAddr returns the local network address.
func (c *UTPConn) LocalAddr() net.Addr {
	return utpAddr{c.mux.conn.LocalAddr()}
}

// RemoteAddr returns the remote network address.
func (c *UTPConn) RemoteAddr() net.Addr {
	return utpAddr{c.remoteAddr}
}

// SetDeadline sets the read and write deadlines.
func (c *UTPConn) SetDeadline(t time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.readDeadline = t
	c.writeDeadline = t
	return nil
}

// SetReadDeadline sets the read deadline.
func (c *UTPConn) SetReadDeadline(t time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.readDeadline = t
	return nil
}

// SetWriteDeadline sets the write deadline.
func (c *UTPConn) SetWriteDeadline(t time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.writeDeadline = t
	return nil
}
