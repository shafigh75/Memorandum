package db

import (
	"Memorandum/config" // Adjust the import path as necessary
	"hash/fnv"
	"sync"
	"time"
)

// ValueWithTTL represents a value with its expiration time.
type ValueWithTTL struct {
	Value      string
	Expiration int64 // Unix timestamp in seconds
}

// mapShard represents a single shard of the in-memory store.
type mapShard struct {
	mu    sync.RWMutex
	store map[string]ValueWithTTL
}

// ShardedInMemoryStore represents a sharded in-memory key-value store with TTL.
type ShardedInMemoryStore struct {
	shards    []*mapShard
	numShards int
}

// NewShardedInMemoryStore creates a new instance of ShardedInMemoryStore.
func NewShardedInMemoryStore(numShards int) *ShardedInMemoryStore {
	shards := make([]*mapShard, numShards)
	for i := 0; i < numShards; i++ {
		shards[i] = &mapShard{
			store: make(map[string]ValueWithTTL),
		}
	}
	return &ShardedInMemoryStore{
		shards:    shards,
		numShards: numShards,
	}
}

// getShard returns the shard for a given key.
func (s *ShardedInMemoryStore) getShard(key string) *mapShard {
	hash := fnv.New32a()
	hash.Write([]byte(key))
	shardIndex := int(hash.Sum32()) % s.numShards
	return s.shards[shardIndex]
}

// Set adds a key-value pair to the store with an optional TTL.
func (s *ShardedInMemoryStore) Set(key, value string, ttl int64) {
	shard := s.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	expiration := time.Now().Add(time.Duration(ttl) * time.Second).Unix()
	shard.store[key] = ValueWithTTL{Value: value, Expiration: expiration}
}

// Get retrieves a value by key from the store, checking for expiration.
func (s *ShardedInMemoryStore) Get(key string) (string, bool) {
	shard := s.getShard(key)
	shard.mu.RLock()
	valueWithTTL, exists := shard.store[key]
	shard.mu.RUnlock() // Unlock before potentially deleting

	if !exists || (valueWithTTL.Expiration > 0 && time.Now().Unix() > valueWithTTL.Expiration) {
		// If the key does not exist or has expired, attempt to delete it
		if exists {
			s.Delete(key) // Delete the expired key
		}
		return "", false
	}

	return valueWithTTL.Value, true
}

// Delete removes a key-value pair from the store.
func (s *ShardedInMemoryStore) Delete(key string) {
	shard := s.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	delete(shard.store, key)
}

// Cleanup removes expired keys from the store concurrently.
func (s *ShardedInMemoryStore) Cleanup() {
	for _, shard := range s.shards {
		shard.mu.Lock()
		now := time.Now().Unix()
		for key, valueWithTTL := range shard.store {
			if valueWithTTL.Expiration > 0 && now > valueWithTTL.Expiration {
				delete(shard.store, key)
			}
		}
		shard.mu.Unlock()
	}
}

// StartCleanupRoutine starts a background goroutine to periodically clean up expired keys.
func (s *ShardedInMemoryStore) StartCleanupRoutine(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			<-ticker.C
			s.Cleanup() // Perform cleanup at regular intervals
		}
	}()
}

// LoadConfigAndCreateStore loads the configuration and creates a new sharded store.
func LoadConfigAndCreateStore(configPath string) (*ShardedInMemoryStore, error) {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, err
	}
	return NewShardedInMemoryStore(cfg.NumShards), nil
}
