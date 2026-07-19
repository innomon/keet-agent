# Implementation Plan - Implement peer-to-peer chat message exchange over secure connections

This plan outlines the steps to implement JSON chat message serialization, automatic live synchronization triggers, and asynchronous IPC broadcasting.

## Phase 1: Chat Message Schema & Serialization [checkpoint: 55352d6]

- [x] Task: Chat Message Serialization Helper (TDD) (358b0a9)
    - [x] Write unit tests for chat message serialization, parsing, and bounds validation
    - [x] Implement `ChatMessage` struct and utility helpers in `pkg/chat/message.go`
    - [x] Verify serialization tests pass
- [x] Task: Conductor - User Manual Verification 'Phase 1: Chat Message Schema & Serialization' (Protocol in workflow.md)

## Phase 2: Live Sync Replication Event Trigger [checkpoint: 0fc220c]

- [x] Task: P2P Auto-Sync Triggers (TDD) (8201a97)
    - [x] Write integration tests for real-time replication propagation between connected PeerManagers
    - [x] Modify `SyncSession` and `PeerManager` to automatically trigger `Have`/`Request` updates when new blocks are local or remote
    - [x] Verify P2P auto-sync propagation tests pass
- [x] Task: Conductor - User Manual Verification 'Phase 2: Live Sync Replication Event Trigger' (Protocol in workflow.md)

## Phase 3: IPC Notification Broadcasting

- [x] Task: Gateway IPC Broadcast Event Loop (TDD) (4c8b2d5)
    - [x] Write integration tests verifying local JSON-RPC socket notifications upon P2P block arrival
    - [x] Implement active socket connection broadcast registry in `pkg/ipc/socket.go`
    - [x] Ingest new block events into the IPC broadcaster, publishing asynchronous frame updates to ADK clients
    - [x] Verify IPC socket broadcast tests pass
- [ ] Task: Conductor - User Manual Verification 'Phase 3: IPC Notification Broadcasting' (Protocol in workflow.md)
