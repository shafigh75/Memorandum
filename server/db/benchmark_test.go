package db

import (
	"fmt"
	"testing"
	"time"
)

func setupBenchmarkStore() *ShardedInMemoryStore {
	wal, err := NewWAL("benchmark_wal.log", 100, 10*time.Second)
	if err != nil {
		panic(err)
	}
	store := NewShardedInMemoryStore(16, wal)
	return store
}

func BenchmarkSet(b *testing.B) {
	store := setupBenchmarkStore()
	defer store.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i)
		value := fmt.Sprintf("value%d", i)
		store.Set(key, value, 30)
	}
}

func BenchmarkGet(b *testing.B) {
	store := setupBenchmarkStore()
	defer store.Close()

	// Prepopulate store with data
	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("key%d", i)
		value := fmt.Sprintf("value%d", i)
		store.Set(key, value, 30)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i%10000)
		store.Get(key)
	}
}

func BenchmarkDelete(b *testing.B) {
	store := setupBenchmarkStore()
	defer store.Close()

	// Prepopulate store with data
	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("key%d", i)
		value := fmt.Sprintf("value%d", i)
		store.Set(key, value, 30)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i%10000)
		store.Delete(key)
	}
}
