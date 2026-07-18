package db

import (
	"context"
	"fmt"
)

type SwarmRepository struct {
	db *DB
}

func NewSwarmRepository(db *DB) *SwarmRepository {
	return &SwarmRepository{db: db}
}

func (r *SwarmRepository) RegisterSwarm(ctx context.Context, topicKey, topicName string) error {
	query := `
	INSERT INTO swarms (topic_key, topic_name) VALUES ($1, $2)
	ON CONFLICT (topic_key) DO UPDATE SET topic_name = $2;`

	_, err := r.db.Pool.Exec(ctx, query, topicKey, topicName)
	if err != nil {
		return fmt.Errorf("db register swarm: %w", err)
	}
	return nil
}

func (r *SwarmRepository) UnregisterSwarm(ctx context.Context, topicKey string) error {
	query := `DELETE FROM swarms WHERE topic_key = $1;`

	_, err := r.db.Pool.Exec(ctx, query, topicKey)
	if err != nil {
		return fmt.Errorf("db unregister swarm: %w", err)
	}
	return nil
}

func (r *SwarmRepository) GetActiveSwarms(ctx context.Context) ([]string, error) {
	query := `SELECT topic_key FROM swarms;`

	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("db get active swarms: %w", err)
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("db scan swarm row: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("db active swarms iteration error: %w", err)
	}

	return keys, nil
}
