# Implementation Plan: Implement HyperDHT Network Transport

## Phase 1: Transport Interface & RPC Codec

- [x] Task: Define `Transport` Interface & `UDPTransport` (TDD) [155f2dd]
    - [x] Write unit tests for `UDPTransport`: bind, send, receive, close lifecycle
    - [x] Define `Transport` interface with `ReadFrom`, `WriteTo`, `Close`, `Addr` methods in `pkg/dht/transport.go`
    - [x] Implement `UDPTransport` backed by `net.ListenPacket("udp", addr)`
    - [x] Verify transport tests pass
- [x] Task: Implement `InProcessTransport` loopback stub (TDD) [1ee8756]
    - [x] Write unit tests verifying two `InProcessTransport` instances can exchange packets in-process
    - [x] Implement `InProcessTransport` with `chan []byte` backing, connecting named endpoints
    - [x] Verify stub transport tests pass
- [ ] Task: Implement Kademlia RPC message codec (TDD)
    - [ ] Write unit tests for encode/decode round-trips of all 7 message types: PING, PONG, FIND_NODE, FIND_NODE_RESP, ANNOUNCE, LOOKUP, LOOKUP_RESP
    - [ ] Implement `pkg/dht/rpc.go` with binary-encoded RPC structs and a 4-byte transaction ID
    - [ ] Verify codec tests pass
- [ ] Task: Conductor - User Manual Verification 'Phase 1: Transport Interface & RPC Codec' (Protocol in workflow.md)

## Phase 2: Kademlia Core — Routing & Request/Response Engine

- [ ] Task: Upgrade `RoutingTable` to K=20 bucket-size enforcement (TDD)
    - [ ] Write unit tests verifying K=20 limit per XOR-distance bucket and LRU eviction
    - [ ] Update `pkg/dht/routing.go` to use per-bucket slices capped at K=20 with LRU eviction of least-recently-seen contacts
    - [ ] Verify routing table tests pass
- [ ] Task: Implement RPC dispatcher & pending-request correlation map (TDD)
    - [ ] Write unit tests: send request, match response by transaction ID, timeout after 5s, concurrent safe
    - [ ] Implement `pkg/dht/dispatcher.go` with a `sync.Map`-backed pending request table and goroutine-driven read loop
    - [ ] Verify dispatcher tests pass
- [ ] Task: Conductor - User Manual Verification 'Phase 2: Kademlia Core — Routing & Request/Response Engine' (Protocol in workflow.md)

## Phase 3: Bootstrap, Announce & Lookup

- [ ] Task: Implement `DHTNode.Start` bootstrap sequence (TDD)
    - [ ] Write integration test using `InProcessTransport`: 2+ nodes, one acts as bootstrap; verify calling `Start` populates routing table with bootstrap node's contact
    - [ ] Implement `DHTNode.Start(ctx)`: bind transport, PING each bootstrap node, issue `FIND_NODE(selfID)`, merge responses into routing table
    - [ ] Verify bootstrap integration test passes
- [ ] Task: Implement `DHTNode.Announce` iterative announce (TDD)
    - [ ] Write integration test: 3-node in-process DHT; node A announces topic, nodes B and C perform `LOOKUP` and find A's address
    - [ ] Implement `DHTNode.Announce(ctx, topic [32]byte)`: iterative `FIND_NODE` for topic, send `ANNOUNCE` to K-closest
    - [ ] Implement periodic re-announce goroutine (default 10-minute interval, configurable via config)
    - [ ] Verify announce integration test passes
- [ ] Task: Implement `DHTNode.Lookup` iterative lookup (TDD)
    - [ ] Write integration test: 3-node in-process DHT; node A announces, node B calls `Lookup`, receives A's peer address
    - [ ] Implement `DHTNode.Lookup(ctx, topic [32]byte) ([]string, error)`: iterative `FIND_NODE` + `LOOKUP` RPCs, collect + deduplicate peer addresses
    - [ ] Verify lookup integration test passes
- [ ] Task: Implement `DHTNode.Leave` graceful unannounce (TDD)
    - [ ] Write test: node A announces, then calls `Leave`; subsequent `Lookup` from node B does not return A's address
    - [ ] Implement `DHTNode.Leave(ctx, topic [32]byte)`: remove from local announce set, cancel periodic re-announce
    - [ ] Verify leave test passes
- [ ] Task: Conductor - User Manual Verification 'Phase 3: Bootstrap, Announce & Lookup' (Protocol in workflow.md)

## Phase 4: IPC Integration & Swarm Re-hydration

- [ ] Task: Wire `DHTNode` into IPC `join_swarm` / `leave_swarm` handlers (TDD)
    - [ ] Write integration test: IPC client sends `join_swarm`; verify `DHTNode.Announce` and `DHTNode.Lookup` are called and discovered peers are registered in `SwarmRegistry`
    - [ ] Update `pkg/ipc/socket.go` `HandleClient`: in `join_swarm`, call `dhtNode.Announce(ctx, topicKey)` then `dhtNode.Lookup(ctx, topicKey)` and feed results into `swarmRegistry.RegisterPeer`
    - [ ] Update `leave_swarm` to call `dhtNode.Leave(ctx, topicKey)`
    - [ ] Verify IPC integration tests pass
- [ ] Task: Implement swarm re-hydration on gateway restart (TDD)
    - [ ] Write integration test: persist a swarm to DB, restart `DHTNode`, verify `Announce` + `Lookup` are called for the persisted topic
    - [ ] Update `DHTNode.Start`: after bootstrap, call `swarmRepo.GetActiveSwarms(ctx)`, for each topic call `Announce` + `Lookup`
    - [ ] Thread `swarmRepo *db.SwarmRepository` through to `DHTNode` (or accept it as a `Start` parameter)
    - [ ] Verify re-hydration integration tests pass
- [ ] Task: Conductor - User Manual Verification 'Phase 4: IPC Integration & Swarm Re-hydration' (Protocol in workflow.md)
