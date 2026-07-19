# Implementation Plan: Implement UTP/µTP Reliable Transport over UDP

## Phase 1: Packet Protocol and Simulation Layer [checkpoint: e9c44fc]

- [x] Task: Design and Implement µTP Packet Structure and Serialization [9514064]
    - [x] Write unit tests verifying packet serialization, deserialization, and header validation (red phase).
    - [x] Implement packet header structure and encoding/decoding functions in `pkg/utp/packet.go` (green phase).
- [x] Task: Create UDP Lossy Network Simulator [ce08a0a]
    - [x] Write unit tests verifying packet drop, delay jitter, and out-of-order delivery logic under the simulator (red phase).
    - [x] Implement `pkg/utp/simulator.go` wrapping UDP PacketConn to simulate lossy networks (green phase).
- [x] Task: Conductor - User Manual Verification 'Phase 1: Packet Protocol and Simulation Layer' (Protocol in workflow.md) [e9c44fc]

## Phase 2: Connection Management and State Machine [checkpoint: 47f70ee]

- [x] Task: Implement Socket Multiplexer (SocketMux) [4204631]
    - [x] Write unit tests verifying UDP packet demultiplexing to correct connections by source address and Connection ID (red phase).
    - [x] Implement connection tracking, incoming packet dispatching, and local binding in `pkg/utp/mux.go` (green phase).
- [x] Task: Implement Connection Handshake & Lifecycle [77150df]
    - [x] Write unit tests verifying state transitions for client connection (SYN -> STATE/SYN-ACK -> Connected) and listener (SYN -> SYN-ACK -> Connected) (red phase).
    - [x] Implement connection creation, handshake processing, and listener state transition logic in `pkg/utp/conn.go` and `pkg/utp/listener.go` (green phase).
- [x] Task: Implement Connection Teardown [ffd7d69]
    - [x] Write unit tests verifying FIN handshakes, ACK confirmation, and RST cleanup flows (red phase).
    - [x] Implement connection termination logic and connection map cleanup (green phase).
- [x] Task: Conductor - User Manual Verification 'Phase 2: Connection Management and State Machine' (Protocol in workflow.md) [47f70ee]

## Phase 3: Reliability, LEDBAT Congestion Control, and Flow Control [checkpoint: b96353b]

- [x] Task: Implement Reliable Data Transfer and Window Management [be334e9]
    - [x] Write unit tests verifying packet ACK matching, duplicate ACK count, Fast Retransmit, and Retransmission Timeout (RTO) calculations (red phase).
    - [x] Implement sequence buffers, retransmission queues, and acknowledgment logic in `pkg/utp/reliability.go` (green phase).
- [x] Task: Implement LEDBAT Congestion Control [8e35476]
    - [x] Write unit tests verifying one-way delay calculation, base delay tracking, queuing delay calculation, and cwnd scaling (red phase).
    - [x] Implement LEDBAT congestion window calculation and window adjustments per incoming ACK in `pkg/utp/ledbat.go` (green phase).
- [x] Task: Implement net.Conn and net.Listener standard interfaces [5c04dfa]
    - [x] Write unit tests verifying standard `Read`/`Write` stream interface behavior, read/write deadlines, and concurrent call safety (red phase).
    - [x] Implement standard `net.Conn` and `net.Listener` methods on `UTPConn` and `UTPListener` (green phase).
- [x] Task: Conductor - User Manual Verification 'Phase 3: Reliability, LEDBAT Congestion Control, and Flow Control' (Protocol in workflow.md) [b96353b]

## Phase 4: Swarming Integration and End-to-End Testing

- [x] Task: Integrate UTP transport with connection and DHT swarming layers [1af9f41]
    - [x] Write unit tests verifying connection dialing and swarming peer connections over UTP (red phase).
    - [x] Replace or extend standard socket dialing/listening logic in the P2P wiring and gateway manager to use UTP (green phase).
- [ ] Task: Complete End-to-End P2P gateway validation
    - [ ] Write integration tests validating Noise handshake, Hypercore replication, and chat message sync over UTP transport (red phase).
    - [ ] Verify everything compiles cleanly and passes all test suites with >80% coverage (green phase).
- [ ] Task: Conductor - User Manual Verification 'Phase 4: Swarming Integration and End-to-End Testing' (Protocol in workflow.md)
