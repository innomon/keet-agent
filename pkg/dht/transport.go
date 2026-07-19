package dht

import (
	"net"
)

// Transport defines the network interface for DHT message exchange.
type Transport interface {
	// ReadFrom reads a packet from the transport.
	ReadFrom(p []byte) (n int, addr net.Addr, err error)
	// WriteTo writes a packet to the specified address.
	WriteTo(p []byte, addr net.Addr) (n int, err error)
	// Addr returns the local address of the transport.
	Addr() net.Addr
	// Close closes the transport.
	Close() error
}

// UDPTransport is a concrete implementation of Transport using a UDP network connection.
type UDPTransport struct {
	conn net.PacketConn
}

// NewUDPTransport creates a new UDPTransport bound to the specified address.
func NewUDPTransport(addr string) (*UDPTransport, error) {
	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		return nil, err
	}
	return &UDPTransport{conn: conn}, nil
}

// ReadFrom reads a packet from the UDP connection.
func (u *UDPTransport) ReadFrom(p []byte) (int, net.Addr, error) {
	return u.conn.ReadFrom(p)
}

// WriteTo writes a packet to the specified UDP address.
func (u *UDPTransport) WriteTo(p []byte, addr net.Addr) (int, error) {
	return u.conn.WriteTo(p, addr)
}

// Addr returns the local address of the UDP connection.
func (u *UDPTransport) Addr() net.Addr {
	return u.conn.LocalAddr()
}

// Close closes the UDP connection.
func (u *UDPTransport) Close() error {
	return u.conn.Close()
}
