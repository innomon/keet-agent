# Implementation Plan - Implement peer-to-peer chat message exchange over secure connections

This plan outlines the steps to implement JSON chat message serialization, automatic live synchronization triggers, and asynchronous IPC broadcasting.

## Phase 1: Chat Message Schema & Serialization

- [ ] Task: Chat Message Serialization Helper (TDD)
    - [ ] Write unit tests for chat message serialization, parsing, and bounds validation
    - [ ] Implement `ChatMessage` struct and utility helpers in `pkg/chat/message.go`
    - [ ] Verify serialization tests pass
- [ ] Task: Conductor - User Manual Verification 'Phase 1: Chat Message Schema & Serialization' (Protocol in workflow.md)

## Phase 2: Live Sync Replication Event Trigger

- [ ] Task: P2P Auto-Sync Triggers (TDD)
    - [ ] Write integration tests for real-time replication propagation between connected PeerManagers
    - [ ] Modify `SyncSession` and `PeerManager` to automatically trigger `Have`/`Request` updates when new blocks are local or remote
    - [ ] Verify P2P auto-sync propagation tests pass
- [ ] Task: Conductor - User Manual Verification 'Phase 2: Live Sync Replication Event Trigger' (Protocol in workflow.md)

## Phase 3: IPC Notification Broadcasting

- [ ] Task: Gateway IPC Broadcast Event Loop (TDD)
    - [ ] Write integration tests verifying local JSON-RPC socket notifications upon P2P block arrival
    - [ ] Implement active socket connection broadcast registry in `pkg/ipc/socket.go`
    - [ ] Ingest new block events into the IPC broadcaster, publishing asynchronous frame updates to ADK clients
    - [ ] Verify IPC socket broadcast tests pass
- [ ] Task: Conductor - User Manual Verification 'Phase 3: IPC Notification Broadcasting' (Protocol in workflow.md)
