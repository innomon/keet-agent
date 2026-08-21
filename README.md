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
* **Go**: Version `1.21` or higher installed.
* **Make**: (Optional) For convenient shortcut commands.
* **Git**: Used for automatic version tagging and build metadata injection.

---

## 🏗️ Build Instructions

The gateway provides a unified build script ([`build.sh`](file:///Volumes/mac1tb/dev/keet-agent/build.sh)) and a [`Makefile`](file:///Volumes/mac1tb/dev/keet-agent/Makefile) that compile pure-Go, statically linked binaries (`CGO_ENABLED=0`) stripped of debug symbols (`-s -w -trimpath`) for zero external library dependencies.

### Supported Target Platforms

| Platform | Target Identifier | Architecture (`GOOS/GOARCH`) | Output Binary |
|---|---|---|---|
| **macOS Apple Silicon** | `macos-arm64` / `darwin-arm64` | `darwin/arm64` (M1/M2/M3/M4) | `bin/gateway-darwin-arm64` |
| **Debian Linux ARM64** | `debian-arm64` / `linux-arm64` | `linux/arm64` (Raspberry Pi 5, SBCs, ARM VMs) | `bin/gateway-linux-arm64` |
| **Ubuntu Linux AMD64** | `ubuntu-amd64` / `linux-amd64` | `linux/amd64` (x86_64 Cloud & Servers) | `bin/gateway-linux-amd64` |

---

### 1. Build for All Platforms

Compile static binaries for all three platforms at once:

```bash
./build.sh all
# Or with make:
make build-all
```

The resulting binaries will be placed in the `bin/` directory:
* `bin/gateway-darwin-arm64` (macOS Apple Silicon)
* `bin/gateway-linux-arm64` (Debian ARM64)
* `bin/gateway-linux-amd64` (Ubuntu AMD64)

### 2. Build for a Specific Target

```bash
# macOS Apple Silicon (M1/M2/M3/M4)
./build.sh macos-arm64
# Or: make macos-arm64

# Debian Linux ARM64 (Raspberry Pi 4/5, aarch64 SBCs, Debian Cloud)
./build.sh debian-arm64
# Or: make debian-arm64

# Ubuntu Linux AMD64 (x86_64 Servers, Cloud Instances)
./build.sh ubuntu-amd64
# Or: make ubuntu-amd64

# Native Host (builds for your current development machine OS & Arch)
./build.sh host
# Or: make build
```

### 3. Build & Package for Distribution

Create standalone release archives (`.tar.gz`) containing the binary, documentation, sample configuration, and a `SHA256SUMS` checksum manifest:

```bash
./build.sh package
# Or with make:
make package
```

Archives and checksums will be saved in `dist/`:
```text
dist/
├── SHA256SUMS
├── gateway-v1.0.0-darwin-arm64.tar.gz
├── gateway-v1.0.0-linux-arm64.tar.gz
└── gateway-v1.0.0-linux-amd64.tar.gz
```

### 4. Build Script Options & Flags

```bash
Usage: ./build.sh [TARGET|COMMAND] [OPTIONS]

Targets:
  all               Build for all supported platforms (default)
  macos-arm64       Build for macOS Apple Silicon (darwin/arm64)
  debian-arm64      Build for Debian Linux ARM64 (linux/arm64)
  ubuntu-amd64      Build for Ubuntu AMD64 (linux/amd64)
  host              Build for current host OS and architecture

Commands:
  clean             Remove all built binaries and distribution archives
  package           Build and package (.tar.gz + SHA256 checksums)

Options:
  -o, --output DIR  Specify custom binary output directory (default: bin)
  -p, --package     Create compressed tarballs and checksums
  -c, --clean       Clean output directory before building
  -v, --verbose     Enable verbose compiler output
  -h, --help        Display help message
```

---

## 🚀 Running the Gateway

### 1. Quick Start

Run the binary corresponding to your platform:

```bash
# On macOS Apple Silicon:
./bin/gateway-darwin-arm64

# On Debian Linux (ARM64):
./bin/gateway-linux-arm64

# On Ubuntu Linux (AMD64):
./bin/gateway-linux-amd64
```

By default, the gateway:
* Initializes local storage at `./storage/`
* Creates an embedded BoltDB file at `./storage/gateway.db`
* Binds the ADK Unix IPC socket at `/tmp/keet-adk.sock`
* Starts the P2P DHT node and logs to stdout and `./logs/gateway.log`

---

### 2. Running with Custom Configuration

Create or modify `config.yaml` (see [`config.example.yaml`](file:///Volumes/mac1tb/dev/keet-agent/config.example.yaml)):

```bash
# Copy example configuration
cp config.example.yaml config.yaml

# Run specifying config file explicitly
./bin/gateway-darwin-arm64 --config config.yaml

# Or place config.yaml in the working directory (loaded automatically)
./bin/gateway-darwin-arm64
```

#### Configuration Precedence:
1. `--config <path>` command-line flag
2. `config.yaml` in the current working directory
3. `config.yaml` next to the executable binary
4. Environment variables (`SOCKET_PATH`, `DB_TYPE`, `LOG_LEVEL`, etc.)

---

### 3. Running with Environment Variables

You can override settings inline without modifying files:

```bash
# Enable network access via TCP socket and set DEBUG logging
SOCKET_PATH="0.0.0.0:12345" LOG_LEVEL="DEBUG" ./bin/gateway-linux-amd64

# Use PostgreSQL backend instead of BoltDB
DB_TYPE="postgres" DB_HOST="localhost" DB_PORT="5432" DB_USER="postgres" DB_PASSWORD="secret" ./bin/gateway-linux-arm64
```

---

### 4. Running as a Systemd Service (Debian / Ubuntu Linux)

To run the gateway as a background persistent service on Linux:

1. Copy the binary to `/usr/local/bin`:
   ```bash
   sudo cp bin/gateway-linux-arm64 /usr/local/bin/gateway  # for ARM64
   # or
   sudo cp bin/gateway-linux-amd64 /usr/local/bin/gateway  # for AMD64
   sudo chmod +x /usr/local/bin/gateway
   ```

2. Create a service unit file at `/etc/systemd/system/keet-gateway.service`:
   ```ini
   [Unit]
   Description=Keet ADK Gateway Daemon
   After=network.target

   [Service]
   Type=simple
   User=keet
   WorkingDirectory=/var/lib/keet-gateway
   ExecStart=/usr/local/bin/gateway --config /etc/keet-gateway/config.yaml
   Restart=always
   RestartSec=5
   LimitNOFILE=65535

   [Install]
   WantedBy=multi-user.target
   ```

3. Enable and start the service:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable --now keet-gateway
   sudo systemctl status keet-gateway
   ```

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

* 🏗️ **[System Architecture Guide](docs/ARCHITECTURE.md)** — Core service topology, packet routing, and database layout.
* 🐻 **[Bare Go Stack & Architecture Comparison](docs/bare-go.md)** — Comparative analysis of `bareclaw`, `bare-rpc-golang`, and the Keet ADK Gateway architecture.
* 💾 **[Database Design & Interfaces](docs/embedded-db.md)** — Structural details regarding BoltDB binary serialization and repository abstractions.
* 📶 **[Production Raspberry Pi 5 Guide](docs/production_setup_guide.md)** — Step-by-step tutorial on local Wi-Fi mobile routing, Whitelist setup, and local LLM orchestration with **Ollama** and **IBM Granite**.

---

## 📄 License
This project is licensed under the MIT License. See `LICENSE` for more information.
