package network

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"github.com/innomon/keet-adk-gateway/pkg/crypto"
	"github.com/innomon/keet-adk-gateway/pkg/db"
	"github.com/innomon/keet-adk-gateway/pkg/hypercore"
)

type PeerManager struct {
	localPriv ed25519.PrivateKey
	storage   *hypercore.Storage
	blockRepo *db.BlockRepository
	feedKey   string
	listener  net.Listener
	mu        sync.Mutex
	conns     map[string]net.Conn
	wg        sync.WaitGroup
	cancel    context.CancelFunc
}

func NewPeerManager(localPriv ed25519.PrivateKey, storage *hypercore.Storage, blockRepo *db.BlockRepository, feedKey string) *PeerManager {
	return &PeerManager{
		localPriv: localPriv,
		storage:   storage,
		blockRepo: blockRepo,
		feedKey:   feedKey,
		conns:     make(map[string]net.Conn),
	}
}

func (pm *PeerManager) StartListener(ctx context.Context, addr string) error {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	pm.listener = l

	ctx, cancel := context.WithCancel(ctx)
	pm.cancel = cancel

	pm.wg.Add(1)
	go func() {
		defer pm.wg.Done()
		for {
			conn, err := l.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					slog.Error("Failed to accept connection", "err", err)
					continue
				}
			}

			go pm.handleIncoming(ctx, conn)
		}
	}()

	return nil
}

func (pm *PeerManager) DialPeer(ctx context.Context, addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	pm.wg.Add(1)
	go func() {
		defer pm.wg.Done()
		pm.handleOutgoing(ctx, conn)
	}()

	return nil
}

func (pm *PeerManager) handleIncoming(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	secureConn, remotePub, err := crypto.NewSecureConnection(conn, pm.localPriv, false)
	if err != nil {
		slog.Error("Incoming Noise handshake failed", "err", err)
		return
	}
	defer secureConn.Close()

	peerKey := fmt.Sprintf("%x", remotePub)
	pm.mu.Lock()
	pm.conns[peerKey] = secureConn
	pm.mu.Unlock()

	defer func() {
		pm.mu.Lock()
		delete(pm.conns, peerKey)
		pm.mu.Unlock()
	}()

	session := hypercore.NewSyncSession(secureConn, pm.storage, pm.blockRepo, pm.feedKey, pm.localPriv, remotePub, false)
	if err := session.Run(ctx); err != nil {
		slog.Error("Incoming sync session error", "peer", peerKey, "err", err)
	}
}

func (pm *PeerManager) handleOutgoing(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	secureConn, remotePub, err := crypto.NewSecureConnection(conn, pm.localPriv, true)
	if err != nil {
		slog.Error("Outgoing Noise handshake failed", "err", err)
		return
	}
	defer secureConn.Close()

	peerKey := fmt.Sprintf("%x", remotePub)
	pm.mu.Lock()
	pm.conns[peerKey] = secureConn
	pm.mu.Unlock()

	defer func() {
		pm.mu.Lock()
		delete(pm.conns, peerKey)
		pm.mu.Unlock()
	}()

	session := hypercore.NewSyncSession(secureConn, pm.storage, pm.blockRepo, pm.feedKey, pm.localPriv, remotePub, true)
	if err := session.Run(ctx); err != nil {
		slog.Error("Outgoing sync session error", "peer", peerKey, "err", err)
	}
}

func (pm *PeerManager) Close() {
	if pm.cancel != nil {
		pm.cancel()
	}
	if pm.listener != nil {
		pm.listener.Close()
	}
	pm.mu.Lock()
	for _, conn := range pm.conns {
		conn.Close()
	}
	pm.mu.Unlock()
	pm.wg.Wait()
}
