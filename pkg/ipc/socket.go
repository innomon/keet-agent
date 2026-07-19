package ipc

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/innomon/keet-adk-gateway/pkg/db"
	"github.com/innomon/keet-adk-gateway/pkg/dht"
	"github.com/innomon/keet-adk-gateway/pkg/hypercore"
)

type ClientRegistry struct {
	mu      sync.Mutex
	clients map[net.Conn]*json.Encoder
}

var ActiveClients = &ClientRegistry{
	clients: make(map[net.Conn]*json.Encoder),
}

func (cr *ClientRegistry) Register(conn net.Conn, encoder *json.Encoder) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.clients[conn] = encoder
}

func (cr *ClientRegistry) Unregister(conn net.Conn) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	delete(cr.clients, conn)
}

func (cr *ClientRegistry) Broadcast(msg map[string]interface{}) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	for conn, encoder := range cr.clients {
		if err := encoder.Encode(msg); err != nil {
			slog.Error("Failed to broadcast message to client", "remote", conn.RemoteAddr(), "err", err)
		}
	}
}

func BroadcastChatMessage(feedKey string, index uint64, value []byte) {
	var payload map[string]interface{}
	if err := json.Unmarshal(value, &payload); err == nil {
		// Verify if it contains the required keys of a ChatMessage
		_, hasSender := payload["sender"]
		_, hasTimestamp := payload["timestamp"]
		_, hasContent := payload["content"]

		if hasSender && hasTimestamp && hasContent {
			notification := map[string]interface{}{
				"command":   "chat_message_received",
				"feed_key":  feedKey,
				"index":     index,
				"sender":    payload["sender"],
				"timestamp": payload["timestamp"],
				"content":   payload["content"],
			}
			ActiveClients.Broadcast(notification)
		}
	}
}

type SocketListener struct {
	listener net.Listener
	network  string
	path     string
}

func NewSocketListener(addr string) (*SocketListener, error) {
	network := "unix"
	path := addr

	if strings.HasPrefix(addr, "tcp://") {
		network = "tcp"
		path = strings.TrimPrefix(addr, "tcp://")
	} else if strings.Contains(addr, ":") {
		// Try TCP if there is a colon and no forward slashes
		if !strings.Contains(addr, "/") {
			network = "tcp"
		}
	}

	if network == "unix" {
		// Clean up stale socket file if it exists
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to clear stale socket: %w", err)
		}
	}

	listener, err := net.Listen(network, path)
	if err != nil {
		return nil, fmt.Errorf("failed to bind %s listener on %s: %w", network, path, err)
	}

	return &SocketListener{
		listener: listener,
		network:  network,
		path:     path,
	}, nil
}

func (s *SocketListener) Accept() (net.Conn, error) {
	return s.listener.Accept()
}

func (s *SocketListener) Close() error {
	var err error
	if s.listener != nil {
		err = s.listener.Close()
	}
	if s.network == "unix" {
		// Clean up socket file on close
		_ = os.Remove(s.path)
	}
	return err
}


