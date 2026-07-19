package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/innomon/keet-adk-gateway/pkg/config"
	"github.com/innomon/keet-adk-gateway/pkg/crypto"
	"github.com/innomon/keet-adk-gateway/pkg/db"
	"github.com/innomon/keet-adk-gateway/pkg/dht"
	"github.com/innomon/keet-adk-gateway/pkg/hypercore"
	"github.com/innomon/keet-adk-gateway/pkg/ipc"
	"github.com/innomon/keet-adk-gateway/pkg/logger"
	"github.com/innomon/keet-adk-gateway/pkg/network"
)

// defaultFeedKey is the Hypercore feed identifier used by this node for P2P sync and IPC broadcast.
const defaultFeedKey = "default_feed"

func main() {
	// Optimize execution profile for Apple Silicon M4 multicore distribution
	runtime.GOMAXPROCS(runtime.NumCPU())

	cfg := config.LoadConfig()

	// Initialize structured multiplexed logging
	log, err := logger.Init(cfg)
	if err != nil {
		slog.Error("Failed to initialize structured logging", "err", err)
		os.Exit(1)
	}

	cl := logger.NewCustomLogger("SYSTEM")

	cl.Infof("Starting Keet-ADK Gateway Backend (Arch: %s, OS: %s, Cores: %d)",
		runtime.GOARCH,
		runtime.GOOS,
		runtime.NumCPU(),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Capture OS signals for graceful degradation
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	listener, err := ipc.NewSocketListener(cfg.SocketPath)
	if err != nil {
		cl.Errorf("Failed to initialize Unix domain socket: %v", err)
		os.Exit(1)
	}
	defer listener.Close()

	// Initialize DHT and Swarm Registry
	dhtNode, err := dht.NewDHTNode(nil)
	if err != nil {
		cl.Errorf("Failed to initialize DHT Node: %v", err)
		os.Exit(1)
	}
	swarmRegistry := dht.NewSwarmRegistry()

		// Initialize database storage backend (PostgreSQL or BBolt)
	swarmRepo, blockRepo, dbClose, err := db.InitDatabase(ctx, cfg)
	if err != nil {
		cl.Errorf("Failed to initialize database: %v", err)
		os.Exit(1)
	}
	defer dbClose()
	cl.Infof("Successfully connected to database storage backend (type: %s)", cfg.DBType)


	// Initialize Hypercore flat-file Storage
	hypercoreStorage, err := hypercore.NewStorage(cfg.StorageDir)
	if err != nil {
		cl.Errorf("Failed to initialize Hypercore Storage: %v", err)
		os.Exit(1)
	}
	defer hypercoreStorage.Close()

	// Load or generate node static identity private key
	nodePrivKey, err := crypto.LoadOrGenerateNodeKey(cfg.StorageDir)
	if err != nil {
		cl.Errorf("Failed to load or generate node key: %v", err)
		os.Exit(1)
	}

	// Initialize PeerManager
	pm := network.NewPeerManager(nodePrivKey, hypercoreStorage, blockRepo, defaultFeedKey)

	// Start PeerManager TCP Listener
	p2pAddr := fmt.Sprintf("%s:%s", cfg.P2PListenAddr, cfg.P2PPort)
	if err := pm.StartListener(ctx, p2pAddr); err != nil {
		cl.Errorf("Failed to start PeerManager listener: %v", err)
		os.Exit(1)
	}
	defer pm.Close()
	cl.Infof("P2P Listener running at address: %s", pm.Addr().String())

	p2pPort := uint16(pm.Addr().(*net.TCPAddr).Port)
	swarmRegistry.P2PPort = p2pPort
	dhtNode.SetP2PPort(p2pPort)

	// Start the DHT Node with re-hydration repo!
	if err := dhtNode.Start(ctx, swarmRepo); err != nil {
		cl.Errorf("Failed to start DHT Node: %v", err)
		os.Exit(1)
	}
	defer dhtNode.Stop()

	// Wire Swarm Discovery to PeerManager Auto-Dialing
	swarmRegistry.OnRegisterPeer = func(topic [32]byte, peerAddr string) {
		cl.Infof("Discovered new swarm peer: %s. Dialing...", peerAddr)
		go func() {
			if err := pm.DialPeer(ctx, peerAddr); err != nil {
				cl.Errorf("Failed to dial discovered swarm peer %s: %v", peerAddr, err)
			}
		}()
	}

	// Wire Replication to IPC notification broadcast
	pm.OnAppendBlock = func(index uint64, value []byte) {
		cl.Infof("Replicated block index %d received from peer. Broadcasting to ADK clients...", index)
		ipc.BroadcastChatMessage(defaultFeedKey, index, value)
	}

	cl.Infof("ADK Communication Socket Ready at path: %s", cfg.SocketPath)

	go func() {
		sig := <-sigChan
		cl.Infof("Termination signal %v received. Shutting down gateway gracefully...", sig)
		listener.Close()
		cancel()
		os.Exit(0)
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				log.Handler().Handle(context.Background(), slog.NewRecord(time.Now(), slog.LevelError, fmt.Sprintf("Socket connection accept failure: %v", err), 0))
				continue
			}
		}
		go ipc.HandleClient(ctx, conn, dhtNode, swarmRegistry, hypercoreStorage, swarmRepo, blockRepo)
	}
}
