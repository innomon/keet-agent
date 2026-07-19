# DB 

  docker run --name keet-postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=keet_gateway -p 5435:5432 -d postgres:17-alpine


In this project, PostgreSQL serves as the central persistent relational storage backend for the gateway, implemented within the pkg/db package using the high-performance `pgxpool`        
  connection pool library.
  
  It stores and manages two main categories of data:
  
  ### 1. Swarm Management (swarms table)
  
  • Purpose: Persists information about Kademlia-based peer-to-peer swarms and discovery topics.
  • Schema:
      • topic_key (Text, Primary Key): Unique hex representation of the swarm's topic.
      • topic_name (Text): The descriptive, human-readable name of the topic.
      • created_at (Timestamp with timezone): Date/time the swarm was registered on the gateway.
  • Key Functionalities (in swarm_repo.go):
      • Registers swarms and keeps track of active swarms across process lifecycles.
      • Unregisters and cleans up old swarms dynamically.
  

  ### 2. Hypercore Logs Persistence (blocks table)
  
  • Purpose: Persists the append-only logs for Hypercore replication sessions. This ensures that even if the gateway restarts, the downloaded Hypercore feed blocks do not need to be      
  refetched over the network.
  • Schema:
      • feed_key (Text): Public key identifying the Hypercore feed.
      • block_index (BigInt): The sequential sequence number of the block in the feed.
      • value (Bytea / Binary): The actual chunk data payload (stored as raw bytes).
      • signature (Bytea / Binary): Cryptographic Ed25519 signature of the block to guarantee tamper-proof log integrity.
      • created_at (Timestamp with timezone): Date/time the block was appended/replicated.
      • Primary Key: Composite key (feed_key, block_index) to ensure strict sequential uniqueness per feed.
  • Key Functionalities (in block_repo.go):
      • Fast inserts and on-conflict updates (upsert operations) for newly downloaded log blocks.
      • Direct random-index point lookups (SELECT) to serve feed blocks to other replicating peers.
  

  ### Infrastructure & Operations
  
  • Connection Pooling: Uses pgxpool configured with sane resource limit defaults (maximum 10 connections, minimum 2 connections, and idle connection lifespans of up to 30 minutes) to    
  prevent socket starvation under heavy P2P replication concurrency.
  • Migrations: Features transaction-safe, automatic database schema bootstrapping (RunMigrations) that creates the necessary tables on service startup.





## Embedded  db
Choosing a **pure Go** (no CGO) embedded database is a common architectural move when you want zero-dependency, easy cross-compilation, and a dead-simple single-binary deployment.

The prominent pure-Go embedded databases fall into three major paradigms: Key-Value stores, Relational/SQL, and Document/NoSQL engines.

---

### The Contenders

#### 1. bbolt (by etcd-io) — *The Reliability King*

A popular fork of the original BoltDB, `bbolt` is a low-level key-value store optimized for read-heavy workloads. It structures data into nested "buckets" and uses a B+ tree architecture. It is battle-tested, powering core infrastructure like Kubernetes (via `etcd`).

* **Pros:** Bulletproof ACID transactions, incredibly fast and concurrent reads, memory-mapped files (`mmap`) allow fast startups.
* **Cons:** Single-writer limitation (writes block each other), poor write performance for random keys because it has to rebalance the B+ tree.

#### 2. BadgerDB (by Dgraph) — *The Write-Heavy Performance Monster*

`badger` is a high-performance LSM-tree (Log-Structured Merge-tree) key-value store. Unlike traditional LSM engines that store keys and values together, Badger implements *WiscKey* architecture, separating keys from values.

* **Pros:** Unmatched write throughput, highly optimized for SSDs, handles multi-gigabyte datasets smoothly, and supports key-range scanning.
* **Cons:** Higher RAM usage compared to `bbolt`, and the data files require aggressive value-log garbage collection background processes.

#### 3. modernc.org/sqlite — *The Standard SQL Choice*

If you want standard SQLite without dealing with a C toolchain, this is the definitive answer. It is a **100% automated translation of the SQLite C source code into pure Go**. It plugs directly into Go's standard `database/sql` interface.

* **Pros:** It is *literally* SQLite. Full SQL, ACID compliance, complex queries, and native compatibility with ORMs like GORM. Zero configuration cross-compilation.
* **Cons:** Because it's translated code, it has slightly higher memory usage and a minor performance penalty (roughly 10–30% slower) compared to the native CGO `mattn/go-sqlite3` driver.

#### 4. Clover / Genji — *The Document-Oriented NoSQL Alternatives*

For projects needing a NoSQL approach (similar to MongoDB), **Clover** is a pure Go document database that uses JSON-like structures and supports indexing. **Genji** is an alternative that combines document-flexibility with an SQL query layer.

* **Pros:** No need to handle complex object-relational mapping if your application naturally deals with rich JSON data.
* **Cons:** Much smaller ecosystems, less battle-tested, and slower query optimization compared to SQL or raw KV stores.

---

### Comparison Matrix

| Feature | bbolt | BadgerDB | modernc.org/sqlite | Clover |
| --- | --- | --- | --- | --- |
| **Data Model** | Key-Value (Buckets) | Key-Value | Relational (SQL) | Document / NoSQL |
| **Underlying Architecture** | B+ Tree | LSM Tree / WiscKey | B-Tree / SQLite Engine | Memory + Disk Append |
| **Read Performance** | **Exceptional** (via mmap) | Good | Good | Moderate |
| **Write Performance** | Moderate (Slow random writes) | **Exceptional** (Sequential logs) | Moderate | Moderate |
| **Transaction Style** | Fully ACID (1 Writer / Multi-Reader) | Fully ACID (MVCC Concurrent) | Fully ACID | Lightweight Transactions |
| **Best For...** | Read-heavy configs, metadata, small state | High-frequency logging, metrics, massive KV | Standard apps needing relational schemas | Quick prototyping with JSON/BSON payloads |

---

### Which one should you pick?

* **Choose `bbolt` if:** You are building a service configuration store, a local application coordinator, or something that reads data constantly but writes occasionally. If etcd trusts it for Kubernetes, your app will be fine.
* **Choose `BadgerDB` if:** You are building a time-series app, an ingestion pipeline, a cache layer, or need to store large chunks of data (images/blobs as values) with massive write throughput.
* **Choose `modernc.org/sqlite` if:** You already know SQL, want to use standard migrations, or need to query complex relationships between datasets without writing custom index-traversal logic in Go.
* **Choose `Clover` if:** You want a local alternative to MongoDB for a CLI tool or a small desktop application that deals primarily with un-schemed data blocks.

