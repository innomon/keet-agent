# Product Guidelines: Keet Desktop Gateway for ADK

These guidelines define the core operational principles, error handling standards, and performance baselines for the Keet Desktop Gateway backend.

## 1. IPC & API Design Principles
- **JSON-RPC 2.0 Protocol**: The Unix domain socket MUST communicate using strictly compliant JSON-RPC 2.0 frames over a line-delimited stream.
- **Asynchronous Execution**: The socket interface should not block. Long-running P2P actions (e.g. swarm connection, log replication) must return immediately with a job/request ID, and emit corresponding status notifications asynchronously.
- **Socket Lifecycle Safety**: Upon startup, the service must verify and remove stale socket files at `/tmp/keet-adk.sock`. Standard Unix signals (`SIGINT`, `SIGTERM`) must be intercepted to close connections gracefully and clean up the socket file.

## 2. P2P & Cryptographic Standards
- **Standard Cryptographic Packages**: All cryptography (Ed25519, Noise handshakes, Blake2b) must be implemented using pure Go libraries (e.g. `crypto/ed25519`, `golang.org/x/crypto/blake2b`).
- **Swarm Management**: Limit maximum peer connections per topic to prevent resource exhaustion on local Apple Silicon hosts. Connection pools must be managed with thread-safe maps (`sync.Map`).

## 3. Storage & Persistence
- **Postgres Database Only**: SQLite is prohibited. The database schema must handle concurrent connection pools efficiently.
- **Schema Migrations**: Use raw SQL files or a minimal pure Go migration helper. Schema modifications must be backward compatible.

## 4. Operational & Logging Standards
- **Zero-Dependency Log Rotation**: Logging must implement console text output alongside file size-based JSONL rotation.
- **Emitter Traceability**: Custom logs must trace the actual calling frame to accurately log file names and line numbers.
- **No spf13 Cobra/Pflag Libraries**: Any CLI commands or slash arguments must be routed through a custom, handcrafted command registry.
