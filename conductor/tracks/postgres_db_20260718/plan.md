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

## Phase 2: Swarm and Block Repositories [checkpoint: f443a75]

- [x] Task: Swarms Table Repository (TDD) (ce21902)
    - [x] Write unit tests for adding and removing swarms in the database
    - [x] Implement `SwarmRepository` in `pkg/db/swarm_repo.go` handling database inserts and deletions
    - [x] Verify swarm repository tests pass
- [x] Task: Replicated Blocks Repository (TDD) (77b11fa)
    - [x] Write unit tests for inserting and retrieving Hypercore log blocks from the database
    - [x] Implement `BlockRepository` in `pkg/db/block_repo.go` handling database inserts and queries
    - [x] Verify block repository tests pass
- [x] Task: Conductor - User Manual Verification 'Phase 2: Swarm and Block Repositories' (Protocol in workflow.md)

## Phase 3: Integration and Gateway Run Verification [checkpoint: 45b235d]

- [x] Task: Database Integration into Main Loop (b249886)
    - [x] Integrate DB migrations and Postgres connection pool into `cmd/gateway/main.go` on startup
    - [x] Update socket connections to persist joined swarms and blocks into database instead of solely in-memory/flat-file
- [x] Task: Gateway Run Verification (04ee436)
    - [x] Verify gateway builds successfully, connects to local/test Postgres database, automatically executes migrations, and persists socket commands
- [x] Task: Conductor - User Manual Verification 'Phase 3: Integration and Gateway Run Verification' (Protocol in workflow.md)

## Phase: Review Fixes
- [x] Task: Apply review suggestions 2effece
