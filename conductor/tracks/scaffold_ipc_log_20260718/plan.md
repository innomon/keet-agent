# Implementation Plan - Setup Go project scaffold, Unix socket IPC listener, and structured log rotator

This plan outlines the steps to initialize the Go module, configure the structured logging with rotator, and establish the Unix domain socket listener.

## Phase 1: Project Scaffolding and Structured Logging

- [ ] Task: Go Module Scaffolding
    - [ ] Initialize Go 1.24 module
    - [ ] Set up basic directory structure (cmd/gateway, pkg/logger, pkg/ipc)
    - [ ] Add basic configuration file structure
    - [ ] Commit scaffolding changes
- [ ] Task: Thread-Safe Log Rotator Implementation (TDD)
    - [ ] Write tests for LogRotator (file creation, size-based rotation, backup pruning)
    - [ ] Implement LogRotator in `pkg/logger/rotator.go` using only standard library
    - [ ] Verify LogRotator tests pass
- [ ] Task: Structured Multiplexed Logger (TDD)
    - [ ] Write tests for MultiHandler and Logger Init
    - [ ] Implement MultiHandler and Logger configuration in `pkg/logger/logger.go`
    - [ ] Implement caller PC tracing in custom logging wrapper to preserve source code locations
    - [ ] Verify logger and wrapper tests pass
- [ ] Task: Conductor - User Manual Verification 'Phase 1: Project Scaffolding and Structured Logging' (Protocol in workflow.md)

## Phase 2: IPC Unix Domain Socket

- [ ] Task: Unix Domain Socket Listener (TDD)
    - [ ] Write tests for socket initialization, cleanup of stale descriptors, and basic connection acceptance
    - [ ] Implement Unix domain socket listener binding to `/tmp/keet-adk.sock` in `pkg/ipc/socket.go`
    - [ ] Implement stale socket deletion on startup
    - [ ] Verify socket listener tests pass
- [ ] Task: Multi-Core Concurrency & Client Handler (TDD)
    - [ ] Write tests for concurrent client routing and asynchronous worker execution
    - [ ] Implement concurrent client socket reader/writer loops with goroutine worker pools
    - [ ] Handle graceful termination signals (SIGINT, SIGTERM) to close socket listener and clean up socket files
    - [ ] Verify concurrency and client handler tests pass
- [ ] Task: Main Gateway Loop
    - [ ] Implement main entry point `cmd/gateway/main.go` integrating logger and IPC listener
    - [ ] Configure `runtime.GOMAXPROCS` to use all M4 CPU cores
    - [ ] Verify the complete gateway runs, logs to console + files, and listens on the Unix socket
- [ ] Task: Conductor - User Manual Verification 'Phase 2: IPC Unix Domain Socket' (Protocol in workflow.md)
