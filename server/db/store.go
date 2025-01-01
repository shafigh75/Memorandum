package db

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"hash/fnv"
	"io"
	"os"
	"sync"
	"time"

	"github.com/shafigh75/Memorandum/config" // Adjust the import path as necessary
)

// ValueWithTTL represents a value with its expiration time.
type ValueWithTTL struct {
	Value      string
	Expiration int64 // Unix timestamp in seconds
}

// WriteAheadLogEntry represents a log entry for WAL.
type WriteAheadLogEntry struct {
	Action    string `json:"action"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	TTL       int64  `json:"ttl"`
	Timestamp int64  `json:"timestamp"`
	Checksum  uint32 `json:"checksum"` // Integrity check using CRC32
}

// WAL represents the Write-Ahead Log.
type WAL struct {
	file        *os.File
	mu          sync.Mutex
	buffer      []WriteAheadLogEntry
	bufferSize  int
	flushTicker *time.Ticker
	flushDone   chan struct{}
	gzipWriter  *gzip.Writer
}

// NewWAL creates a new WAL instance.
func NewWAL(filename string, bufferSize int, flushInterval time.Duration) (*WAL, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	gzipWriter := gzip.NewWriter(file)

	wal := &WAL{
		file:        file,
		buffer:      make([]WriteAheadLogEntry, 0, bufferSize),
		bufferSize:  bufferSize,
		flushTicker: time.NewTicker(flushInterval),
		flushDone:   make(chan struct{}),
		gzipWriter:  gzipWriter,
	}

	go wal.startFlushRoutine()
	return wal, nil
}

// DummyWAL is a no-op WAL implementation.
type DummyWAL struct{}

func (d *DummyWAL) Log(entry WriteAheadLogEntry) error { return nil }
func (d *DummyWAL) Close() error                       { return nil }

var isWalRecovery bool

// Log writes a log entry to the WAL.
func (wal *WAL) Log(entry WriteAheadLogEntry) error {
	wal.mu.Lock()
	defer wal.mu.Unlock()

	// Calculate checksum for the entry
	entry.Checksum = crc32.ChecksumIEEE([]byte(entry.Key + entry.Value))
	wal.buffer = append(wal.buffer, entry)
	if len(wal.buffer) >= wal.bufferSize {
		return wal.flush()
	}
	return nil
}

// flush writes the buffered log entries to the WAL file.
func (wal *WAL) flush() error {
	if len(wal.buffer) == 0 {
		return nil
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	for _, entry := range wal.buffer {
		if err := encoder.Encode(entry); err != nil {
			return err
		}
	}

	// Write the compressed data to the gzip writer
	if _, err := wal.gzipWriter.Write(buf.Bytes()); err != nil {
		return err
	}

	wal.buffer = wal.buffer[:0] // Clear the buffer
	return nil
}

// startFlushRoutine periodically flushes the log entries.
func (wal *WAL) startFlushRoutine() {
	defer close(wal.flushDone)
	for {
		select {
		case <-wal.flushTicker.C:
			wal.mu.Lock()
			if err := wal.flush(); err != nil {
				fmt.Println("Error flushing WAL: ", err.Error())
			}
			wal.mu.Unlock()
		}
	}
}

// Close closes the WAL file and flushes any remaining entries.
func (wal *WAL) Close() error {
	wal.flushTicker.Stop()
	wal.mu.Lock()
	defer wal.mu.Unlock()
	if err := wal.flush(); err != nil {
		return err
	}
	if err := wal.gzipWriter.Close(); err != nil {
		return err
	}
	return wal.file.Close()
}

// mapShard represents a single shard of the in-memory store.
type mapShard struct {
	mu    sync.RWMutex
	store map[string]ValueWithTTL
}

type WALInterface interface {
	Log(WriteAheadLogEntry) error
	Close() error
}

// ShardedInMemoryStore represents a sharded in-memory key-value store with TTL.
type ShardedInMemoryStore struct {
	shards    []*mapShard
	numShards int
	wal       WALInterface
}

// NewShardedInMemoryStore creates a new instance of ShardedInMemoryStore.
func NewShardedInMemoryStore(numShards int, wal WALInterface) *ShardedInMemoryStore {
	shards := make([]*mapShard, numShards)
	for i := 0; i < numShards; i++ {
		shards[i] = &mapShard{
			store: make(map[string]ValueWithTTL),
		}
	}
	return &ShardedInMemoryStore{
		shards:    shards,
		numShards: numShards,
		wal:       wal,
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
	var expiration int64 
	if ttl == 0 {
		expiration = 0
	} else {
		expiration = time.Now().Add(time.Duration(ttl) * time.Second).Unix()
	}
	shard.store[key] = ValueWithTTL{Value: value, Expiration: expiration}

	// Log the operation
	entry := WriteAheadLogEntry{
		Action:    "set",
		Key:       key,
		Value:     value,
		TTL:       ttl,
		Timestamp: time.Now().Unix(),
	}
	if !isWalRecovery {
		if err := s.wal.Log(entry); err != nil {
			// Handle logging error (e.g., log it)
			fmt.Println("Error writing to WAL: ", err.Error())
		}
	}
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

	// Log the delete operation
	entry := WriteAheadLogEntry{
		Action:    "delete",
		Key:       key,
		Timestamp: time.Now().Unix(),
	}
	if !isWalRecovery {
		if err := s.wal.Log(entry); err != nil {
			// Handle logging error (e.g., log it)
			fmt.Println("Error writing to WAL: ", err.Error())
		}
	}
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

// RecoverFromWAL replays the WAL to restore the state of the store.
// RecoverFromWAL replays the WAL to restore the state of the store.
func (s *ShardedInMemoryStore) RecoverFromWAL(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil && err.Error() != "EOF" {
		return err
	} else if err != nil && err.Error() == "EOF" {
		return nil
	}
	defer gzipReader.Close()

	decoder := json.NewDecoder(gzipReader)
	isWalRecovery = true
	for {
		var entry WriteAheadLogEntry
		if err := decoder.Decode(&entry); err != nil {
			if err == io.EOF {
				break // End of file reached, exit the loop gracefully
			}
			return err // Return any other error
		}

		// Validate checksum to ensure entry integrity
		expectedChecksum := crc32.ChecksumIEEE([]byte(entry.Key + entry.Value))
		if entry.Checksum != expectedChecksum {
			return fmt.Errorf("invalid checksum for entry: %v", entry)
		}

		switch entry.Action {
		case "set":
			expiredAt := time.Unix(entry.Timestamp, 0).Add(time.Duration(entry.TTL) * time.Second).Unix()
			nowDate := time.Now().Unix()
			if entry.TTL != 0 && nowDate > expiredAt {
				break
			}
			s.Set(entry.Key, entry.Value, entry.TTL)
		case "delete":
			expiredAt := time.Unix(entry.Timestamp, 0).Add(time.Duration(entry.TTL) * time.Second).Unix()
			nowDate := time.Now().Unix()
			if entry.TTL != 0 && nowDate > expiredAt {
				break
			}
			s.Delete(entry.Key)
		}
	}
	isWalRecovery = false
	return nil
}

func LoadConfigAndCreateStore(configPath string) (*ShardedInMemoryStore, error) {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, err
	}

	var wal WALInterface
	if cfg.WalEnabled {
		wal, err = NewWAL(cfg.WalPath, cfg.WalBufferSize, time.Duration(cfg.WalFlushInterval)*time.Second)
		if err != nil {
			return nil, err
		}
	} else {
		wal = &DummyWAL{}
	}

	store := NewShardedInMemoryStore(cfg.NumShards, wal)

	if cfg.WalEnabled {
		if err := store.RecoverFromWAL(cfg.WalPath); err != nil {
			return nil, err
		}
	}

	return store, nil
}

// Close cleans up resources, including closing the WAL.
func (s *ShardedInMemoryStore) Close() error {
	s.Cleanup()
	return s.wal.Close()
}
