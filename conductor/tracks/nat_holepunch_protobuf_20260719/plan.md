# Implementation Plan: NAT Hole Punching and Hypercore Protobuf Wire Compliance

## Phase 1: STUN & TURN Client Helpers [checkpoint: 1d133fe]

- [x] Task: Implement STUN Binding Client Protocol [bc8ede8]
    - [x] Write unit tests verifying STUN binding request generation and response parsing (Mapped Address attribute extraction) (red phase).
    - [x] Implement STUN packet parsing, header definitions, and binding transaction logic in `pkg/utp/stun.go` (green phase).
- [x] Task: Implement TURN Client Protocol [df8a1c8]
    - [x] Write unit tests verifying TURN allocation request, permission creation, and Send/Data indication encapsulation (red phase).
    - [x] Implement TURN protocol state transitions, allocation management, and UDP relay framing in `pkg/utp/turn.go` (green phase).
- [x] Task: Conductor - User Manual Verification 'Phase 1: STUN & TURN Client Helpers' (Protocol in workflow.md) [1d133fe]

## Phase 2: ICE Hole Punching and Connection Fallbacks [checkpoint: 51a4518]

- [x] Task: Implement ICE Candidate Exchange and Hole Punching [2898ede]
    - [x] Write unit tests verifying candidate exchange sequence, direct socket hole-punching attempts, and successful direct binding (red phase).
    - [x] Implement UDP punch packet exchanges, candidate ranking/discovery, and direct UDP socket association in `pkg/utp/ice.go` (green phase).
- [x] Task: Implement TURN UDP Relay Fallback [aaa30ab]
    - [x] Write unit tests verifying relay fallback switching logic upon direct connection timeout or failure (red phase).
    - [x] Integrate fallback relay transport path into client dialing and server listener handling in `pkg/utp/conn.go` (green phase).
- [x] Task: Conductor - User Manual Verification 'Phase 2: ICE Hole Punching and Connection Fallbacks' (Protocol in workflow.md) [51a4518]

## Phase 3: Hypercore Wire Protobuf Spec Compliance [checkpoint: 03b7db4]

- [x] Task: Define compliant Hypercore Wire Protobuf Schema and Serialization [73919fb]
    - [x] Write unit tests verifying Hypercore wire message validation rules (max frame length, missing fields constraint, capability field length checking) (red phase).
    - [x] Define hypercore wire messages (Feed, Handshake, Request, Data, Cancel) using a compatible lightweight protobuf schema/generator or manual parser in `pkg/network/protobuf.go` (green phase).
- [x] Task: Implement Extension Messages and Compression [f24193e]
    - [x] Write unit tests verifying capabilities exchange and Inflate/Deflate compression/decompression on wire frames (red phase).
    - [x] Implement compression algorithms and extension payload parsing on replication sessions in `pkg/hypercore/sync.go` (green phase).
- [x] Task: Conductor - User Manual Verification 'Phase 3: Hypercore Wire Protobuf Spec Compliance' (Protocol in workflow.md) [03b7db4]

## Phase 4: Integration and End-to-End Traversal Testing [checkpoint: 7e9580d]

- [x] Task: Integrate NAT Traversal with Peer Discovery and Replication [7bbfedf]
    - [x] Write unit tests verifying that PeerManager automatically initiates STUN/ICE traversal upon discovering a peer via HyperDHT, falling back to TURN relay connection (red phase).
    - [x] Implement ICE negotiation flow triggering and fallback dial integration inside `pkg/network/tcp.go` PeerManager dial loop (green phase).
- [x] Task: End-to-End Verification and Coverage Gates [977d329]
    - [x] Write integration tests verifying E2E traversal and replication over simulated restricted NAT routers (red phase).
    - [x] Verify clean builds, pass all tests, and check quality gate metrics with >80% coverage (green phase).
- [x] Task: Conductor - User Manual Verification 'Phase 4: Integration and End-to-End Traversal Testing' (Protocol in workflow.md) [7e9580d]
