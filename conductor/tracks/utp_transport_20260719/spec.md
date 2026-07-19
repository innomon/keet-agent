# Specification: Implement UTP/µTP Reliable Transport over UDP

## 1. Overview
This track implements a custom, lightweight Go µTP (Micro Transport Protocol) implementation from scratch inside a new package `pkg/utp`. This implementation provides a reliable stream socket transport over UDP using `net.Conn` and `net.Listener` interfaces. It also implements the standard LEDBAT (Low Extra Delay Background Transport) congestion control algorithm. Finally, it integrates this transport into the gateway's swarming/connection management layers to replace plain TCP or mock transport, aligning with Holepunch's production-grade network topology.

---

## 2. Functional Requirements

### 2.1 µTP Packet Format & Protocol Engine
- Implement the µTP packet header structure (pure Go, binary encoding):
  - **Type (4 bits)**: `ST_DATA` (0), `ST_FIN` (1), `ST_STATE` (2), `ST_RESET` (3), `ST_SYN` (4).
  - **Version (4 bits)**: Version 1.
  - **Extension (8 bits)**: Extension type (0 = none, supporting basic headers first).
  - **Connection ID (16 bits)**: Flow identifier. A connection uses `recv_id` for incoming packets and `send_id = recv_id + 1` (or vice versa) for outgoing packets.
  - **Timestamp Microseconds (32 bits)**: Microsecond clock of the sender at transmission time.
  - **Timestamp Difference Microseconds (32 bits)**: Microsecond difference between local receive time and remote send time (`difference = local_time - remote_timestamp`).
  - **Window Size (32 bits)**: Advertised receive window (bytes) to prevent buffer overflow.
  - **Sequence Number (16 bits)**: Packet sequence number (starts randomly, increments per data/FIN packet).
  - **Acknowledgment Number (16 bits)**: Sequence number of the last successfully received packet.
- Implement Connection States: `STATE_NONE`, `STATE_SYN_SENT`, `STATE_CONNECTED`, `STATE_FIN_SENT`, `STATE_CLOSED`.
- Implement reliable transport features:
  - Packet acknowledgment: ACKs are carried inside `ST_STATE` or data packets.
  - Fast Retransmit: Trigger packet retransmission upon receiving 3 duplicate ACKs.
  - Retransmission Timeout (RTO): Exponential backoff based on measured RTT (Round Trip Time) and variance.

### 2.2 LEDBAT Congestion Control
- Calculate current one-way delay: `delay = local_time - timestamp_microseconds`.
- Maintain a running base delay (minimum observed delay over a sliding window of the last 2 minutes).
- Calculate current queuing delay: `queuing_delay = current_delay - base_delay`.
- Implement dynamic congestion window (`cwnd`) adjustment:
  - Target queuing delay (`TARGET`): 100 milliseconds (100,000 microseconds).
  - Delay difference (`off_target`): `TARGET - queuing_delay`.
  - Adjust `cwnd` proportional to `off_target`:
    - If `off_target > 0` (delay is below target), increase `cwnd`.
    - If `off_target < 0` (delay is above target), decrease `cwnd`.
    - Update rule: `cwnd += GAIN * off_target / cwnd` per ACK.
  - Clamp `cwnd` to a minimum of 2 packets and maximum based on receiver advertised window.

### 2.3 Socket Multiplexer (`net.Listener` and `net.Conn`)
- Implement a UDP socket multiplexer (`SocketMux`) running a single background read loop on the UDP port.
- Demultiplex incoming packets to active connection objects (`UTPConn`) based on their Connection ID and source address.
- Provide `UTPListener` implementing `net.Listener`:
  - `Accept()` returns an established `net.Conn`.
  - Handshake protocol:
    - Receive `ST_SYN` -> allocate new `UTPConn` in `STATE_SYN_RECEIVED` state -> reply with `ST_STATE` (ACK) -> wait for first data/state from remote to transition to `STATE_CONNECTED`.
- Provide `UTPConn` implementing `net.Conn`:
  - `Read()`, `Write()`, `Close()`, `LocalAddr()`, `RemoteAddr()`, `SetDeadline()`, `SetReadDeadline()`, `SetWriteDeadline()`.
  - Handle buffer fragmentation, out-of-order packet reordering, and flow control buffers.

### 2.4 Noise and Peer Connection Integration
- Update `pkg/peer` (or equivalent connection manager) to dial peers using `utp.Dial` and listen on a UDP port using `utp.Listen`.
- Verify the Noise handshake successfully completes over `UTPConn`.
- Verify Hypercore replication stream operates reliably over the new transport layer.

### 2.5 In-Memory Lossy UDP Simulator
- Create `pkg/utp/simulator` providing a simulated UDP transport layer interface.
- Support configurable loss parameters:
  - Packet Drop Rate (e.g., 0% to 50%).
  - Latency / Jitter (e.g., minimum latency, maximum variance).
  - Out-of-Order Packet Injection.
- Provide tests verifying:
  - Fast retransmit behaves correctly under 10% packet drop.
  - RTO-based recovery operates under higher packet loss rates.
  - Connection teardown (FIN handshake) completes gracefully.

---

## 3. Non-Functional Requirements
- **Pure Go**: Strictly zero external dependencies. Avoid importing existing uTP C bindings or non-stdlib libraries.
- **Concurrency & Resource Safety**:
  - Thread-safe multiplexer with fine-grained locking or channel-based synchronization.
  - Prevent goroutine leaks when connections time out or disconnect.
  - Graceful cleanup of resources (timers, channel queues, buffers).
- **Code Coverage**: Target >80% code coverage for `pkg/utp`.
- **API Documentation**: All exported structs, interfaces, and public methods must have clear GoDoc comments.

---

## 4. Acceptance Criteria
- Unit tests verify µTP header serialization and deserialization.
- Core µTP connection handshake (SYN -> SYN-ACK -> Connected) succeeds.
- Graceful connection closing (FIN -> ACK -> FIN -> ACK) succeeds.
- Reliable packet transmission works over the `LossyUDP` simulator, correcting for 10% packet loss.
- LEDBAT window increases when delay is below 100ms and shrinks when queuing delay exceeds 100ms.
- High-level P2P Noise handshake and Hypercore replication pass local test suites when running over UTP instead of TCP/mock network.

---

## 5. Out of Scope
- NAT Hole Punching (ICE/STUN/TURN) — handled in a separate track.
- UDP Relay / TURN fallback — handled in a separate track.
- Multi-streaming multiplexing (e.g. Yamux over UTP) — we run UTP connections directly.
- Compatibility with classic Bittorrent clients (we only target our own Go gateway peers).
