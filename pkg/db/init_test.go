package db

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/innomon/keet-adk-gateway/pkg/config"
)

func TestInitDatabase(t *testing.T) {
	ctx := context.Background()

	t.Run("BBolt", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "bbolt-init-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		cfg := config.Config{
			DBType:    "bbolt",
			BBoltPath: filepath.Join(tmpDir, "init.db"),
		}

		swarmRepo, blockRepo, closeFunc, err := InitDatabase(ctx, cfg)
		if err != nil {
			t.Fatalf("failed to init bbolt database: %v", err)
		}
		defer closeFunc()

		if swarmRepo == nil {
			t.Error("expected non-nil swarmRepo")
		}
		if blockRepo == nil {
			t.Error("expected non-nil blockRepo")
		}
	})

	t.Run("PostgreSQL Skip or Run", func(t *testing.T) {
		cfg := config.LoadConfig()
		cfg.DBType = "postgres"

		swarmRepo, blockRepo, closeFunc, err := InitDatabase(ctx, cfg)
		if err != nil {
			if os.Getenv("DB_HOST") == "" {
				t.Skipf("PostgreSQL not running, skipping: %v", err)
			} else {
				t.Fatalf("failed to init postgres database: %v", err)
			}
		}
		defer closeFunc()

		if swarmRepo == nil {
			t.Error("expected non-nil swarmRepo")
		}
		if blockRepo == nil {
			t.Error("expected non-nil blockRepo")
		}
	})
}
