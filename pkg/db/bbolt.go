package db

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"

	"go.etcd.io/bbolt"
)

type BoltDB struct {
	db *bbolt.DB
}

var (
	swarmsBucket = []byte("swarms")
	blocksBucket = []byte("blocks")
)

// NewBoltDB opens and initializes a BoltDB database.
func NewBoltDB(path string) (*BoltDB, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("open bbolt: %w", err)
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(swarmsBucket); err != nil {
			return fmt.Errorf("create swarms bucket: %w", err)
		}
		if _, err := tx.CreateBucketIfNotExists(blocksBucket); err != nil {
			return fmt.Errorf("create blocks bucket: %w", err)
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize buckets: %w", err)
	}

	return &BoltDB{db: db}, nil
}

// Close closes the BoltDB.
func (b *BoltDB) Close() error {
	return b.db.Close()
}

// BoltSwarmRepository implements SwarmRepository using BoltDB.
type BoltSwarmRepository struct {
	b *BoltDB
}

func NewBoltSwarmRepository(b *BoltDB) SwarmRepository {
	return &BoltSwarmRepository{b: b}
}

func (r *BoltSwarmRepository) RegisterSwarm(ctx context.Context, topicKey, topicName string) error {
	return r.b.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(swarmsBucket)
		if bucket == nil {
			return fmt.Errorf("swarms bucket not found")
		}
		return bucket.Put([]byte(topicKey), []byte(topicName))
	})
}

func (r *BoltSwarmRepository) UnregisterSwarm(ctx context.Context, topicKey string) error {
	return r.b.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(swarmsBucket)
		if bucket == nil {
			return fmt.Errorf("swarms bucket not found")
		}
		return bucket.Delete([]byte(topicKey))
	})
}

func (r *BoltSwarmRepository) GetActiveSwarms(ctx context.Context) ([]string, error) {
	var active []string
	err := r.b.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(swarmsBucket)
		if bucket == nil {
			return fmt.Errorf("swarms bucket not found")
		}
		return bucket.ForEach(func(k, v []byte) error {
			active = append(active, string(k))
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return active, nil
}

// BoltBlockRepository implements BlockRepository using BoltDB.
type BoltBlockRepository struct {
	b *BoltDB
}

func NewBoltBlockRepository(b *BoltDB) BlockRepository {
	return &BoltBlockRepository{b: b}
}

func (r *BoltBlockRepository) PutBlock(ctx context.Context, feedKey string, index uint64, value, signature []byte) error {
	return r.b.db.Update(func(tx *bbolt.Tx) error {
		blocksB := tx.Bucket(blocksBucket)
		if blocksB == nil {
			return fmt.Errorf("blocks bucket not found")
		}

		feedB, err := blocksB.CreateBucketIfNotExists([]byte(feedKey))
		if err != nil {
			return fmt.Errorf("create feed bucket: %w", err)
		}

		key := make([]byte, 8)
		binary.BigEndian.PutUint64(key, index)

		encoded := encodeBlock(value, signature)
		return feedB.Put(key, encoded)
	})
}

func (r *BoltBlockRepository) GetBlock(ctx context.Context, feedKey string, index uint64) ([]byte, []byte, error) {
	var value, signature []byte
	err := r.b.db.View(func(tx *bbolt.Tx) error {
		blocksB := tx.Bucket(blocksBucket)
		if blocksB == nil {
			return fmt.Errorf("blocks bucket not found")
		}

		feedB := blocksB.Bucket([]byte(feedKey))
		if feedB == nil {
			return fmt.Errorf("feed bucket not found")
		}

		key := make([]byte, 8)
		binary.BigEndian.PutUint64(key, index)

		data := feedB.Get(key)
		if data == nil {
			return fmt.Errorf("block not found")
		}

		var err error
		value, signature, err = decodeBlock(data)
		return err
	})
	if err != nil {
		return nil, nil, err
	}
	return value, signature, nil
}

func encodeBlock(value, signature []byte) []byte {
	valLen := len(value)
	sigLen := len(signature)
	buf := make([]byte, 8+valLen+sigLen)
	binary.BigEndian.PutUint32(buf[0:4], uint32(valLen))
	copy(buf[4:4+valLen], value)
	binary.BigEndian.PutUint32(buf[4+valLen:8+valLen], uint32(sigLen))
	copy(buf[8+valLen:], signature)
	return buf
}

func decodeBlock(data []byte) ([]byte, []byte, error) {
	if len(data) < 8 {
		return nil, nil, fmt.Errorf("data too short")
	}
	valLen := int(binary.BigEndian.Uint32(data[0:4]))
	if len(data) < 8+valLen {
		return nil, nil, fmt.Errorf("data too short for value of length %d", valLen)
	}
	value := make([]byte, valLen)
	copy(value, data[4:4+valLen])

	sigLen := int(binary.BigEndian.Uint32(data[4+valLen : 8+valLen]))
	if len(data) < 8+valLen+sigLen {
		return nil, nil, fmt.Errorf("data too short for signature of length %d", sigLen)
	}
	signature := make([]byte, sigLen)
	copy(signature, data[8+valLen:8+valLen+sigLen])

	return value, signature, nil
}
