# System Architecture Guide

The Keet ADK Gateway serves as a localized, high-performance coordination conduit. It bridges external application nodes and mobile clients using the Keet Application Development Kit (ADK) into decentralized, DHT-backed overlay networks.

This document describes the core service topology, packet routing, and internal subsystem designs of the gateway.

---

## 1. System Topology Overview

The gateway is built on a multi-layered, asynchronous architecture structured into decoupled modules:

```mermaid
graph TD
    subgraph Client Layer
        KC[Keet Client / Mobile App]
        CS[Custom ADK Script / Consumer]
    end

    subgraph Interface & Gatekeeper
        SL[SocketListener - TCP or Unix]
        IPCP[IPC Socket Handler & Router]
        WLG[Whitelist Security Gatekeeper]
    end

    subgraph Data Subsystem
        HS[Hypercore Local Storage]
        DB[Unified Database Dispatcher]
        BB[BoltDB Embedded Bucket Cache]
        PG[PostgreSQL Relational Storage]
    end

    subgraph P2P Network
        DHTN[Distributed Hash Table Node]
        SR[Swarm Registry]
        UTP[uTP Holepunch / TCP Sync Engine]
    end

    KC -->|Connect| SL
    CS -->|Connect| SL
    SL --> IPCP
    IPCP --> WLG
    WLG -->|Authorized Commands| HS
    WLG -->|Register Swarms / Cache Blocks| DB
    DB -->|Pure Go| BB
    DB -->|Relational SQL| PG
    
    IPCP -->|Announce / Lookup Swarms| DHTN
    DHTN <-->|Find Peers| SR
    HS <-->|Replicate Blocks| UTP
```

---

## 2. Subsystem Breakdowns

### A. Interface & IPC Handler (`pkg/ipc`)
* **Multi-Protocol Socket Listener (`SocketListener`):** Detects whether the configured socket address is a local file path (Unix Socket) or an network port/prefix (`tcp://` or `:port`). If a TCP port is requested, it binds to a network-facing TCP interface, enabling wireless connections from local networks (e.g., phones on local Wi-Fi).
* **Security Gatekeeper:** If a `client_whitelist` is present in the configuration, all connections are initially placed in a locked, unauthenticated state. The gatekeeper checks every incoming frame for a valid whitelisted `peer_key` (public key in hex) or an explicit `auth` handshake command. Unwhitelisted clients are instantly disconnected.

### B. Configuration System (`pkg/config`)
* Evaluates parameters dynamically across **four tiers of precedence**:
  1. Handcrafted Command-Line overrides (e.g. `--config /path/to/yaml` or `-config=`).
  2. Local directory `./config.yaml`.
  3. Binary directory location `/path/to/binary/config.yaml`.
  4. Environment-level default variables.

### C. Database Persistence Engine (`pkg/db`)
* Decoupled using an abstract repository pattern:
  * `SwarmRepository`: Manages swarm topic metadata, tracking currently active uTP channel handshakes.
  * `BlockRepository`: Stores and caches full, raw, authenticated Hypercore block structures.
* **BBolt Persistence (Embedded default):** Utilizes key-value bucket nesting inside memory-mapped files. The blocks are packaged alongside their secure signatures via an ultra-fast length-prefixed binary serialization mechanism.
* **PostgreSQL Backup Driver:** Structured for scale. Connects using connection pooling and automatically manages table definitions and index migrations on startup.

### D. Hypercore & DHT P2P Overlay (`pkg/hypercore`, `pkg/dht`, `pkg/network`)
* **DHT Node:** Operates a distributed Kademlia-based hash table. Orchestrates room announcements and peer lookups.
* **Hypercore Replication Engine:** Governs bitfield negotiation, block requests, and remote stream replication. Uses custom uTP holepunch protocols to pipe binary feeds asynchronously across local subnets and WANs.

---

## 3. Data Flow & Sequence Diagram

The diagram below outlines the standard flow when an authorized client registers a swarm topic, appends a block, and replicates it:

```mermaid
sequenceDiagram
    autonumber
    actor Client as Keet Client
    participant IPC as IPC Socket
    participant SEC as Security Gatekeeper
    participant DB as DB Repository
    participant DHT as DHT Node
    participant HC as Hypercore / uTP

    Client->>IPC: Connect (TCP/Unix)
    Client->>IPC: Send auth command {peer_key: "0123..."}
    IPC->>SEC: Is "0123..." in Whitelist?
    SEC-->>IPC: Authorized
    IPC-->>Client: {"status": "success", "command": "auth"}

    rect rgb(240, 248, 255)
        Note over Client, DHT: Swarm Discovery Flow
        Client->>IPC: Send join_swarm {topic: "room-abc"}
        IPC->>DB: Register Swarm Topic Metadata
        IPC->>DHT: Announce topic "room-abc" to DHT network
        DHT-->>IPC: Peer lookup complete (found peer X, Y, Z)
        IPC-->>Client: {"status": "success", "resolved_topic_key": "feed123..."}
    end

    rect rgb(245, 245, 245)
        Note over Client, HC: Append & Replication Flow
        Client->>IPC: Send append_block {data: "base64...", signature: "..."}
        IPC->>HC: Append raw data into active Hypercore Storage
        IPC->>DB: PutBlock (cache block data & signatures in BBolt)
        HC->>HC: Trigger OnAppendBlock Event
        HC->>Client: Replicate and sync blocks with Peer X, Y, Z via uTP
        IPC-->>Client: {"status": "success", "index": 42}
    end
```
