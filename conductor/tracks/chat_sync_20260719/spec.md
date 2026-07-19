# Specification: Implement peer-to-peer chat message exchange over secure connections

## 1. Goal
Implement structured peer-to-peer chat message exchange over secured Noise and Hypercore replicator connections. Support JSON-formatted chat message parsing (sender, timestamp, content), real-time automatic replication sync upon block append events, and asynchronous JSON-RPC socket notifications broadcasted to connected ADK gateway clients.

## 2. Scope
### In-Scope
- **JSON Chat Message Schema**:
  - Define chat message structure: Sender public key, Unix timestamp, and UTF-8 content string.
  - Implement parsing, serialization, and validation helpers in `pkg/chat/message.go`.
- **Real-Time Replication Event Integration**:
  - Integrate P2P sync session to immediately announce new chat blocks using `Have` frame writes.
  - Automatically fetch incoming block logs on receiving peer announcements.
- **Asynchronous IPC Gateway Broadcasting**:
  - Upon receiving and verifying a new chat log block, broadcast a JSON-RPC `chat_message_received` notification frame to all connected local Unix Domain Socket clients.
  - Ensure thread-safe socket broadcast loop across client connections.

### Out-of-Scope
- End-to-end chat message encryption (messages are plaintext inside the Noise channel).
- Multi-room routing and private messaging (all messages are swarm-wide broadcast).

## 3. Tech Stack
- Go 1.24+ standard library.
- Database & Storage: `pkg/db`, `pkg/hypercore` (flat-file + pgx).
- Transport & IPC: `pkg/ipc`, `pkg/network`.
