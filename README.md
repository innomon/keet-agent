# Keet ADK Gateway

A high-performance, lightweight, pure-Go implementation of the **Keet Application Development Kit (ADK) Gateway**. 

This gateway acts as a local bridge between your custom applications or mobile clients (running the Keet app) and a distributed, secure peer-to-peer DHT network overlay. It manages swarm orchestration, secure public key whitelisting, and ultra-fast local binary append caching.

---

## 🚀 Key Features

* **Pure-Go Local Storage (Default):** Features a memory-mapped database powered by **BBolt (BoltDB)** with custom binary block packing and feed caching.
* **Optional Relational Database:** Supports PostgreSQL as a production-configurable database backend, running full auto-migrations.
* **Precedence-Based Configuration:** Loads `config.yaml` with a robust three-tier lookup (CLI `--config` override ➔ current working directory ➔ executable's directory ➔ environment variable fallbacks).
* **Multi-Protocol Socket Listener:** Accepts local connections via **Unix Domain Sockets** or network-wide connections via **TCP Sockets** (allowing mobile devices on Wi-Fi to securely connect).
* **Security & Whitelisting:** Features an access-control pipeline that requires connected clients to be explicitly authorized via a public key whitelist.
* **Structured, Enterprise Logging:** Integrates robust logging featuring configurable sizes, rotational logs, back-ups, and console-compatible formatted printouts.

---

## 🛠️ Getting Started

### Prerequisites
* Go `1.21` or higher.
* Node.js (if running client consumer scripts or test frameworks).

### Installation & Build
Clone the repository and build the gateway executable:

```bash
# Build the binary
go build -o bin/gateway cmd/gateway/main.go
```

### Running the Gateway
Running the gateway is as simple as launching the executable. It will automatically initialize the local `storage/` directory and spin up the DHT node:

```bash
# Run with default environment variables (e.g. Unix socket path /tmp/keet-adk.sock)
./bin/gateway

# Run using a specific configuration file
./bin/gateway --config path/to/config.yaml
```

---

## ⚙️ Configuration Reference

You can customize the gateway's behavior by placing a `config.yaml` next to your binary or in your working directory. Here is an overview of the key properties:

| Key | Default Value | Description |
|---|---|---|
| `socket_path` | `/tmp/keet-adk.sock` | Socket to bind to. Use a TCP port (e.g., `0.0.0.0:12345` or `tcp://0.0.0.0:12345`) to enable network access. |
| `db_type` | `bbolt` | Backend persistence driver (`bbolt` or `postgres`). |
| `bbolt_path` | `storage/gateway.db` | Directory path where the embedded BBolt file will be created. |
| `client_whitelist` | `[]` | List of authorized client public keys in hex. If empty, access control is disabled. |
| `p2p_listen_addr`| `127.0.0.1` | Local IP address for the DHT node bind interface. |
| `log_level` | `INFO` | Output log verbosity (`DEBUG`, `INFO`, `WARN`, `ERROR`). |

---

## 📚 Documentation Index

To help you get the most out of the Keet ADK Gateway, check out the specialized guides in our `docs/` directory:

* 🏗️ **[System Architecture Guide](file:///home/innomon/orez/apps/keet-agent/docs/ARCHITECTURE.md)** — Core service topology, packet routing, and database layout.
* 💾 **[Database Design & Interfaces](file:///home/innomon/orez/apps/keet-agent/docs/embedded-db.md)** — Structural details regarding BoltDB binary serialization and repository abstractions.
* 📶 **[Production Raspberry Pi 5 Guide](file:///home/innomon/orez/apps/keet-agent/docs/production_setup_guide.md)** — Step-by-step tutorial on local Wi-Fi mobile routing, Whitelist setup, and local LLM orchestration with **Ollama** and **IBM Granite**.

---

## 📄 License
This project is licensed under the MIT License. See `LICENSE` for more information.
