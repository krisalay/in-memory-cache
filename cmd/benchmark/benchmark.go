package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	cache "github.com/krisalay/in-memory-cache"
	"github.com/krisalay/in-memory-cache/engine"
	"github.com/krisalay/in-memory-cache/eviction"
	"github.com/krisalay/in-memory-cache/expiration"
	"github.com/krisalay/in-memory-cache/writepolicy"
)

// ================= BACKING STORE =================

type InMemoryStore struct {
	mu   sync.RWMutex
	data map[string]any
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{data: make(map[string]any)}
}

func (s *InMemoryStore) Load(ctx context.Context, key string) (any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data[key], nil
}

func (s *InMemoryStore) Put(ctx context.Context, key string, value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
	return nil
}

// ================= BENCHMARK =================

func main() {
	ctx := context.Background()

	fmt.Println("\n================ CACHE LOAD BENCHMARK =================")

	// ---------------- Cache Config ----------------
	const (
		shards      = 8
		capacity    = 200000
		preloadKeys = 100000
		goroutines  = 200
		opsPerG     = 5000
	)

	fmt.Println("CONFIG")
	fmt.Println("---------------------------------")
	fmt.Println("Shards       :", shards)
	fmt.Println("Capacity     :", capacity)
	fmt.Println("Preload Keys :", preloadKeys)
	fmt.Println("Goroutines   :", goroutines)
	fmt.Println("Ops/Goroutine:", opsPerG)
	fmt.Println("---------------------------------")

	// ---------------- Backing Store ----------------
	store := NewInMemoryStore()

	// ---------------- Cache Engine ----------------
	exp := &expiration.ExpireAfterAccess{TTL: 60 * time.Second}
	writePolicy := writepolicy.NewWriteBackPolicy(store, 4096)

	engine := engine.NewCacheEngine(
		exp,
		nil,
		store,
		writePolicy,
		nil,
	)

	c := cache.NewShardedCache(
		shards,
		capacity,
		eviction.LRU,
		engine,
	)

	// ---------------- Preload Cache ----------------
	fmt.Println("Preloading cache...")
	for i := 0; i < preloadKeys; i++ {
		key := fmt.Sprintf("key-%d", i)
		c.Put(ctx, key, i)
	}
	fmt.Println("Preload complete.")

	// ---------------- Warmup ----------------
	fmt.Println("Warming up cache...")
	for i := 0; i < 10000; i++ {
		c.Get(ctx, fmt.Sprintf("key-%d", i%preloadKeys))
	}
	fmt.Println("Warmup complete.")

	// ---------------- Load Test ----------------
	fmt.Println("Running concurrency benchmark...")

	start := time.Now()

	wg := sync.WaitGroup{}
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerG; j++ {
				key := fmt.Sprintf("key-%d", j%preloadKeys)
				c.Get(ctx, key)
			}
		}(i)
	}

	wg.Wait()

	duration := time.Since(start)
	totalOps := goroutines * opsPerG

	fmt.Println("\n================ CACHE LOAD BENCHMARK =================")

	fmt.Println("CONFIG")
	fmt.Println("---------------------------------")
	fmt.Println("Shards       :", shards)
	fmt.Println("Capacity     :", capacity)
	fmt.Println("Preload Keys :", preloadKeys)
	fmt.Println("Goroutines   :", goroutines)
	fmt.Println("Ops/Goroutine:", opsPerG)
	fmt.Println("---------------------------------")

	fmt.Println("\nPreloading cache...")
	fmt.Println("Preload complete.")

	fmt.Println("\nWarming up cache...")
	fmt.Println("Warmup complete.")

	fmt.Println("\nRunning concurrency benchmark...")

	fmt.Println("\n================ RESULTS =================")
	fmt.Printf("Total Operations : %d\n", totalOps)
	fmt.Printf("Total Time       : %v\n", duration)
	fmt.Printf("Throughput       : %.2f ops/sec\n", float64(totalOps)/duration.Seconds())
	fmt.Println("=========================================")

	c.Close()
}
