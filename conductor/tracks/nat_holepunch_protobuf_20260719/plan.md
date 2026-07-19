# Implementation Plan: NAT Hole Punching and Hypercore Protobuf Wire Compliance

## Phase 1: STUN & TURN Client Helpers

- [x] Task: Implement STUN Binding Client Protocol [bc8ede8]
    - [x] Write unit tests verifying STUN binding request generation and response parsing (Mapped Address attribute extraction) (red phase).
    - [x] Implement STUN packet parsing, header definitions, and binding transaction logic in `pkg/utp/stun.go` (green phase).
- [x] Task: Implement TURN Client Protocol [df8a1c8]
    - [x] Write unit tests verifying TURN allocation request, permission creation, and Send/Data indication encapsulation (red phase).
    - [x] Implement TURN protocol state transitions, allocation management, and UDP relay framing in `pkg/utp/turn.go` (green phase).
- [ ] Task: Conductor - User Manual Verification 'Phase 1: STUN & TURN Client Helpers' (Protocol in workflow.md)

## Phase 2: ICE Hole Punching and Connection Fallbacks

- [ ] Task: Implement ICE Candidate Exchange and Hole Punching
    - [ ] Write unit tests verifying candidate exchange sequence, direct socket hole-punching attempts, and successful direct binding (red phase).
    - [ ] Implement UDP punch packet exchanges, candidate ranking/discovery, and direct UDP socket association in `pkg/utp/ice.go` (green phase).
- [ ] Task: Implement TURN UDP Relay Fallback
    - [ ] Write unit tests verifying relay fallback switching logic upon direct connection timeout or failure (red phase).
    - [ ] Integrate fallback relay transport path into client dialing and server listener handling in `pkg/utp/conn.go` (green phase).
- [ ] Task: Conductor - User Manual Verification 'Phase 2: ICE Hole Punching and Connection Fallbacks' (Protocol in workflow.md)

## Phase 3: Hypercore Wire Protobuf Spec Compliance

- [ ] Task: Define compliant Hypercore Wire Protobuf Schema and Serialization
    - [ ] Write unit tests verifying Handshake, Feed, Request, Data, and Cancel message layouts match standard protobuf layout precisely (red phase).
    - [ ] Generate or integrate compliant protobuf message structures, size validations, and strict constraint checks in `pkg/hypercore/protobuf.go` (green phase).
- [ ] Task: Implement Extension Messages and Compression
    - [ ] Write unit tests verifying capabilities exchange and Inflate/Deflate compression/decompression on wire frames (red phase).
    - [ ] Implement compression algorithms and extension payload parsing on replication sessions in `pkg/hypercore/sync.go` (green phase).
- [ ] Task: Conductor - User Manual Verification 'Phase 3: Hypercore Wire Protobuf Spec Compliance' (Protocol in workflow.md)

## Phase 4: Integration and End-to-End Traversal Testing

- [ ] Task: Integrate NAT Traversal with Peer Discovery and Replication
    - [ ] Write unit tests verifying automated candidate exchange upon DHT node registration and complete sync over traversal paths (red phase).
    - [ ] Wire the STUN/TURN/ICE traversal engine to `PeerManager` and DHT node callbacks in `pkg/network/tcp.go` (green phase).
- [ ] Task: End-to-End Verification and Coverage Gates
    - [ ] Write integration tests verifying E2E traversal and replication over simulated restricted NAT routers (red phase).
    - [ ] Verify clean builds, pass all tests, and check quality gate metrics with >80% coverage (green phase).
- [ ] Task: Conductor - User Manual Verification 'Phase 4: Integration and End-to-End Traversal Testing' (Protocol in workflow.md)
