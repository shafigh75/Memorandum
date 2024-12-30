package db

import (
	"fmt"
	"testing"
)

func setupBenchmarkStore() *ShardedInMemoryStore {
	store, err := LoadConfigAndCreateStore("../../config/config.json")
	if err != nil {
		panic("ERROR starting the benchmark" + err.Error())
	}
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
