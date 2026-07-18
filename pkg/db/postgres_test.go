package db

import (
	"context"
	"os"
	"testing"

	"github.com/innomon/keet-adk-gateway/pkg/config"
)

func TestDB_Connect(t *testing.T) {
	// If TEST_DB_CONN is not set or empty, we skip live DB connection test unless localhost connection works.
	cfg := config.LoadConfig()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := Connect(ctx, cfg)
	if err != nil {
		// Sane check: skip if database isn't running, but error out if the user explicitly set DB envs indicating they expected a running instance.
		if os.Getenv("DB_HOST") == "" {
			t.Skipf("PostgreSQL is not running on localhost, skipping connection test: %v", err)
		} else {
			t.Fatalf("failed to connect to configured database: %v", err)
		}
	}
	defer db.Close()

	if err := db.Pool.Ping(ctx); err != nil {
		t.Fatalf("ping failed: %v", err)
	}
}
