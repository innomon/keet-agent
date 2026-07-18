# Initial Concept

A pure Go backend gateway that emulates a Keet chat peer and acts as a communications gateway for the Agent Development Kit (ADK)

---

# Product Guide: Keet Desktop Gateway for ADK (Pure Go)

## Vision
To build a high-performance, lightweight, pure Go backend gateway service that compiles natively for macOS (darwin/arm64) running on Apple Silicon. This service emulates a Keet chat peer, connecting directly to the Holepunch P2P network, and acts as a decentralized communications bridge for Agent Development Kit (ADK) clients over a local Unix domain socket interface.

## Core Features
1. **Autonomous P2P Networking (Downstream)**:
   - Emulate a Keet client to join swarms via HyperDHT using 32-byte hash keys (chat room topics).
   - Exchange messages using Hypercore protocol wire format (v10) with append-only distributed logs.
   - Use Ed25519 Merkle tree signatures and Blake2b hashing for block verification.
   - Secure P2P communication channels using the Noise handshake protocol.

2. **IPC Gateway Interface (Upstream)**:
   - Expose a local Unix domain socket interface at `/tmp/keet-adk.sock`.
   - Support asynchronous JSON-RPC or text-delimited stream protocol for Agent Development Kit (ADK) clients.
   - Graceful socket descriptor handling, cleanup on startup, and recovery from unexpected terminations.

3. **Storage & Configuration**:
   - Utilize a Postgres database for configuration, swarm registry, and metadata cache (SQLite is prohibited).
   - Store log histories and session metadata securely.

4. **Multi-core & Zero-Dependency Design**:
   - Leverage Go 1.24+ concurrency features (goroutine worker pools, channels, sync.Map) optimized for multi-core Apple Silicon (M4).
   - Minimal external dependency footprint, compiling to a single native binary.
   - Pure Go implementation of structured logging with size-based log rotation and caller frame preservation.
