package utp

import (
	"errors"
	"net"
)

// UTPListener implements the net.Listener interface for accepting incoming µTP connections.
type UTPListener struct {
	mux        *SocketMux
	acceptChan chan *UTPConn
	closeChan  chan struct{}
}

// NewUTPListener creates a UTPListener registered on the provided SocketMux.
func NewUTPListener(mux *SocketMux) *UTPListener {
	l := &UTPListener{
		mux:        mux,
		acceptChan: make(chan *UTPConn, 100),
		closeChan:  make(chan struct{}),
	}
	mux.RegisterListener(l)
	return l
}

// Accept waits for and returns the next connection to the listener.
func (l *UTPListener) Accept() (net.Conn, error) {
	select {
	case conn := <-l.acceptChan:
		return conn, nil
	case <-l.closeChan:
		return nil, errors.New("listener closed")
	}
}

// Close closes the listener.
func (l *UTPListener) Close() error {
	l.mux.DeregisterListener()
	close(l.closeChan)
	return nil
}

// Addr returns the listener's network address.
func (l *UTPListener) Addr() net.Addr {
	return l.mux.conn.LocalAddr()
}
