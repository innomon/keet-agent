The Noise channel used by Keet and the underlying Pear/Holepunch runtime is **fully encrypted**.

The connection architecture relies on **Hyperswarm** and **HyperDHT** for peer discovery and NAT traversal. Once a direct peer-to-peer connection is successfully established, the connection is instantly upgraded using the **Noise Protocol**.

Specifically, the cryptographic implementation utilizes:

* **Handshake/Key Exchange:** The **Noise XX pattern** using ephemeral Diffie-Hellman keys to authenticate the session and derive a shared secret.
* **Symmetric Encryption:** **XChaCha20-Poly1305** (via `libsodium`'s secretstream layer) to securely encrypt all text, data, and multimedia payloads flying over the connection.

Because it is end-to-end encrypted by default, no intermediate DHT nodes or external servers can intercept or read the traffic.