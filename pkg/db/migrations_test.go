package db

import (
	"context"
	"os"
	"testing"

	"github.com/innomon/keet-adk-gateway/pkg/config"
)

func TestMigrations_Run(t *testing.T) {
	cfg := config.LoadConfig()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := Connect(ctx, cfg)
	if err != nil {
		if os.Getenv("DB_HOST") == "" {
			t.Skipf("PostgreSQL is not running, skipping migrations test: %v", err)
		} else {
			t.Fatalf("failed to connect: %v", err)
		}
	}
	defer db.Close()

	if err := RunMigrations(ctx, db); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Verify tables exist by attempting to insert a test entry
	_, err = db.Pool.Exec(ctx, "INSERT INTO swarms (topic_key, topic_name) VALUES ($1, $2)", "test_topic_key", "test_topic")
	if err != nil {
		t.Fatalf("failed to insert test swarm row (table swarms likely missing or broken): %v", err)
	}

	_, err = db.Pool.Exec(ctx, "DELETE FROM swarms WHERE topic_key = $1", "test_topic_key")
	if err != nil {
		t.Fatalf("failed to clean up test swarm row: %v", err)
	}
}
