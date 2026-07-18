package db

import (
	"context"
	"fmt"
)

type BlockRepository struct {
	db *DB
}

func NewBlockRepository(db *DB) *BlockRepository {
	return &BlockRepository{db: db}
}

func (r *BlockRepository) PutBlock(ctx context.Context, feedKey string, index uint64, value, signature []byte) error {
	query := `
	INSERT INTO blocks (feed_key, block_index, value, signature)
	VALUES ($1, $2, $3, $4)
	ON CONFLICT (feed_key, block_index) DO UPDATE SET value = $3, signature = $4;`

	_, err := r.db.Pool.Exec(ctx, query, feedKey, int64(index), value, signature)
	if err != nil {
		return fmt.Errorf("db put block: %w", err)
	}
	return nil
}

func (r *BlockRepository) GetBlock(ctx context.Context, feedKey string, index uint64) ([]byte, []byte, error) {
	query := `SELECT value, signature FROM blocks WHERE feed_key = $1 AND block_index = $2;`

	var value, signature []byte
	err := r.db.Pool.QueryRow(ctx, query, feedKey, int64(index)).Scan(&value, &signature)
	if err != nil {
		return nil, nil, fmt.Errorf("db get block: %w", err)
	}

	return value, signature, nil
}
