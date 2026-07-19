# Specification: Implement Production P2P Gateway Wiring

## 1. Overview
This specification defines the integration of the P2P networking layer (`PeerManager`, `SyncSession`, Noise `XX` handshakes) with the production backend gateway (`main.go`). It binds the local Unix socket client handler, DHT peer discovery, PostgreSQL caching, and loopback socket listeners together.

## 2. Functional Requirements
1. **Node Identity Management**:
   - The gateway must load an Ed25519 static identity key pair from a file named `node_key.priv` inside the configured storage directory.
   - If the file does not exist, a new key pair must be generated, stored securely in `node_key.priv`, and used for Noise XX handshake static key validation.
2. **P2P Socket Listener**:
   - Start a TCP socket listener on `P2P_PORT` (configured via configuration/environment).
   - Default to random port allocation (`:0`) if no specific port is configured, and output the resolved listening address to structured logs.
3. **Automated Swarm Dialing & Syncing**:
   - When a local client issues a `join_swarm` command, the DHT discovers swarm peer addresses.
   - The gateway must automatically trigger `DialPeer` on the `PeerManager` to initiate handshakes with newly discovered peers.
4. **Live IPC Sync Notifications**:
   - Bind `OnAppendBlock` in `PeerManager` to trigger `ipc.BroadcastChatMessage` when new log blocks arrive from peers.
   - Ensure the JSON-RPC `chat_message_received` frame is instantly broadcasted to all active Unix socket connections.

## 3. Acceptance Criteria
- Starting `cmd/gateway` boots up the TCP P2P engine and loads/persists the node identity file.
- Discovered peers are dialed automatically.
- Replicated chat messages generate asynchronous IPC notifications to local clients.
