# Specification: Implement P2P log synchronization and replication using Hypercore protocol over Noise secure connections

## 1. Goal
Implement secure P2P stream replication of append-only logs. Use the Noise protocol (XX pattern) to establish encrypted mutual-authenticated transport sessions over TCP or memory-pipes, and execute concurrent Hypercore replication loops exchanging wire-encoded frames (handshake, want, have, request, data).

## 2. Scope
### In-Scope
- **Noise Security Handshake Layer**:
  - Integrate Noise `XX` handshake pattern using `github.com/flynn/noise` to secure the raw TCP or pipe connection before starting replication.
- **Concurrent P2P Replication Protocol**:
  - Implement a peer session sync loop that writes and reads wire messages concurrently (using `pkg/hypercore/wire.go`).
  - Support protocol flow:
    - Exchange `handshake` (protocol ID verification).
    - Exchange `have` and `want` (identifying missing block indexes).
    - Request missing blocks via `request` frames.
    - Serve requested blocks using database or flat-file log blocks via `data` frames.
- **Transports & Testing**:
  - Implement a standard TCP socket dialer/listener replication server.
  - Implement unit/mock tests utilizing in-memory pipes (`net.Pipe`) to validate secure log sync.

### Out-of-Scope (Future Tracks)
- Peer discovery, routing tables, and Kademlia-based DHT integration (discovery of peers).
- Torrent-style pipelined block fetching from multiple peers concurrently.

## 3. Tech Stack
- Go 1.24+ standard library.
- Cryptography: `github.com/flynn/noise` (Noise protocol).
- Core: `pkg/hypercore` (wire encoding and repositories).
