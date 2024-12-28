package db

import (
	"sync"
	"time"
)

// ValueWithTTL represents a value with its expiration time.
type ValueWithTTL struct {
	Value      string
	Expiration int64 // Unix timestamp in seconds
}

// InMemoryStore represents a simple in-memory key-value store with TTL.
type InMemoryStore struct {
	mu    sync.RWMutex
	store map[string]ValueWithTTL
}

// NewInMemoryStore creates a new instance of InMemoryStore.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		store: make(map[string]ValueWithTTL),
	}
}

// Set adds a key-value pair to the store with an optional TTL.
func (s *InMemoryStore) Set(key, value string, ttl int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	expiration := time.Now().Add(time.Duration(ttl) * time.Second).Unix()
	s.store[key] = ValueWithTTL{Value: value, Expiration: expiration}
}

// Get retrieves a value by key from the store, checking for expiration.
func (s *InMemoryStore) Get(key string) (string, bool) {
	s.mu.RLock()
	valueWithTTL, exists := s.store[key]
	s.mu.RUnlock() // Unlock before potentially deleting

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
func (s *InMemoryStore) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.store, key)
}

// Cleanup removes expired keys from the store concurrently.
func (s *InMemoryStore) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().Unix()
	for key, valueWithTTL := range s.store {
		if valueWithTTL.Expiration > 0 && now > valueWithTTL.Expiration {
			delete(s.store, key)
		}
	}
}

// StartCleanupRoutine starts a background goroutine to periodically clean up expired keys.
func (s *InMemoryStore) StartCleanupRoutine(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			<-ticker.C
			s.Cleanup() // Perform cleanup at regular intervals
		}
	}()
}
