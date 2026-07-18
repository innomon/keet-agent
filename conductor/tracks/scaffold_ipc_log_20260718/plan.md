# Implementation Plan - Setup Go project scaffold, Unix socket IPC listener, and structured log rotator

This plan outlines the steps to initialize the Go module, configure the structured logging with rotator, and establish the Unix domain socket listener.

## Phase 1: Project Scaffolding and Structured Logging [checkpoint: ff7d825]

- [x] Task: Go Module Scaffolding (e150631)
    - [x] Initialize Go 1.24 module
    - [x] Set up basic directory structure (cmd/gateway, pkg/logger, pkg/ipc)
    - [x] Add basic configuration file structure
    - [x] Commit scaffolding changes
- [x] Task: Thread-Safe Log Rotator Implementation (TDD) (c15af85)
    - [x] Write tests for LogRotator (file creation, size-based rotation, backup pruning)
    - [x] Implement LogRotator in `pkg/logger/rotator.go` using only standard library
    - [x] Verify LogRotator tests pass
- [x] Task: Structured Multiplexed Logger (TDD) (ebc1ba9)
    - [x] Write tests for MultiHandler and Logger Init
    - [x] Implement MultiHandler and Logger configuration in `pkg/logger/logger.go`
    - [x] Implement caller PC tracing in custom logging wrapper to preserve source code locations
    - [x] Verify logger and wrapper tests pass
- [x] Task: Conductor - User Manual Verification 'Phase 1: Project Scaffolding and Structured Logging' (Protocol in workflow.md)

## Phase 2: IPC Unix Domain Socket

- [x] Task: Unix Domain Socket Listener (TDD) (cc2e35e)
    - [x] Write tests for socket initialization, cleanup of stale descriptors, and basic connection acceptance
    - [x] Implement Unix domain socket listener binding to `/tmp/keet-adk.sock` in `pkg/ipc/socket.go`
    - [x] Implement stale socket deletion on startup
    - [x] Verify socket listener tests pass
- [x] Task: Multi-Core Concurrency & Client Handler (TDD) (d8123b3)
    - [x] Write tests for concurrent client routing and asynchronous worker execution
    - [x] Implement concurrent client socket reader/writer loops with goroutine worker pools
    - [x] Handle graceful termination signals (SIGINT, SIGTERM) to close socket listener and clean up socket files
    - [x] Verify concurrency and client handler tests pass
- [ ] Task: Main Gateway Loop
    - [ ] Implement main entry point `cmd/gateway/main.go` integrating logger and IPC listener
    - [ ] Configure `runtime.GOMAXPROCS` to use all M4 CPU cores
    - [ ] Verify the complete gateway runs, logs to console + files, and listens on the Unix socket
- [ ] Task: Conductor - User Manual Verification 'Phase 2: IPC Unix Domain Socket' (Protocol in workflow.md)
