# Specification: Implement PostgreSQL database persistence, schema migrations, and connection pools

## 1. Goal
Implement a robust, concurrent-safe PostgreSQL connection layer in the gateway. Persist the swarm topic registry and log blocks to the database using `github.com/jackc/pgx/v5`. Provide a simple handcrafted raw SQL migration system to run database setup scripts automatically on startup.

## 2. Scope
### In-Scope
- **Postgres Connection Pool**:
  - Initialize a secure, concurrent-safe Postgres connection pool using `github.com/jackc/pgx/v5/pgxpool`.
  - Parse environment variables: `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE` with sane defaults.
- **Handcrafted Database Migrations**:
  - Implement a raw SQL file reader or embedded SQL strings schema loader executing sequentially on startup to build/maintain tables.
- **Database Tables**:
  - `swarms`: Stores active joined swarm topic entries.
    - Fields: `topic_key` (text primary key), `topic_name` (text), `created_at` (timestamp).
  - `blocks`: Caches replicated Hypercore log blocks.
    - Fields: `feed_key` (text), `block_index` (bigint), `value` (bytea), `signature` (bytea), `created_at` (timestamp), Primary Key: (`feed_key`, `block_index`).
- **Data Access Layer**:
  - Implement functions to save/remove active swarms.
  - Implement functions to save/retrieve cached blocks.

### Out-of-Scope (Future Tracks)
- Complex relational index lookups or full text search in logs.
- SQLite or file-based database fallbacks (Postgres is strictly mandatory).

## 3. Tech Stack
- Go 1.24+ standard library.
- Database Driver: `github.com/jackc/pgx/v5/pgxpool` (pure Go).
- Database: PostgreSQL (v16+).
