# Implementation Plan - Implement Kademlia-based HyperDHT P2P swarming and peer discovery in Go

This plan details the steps to implement HyperDHT bootstrapping, topic hashing, Noise-secured channels, and socket IPC endpoints.

## Phase 1: HyperDHT Node and Bootstrapping

- [ ] Task: DHT Configuration and Structs (TDD)
    - [ ] Write tests for configuration loading and custom bootstrap nodes
    - [ ] Define DHT node structure and custom bootstrapper in `pkg/dht/dht.go`
    - [ ] Verify configuration tests pass
- [ ] Task: Topic Hashing and Swarm Resolution (TDD)
    - [ ] Write tests for Blake2b topic hashing (validating string topics vs 32-byte raw topics)
    - [ ] Implement Blake2b topic hashing in `pkg/dht/topic.go`
    - [ ] Verify hashing tests pass
- [ ] Task: Kademlia Routing and Node Discovery (TDD)
    - [ ] Write tests for routing table updates and lookup functions
    - [ ] Implement local Kademlia routing table and bootstrap node discovery protocol
    - [ ] Verify routing table and discovery tests pass
- [ ] Task: Conductor - User Manual Verification 'Phase 1: HyperDHT Node and Bootstrapping' (Protocol in workflow.md)

## Phase 2: Noise Handshake and Channel Security

- [ ] Task: Noise Protocol Handshake (TDD)
    - [ ] Write tests for secure handshakes between mock peers
    - [ ] Implement Noise handshakes (`Noise_XK_25519_ChaChaPoly_BLAKE2b`) using Ed25519 keys in `pkg/crypto/noise.go`
    - [ ] Verify Noise handshake tests pass
- [ ] Task: In-Memory Swarm Registry (TDD)
    - [ ] Write tests for thread-safe swarm registry updates
    - [ ] Implement in-memory registry (`sync.Map`) for active swarms and connections in `pkg/dht/registry.go`
    - [ ] Verify registry tests pass
- [ ] Task: Conductor - User Manual Verification 'Phase 2: Noise Handshake and Channel Security' (Protocol in workflow.md)

## Phase 3: IPC Command API Integration

- [ ] Task: IPC command endpoint tests (TDD)
    - [ ] Write tests for `join_swarm` and `leave_swarm` JSON-RPC socket frames
    - [ ] Extend socket IPC router in `pkg/ipc/socket.go` to support `join_swarm` and `leave_swarm` commands
    - [ ] Verify IPC command tests pass
- [ ] Task: Integration and Gateway Run Verification
    - [ ] Integrate DHT nodes and swarm registry into `cmd/gateway/main.go`
    - [ ] Verify that running the gateway, sending socket join command, and swarming on a test topic discovery works and logs properly
- [ ] Task: Conductor - User Manual Verification 'Phase 3: IPC Command API Integration' (Protocol in workflow.md)
