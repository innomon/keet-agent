# Implementation Plan: Implement Production P2P Gateway Wiring

## Phase 1: Node Identity & P2P Listener Lifecycle [checkpoint: baa3b64]

- [x] Task: Node Private Key Persistence (TDD) (95dd289)
    - [x] Write unit test for loading/saving node private key in `pkg/crypto`
    - [x] Implement `LoadOrGenerateNodeKey(storageDir string) (ed25519.PrivateKey, error)` in `pkg/crypto/keys.go`
    - [x] Verify key persistence tests pass
- [x] Task: Gateway Main Integration & TCP Listener (TDD) (7abcdce)
    - [x] Write integration test verifying gateway boots `PeerManager` with local key and tcp listener
    - [x] Wire `PeerManager` initialization into `cmd/gateway/main.go` and start the TCP listener
    - [x] Verify listener tests pass
- [x] Task: Conductor - User Manual Verification 'Phase 1: Node Identity & P2P Listener Lifecycle' (Protocol in workflow.md)

## Phase 2: DHT Peer Discovery Auto-Dialing [checkpoint: 743cac4]

- [x] Task: DHT Peer Swarm Integration (TDD) (2ad3d5e)
    - [x] Write integration test verifying that when DHT registers a new peer, the gateway automatically dials the peer
    - [x] Wire DHT swarm registry additions to invoke `pm.DialPeer` in `cmd/gateway/main.go`
    - [x] Verify auto-dialing integration tests pass
- [x] Task: Conductor - User Manual Verification 'Phase 2: DHT Peer Discovery Auto-Dialing' (Protocol in workflow.md)

## Phase 3: P2P Socket Event Notification Integration

- [ ] Task: End-to-End P2P Sync & Socket Broadcast (TDD)
    - [ ] Write integration test verifying that P2P replication appends automatically broadcast `chat_message_received` notifications to Unix socket clients in the fully assembled gateway
    - [ ] Link `pm.OnAppendBlock` to `ipc.BroadcastChatMessage` inside `cmd/gateway/main.go`
    - [ ] Verify end-to-end integration tests pass
- [ ] Task: Conductor - User Manual Verification 'Phase 3: P2P Socket Event Notification Integration' (Protocol in workflow.md)
