package db

import (
	"bytes"
	"container/heap"
	"encoding/binary"
	"fmt"
	"hash/crc32"
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

// WriteAheadLogEntry represents a binary log entry for WAL.
type WriteAheadLogEntry struct {
	Action    string
	Key       string
	Value     string
	TTL       int64
	Timestamp int64
	Checksum  uint32 // Integrity check using CRC32
}

// WAL represents the Write-Ahead Log.
type WAL struct {
	file        *os.File
	mu          sync.Mutex
	buffer      []WriteAheadLogEntry
	bufferSize  int
	flushTicker *time.Ticker
	flushDone   chan struct{}
}

// NewWAL creates a new WAL instance.
func NewWAL(filename string, bufferSize int, flushInterval time.Duration) (*WAL, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	wal := &WAL{
		file:        file,
		buffer:      make([]WriteAheadLogEntry, 0, bufferSize),
		bufferSize:  bufferSize,
		flushTicker: time.NewTicker(flushInterval),
		flushDone:   make(chan struct{}),
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

// flush writes the buffered log entries to the WAL file in binary format.
func (wal *WAL) flush() error {
	if len(wal.buffer) == 0 {
		return nil
	}

	var buf bytes.Buffer
	for _, entry := range wal.buffer {
		if err := encodeEntry(&buf, entry); err != nil {
			return err
		}
	}

	// Write binary data to the WAL file
	if _, err := wal.file.Write(buf.Bytes()); err != nil {
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
	return wal.file.Close()
}

// encodeEntry encodes a WriteAheadLogEntry into binary format.
func encodeEntry(buf *bytes.Buffer, entry WriteAheadLogEntry) error {
	if err := binary.Write(buf, binary.LittleEndian, int32(len(entry.Action))); err != nil {
		return err
	}
	if _, err := buf.WriteString(entry.Action); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.LittleEndian, int32(len(entry.Key))); err != nil {
		return err
	}
	if _, err := buf.WriteString(entry.Key); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.LittleEndian, int32(len(entry.Value))); err != nil {
		return err
	}
	if _, err := buf.WriteString(entry.Value); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.LittleEndian, entry.TTL); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.LittleEndian, entry.Timestamp); err != nil {
		return err
	}
	return binary.Write(buf, binary.LittleEndian, entry.Checksum)
}

// decodeEntry decodes a binary WAL entry from the given reader.
func decodeEntry(r io.Reader) (WriteAheadLogEntry, error) {
	var entry WriteAheadLogEntry

	var actionLen, keyLen, valueLen int32
	if err := binary.Read(r, binary.LittleEndian, &actionLen); err != nil {
		return entry, err
	}

	action := make([]byte, actionLen)
	if _, err := io.ReadFull(r, action); err != nil {
		return entry, err
	}
	entry.Action = string(action)

	if err := binary.Read(r, binary.LittleEndian, &keyLen); err != nil {
		return entry, err
	}

	key := make([]byte, keyLen)
	if _, err := io.ReadFull(r, key); err != nil {
		return entry, err
	}
	entry.Key = string(key)

	if err := binary.Read(r, binary.LittleEndian, &valueLen); err != nil {
		return entry, err
	}

	value := make([]byte, valueLen)
	if _, err := io.ReadFull(r, value); err != nil {
		return entry, err
	}
	entry.Value = string(value)

	if err := binary.Read(r, binary.LittleEndian, &entry.TTL); err != nil {
		return entry, err
	}

	if err := binary.Read(r, binary.LittleEndian, &entry.Timestamp); err != nil {
		return entry, err
	}

	if err := binary.Read(r, binary.LittleEndian, &entry.Checksum); err != nil {
		return entry, err
	}

	return entry, nil
}

// ShardedInMemoryStore represents a sharded in-memory key-value store with TTL.
type ShardedInMemoryStore struct {
	shards    []*mapShard
	numShards int
	wal       WALInterface
}

// mapShard represents a single shard of the in-memory store.
type mapShard struct {
	mu       sync.RWMutex
	store    map[string]ValueWithTTL
	ttlHeap  TTLHeap
	heapLock sync.Mutex // Lock for the heap to avoid race conditions
}

type WALInterface interface {
	Log(WriteAheadLogEntry) error
	Close() error
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
	hash := crc32.ChecksumIEEE([]byte(key))
	return s.shards[int(hash)%s.numShards]
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
	// Add to the TTL heap if TTL is set

	if ttl > 0 {
		shard.heapLock.Lock()
		heap.Push(&shard.ttlHeap, &HeapNode{
			Expiration: expiration,
			Key:        key,
		})
		shard.heapLock.Unlock()
	}

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
			fmt.Println("Error writing to WAL: ", err.Error())
		}
	}
}

// Get retrieves a value by key from the store, checking for expiration.
func (s *ShardedInMemoryStore) Get(key string) (string, bool) {
	shard := s.getShard(key)
	shard.mu.RLock()
	valueWithTTL, exists := shard.store[key]
	shard.mu.RUnlock()

	if !exists || (valueWithTTL.Expiration > 0 && time.Now().Unix() > valueWithTTL.Expiration) {
		if exists {
			s.Delete(key)
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
			fmt.Println("Error writing to WAL: ", err.Error())
		}
	}
}

// Cleanup removes expired keys from the store concurrently.
func (s *ShardedInMemoryStore) Cleanup() {
	for _, shard := range s.shards {
		now := time.Now().Unix()
		shard.heapLock.Lock()
		defer shard.heapLock.Unlock()
		for shard.ttlHeap.Len() > 0 {
			// Peek at the root of the heap
			node := shard.ttlHeap[0]
			if node.Expiration > now {
				break // Stop if the nearest expiration is in the future
			}
			// Remove expired entry from the heap
			heap.Pop(&shard.ttlHeap)
			// Delete the corresponding key from the store
			shard.mu.Lock()
			delete(shard.store, node.Key)
			shard.mu.Unlock()
		}
	}
}

// StartCleanupRoutine starts a background goroutine to periodically clean up expired keys.
func (s *ShardedInMemoryStore) StartCleanupRoutine(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			<-ticker.C
			s.Cleanup()
		}
	}()
}

// RecoverFromWAL replays the WAL to restore the state of the store.
func (s *ShardedInMemoryStore) RecoverFromWAL(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	isWalRecovery = true
	for {
		entry, err := decodeEntry(file)
		if err != nil {
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
			s.Set(entry.Key, entry.Value, entry.TTL)
		case "delete":
			s.Delete(entry.Key)
		}
	}
	isWalRecovery = false
	return nil
}

// LoadConfigAndCreateStore loads the config file and initializes the store.
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

// HeapNode represents a single node in the TTL heap.

type HeapNode struct {
	Expiration int64  // Unix timestamp of expiration
	Key        string // Key associated with this TTL
	Index      int    // Index in the heap (used for updates)
}

// TTLHeap implements a min-heap for managing expirations.

type TTLHeap []*HeapNode

func (h TTLHeap) Len() int { return len(h) }

func (h TTLHeap) Less(i, j int) bool { return h[i].Expiration < h[j].Expiration }

func (h TTLHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].Index = i
	h[j].Index = j
}

// Push adds a new node to the heap.

func (h *TTLHeap) Push(x interface{}) {
	node := x.(*HeapNode)
	node.Index = len(*h)
	*h = append(*h, node)
}

// Pop removes the node with the smallest expiration.

func (h *TTLHeap) Pop() interface{} {
	old := *h
	n := len(old)
	node := old[n-1]
	node.Index = -1 // Mark as removed
	*h = old[0 : n-1]
	return node
}
