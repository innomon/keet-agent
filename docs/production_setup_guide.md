# Keet ADK Gateway: Production Deployment & Raspberry Pi 5 Guide

This guide describes how to configure, run, and secure the Keet ADK Gateway on a **Raspberry Pi 5 (16GB)** in a real-world production topology:
1. **Raspberry Pi 5 is behind a NAT firewall**, connected to the public Internet.
2. The **ADK Gateway runs locally** on the Pi 5.
3. The **Agentic runtime** ([agentic](https://github.com/innomon/agentic)) runs locally on the Pi 5, initialized with the command:
   ```bash
   ./agentic -webui -a2a -host=192.168.1.10
   ```
   and listens on an HTTP port using the open `adk.dev` ADK protocol specification.
4. The **Keet ADK Gateway acts as a chat proxy and ADK.dev protocol client**, connecting to `agentic`'s HTTP endpoint. It acts as a standard peer in the chat swarm room, transparently proxying messages between the Keet Mobile app and the `agentic` service.
5. The **Keet Mobile App runs on the public Internet**, communicating with the RPi5 gateway securely and peer-to-peer via DHT swarm orchestration and uTP NAT holepunching. To the mobile app, it is as if it is communicating with another standard Keet mobile client.

No public port forwarding, dynamic DNS, or local IPC socket listeners are required for the agent-to-mobile communication!

---

## 1. Production Architecture Topology

Instead of exposing any socket interface to the public network, the gateway uses the strength of decentralized peer-to-peer protocols (DHT + uTP + Hypercore) to bridge firewalls, acting as a secure gateway proxy to your local HTTP `agentic` instances.

```
+------------------------------------------+                 +------------------------------+
|            Raspberry Pi 5                |                 |        Public Internet       |
|          (Behind NAT/Firewall)           |                 |                              |
|                                          |                 |                              |
|  +------------------------------------+  |                 |                              |
|  | agentic                            |  |                 |                              |
|  | (http://192.168.1.10:8080/chat)    |  |                 |                              |
|  +------------------------------------+  |                 |                              |
|                   ^                      |                 |                              |
|                   | HTTP POST (adk.dev)  |                 |                              |
|                   v                      |                 |                              |
|  +------------------------------------+  |  Holepunching   |  +------------------------+  |
|  |        Keet ADK Gateway            |<-+================>|  |     Keet Mobile App    |  |
|  | (DHT Node / uTP Sync / BBolt Cache) |  |   uTP Sync /    |  | (P2P Node on 4G/5G/WAN)|  |
|  +------------------------------------+  |  DHT Swarm Room |  +------------------------+  |
+------------------------------------------+                 +------------------------------+
```

### Operational Workflow:
1. The **Keet ADK Gateway** on the RPi5 runs locally and announces a unique DHT swarm room topic.
2. The **Keet Mobile App** (on WAN/Cellular) joins the same DHT swarm room topic.
3. The built-in DHT Node (`pkg/dht`) and NAT holepunch protocol (`pkg/utp`) resolve paths through firewalls to establish a direct uTP stream connection. To the mobile app, the gateway appears as another normal Keet peer.
4. When a user sends a message on the mobile app, it replicates automatically to the RPi5's local Hypercore feed over uTP.
5. The gateway's `OnAppendBlock` replication hook intercepts the block, parses the JSON chat payload, and proxies a **JSON HTTP POST** request to your locally running `agentic` service (`http://192.168.1.10:8080`):
   ```json
   {
     "sender": "012345...",
     "content": "Hello Agent!",
     "feed_key": "default_feed"
   }
   ```
6. The `agentic` service processes the message and returns the response:
   ```json
   {
     "response": "Hello from the Pi 5 Agent!"
   }
   ```
7. The gateway captures the response and appends it to the local Hypercore storage.
8. The uTP synchronization engine automatically replicates the new block directly back to your mobile client over the public WAN!

---

## 2. Gateway Production Configuration (`config.yaml`)

Configure your gateway to enable the HTTP proxy bridge by defining the `agentic_url` property in your `config.yaml`.

```yaml
# log settings
log_level: INFO
console_log_enabled: true
file_log_enabled: true

# database settings (Pure Go BBolt is default and recommended for RPi5)
db_type: bbolt
bbolt_path: storage/gateway.db

# Agentic HTTP Bridge Proxy URL (ADK Protocol endpoint)
agentic_url: http://192.168.1.10:8080/api/v1/chat

# DHT / Peer-to-Peer Settings
p2p_port: "0" # 0 means use random ephemeral port for holepunching
p2p_listen_addr: 0.0.0.0 # Bind P2P engine to all interfaces to listen for incoming WAN holepunches

# Access control whitelist (only allow specific Keet mobile keys to replicate blocks)
client_whitelist:
  - "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" # Replace with your Keet Mobile App's public key
```

---

## 3. Whitelist Verification & Security

To prevent unauthorized internet nodes from synchronizing with your RPi5 or commanding your local model, the whitelisting mechanism acts directly at the replication and command level.

1. **Replication Guard:** The ADK Gateway verifies peer identities during the initial uTP handshake inside `join_swarm` and block requests.
2. **Access Protection:** Only feeds and blocks carrying signatures associated with whitelisted keys are verified, processed, and persisted in the local BoltDB database.

---

## 4. Running the Agentic Loop with `agentic`

Launch your agentic runtime on the local network IP:

```bash
# Execute agentic framework on local IP with WebUI and Agent-to-Agent protocol enabled
./agentic -webui -a2a -host=192.168.1.10
```

The gateway's HTTP bridge parses the response payload from `agentic` in order of common JSON keys: `response`, `content`, or `message`. This makes it compatible with any standard ADK HTTP specifications.

---

## 5. Deployment Summary Checklist

1. [ ] **DHT Port Accessibility:** Ensure that your router allows UDP egress so the gateway can query external bootstrap DHT nodes and announce topics.
2. [ ] **Whitelist Key:** Capture your Keet mobile app’s public key from the app settings, add it to `client_whitelist` in `config.yaml`, and restart the gateway on your Pi.
3. [ ] **Agentic Host:** Ensure `agentic_url` in your `config.yaml` points correctly to the IP/port exposed by your `./agentic -host=192.168.1.10` runner.
