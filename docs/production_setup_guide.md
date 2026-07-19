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

No public port forwarding or dynamic DNS is required for the client-gateway connection!

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

## 2. Dual-Mode Operation (Usecase Scenarios)

The gateway supports two operational modes simultaneously. This provides extreme versatility depending on your integration needs.

### 🌟 Usecase Scenario 1: Standalone P2P Chat Proxy (HTTP Bridge Mode)
* **Goal:** Run an automated, hands-free assistant on your Pi 5 behind firewalls, making your local LLM accessible to your phone anywhere in the world.
* **How it works:** You run `./agentic` on HTTP port `8080` and the gateway daemon. There is **no need for any socket client**. The gateway automatically captures inbound P2P chat messages, translates them into ADK HTTP post requests, gets the response, and publishes it back to the swarm.

### 🌟 Usecase Scenario 2: Interactive Socket Control (IPC Local Mode)
* **Goal:** Perform real-time administrative commands, monitor live network swarm sessions, or build local custom dashboards side-by-side with your active agent.
* **How it works:** In addition to forwarding messages via HTTP, the gateway keeps its local Unix domain socket `/tmp/keet-adk.sock` active. You can run local diagnostic scripts or direct command-line clients to verify database caches, inspect feeds, or broadcast manual broadcast notifications.

---

## 3. Step-by-Step Runnable Examples

Below are concrete, terminal-ready commands you can run directly on your Raspberry Pi 5 to test and interact with the gateway in both modes.

### Example A: Running in Standalone HTTP Proxy Mode

1. **Configure `config.yaml`:**
   Create a `config.yaml` file in your directory:
   ```yaml
   log_level: INFO
   db_type: bbolt
   bbolt_path: storage/gateway.db
   socket_path: /tmp/keet-adk.sock
   agentic_url: http://192.168.1.10:8080/api/v1/chat
   client_whitelist:
     - "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
   ```

2. **Start your `agentic` runtime:**
   ```bash
   ./agentic -webui -a2a -host=192.168.1.10
   ```

3. **Start the Keet ADK Gateway:**
   ```bash
   ./bin/gateway --config config.yaml
   ```
   Now, any message sent from your Whitelisted Mobile App in the swarm room is automatically routed over HTTP to `agentic`, resolved, and synced back to your mobile client.

---

### Example B: Testing & Interacting via Local Unix Socket (IPC Mode)

You can use the native utility `nc` (netcat) to establish an interactive, real-time command session directly into the local Unix domain socket. This is incredibly useful for manual diagnostics.

1. **Connect to the socket using `netcat`:**
   ```bash
   nc -U /tmp/keet-adk.sock
   ```

2. **Send an explicit Authentication Command:**
   Copy, paste, and press Enter to authenticate your session:
   ```json
   {"command": "auth", "peer_key": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}
   ```
   *Expected Response:*
   ```json
   {"status":"success","command":"auth"}
   ```

3. **Manually Join a Swarm Room:**
   Send the join command to register the topic:
   ```json
   {"command": "join_swarm", "topic": "my-secure-room", "peer_key": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}
   ```
   *Expected Response:*
   ```json
   {"status":"success","command":"join_swarm","resolved_topic_key":"..."}
   ```

4. **Append a Message Block Manually:**
   Package a Base64-encoded message and send it:
   ```json
   {"command": "append_block", "feed_key": "default_feed", "data": "eyJzZW5kZXIiOiJtYW51YWwiLCJjb250ZW50IjoiSGVsbG8gZnJvbSBOZXRjYXQhIn0="}
   ```
   *Expected Response:*
   ```json
   {"status":"success","command":"append_block","index":0}
   ```

---

## 4. Deployment Summary Checklist

1. [ ] **DHT Port Accessibility:** Ensure that your router allows UDP egress so the gateway can query external bootstrap DHT nodes and announce topics.
2. [ ] **Whitelist Key:** Capture your Keet mobile app’s public key from the app settings, add it to `client_whitelist` in `config.yaml`, and restart the gateway on your Pi.
3. [ ] **Agentic Host:** Ensure `agentic_url` in your `config.yaml` points correctly to the IP/port exposed by your `./agentic -host=192.168.1.10` runner.
