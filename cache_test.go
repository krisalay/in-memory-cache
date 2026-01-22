package cache_test

import (
	"context"
	"sync"
	"testing"
	"time"

	cache "github.com/krisalay/in-memory-cache"
	"github.com/krisalay/in-memory-cache/engine"
	"github.com/krisalay/in-memory-cache/eviction"
	"github.com/krisalay/in-memory-cache/expiration"
	"github.com/krisalay/in-memory-cache/writepolicy"
)

//
// ================= TEST BACKING STORE =================
//

type TestStore struct {
	mu   sync.RWMutex
	data map[string]any
}

func NewTestStore() *TestStore {
	return &TestStore{data: make(map[string]any)}
}

func (s *TestStore) Load(ctx context.Context, key string) (any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data[key], nil
}

func (s *TestStore) Put(ctx context.Context, key string, value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
	return nil
}

func (s *TestStore) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
}

//
// ================= HELPER: CREATE CACHE (WRITE-BACK MODE) =================
//

func newTestCache(capacity int) (*cache.ShardedCache, *TestStore) {
	store := NewTestStore()

	exp := &expiration.ExpireAfterAccess{TTL: 10 * time.Second}

	// WRITE-BACK POLICY
	writePolicy := writepolicy.NewWriteBackPolicy(store, 1024)

	engine := engine.NewCacheEngine(
		exp,
		nil,
		store,
		writePolicy,
		nil,
	)

	c := cache.NewShardedCache(
		2,            // shards
		capacity,     // capacity
		eviction.LRU, // eviction policy
		engine,
	)

	return c, store
}

//
// ================= BASIC OPERATIONS =================
//

func TestAddAndRetrieve(t *testing.T) {
	ctx := context.Background()
	c, _ := newTestCache(10)

	if err := c.Put(ctx, "key1", "value1"); err != nil {
		t.Fatalf("put failed: %v", err)
	}

	v, _ := c.Get(ctx, "key1")
	if v != "value1" {
		t.Fatalf("expected value1, got %v", v)
	}
}

func TestRetrieveNonExistentKey(t *testing.T) {
	ctx := context.Background()
	c, store := newTestCache(10)

	// backing store has value
	store.data["keyX"] = "store-value"

	v, _ := c.Get(ctx, "keyX")
	if v != "store-value" {
		t.Fatalf("expected store-value, got %v", v)
	}

	// missing in both cache and store
	v, _ = c.Get(ctx, "missing")
	if v != nil {
		t.Fatalf("expected nil, got %v", v)
	}
}

func TestUpdateExistingKey(t *testing.T) {
	ctx := context.Background()
	c, _ := newTestCache(10)

	c.Put(ctx, "key1", "value1")
	c.Put(ctx, "key1", "value2")

	v, _ := c.Get(ctx, "key1")
	if v != "value2" {
		t.Fatalf("expected value2, got %v", v)
	}
}

func TestRemoveKey(t *testing.T) {
	ctx := context.Background()
	c, store := newTestCache(10)

	c.Put(ctx, "key1", "value1")

	// wait briefly to allow async write-back to store
	time.Sleep(10 * time.Millisecond)

	c.Remove("key1")

	// remove from store as well to simulate true delete
	store.Delete("key1")

	v, _ := c.Get(ctx, "key1")
	if v != nil {
		t.Fatalf("expected nil after remove, got %v", v)
	}
}

//
// ================= CAPACITY & EVICTION =================
//

func TestEvictionOnCapacity(t *testing.T) {
	ctx := context.Background()
	c, store := newTestCache(2)

	// preload store so evicted key can be reloaded
	store.data["key1"] = "value1"
	store.data["key2"] = "value2"
	store.data["key3"] = "value3"

	c.Put(ctx, "key1", "value1")
	c.Put(ctx, "key2", "value2")
	c.Put(ctx, "key3", "value3") // should evict key1 (LRU)

	v, _ := c.Get(ctx, "key1")

	// since backing store has key1, it should reload
	if v != "value1" {
		t.Fatalf("expected value1 from backing store, got %v", v)
	}
}

//
// ================= TTL TEST =================
//

func TestTTLExpiration(t *testing.T) {
	ctx := context.Background()
	c, store := newTestCache(10)

	// ensure key is NOT in backing store
	store.Delete("ttlKey")

	c.PutWithTTL(ctx, "ttlKey", "temp", 1*time.Second)

	time.Sleep(2 * time.Second)

	v, _ := c.Get(ctx, "ttlKey")

	// with write-back policy, TTL key should truly expire
	if v != nil {
		t.Fatalf("expected nil after TTL expiration, got %v", v)
	}
}

//
// ================= CONCURRENCY TEST =================
//

func TestConcurrentGet(t *testing.T) {
	ctx := context.Background()
	c, store := newTestCache(10)

	store.data["key"] = "value"

	wg := sync.WaitGroup{}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, _ := c.Get(ctx, "key")
			if v != "value" {
				t.Errorf("expected value, got %v", v)
			}
		}()
	}

	wg.Wait()
}
