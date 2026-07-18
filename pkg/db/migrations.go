package db

import (
	"context"
	"fmt"
)

func RunMigrations(ctx context.Context, db *DB) error {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Create swarms table
	swarmsSchema := `
	CREATE TABLE IF NOT EXISTS swarms (
		topic_key TEXT PRIMARY KEY,
		topic_name TEXT NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err := tx.Exec(ctx, swarmsSchema); err != nil {
		return fmt.Errorf("create swarms table: %w", err)
	}

	// Create blocks table
	blocksSchema := `
	CREATE TABLE IF NOT EXISTS blocks (
		feed_key TEXT NOT NULL,
		block_index BIGINT NOT NULL,
		value BYTEA NOT NULL,
		signature BYTEA NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (feed_key, block_index)
	);`
	if _, err := tx.Exec(ctx, blocksSchema); err != nil {
		return fmt.Errorf("create blocks table: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
