# Specification: NAT Hole Punching and Hypercore Protobuf Wire Compliance

## Overview
Implement lightweight, pure Go STUN/TURN client helpers to enable NAT hole punching (ICE) and UDP relay fallback. Ensure the Hypercore wire protocol (v10) is fully compliant with the standard protobuf specification, including core protocol messages, capability exchanges, compression, and strict limit validation.

## Functional Requirements
1. **Lightweight STUN/TURN Client Helpers**:
   - Implement STUN (RFC 5389 / RFC 8489) binding request parsing and public IP/port mapping discovery.
   - Implement TURN (RFC 5766 / RFC 8656) allocation requests, permission management, and UDP packet relay encapsulation.
   - Zero external library dependencies for STUN/TURN parsing (custom Go implementation).
2. **ICE and UDP Relay Fallback**:
   - Integrate STUN discovery directly into `PeerManager`.
   - Perform automatic UDP hole punching (ICE candidate exchanges) when peer connections are discovered.
   - Fall back to TURN relay server communication if direct UDP traversal fails.
   - Support both public STUN servers (e.g. Google's public STUN) and custom private TURN relays via gateway configuration.
3. **Hypercore Wire Protobuf Spec Compliance**:
   - Encode/decode all standard Hypercore wire messages (Feed, Handshake, Request, Data, Cancel) conforming to protobuf specifications.
   - Implement extension messages (capability exchanges and Inflate/Deflate compression options).
   - Enforce strict message size limits and field constraint validation on deserialization.

## Non-Functional Requirements
- **Performance**: Zero-copy packet parsing where possible to minimize throughput overhead on relays.
- **Coverage**: Maintain >80% statement coverage for new STUN/TURN and Protobuf serialization packages.

## Acceptance Criteria
- Unit tests verify STUN mapping extraction and TURN relay framing correctly.
- Integration tests verify successful peer synchronization over UDP hole punched sockets and TURN relays.
- Hypercore wire packets conform to the protobuf layout and decode successfully.
