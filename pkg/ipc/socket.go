package ipc

import (
	"fmt"
	"net"
	"os"
)

type SocketListener struct {
	listener net.Listener
	path     string
}

func NewSocketListener(path string) (*SocketListener, error) {
	// Clean up stale socket file if it exists
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to clear stale socket: %w", err)
	}

	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, fmt.Errorf("failed to bind Unix socket: %w", err)
	}

	return &SocketListener{
		listener: listener,
		path:     path,
	}, nil
}

func (s *SocketListener) Accept() (net.Conn, error) {
	return s.listener.Accept()
}

func (s *SocketListener) Close() error {
	var err error
	if s.listener != nil {
		err = s.listener.Close()
	}
	// Clean up socket file on close
	_ = os.Remove(s.path)
	return err
}
