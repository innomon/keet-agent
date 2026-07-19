package network

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/innomon/keet-adk-gateway/pkg/crypto"
	"github.com/innomon/keet-adk-gateway/pkg/db"
	"github.com/innomon/keet-adk-gateway/pkg/hypercore"
	"github.com/innomon/keet-adk-gateway/pkg/utp"
)

// PeerManager coordinates active remote peer sessions and local block synchronization.
type PeerManager struct {
	localPriv     ed25519.PrivateKey
	storage       *hypercore.Storage
	blockRepo     db.BlockRepository
	feedKey       string
	listener      net.Listener
	mu            sync.Mutex
	conns         map[string]net.Conn
	sessions      map[string]*hypercore.SyncSession
	wg            sync.WaitGroup
	cancel        context.CancelFunc
	OnAppendBlock func(index uint64, value []byte)
	relayServer   string
	stunServer    string
}

// NewPeerManager instantiates a new PeerManager with the given credentials.
func NewPeerManager(localPriv ed25519.PrivateKey, storage *hypercore.Storage, blockRepo db.BlockRepository, feedKey string) *PeerManager {

	return &PeerManager{
		localPriv: localPriv,
		storage:   storage,
		blockRepo: blockRepo,
		feedKey:   feedKey,
		conns:     make(map[string]net.Conn),
		sessions:  make(map[string]*hypercore.SyncSession),
	}
}

func (pm *PeerManager) SetRelayServer(addr string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.relayServer = addr
}

func (pm *PeerManager) SetStunServer(addr string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.stunServer = addr
}

// StartListener starts the UDP socket, running a SocketMux and UTPListener to accept incoming connections.
func (pm *PeerManager) StartListener(ctx context.Context, addr string) error {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}
	packetConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}

	mux := utp.NewSocketMux(packetConn)
	if pm.relayServer != "" {
		mux.SetRelayServer(pm.relayServer)
	}
	mux.Start()

	l := utp.NewUTPListener(mux)
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

type clientUTPConn struct {
	net.Conn
	mux *utp.SocketMux
}

// Close overrides net.Conn.Close to also stop the client-side SocketMux.
func (c *clientUTPConn) Close() error {
	err := c.Conn.Close()
	c.mux.Stop()
	return err
}

// DialPeer establishes an outbound UTP connection to the target remote address.
func (pm *PeerManager) DialPeer(ctx context.Context, addr string) error {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}

	localConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		return err
	}

	mux := utp.NewSocketMux(localConn)
	if pm.relayServer != "" {
		mux.SetRelayServer(pm.relayServer)
	}
	mux.Start()

	conn, err := utp.DialUTPWithTimeoutAndRelay(mux, udpAddr, 1*time.Second)
	if err != nil {
		mux.Stop()
		return err
	}

	wrappedConn := &clientUTPConn{
		Conn: conn,
		mux:  mux,
	}

	pm.wg.Add(1)
	go func() {
		defer pm.wg.Done()
		pm.handleOutgoing(ctx, wrappedConn)
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

	session := hypercore.NewSyncSession(secureConn, pm.storage, pm.blockRepo, pm.feedKey, pm.localPriv, remotePub, false)
	session.OnAppendBlock = pm.OnAppendBlock

	peerKey := fmt.Sprintf("%x", remotePub)
	pm.mu.Lock()
	pm.conns[peerKey] = secureConn
	pm.sessions[peerKey] = session
	pm.mu.Unlock()

	defer func() {
		pm.mu.Lock()
		delete(pm.conns, peerKey)
		delete(pm.sessions, peerKey)
		pm.mu.Unlock()
	}()

	if err := session.Run(ctx); err != nil {
		errStr := err.Error()
		if ctx.Err() != nil || strings.Contains(errStr, "use of closed network connection") || strings.Contains(errStr, "EOF") || strings.Contains(errStr, "connection reset by peer") {
			slog.Debug("Incoming sync session stopped gracefully", "peer", peerKey, "err", err)
		} else {
			slog.Error("Incoming sync session error", "peer", peerKey, "err", err)
		}
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

	session := hypercore.NewSyncSession(secureConn, pm.storage, pm.blockRepo, pm.feedKey, pm.localPriv, remotePub, true)
	session.OnAppendBlock = pm.OnAppendBlock

	peerKey := fmt.Sprintf("%x", remotePub)
	pm.mu.Lock()
	pm.conns[peerKey] = secureConn
	pm.sessions[peerKey] = session
	pm.mu.Unlock()

	defer func() {
		pm.mu.Lock()
		delete(pm.conns, peerKey)
		delete(pm.sessions, peerKey)
		pm.mu.Unlock()
	}()

	if err := session.Run(ctx); err != nil {
		errStr := err.Error()
		if ctx.Err() != nil || strings.Contains(errStr, "use of closed network connection") || strings.Contains(errStr, "EOF") || strings.Contains(errStr, "connection reset by peer") {
			slog.Debug("Outgoing sync session stopped gracefully", "peer", peerKey, "err", err)
		} else {
			slog.Error("Outgoing sync session error", "peer", peerKey, "err", err)
		}
	}
}

// BroadcastHave broadcasts the current hypercore feed length to all active sync sessions.
func (pm *PeerManager) BroadcastHave(length uint64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	for _, s := range pm.sessions {
		if err := s.NotifyHave(length); err != nil {
			slog.Error("Failed to send Have broadcast to session", "err", err)
		}
	}
}

// Addr returns the listener address if active.
func (pm *PeerManager) Addr() net.Addr {
	if pm.listener != nil {
		return pm.listener.Addr()
	}
	return nil
}

// ConnCount returns the number of active peer connections. Used for testing.
func (pm *PeerManager) ConnCount() int {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return len(pm.conns)
}

// Close gracefully closes the listener and cleans up all active peer connections.
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
