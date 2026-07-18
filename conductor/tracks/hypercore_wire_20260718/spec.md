# Specification: Implement Hypercore v10 protocol wire implementation in Go

## 1. Goal
Implement the Hypercore v10 protocol serialization, deserialization, and replication state machine in Go. Cryptographically verify block integrity using Ed25519 Merkle tree signatures and Blake2b hashes. Persist blocks to flat files, and expose block get/append endpoints on the Unix socket.

## 2. Scope
### In-Scope
- **Protocol Message Coding**:
  - Implement protocol message serialization/deserialization for Hypercore v10 using protocol buffers (or standard binary encoding/decoding matching the spec: `handshake`, `want`, `have`, `request`, and `data`).
- **Merkle Tree & Crypto Verification**:
  - Implement Merkle tree hashing using Blake2b (leaf and parent node hashing).
  - Cryptographically verify block signatures against feed Ed25519 public keys.
- **Replication State Machine**:
  - Implement the protocol exchange state machine: exchanging handshakes, announcing possessed blocks (`have`), requesting missing blocks (`request`), and sending data payloads (`data`).
- **Log Block Flat-File Storage**:
  - Implement a flat-file block storage driver that appends blocks to a data file and maps byte offsets in an index file.
- **IPC Command endpoints**:
  - Expose JSON-RPC commands `get_block` (retrieving block by index) and `append_block` (appending data to local log) via `/tmp/keet-adk.sock`.

### Out-of-Scope (Future Tracks)
- PostgreSQL database persistence of log blocks.
- Swarm swarming integration with dynamic peer logs replication (handled in a future synchronization track).

## 3. Tech Stack
- Go 1.24+ standard library.
- Cryptography: `crypto/ed25519`, `crypto/sha512`, `golang.org/x/crypto/blake2b`.
- Protocol Buffers: standard protobuf library (if using protobufs) or binary structures.
