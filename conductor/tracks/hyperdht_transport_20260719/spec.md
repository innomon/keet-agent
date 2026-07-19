# Specification: Implement HyperDHT Network Transport

## 1. Overview
This track implements a real, network-capable Kademlia DHT transport layer in pure Go inside `pkg/dht`, enabling the gateway to autonomously discover peers on the Holepunch P2P network. The existing `dht.DHTNode` struct is promoted from a configuration holder into an active UDP network participant. A transport interface is introduced to make the implementation fully testable without a live network.

## 2. Functional Requirements

### 2.1 Transport Interface
- Define a `Transport` interface in `pkg/dht` that abstracts UDP socket operations (`ReadFrom`, `WriteTo`, `Close`, `Addr`).
- Provide a `UDPTransport` concrete implementation backed by `net.PacketConn`.
- Provide an `InProcessTransport` (loopback stub) for hermetic unit/integration testing without a real network.

### 2.2 Kademlia RPC Layer
- Implement the following Kademlia RPC message types as binary-encoded structs (pure Go, no protobuf):
  - `PING` / `PONG` — liveness check
  - `FIND_NODE(target [32]byte)` — find the K closest contacts to a target ID
  - `FIND_NODE_RESP` — returns up to K=20 contacts
  - `ANNOUNCE(topic [32]byte, port uint16)` — announce this node as a peer for a topic
  - `LOOKUP(topic [32]byte)` — find peers registered under a topic
  - `LOOKUP_RESP` — returns a list of peer addresses for the topic
- Each RPC carries a random 4-byte transaction ID for request/response correlation.
- Responses must be matched to pending requests via a correlation map with a configurable timeout (default 5s).

### 2.3 Bootstrap & Routing Table Integration
- On `DHTNode.Start(ctx)`, dial each bootstrap node address with a `PING` to confirm liveness, then issue `FIND_NODE(selfID)` to each to populate the routing table.
- Integrate responses into the existing `RoutingTable` (update `AddContact` to respect K=20 bucket-size limit per XOR-distance bucket).

### 2.4 Peer Announce & Lookup
- `DHTNode.Announce(ctx, topic [32]byte)` — iteratively find the K closest nodes to `topic`, send `ANNOUNCE` to each, and store the topic in the node's local announce set.
- `DHTNode.Lookup(ctx, topic [32]byte) ([]string, error)` — iteratively find the K closest nodes to `topic`, send `LOOKUP` to each, collect and deduplicate peer address results.
- Periodically re-announce active topics (configurable interval, default 10 minutes).

### 2.5 Graceful Leave / Unannounce
- `DHTNode.Leave(ctx, topic [32]byte)` — remove topic from local announce set, stop periodic re-announce for that topic.

### 2.6 IPC `join_swarm` / `leave_swarm` Integration
- Wire `DHTNode.Announce` into the IPC `join_swarm` command handler: after registering the peer in the swarm registry, trigger `dhtNode.Announce(ctx, topicKey)`.
- Wire `DHTNode.Lookup` into the `join_swarm` flow: after announcing, call `dhtNode.Lookup(ctx, topicKey)` and register each discovered peer address via `SwarmRegistry.RegisterPeer`, which triggers `OnRegisterPeer → pm.DialPeer`.
- Wire `DHTNode.Leave` into the `leave_swarm` command handler.

### 2.7 Swarm Re-hydration on Restart
- On `DHTNode.Start`, after bootstrap is complete, call `swarmRepo.GetActiveSwarms(ctx)` and for each stored topic key, call `DHTNode.Announce` + `DHTNode.Lookup` to re-establish swarm membership.

## 3. Non-Functional Requirements
- Pure Go only — no cgo, no external DHT libraries.
- The `Transport` interface must be the single seam for testability; all DHT logic operates against the interface.
- `DHTNode` must be safe for concurrent use (goroutine-safe announce, lookup, routing table updates).
- All public methods must have GoDoc comments per the project Go style guide.
- Target >80% test coverage for `pkg/dht`.

## 4. Acceptance Criteria
- `DHTNode.Start` successfully pings and populates the routing table from at least one bootstrap node (verified via integration test with `InProcessTransport`).
- `DHTNode.Announce` + `DHTNode.Lookup` discover peers in a hermetic multi-node in-process test (3+ nodes, 2+ topics).
- After `leave_swarm`, `DHTNode.Lookup` no longer returns the left node's address.
- Gateway restart with persisted swarms in DB triggers automatic re-announce and re-lookup.
- IPC `join_swarm` with a valid topic causes `chat_message_received` to flow from a peer's block append (end-to-end, using `InProcessTransport`).

## 5. Out of Scope
- UTP/µTP reliable transport over UDP (Holepunch's actual production layer) — a future track.
- NAT hole-punching — a future track.
- Relay server integration — a future track.
- Hypercore wire protobuf spec compliance — a separate track.
