package ipc

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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

func HandleClient(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	slog.Info("New ADK Client pipeline bound successfully", "remote", conn.RemoteAddr())

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			var req map[string]interface{}
			if err := decoder.Decode(&req); err != nil {
				slog.Warn("Disconnected or malformed ADK frames detected", "err", err)
				return
			}

			slog.Debug("ADK command intercept", "payload", req)

			resp := map[string]interface{}{"status": "acknowledged", "origin": "keet_peer"}
			if err := encoder.Encode(&resp); err != nil {
				slog.Error("Failed response serialization upstream to client", "err", err)
				return
			}
		}
	}
}
