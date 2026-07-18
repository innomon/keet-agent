# Implementation Plan - Implement Kademlia-based HyperDHT P2P swarming and peer discovery in Go

This plan details the steps to implement HyperDHT bootstrapping, topic hashing, Noise-secured channels, and socket IPC endpoints.

## Phase 1: HyperDHT Node and Bootstrapping [checkpoint: a62ad41]

- [x] Task: DHT Configuration and Structs (TDD) (33a6e50)
    - [x] Write tests for configuration loading and custom bootstrap nodes
    - [x] Define DHT node structure and custom bootstrapper in `pkg/dht/dht.go`
    - [x] Verify configuration tests pass
- [x] Task: Topic Hashing and Swarm Resolution (TDD) (c5af8a5)
    - [x] Write tests for Blake2b topic hashing (validating string topics vs 32-byte raw topics)
    - [x] Implement Blake2b topic hashing in `pkg/dht/topic.go`
    - [x] Verify hashing tests pass
- [x] Task: Kademlia Routing and Node Discovery (TDD) (df8fcd6)
    - [x] Write tests for routing table updates and lookup functions
    - [x] Implement local Kademlia routing table and bootstrap node discovery protocol
    - [x] Verify routing table and discovery tests pass
- [x] Task: Conductor - User Manual Verification 'Phase 1: HyperDHT Node and Bootstrapping' (Protocol in workflow.md)

## Phase 2: Noise Handshake and Channel Security [checkpoint: 8a4e179]

- [x] Task: Noise Protocol Handshake (TDD) (94fb78d)
    - [x] Write tests for secure handshakes between mock peers
    - [x] Implement Noise handshakes (`Noise_XK_25519_ChaChaPoly_BLAKE2b`) using Ed25519 keys in `pkg/crypto/noise.go`
    - [x] Verify Noise handshake tests pass
- [x] Task: In-Memory Swarm Registry (TDD) (eaaba26)
    - [x] Write tests for thread-safe swarm registry updates
    - [x] Implement in-memory registry (`sync.Map`) for active swarms and connections in `pkg/dht/registry.go`
    - [x] Verify registry tests pass
- [x] Task: Conductor - User Manual Verification 'Phase 2: Noise Handshake and Channel Security' (Protocol in workflow.md)

## Phase 3: IPC Command API Integration

- [ ] Task: IPC command endpoint tests (TDD)
    - [ ] Write tests for `join_swarm` and `leave_swarm` JSON-RPC socket frames
    - [ ] Extend socket IPC router in `pkg/ipc/socket.go` to support `join_swarm` and `leave_swarm` commands
    - [ ] Verify IPC command tests pass
- [ ] Task: Integration and Gateway Run Verification
    - [ ] Integrate DHT nodes and swarm registry into `cmd/gateway/main.go`
    - [ ] Verify that running the gateway, sending socket join command, and swarming on a test topic discovery works and logs properly
- [ ] Task: Conductor - User Manual Verification 'Phase 3: IPC Command API Integration' (Protocol in workflow.md)
