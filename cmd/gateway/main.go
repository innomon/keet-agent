package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/innomon/keet-adk-gateway/pkg/config"
	"github.com/innomon/keet-adk-gateway/pkg/dht"
	"github.com/innomon/keet-adk-gateway/pkg/hypercore"
	"github.com/innomon/keet-adk-gateway/pkg/ipc"
	"github.com/innomon/keet-adk-gateway/pkg/logger"
)

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

	// Initialize Hypercore flat-file Storage
	hypercoreStorage, err := hypercore.NewStorage(cfg.StorageDir)
	if err != nil {
		cl.Errorf("Failed to initialize Hypercore Storage: %v", err)
		os.Exit(1)
	}
	defer hypercoreStorage.Close()

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
		go ipc.HandleClient(ctx, conn, dhtNode, swarmRegistry, hypercoreStorage)
	}
}
