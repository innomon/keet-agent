# Specification: Implement Kademlia-based HyperDHT P2P swarming and peer discovery in Go

## 1. Goal
Implement Kademlia-based HyperDHT peer discovery and connection routing in Go. Secure the resulting peer channels with Noise protocol handshakes using Ed25519 keys, mapping active swarms in-memory, and expose DHT join/leave commands through the Unix socket.

## 2. Scope
### In-Scope
- **DHT Routing & Bootstrapping**:
  - Configure default public Holepunch DHT bootstrap nodes.
  - Support env variables `DHT_BOOTSTRAP_NODES` to override bootstrap node lists.
  - Implement Kademlia DHT target discovery using 32-byte hash keys.
- **Topic Key Resolution**:
  - Support raw 32-byte cryptographic hashes for topics.
  - Support human-readable strings as topics, automatically hashing them with Blake2b (32-byte output) to join swarms.
- **Noise Cryptographic Security**:
  - Secure DHT connection channels using Noise handshakes (`Noise_XK_25519_ChaChaPoly_BLAKE2b`) matching the Holepunch protocol.
- **IPC Protocol API Integration**:
  - Add JSON-RPC methods `join_swarm` and `leave_swarm` to `/tmp/keet-adk.sock`.
  - Handle inputs containing topic (raw hex or plain string) and local peer public key.
- **In-Memory Registry**:
  - Manage active swarms, connection handles, and peer metadata thread-safely in-memory using `sync.Map`.

### Out-of-Scope (Future Tracks)
- Hypercore append-only log replication (Phase 2 wire protocol).
- PostgreSQL persistence of swarm histories or peer directory caches.

## 3. Tech Stack
- Go 1.24+ standard library.
- Cryptographic packages: `golang.org/x/crypto/noise`, `golang.org/x/crypto/blake2b`, `crypto/ed25519`.
- Unix domain socket IPC framework.
