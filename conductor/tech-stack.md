# Tech Stack: Keet Desktop Gateway for ADK

This document outlines the selected technology stack and system components for the Keet Desktop Gateway.

## 1. Core Development Platform
- **Programming Language**: Go (Golang 1.24+)
  - *Rationale*: Pure Go ensures static linking, zero dependencies on Javascript/NodeJS runtimes, and direct compilation to a native Darwin/ARM64 binary.
- **Target Platform**: macOS (GOOS=darwin, GOARCH=arm64)
  - *Rationale*: Optimized to run natively on Apple Silicon (M4 Desktop).

## 2. Peer-to-Peer Networking
- **DHT Discovery**: Custom Kademlia-based HyperDHT protocol handler in pure Go.
- **Data Replication**: Hypercore Protocol (v10) implementation in Go for parsing append-only cryptographic logs.
- **Cryptography**: Standard Go libraries (`crypto/ed25519`, `golang.org/x/crypto/noise` or standard Noise handshakes, `golang.org/x/crypto/blake2b`).

## 3. Inter-Process Communication (IPC)
- **Local Transport**: Unix Domain Sockets (`net.Listen("unix", "/tmp/keet-adk.sock")`).
- **Wire Format**: Line-delimited JSON-RPC 2.0 frames.

## 4. Storage & Persistence
- **Database**: PostgreSQL (v16+)
  - *Driver*: `github.com/jackc/pgx/v5` (High-performance, pure Go PostgreSQL driver).
  - *Rationale*: Safe concurrent connection pooling, standard compliance, robust and secure data handling (SQLite is not used).

## 5. Operations & Tooling
- **Structured Logging**: Go standard library `log/slog` for structured logs.
- **Log Rotation**: Handcrafted size-based file rotation in Go (dependency-free).
- **CLI/Slash Commands**: Handcrafted command registry (strictly avoiding Cobra/Pflag).
- **Concurrency**: Goroutine worker pools, channel multiplexing, and `sync.Map` for thread-safe session tracking.
