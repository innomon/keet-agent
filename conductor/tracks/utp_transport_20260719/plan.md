# Implementation Plan: Implement UTP/µTP Reliable Transport over UDP

## Phase 1: Packet Protocol and Simulation Layer [checkpoint: e9c44fc]

- [x] Task: Design and Implement µTP Packet Structure and Serialization [9514064]
    - [x] Write unit tests verifying packet serialization, deserialization, and header validation (red phase).
    - [x] Implement packet header structure and encoding/decoding functions in `pkg/utp/packet.go` (green phase).
- [x] Task: Create UDP Lossy Network Simulator [ce08a0a]
    - [x] Write unit tests verifying packet drop, delay jitter, and out-of-order delivery logic under the simulator (red phase).
    - [x] Implement `pkg/utp/simulator.go` wrapping UDP PacketConn to simulate lossy networks (green phase).
- [x] Task: Conductor - User Manual Verification 'Phase 1: Packet Protocol and Simulation Layer' (Protocol in workflow.md) [e9c44fc]

## Phase 2: Connection Management and State Machine

- [x] Task: Implement Socket Multiplexer (SocketMux) [4204631]
    - [x] Write unit tests verifying UDP packet demultiplexing to correct connections by source address and Connection ID (red phase).
    - [x] Implement connection tracking, incoming packet dispatching, and local binding in `pkg/utp/mux.go` (green phase).
- [x] Task: Implement Connection Handshake & Lifecycle [77150df]
    - [x] Write unit tests verifying state transitions for client connection (SYN -> STATE/SYN-ACK -> Connected) and listener (SYN -> SYN-ACK -> Connected) (red phase).
    - [x] Implement connection creation, handshake processing, and listener state transition logic in `pkg/utp/conn.go` and `pkg/utp/listener.go` (green phase).
- [ ] Task: Implement Connection Teardown
    - [ ] Write unit tests verifying FIN handshakes, ACK confirmation, and RST cleanup flows (red phase).
    - [ ] Implement connection termination logic and connection map cleanup (green phase).
- [ ] Task: Conductor - User Manual Verification 'Phase 2: Connection Management and State Machine' (Protocol in workflow.md)

## Phase 3: Reliability, LEDBAT Congestion Control, and Flow Control

- [ ] Task: Implement Reliable Data Transfer and Window Management
    - [ ] Write unit tests verifying packet ACK matching, duplicate ACK count, Fast Retransmit, and Retransmission Timeout (RTO) calculations (red phase).
    - [ ] Implement sequence buffers, retransmission queues, and acknowledgment logic in `pkg/utp/reliability.go` (green phase).
- [ ] Task: Implement LEDBAT Congestion Control
    - [ ] Write unit tests verifying one-way delay calculation, base delay tracking, queuing delay calculation, and cwnd scaling (red phase).
    - [ ] Implement LEDBAT congestion window calculation and window adjustments per incoming ACK in `pkg/utp/ledbat.go` (green phase).
- [ ] Task: Implement net.Conn and net.Listener standard interfaces
    - [ ] Write unit tests verifying standard `Read`/`Write` stream interface behavior, read/write deadlines, and concurrent call safety (red phase).
    - [ ] Implement standard `net.Conn` and `net.Listener` methods on `UTPConn` and `UTPListener` (green phase).
- [ ] Task: Conductor - User Manual Verification 'Phase 3: Reliability, LEDBAT Congestion Control, and Flow Control' (Protocol in workflow.md)

## Phase 4: Swarming Integration and End-to-End Testing

- [ ] Task: Integrate UTP transport with connection and DHT swarming layers
    - [ ] Write unit tests verifying connection dialing and swarming peer connections over UTP (red phase).
    - [ ] Replace or extend standard socket dialing/listening logic in the P2P wiring and gateway manager to use UTP (green phase).
- [ ] Task: Complete End-to-End P2P gateway validation
    - [ ] Write integration tests validating Noise handshake, Hypercore replication, and chat message sync over UTP transport (red phase).
    - [ ] Verify everything compiles cleanly and passes all test suites with >80% coverage (green phase).
- [ ] Task: Conductor - User Manual Verification 'Phase 4: Swarming Integration and End-to-End Testing' (Protocol in workflow.md)
