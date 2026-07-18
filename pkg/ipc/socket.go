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

	"github.com/innomon/keet-adk-gateway/pkg/db"
	"github.com/innomon/keet-adk-gateway/pkg/dht"
	"github.com/innomon/keet-adk-gateway/pkg/hypercore"
)

type SocketListener struct {
	listener net.Listener
	path     string
}

func NewSocketListener(path string) (*SocketListener, error) {
	// Clean up stale socket file if it exists
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to clear stale socket: %w", err)
	}

	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, fmt.Errorf("failed to bind Unix socket: %w", err)
	}

	return &SocketListener{
		listener: listener,
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
	// Clean up socket file on close
	_ = os.Remove(s.path)
	return err
}

func HandleClient(ctx context.Context, conn net.Conn, node *dht.DHTNode, reg *dht.SwarmRegistry, store *hypercore.Storage, swarmRepo *db.SwarmRepository, blockRepo *db.BlockRepository) {
	defer conn.Close()
	slog.Info("New ADK Client pipeline bound successfully", "remote", conn.RemoteAddr())

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

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

			switch cmd {
			case "join_swarm":
				topic, _ := req["topic"].(string)
				peerKey, _ := req["peer_key"].(string)
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
							if blockRepo != nil {
								feedKey, _ := req["feed_key"].(string)
								if feedKey == "" {
									feedKey = "default_feed"
								}
								var sig []byte
								if sigStr, ok := req["signature"].(string); ok {
									sig, _ = base64.StdEncoding.DecodeString(sigStr)
								}
								if err := blockRepo.PutBlock(ctx, feedKey, currIndex, decoded, sig); err != nil {
									slog.Error("Failed to cache block in db", "err", err)
								}
							}
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
