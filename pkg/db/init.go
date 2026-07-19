package db

import (
	"context"
	"fmt"

	"github.com/innomon/keet-adk-gateway/pkg/config"
)

// InitDatabase initializes the configured database (postgres or bbolt)
// and returns the SwarmRepository, BlockRepository, a close function, and any error.
func InitDatabase(ctx context.Context, cfg config.Config) (SwarmRepository, BlockRepository, func() error, error) {
	if cfg.DBType == "bbolt" {
		boltDB, err := NewBoltDB(cfg.BBoltPath)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("init bbolt database: %w", err)
		}
		return NewBoltSwarmRepository(boltDB), NewBoltBlockRepository(boltDB), boltDB.Close, nil
	}

	// Default: postgres
	connPool, err := Connect(ctx, cfg)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("connect postgres database: %w", err)
	}

	if err := RunMigrations(ctx, connPool); err != nil {
		connPool.Close()
		return nil, nil, nil, fmt.Errorf("run postgres migrations: %w", err)
	}

	return NewSwarmRepository(connPool), NewBlockRepository(connPool), func() error {
		connPool.Close()
		return nil
	}, nil
}
