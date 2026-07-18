package db

import (
	"context"
	"os"
	"testing"

	"github.com/innomon/keet-adk-gateway/pkg/config"
)

func TestSwarmRepo_RegisterUnregister(t *testing.T) {
	cfg := config.LoadConfig()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := Connect(ctx, cfg)
	if err != nil {
		if os.Getenv("DB_HOST") == "" {
			t.Skipf("PostgreSQL is not running, skipping swarm repo test: %v", err)
		} else {
			t.Fatalf("failed to connect: %v", err)
		}
	}
	defer db.Close()

	if err := RunMigrations(ctx, db); err != nil {
		t.Fatalf("migrations failed: %v", err)
	}

	repo := NewSwarmRepository(db)

	topicKey := "test_repo_topic_key"
	topicName := "Test Swarm Room"

	// Register
	if err := repo.RegisterSwarm(ctx, topicKey, topicName); err != nil {
		t.Fatalf("failed to register swarm: %v", err)
	}

	// Retrieve
	active, err := repo.GetActiveSwarms(ctx)
	if err != nil {
		t.Fatalf("failed to retrieve active swarms: %v", err)
	}

	found := false
	for _, k := range active {
		if k == topicKey {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected registered swarm topic %q in active swarms list %+v", topicKey, active)
	}

	// Unregister
	if err := repo.UnregisterSwarm(ctx, topicKey); err != nil {
		t.Fatalf("failed to unregister swarm: %v", err)
	}

	// Retrieve again
	active, err = repo.GetActiveSwarms(ctx)
	if err != nil {
		t.Fatalf("failed to retrieve active swarms: %v", err)
	}

	found = false
	for _, k := range active {
		if k == topicKey {
			found = true
			break
		}
	}
	if found {
		t.Errorf("expected unregistered swarm topic %q to be removed from active swarms list %+v", topicKey, active)
	}
}