func HandleClient(ctx context.Context, conn net.Conn, node *dht.DHTNode, reg *dht.SwarmRegistry, store *hypercore.Storage, swarmRepo db.SwarmRepository, blockRepo db.BlockRepository, whitelist []string) {
	defer conn.Close()
	slog.Info("New ADK Client pipeline bound successfully", "remote", conn.RemoteAddr())

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	ActiveClients.Register(conn, encoder)
	defer ActiveClients.Unregister(conn)

	authenticated := len(whitelist) == 0

	for {
		select {
		case <-ctx.Done():
			return
		default:
			var req map[string]interface{}
			if err := decoder.Decode(&req); err != nil {
				slog.Warn("Disconnected or malformed ADK frames detected", "err", err)
				return
			}

			slog.Debug("ADK command intercept", "payload", req)

			cmd, _ := req["command"].(string)
			resp := map[string]interface{}{"status": "acknowledged", "origin": "keet_peer"}

			// Check and extract peer key for inline authentication
			peerKey, _ := req["peer_key"].(string)
			if peerKey != "" && isWhitelisted(peerKey, whitelist) {
				authenticated = true
			}

			if !authenticated {
				resp = map[string]interface{}{
					"status":  "error",
					"command": cmd,
					"error":   "unauthorized client public key or authentication required",
				}
				_ = encoder.Encode(&resp)
				slog.Warn("Rejected unauthorized ADK client attempt", "remote", conn.RemoteAddr())
				return
			}

			switch cmd {
			case "auth":
				resp = map[string]interface{}{
					"status":  "success",
					"command": "auth",
				}
			case "join_swarm":
				topic, _ := req["topic"].(string)
				resolvedKey, err := dht.ResolveTopicKey(topic)
				if err != nil {
					resp = map[string]interface{}{
						"status":  "error",
						"command": "join_swarm",
						"error":   fmt.Sprintf("invalid topic: %v", err),
					}
				} else {
					if reg != nil {
						reg.RegisterPeer(resolvedKey, peerKey)
					}
					if swarmRepo != nil {
						if err := swarmRepo.RegisterSwarm(ctx, hex.EncodeToString(resolvedKey[:]), topic); err != nil {
							slog.Error("Failed to register swarm in db", "err", err)
						}
					}

					if node != nil {
						var p2pPort uint16
						if reg != nil {
							p2pPort = reg.P2PPort
						}
						if err := node.Announce(ctx, resolvedKey, p2pPort); err != nil {
							slog.Error("Failed to announce to DHT on join_swarm", "err", err)
						}

						peers, err := node.Lookup(ctx, resolvedKey)
						if err == nil {
							for _, peer := range peers {
								if reg != nil {
									reg.RegisterPeer(resolvedKey, peer)
								}
							}
						} else {
							slog.Error("Failed to lookup from DHT on join_swarm", "err", err)
						}
					}

					slog.Info("Successfully joined DHT swarm topic", "topic", topic, "key", hex.EncodeToString(resolvedKey[:]))

					resp = map[string]interface{}{
						"status":             "success",
						"command":            "join_swarm",
						"topic":              topic,
						"resolved_topic_key": hex.EncodeToString(resolvedKey[:]),
					}
				}
			case "leave_swarm":
				topic, _ := req["topic"].(string)
				resolvedKey, err := dht.ResolveTopicKey(topic)
				if err == nil {
					if node != nil {
						_ = node.Leave(ctx, resolvedKey)
					}
					if reg != nil {
						reg.ClearSwarm(resolvedKey)
					}
					if swarmRepo != nil {
						if err := swarmRepo.UnregisterSwarm(ctx, hex.EncodeToString(resolvedKey[:])); err != nil {
							slog.Error("Failed to unregister swarm in db", "err", err)
						}
					}
				}
				slog.Info("Successfully left DHT swarm topic", "topic", topic)
				resp = map[string]interface{}{
					"status":  "success",
					"command": "leave_swarm",
					"topic":   topic,
				}
			case "append_block":
				dataStr, _ := req["data"].(string)
				if store == nil {
					resp = map[string]interface{}{
						"status":  "error",
						"command": "append_block",
						"error":   "storage not initialized",
					}
				} else {
					decoded, err := base64.StdEncoding.DecodeString(dataStr)
					if err != nil {
						resp = map[string]interface{}{
							"status":  "error",
							"command": "append_block",
							"error":   fmt.Sprintf("invalid base64: %v", err),
						}
					} else {
						currIndex := store.Len()
						if err := store.Append(decoded); err != nil {
							resp = map[string]interface{}{
								"status":  "error",
								"command": "append_block",
								"error":   fmt.Sprintf("failed to append: %v", err),
							}
						} else {
							feedKey, _ := req["feed_key"].(string)
							if feedKey == "" {
								feedKey = "default_feed"
							}
							if blockRepo != nil {
								var sig []byte
								if sigStr, ok := req["signature"].(string); ok {
									sig, _ = base64.StdEncoding.DecodeString(sigStr)
								}
								if err := blockRepo.PutBlock(ctx, feedKey, currIndex, decoded, sig); err != nil {
									slog.Error("Failed to cache block in db", "err", err)
								}
							}
							BroadcastChatMessage(feedKey, currIndex, decoded)
							resp = map[string]interface{}{
								"status":  "success",
								"command": "append_block",
								"index":   currIndex,
							}
						}
					}
				}
			case "get_block":
				indexVal, ok := req["index"]
				if !ok || store == nil {
					resp = map[string]interface{}{
						"status":  "error",
						"command": "get_block",
						"error":   "invalid request or storage not initialized",
					}
				} else {
					var index uint64
					switch v := indexVal.(type) {
					case float64:
						index = uint64(v)
					case int:
						index = uint64(v)
					case uint64:
						index = v
					}

					block, err := store.Get(index)
					if err != nil {
						// Attempt to fallback to database block cache
						if blockRepo != nil {
							feedKey, _ := req["feed_key"].(string)
							if feedKey == "" {
								feedKey = "default_feed"
							}
							val, _, dbErr := blockRepo.GetBlock(ctx, feedKey, index)
							if dbErr == nil {
								block = val
								err = nil
							}
						}
					}

					if err != nil {
						resp = map[string]interface{}{
							"status":  "error",
							"command": "get_block",
							"error":   fmt.Sprintf("failed to get block: %v", err),
						}
					} else {
						resp = map[string]interface{}{
							"status":  "success",
							"command": "get_block",
							"data":    base64.StdEncoding.EncodeToString(block),
						}
					}
				}
			}

			if err := encoder.Encode(&resp); err != nil {
				slog.Error("Failed response serialization upstream to client", "err", err)
				return
			}
		}
	}
}

func isWhitelisted(key string, whitelist []string) bool {
	if len(whitelist) == 0 {
		return true
	}
	for _, w := range whitelist {
		if w == key {
			return true
		}
	}
	return false
}

