# Implementation Plan - Implement PostgreSQL database persistence, schema migrations, and connection pools

This plan details the steps to implement DB configuration, connection pooling, handcrafted SQL migrations, and repository logic.

## Phase 1: DB Configuration, Connection Pools, and Migrations [checkpoint: 4472d12]

- [x] Task: Database Configuration and Pool Initialization (TDD) (5cbecaf)
    - [x] Write unit tests for database configuration properties
    - [x] Add `DBHost`, `DBPort`, `DBUser`, `DBPassword`, `DBName`, and `DBSSLMode` variables to `pkg/config/config.go`
    - [x] Implement database pool connection in `pkg/db/postgres.go` using `github.com/jackc/pgx/v5/pgxpool`
    - [x] Verify configuration and pool tests pass
- [x] Task: Schema Loader and SQL Migrations (TDD) (e731428)
    - [x] Write unit tests for running database schema migrations sequentially
    - [x] Implement a raw SQL schema migration loader in `pkg/db/migrations.go` executing tables creation query on startup
    - [x] Verify migration loader tests pass
- [x] Task: Conductor - User Manual Verification 'Phase 1: DB Configuration, Connection Pools, and Migrations' (Protocol in workflow.md)

## Phase 2: Swarm and Block Repositories

- [ ] Task: Swarms Table Repository (TDD)
    - [ ] Write unit tests for adding and removing swarms in the database
    - [ ] Implement `SwarmRepository` in `pkg/db/swarm_repo.go` handling database inserts and deletions
    - [ ] Verify swarm repository tests pass
- [ ] Task: Replicated Blocks Repository (TDD)
    - [ ] Write unit tests for inserting and retrieving Hypercore log blocks from the database
    - [ ] Implement `BlockRepository` in `pkg/db/block_repo.go` handling database inserts and queries
    - [ ] Verify block repository tests pass
- [ ] Task: Conductor - User Manual Verification 'Phase 2: Swarm and Block Repositories' (Protocol in workflow.md)

## Phase 3: Integration and Gateway Run Verification

- [ ] Task: Database Integration into Main Loop
    - [ ] Integrate DB migrations and Postgres connection pool into `cmd/gateway/main.go` on startup
    - [ ] Update socket connections to persist joined swarms and blocks into database instead of solely in-memory/flat-file
- [ ] Task: Gateway Run Verification
    - [ ] Verify gateway builds successfully, connects to local/test Postgres database, automatically executes migrations, and persists socket commands
- [ ] Task: Conductor - User Manual Verification 'Phase 3: Integration and Gateway Run Verification' (Protocol in workflow.md)
