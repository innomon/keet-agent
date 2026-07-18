package ipc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net"
	"os"
	"testing"

	"github.com/innomon/keet-adk-gateway/pkg/config"
	"github.com/innomon/keet-adk-gateway/pkg/db"
	"github.com/innomon/keet-adk-gateway/pkg/hypercore"
)

func TestSocket_DBIntegration(t *testing.T) {
	cfg := config.LoadConfig()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to DB (skip-safe)
	connPool, err := db.Connect(ctx, cfg)
	if err != nil {
		if os.Getenv("DB_HOST") == "" {
			t.Skipf("PostgreSQL is not running, skipping socket DB integration test: %v", err)
		} else {
			t.Fatalf("failed to connect: %v", err)
		}
	}
	defer connPool.Close()

	if err := db.RunMigrations(ctx, connPool); err != nil {
		t.Fatalf("migrations failed: %v", err)
	}

	swarmRepo := db.NewSwarmRepository(connPool)
	blockRepo := db.NewBlockRepository(connPool)

	// Set up temporary storage and socket
	tempDir, err := os.MkdirTemp("", "hypercore-db-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := hypercore.NewStorage(tempDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer storage.Close()

	socketPath := "/tmp/keet-adk-db-test.sock"
	_ = os.Remove(socketPath)

	listener, err := NewSocketListener(socketPath)
	if err != nil {
		t.Fatalf("failed to create socket: %v", err)
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go HandleClient(ctx, conn, nil, nil, storage, swarmRepo, blockRepo)
		}
	}()

	client, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to dial socket: %v", err)
	}
	defer client.Close()

	dec := json.NewDecoder(client)
	enc := json.NewEncoder(client)

	// 1. Join Swarm
	joinReq := map[string]interface{}{
		"command":  "join_swarm",
		"topic":    "chat_topic_db_test",
		"peer_key": "some_peer_key",
	}
	if err := enc.Encode(&joinReq); err != nil {
		t.Fatalf("failed to encode join swarm req: %v", err)
	}

	var joinResp map[string]interface{}
	if err := dec.Decode(&joinResp); err != nil {
		t.Fatalf("failed to decode join swarm resp: %v", err)
	}
	if joinResp["status"] != "success" {
		t.Fatalf("join swarm status not success: %v", joinResp)
	}

	// Verify swarm persisted in DB
	active, err := swarmRepo.GetActiveSwarms(ctx)
	if err != nil {
		t.Fatalf("failed to get active swarms from db: %v", err)
	}
	resolvedTopicKey, _ := joinResp["resolved_topic_key"].(string)
	found := false
	for _, key := range active {
		if key == resolvedTopicKey {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected joined swarm topic key %s in database active swarms", resolvedTopicKey)
	}

	// 2. Append Block
	blockPayload := []byte("database cached log item value")
	appendReq := map[string]interface{}{
		"command":   "append_block",
		"feed_key":  "my_feed_key",
		"data":      base64.StdEncoding.EncodeToString(blockPayload),
		"signature": base64.StdEncoding.EncodeToString([]byte("signature_bytes")),
	}
	if err := enc.Encode(&appendReq); err != nil {
		t.Fatalf("failed to encode append req: %v", err)
	}

	var appendResp map[string]interface{}
	if err := dec.Decode(&appendResp); err != nil {
		t.Fatalf("failed to decode append resp: %v", err)
	}
	if appendResp["status"] != "success" {
		t.Fatalf("append status not success: %v", appendResp)
	}

	// Retrieve block index
	indexVal := uint64(appendResp["index"].(float64))

	// Verify block persisted in DB
	dbVal, dbSig, err := blockRepo.GetBlock(ctx, "my_feed_key", indexVal)
	if err != nil {
		t.Fatalf("failed to get block from db: %v", err)
	}
	if string(dbVal) != string(blockPayload) {
		t.Errorf("expected block value %q from db, got %q", string(blockPayload), string(dbVal))
	}
	if string(dbSig) != "signature_bytes" {
		t.Errorf("expected block signature %q from db, got %q", "signature_bytes", string(dbSig))
	}

	// 3. Get Block
	getReq := map[string]interface{}{
		"command":  "get_block",
		"feed_key": "my_feed_key",
		"index":    indexVal,
	}
	if err := enc.Encode(&getReq); err != nil {
		t.Fatalf("failed to encode get req: %v", err)
	}

	var getResp map[string]interface{}
	if err := dec.Decode(&getResp); err != nil {
		t.Fatalf("failed to decode get resp: %v", err)
	}
	if getResp["status"] != "success" {
		t.Fatalf("get block status not success: %v", getResp)
	}
	retrievedData, _ := base64.StdEncoding.DecodeString(getResp["data"].(string))
	if string(retrievedData) != string(blockPayload) {
		t.Errorf("expected block data %q, got %q", string(blockPayload), string(retrievedData))
	}

	// 4. Leave Swarm
	leaveReq := map[string]interface{}{
		"command": "leave_swarm",
		"topic":   "chat_topic_db_test",
	}
	if err := enc.Encode(&leaveReq); err != nil {
		t.Fatalf("failed to encode leave req: %v", err)
	}

	var leaveResp map[string]interface{}
	if err := dec.Decode(&leaveResp); err != nil {
		t.Fatalf("failed to decode leave resp: %v", err)
	}

	// Verify swarm removed from DB
	active, err = swarmRepo.GetActiveSwarms(ctx)
	if err != nil {
		t.Fatalf("failed to get active swarms from db: %v", err)
	}
	found = false
	for _, key := range active {
		if key == resolvedTopicKey {
			found = true
			break
		}
	}
	if found {
		t.Errorf("expected left swarm topic key %s to be removed from database active swarms", resolvedTopicKey)
	}
}
