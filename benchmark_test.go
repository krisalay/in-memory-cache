package cache_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	cache "github.com/krisalay/in-memory-cache"
	"github.com/krisalay/in-memory-cache/engine"
	"github.com/krisalay/in-memory-cache/eviction"
	"github.com/krisalay/in-memory-cache/expiration"
	"github.com/krisalay/in-memory-cache/writepolicy"
)

func newBenchmarkCache() *cache.ShardedCache {
	store := NewTestStore()

	exp := &expiration.ExpireAfterAccess{TTL: 10 * time.Second}
	writePolicy := writepolicy.NewWriteBackPolicy(store, 1024)

	engine := engine.NewCacheEngine(
		exp,
		nil,
		store,
		writePolicy,
		nil,
	)

	return cache.NewShardedCache(
		8,            // shards
		100000,       // capacity
		eviction.LRU, // eviction
		engine,
	)
}

//
// ================= SINGLE THREAD BENCH =================
//

func BenchmarkCacheGetHit(b *testing.B) {
	ctx := context.Background()
	c := newBenchmarkCache()

	c.Put(ctx, "key", "value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get(ctx, "key")
	}
}

func BenchmarkCacheGetMiss(b *testing.B) {
	ctx := context.Background()
	c := newBenchmarkCache()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("miss-%d", i)
		c.Get(ctx, key)
	}
}

//
// ================= PARALLEL BENCH =================
//

func BenchmarkCacheParallelGet(b *testing.B) {
	ctx := context.Background()
	c := newBenchmarkCache()

	for i := 0; i < 1000; i++ {
		c.Put(ctx, fmt.Sprintf("key-%d", i), i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Get(ctx, "key-42")
		}
	})
}

//
// ================= WRITE BENCH =================
//

func BenchmarkCachePut(b *testing.B) {
	ctx := context.Background()
	c := newBenchmarkCache()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Put(ctx, fmt.Sprintf("key-%d", i), i)
	}
}

//
// ================= HIGH CONCURRENCY TEST =================
//

func BenchmarkCacheHighConcurrency(b *testing.B) {
	ctx := context.Background()
	c := newBenchmarkCache()

	keys := make([]string, 10000)
	for i := range keys {
		keys[i] = fmt.Sprintf("key-%d", i)
		c.Put(ctx, keys[i], i)
	}

	b.ResetTimer()

	wg := sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < b.N/100; j++ {
				c.Get(ctx, keys[j%len(keys)])
			}
		}(i)
	}
	wg.Wait()
}
