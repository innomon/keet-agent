# Specification: Setup Go project scaffold, Unix socket IPC listener, and structured log rotator

## 1. Goal
Initialize the Go codebase for the Keet Desktop Gateway, establish a dependency-free structured logging framework with size-based log rotation and caller trace preservation, and build a local Unix domain socket IPC listener to receive JSON-RPC messages from ADK clients.

## 2. Scope
### In-Scope
- Initialize Go 1.24 module `github.com/innomon/keet-adk-gateway`.
- Implement `pkg/logger`:
  - `LogRotator`: Thread-safe size-based file rotation writer.
  - `MultiHandler`: Log multiplexer console (Text) + file (JSONL).
  - Wrapper API with caller PC location preservation.
- Implement `pkg/ipc`:
  - Unix Domain Socket server at `/tmp/keet-adk.sock`.
  - Stale socket cleanup on start.
  - Asynchronous client handlers mapping requests to goroutine pools.
  - Graceful termination signal handling.
- Comprehensive Unit Tests for logging and IPC socket behaviors.

### Out-of-Scope (Future Tracks)
- HyperDHT swarming and peer discovery.
- Hypercore log replication and binary parsing.
- PostgreSQL database persistence and schema migrations.

## 3. Tech Stack Requirements
- Go 1.24+ standard library only.
- Target: `GOOS=darwin GOARCH=arm64` (Apple Silicon M4).
- Database: Postgres (not used in this track, but planned for future).
