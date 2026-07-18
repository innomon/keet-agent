# Coding Agent Instruction Prompt: Keet Desktop Gateway for ADK (Pure Go)

This document contains a structured prompt optimized for an AI coding agent (e.g., Cursor, Claude Engineer, or GitHub Copilot) to design, build, and optimize a backend gateway written entirely in pure Go. It targets Apple Silicon (Mac Mini M4) and implements the **WhatsADK Architecture Blueprint**, swapping out WhatsApp for the peer-to-peer **Keet** (Holepunch) protocol stack.

---

## Agent System Prompt & Context

```text
You are an expert systems engineer and senior Go developer specializing in high-performance networking, peer-to-peer (P2P) systems, and Unix domain socket IPC. 

### Mission
Your objective is to build a pure Go backend gateway service that compiles for macOS (darwin/arm64) running on an Apple Silicon M4 desktop. This service must natively emulate a Keet chat peer, connect directly to the Holepunch P2P network, and act as a communications gateway for the Agent Development Kit (ADK) framework using a unified Unix domain socket interface—mirroring the architectural concept of 'whatsadk' (https://github.com/innomon/whatsadk).

### Target Architecture Overview
1. Downstream (To Keet Network): Act as a fully autonomous Keet client. Join swarms via HyperDHT, maintain an append-only distributed log using the Hypercore protocol wire format (v10 format, Merkle tree signatures via Ed25519, and Blake2b hashing), and handle peer-to-peer direct/group message exchanges natively without spinning up external Node.js/Pear sidecars.
2. Upstream (To ADK Client): Expose an asynchronous JSON-RPC or text-delimited stream API over a local Unix domain socket (/tmp/keet-adk.sock). This allows AI agents using the ADK format to treat the peer as a standard programmable transport layer.

### Technical Constraints
- Language: 100% Pure Go (Golang 1.24+). Minimal external CGO or JS runtime wrappers.
- Optimization: Leverage the 10-core CPU topology of the Mac Mini M4. Use concurrent goroutine worker pools and channels for highly performant I/O multiplexing.
- Encryption & Protocol compliance: Native implementation of the Hypercore v10 replication protocol, Noise framework handshakes for HyperDHT peer connections, and end-to-end encryption schemas matching Keet.
```

---

## Step-by-Step Implementation Strategy

Execute the creation of this project in 4 distinct phases. Provide production-ready, clean, idiomatic Go code for each step. Do not use pseudo-code placeholders.

### Phase 1: Project Setup & Core Data Structures
1. Initialize a Go module named `github.com/youruser/keet-adk-gateway`.
2. Set up the target platform flags for cross-compilation/native execution on `GOOS=darwin GOARCH=arm64`.
3. Design the core message types matching the `whatsadk` message lifecycle payload:
   - `InboundMessage`: Captures peer cryptographic key, room topic, text contents, timestamp, and signature status.
   - `OutboundMessage`: Structure for sending data to a room topic or explicit peer public key.
   - `GatewayStatus`: Tracks HyperDHT peer counts, active swarms, and ADK socket client attachments.

### Phase 2: Hypercore and HyperDHT Wire Implementation in Go
1. Implement the Kademlia-based **HyperDHT** connection layer:
   - Handle target discovery using 32-byte hash keys ("Topics") representing chat rooms or direct swarms.
   - Integrate an Ed25519-based Noise handshake layer for secure P2P channels.
2. Build or embed a lightweight, pure-Go compatible client for **Hypercore v10**:
   - Parse and write signed, append-only logs utilizing the Merkle tree layout.
   - Ensure cryptographic validation matches the Holepunch reference (Blake2b hashing and Ed25519 signatures).

### Phase 3: ADK-Compliant IPC Layer (Unix Domain Sockets)
1. Write a resilient listener using the `net` package that binds to `/tmp/keet-adk.sock`.
2. Implement auto-cleanup mechanics for old socket descriptors upon initialization and graceful termination signals (`SIGINT`, `SIGTERM`).
3. Handle concurrent stream multiplexing: every ADK client request must map to an asynchronous worker loop. Use thread-safe map structures (`sync.Map`) or coordinator channels to route responses to correct downstream peer targets.

### Phase 4: Main Loop & Multi-Core Concurrency Optimization
1. In `main.go`, instantiate runtime settings using `runtime.GOMAXPROCS(runtime.NumCPU())` to utilize all M4 cores.
2. Setup error handling and automated recovery inside the core loop to avoid panics from faulty peer connections or malformed protocol packets.
3. Add full logging capability using `log/slog` for zero-allocation structured logs tracking P2P traffic and ADK inputs.

---

### Go Development
- **Style**: Adhere strictly to [Effective Go](https://go.dev/doc/effective_go) and Go Code Review Comments.
- **Formatting**: Always use `gofmt` and `goimports`.
- **Error Handling**: Never ignore errors with `_`. Handle them explicitly and return early to reduce nesting.
- **Testing**: Use table-driven tests with `t.Run()`. Tests must reside in the same package as the code they test.
- **Concurrency**: Use `context.Context` as the first parameter for functions involving cancellation or timeouts.
- use on postgres database, **DO NOT** use sqilite for this project.
- never use the spf13 library (Cobra/Pflag). Instead, always implement a handcrafted command registry for CLI and slash commands.


---

## Blueprint Reference Implementation Starter Code

Generate the following scaffold structure to jumpstart the agent's workspace:

```go
// main.go
package main

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

type GatewayConfig struct {
	SocketPath string
	KeyPair    ed25519.PrivateKey
}

func main() {
	// Optimize execution profile for Apple Silicon M4 multicore distribution
	runtime.GOMAXPROCS(runtime.NumCPU())

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.Info("Starting Keet-ADK Gateway Backend", 
		"arch", runtime.GOARCH, 
		"os", runtime.GOOS, 
		"cores", runtime.NumCPU(),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Capture OS signals for graceful degradation
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	socketPath := "/tmp/keet-adk.sock"
	
	// Clean up lingering sockets from ungraceful deaths
	if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
		slog.Error("Failed to clear stale socket", "err", err)
		os.Exit(1)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		slog.Error("Failed to initialize Unix domain socket", "err", err)
		os.Exit(1)
	}
	defer listener.Close()

	slog.Info("ADK Communication Socket Ready", "path", socketPath)

	// TODO: Initialize HyperDHT cluster node and bootstrap local Hypercore store

	go func() {
		<-sigChan
		slog.Info("Termination signal received. Shutting down gateway gracefully...")
		listener.Close()
		os.Remove(socketPath)
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
				slog.Error("Socket connection accept failure", "err", err)
				continue
			}
		}
		go handleADKClient(ctx, conn)
	}
}

func handleADKClient(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	slog.Info("New ADK Client pipeline bound successfully")

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
			
			// Processing loop for inbound instructions from agent frameworks
			slog.Debug("ADK command intercept", "payload", req)
			
			resp := map[string]interface{}{"status": "acknowledged", "origin": "keet_peer"}
			if err := encoder.Encode(&resp); err != nil {
				slog.Error("Failed response serialization upstream to client", "err", err)
				return
			}
		}
	}
}
```

---

## Success Criteria Evaluation
When executing this codebase, evaluate success based on these parameters:
- **Zero JS Dependency**: The build pipeline compiles directly to an independent native `darwin/arm64` executable binary using only the standard `go build` toolchain.
- **P2P Network Discovery**: The Go app can actively broadcast its availability via a mock HyperDHT swarm using target chat topics.
- **IPC Reliability**: The gateway process safely accepts, routes, and flushes concurrent read/write streams coming from external ADK instances through the Unix domain socket interface.
