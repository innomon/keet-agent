package db

import "context"

// SwarmRepository abstracts operations for storing active swarms.
type SwarmRepository interface {
	RegisterSwarm(ctx context.Context, topicKey, topicName string) error
	UnregisterSwarm(ctx context.Context, topicKey string) error
	GetActiveSwarms(ctx context.Context) ([]string, error)
}

// BlockRepository abstracts operations for storing hypercore blocks.
type BlockRepository interface {
	PutBlock(ctx context.Context, feedKey string, index uint64, value, signature []byte) error
	GetBlock(ctx context.Context, feedKey string, index uint64) ([]byte, []byte, error)
}
