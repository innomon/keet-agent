# Implementation Plan - Implement Hypercore v10 protocol wire implementation in Go

This plan details the steps to implement Hypercore message encoding, flat-file storage, Merkle tree cryptographic verification, and IPC socket integration.

## Phase 1: Binary Frame Serialization and Flat File Storage

- [ ] Task: Protocol Message Encoding (TDD)
    - [ ] Write unit tests for protocol message serialization and deserialization
    - [ ] Implement encoding/decoding for `handshake`, `want`, `have`, `request`, and `data` messages in `pkg/hypercore/wire.go`
    - [ ] Verify serialization tests pass
- [ ] Task: Flat-File Log Block Storage (TDD)
    - [ ] Write unit tests for appends and offset retrievals on flat file storage
    - [ ] Implement flat-file block driver with index offsets in `pkg/hypercore/storage.go`
    - [ ] Verify storage driver tests pass
- [ ] Task: Conductor - User Manual Verification 'Phase 1: Binary Frame Serialization and Flat File Storage' (Protocol in workflow.md)

## Phase 2: Merkle Tree and Cryptographic Verification

- [ ] Task: Merkle Tree Math and Node Hashing (TDD)
    - [ ] Write unit tests for leaf and parent node calculations
    - [ ] Implement Merkle tree leaf/parent hashing using Blake2b in `pkg/hypercore/merkle.go`
    - [ ] Verify Merkle hashing tests pass
- [ ] Task: Ed25519 Signature Verification (TDD)
    - [ ] Write unit tests validating valid/invalid signatures on Merkle tree root hashes
    - [ ] Implement Ed25519 signature checks using public keys in `pkg/hypercore/crypto.go`
    - [ ] Verify cryptographic verification tests pass
- [ ] Task: Conductor - User Manual Verification 'Phase 2: Merkle Tree and Cryptographic Verification' (Protocol in workflow.md)

## Phase 3: IPC Socket Commands and Integration

- [ ] Task: IPC socket command extensions (TDD)
    - [ ] Write unit tests for `get_block` and `append_block` JSON-RPC frames over Unix socket
    - [ ] Extend socket client worker loop in `pkg/ipc/socket.go` to support `get_block` and `append_block`
    - [ ] Verify socket command tests pass
- [ ] Task: Gateway Integration and Verify Execution
    - [ ] Integrate Hypercore log storage and verification layer into `cmd/gateway/main.go`
    - [ ] Verify gateway starts, processes get/append commands, and logs block operations successfully
- [ ] Task: Conductor - User Manual Verification 'Phase 3: IPC Socket Commands and Integration' (Protocol in workflow.md)
