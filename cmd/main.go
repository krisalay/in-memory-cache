package main

import (
	"context"
	"fmt"
	"strings"
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
	if !strings.HasPrefix(key, "k") {
		fmt.Println("STORE  → load:", key)
	}
	return s.data[key], nil
}

func (s *InMemoryStore) Put(ctx context.Context, key string, value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !strings.HasPrefix(key, "k") {
		fmt.Println("STORE  → put:", key)
	}
	s.data[key] = value
	return nil
}

func (s *InMemoryStore) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
}

// ================= METRICS =================
type Metrics struct {
	mu        sync.Mutex
	hits      int
	misses    int
	evictions int
	expired   int
}

func (m *Metrics) Hit()      { m.mu.Lock(); m.hits++; m.mu.Unlock() }
func (m *Metrics) Miss()     { m.mu.Lock(); m.misses++; m.mu.Unlock() }
func (m *Metrics) Eviction() { m.mu.Lock(); m.evictions++; m.mu.Unlock() }
func (m *Metrics) Expire()   { m.mu.Lock(); m.expired++; m.mu.Unlock() }
func (m *Metrics) Refresh()  {}

func (m *Metrics) Print() {
	fmt.Println("\n==================== METRICS ====================")
	fmt.Printf("HITS      : %d\n", m.hits)
	fmt.Printf("MISSES    : %d\n", m.misses)
	fmt.Printf("EVICTIONS : %d\n", m.evictions)
	fmt.Printf("EXPIRED   : %d\n", m.expired)
}

// ================= MAIN =================

func main() {
	ctx := context.Background()

	fmt.Println("\n==================== SYSTEM BOOT ====================")

	// ---------------- System Config ----------------
	fmt.Println("CACHE MODE      : WRITE-BACK")
	fmt.Println("EVICTION POLICY : LRU")
	fmt.Println("SHARDS          : 4")
	fmt.Println("TTL STRATEGY    : ExpireAfterAccess")
	fmt.Println("CAPACITY        : 20 keys")

	// ---------------- Backing Store ----------------
	store := NewInMemoryStore()
	store.Put(ctx, "a", "alpha")
	store.Put(ctx, "b", "beta")

	// ---------------- Metrics ----------------
	metrics := &Metrics{}

	// ---------------- Cache Engine ----------------
	exp := &expiration.ExpireAfterAccess{TTL: 2 * time.Second}
	writePolicy := writepolicy.NewWriteBackPolicy(store, 1024)

	engine := engine.NewCacheEngine(
		exp,
		nil,
		store,
		writePolicy,
		metrics,
	)

	cache := cache.NewShardedCache(
		4,
		20,
		eviction.LRU,
		engine,
	)

	// ====================================================
	fmt.Println("\n==================== 1) CACHE MISS ====================")
	v, _ := cache.Get(ctx, "a")
	fmt.Println("CACHE  → GET a =", v)

	// ====================================================
	fmt.Println("\n==================== 2) CACHE HIT ====================")
	v, _ = cache.Get(ctx, "a")
	fmt.Println("CACHE  → GET a =", v)

	// ====================================================
	fmt.Println("\n==================== 3) TTL EXPIRATION ====================")
	store.Delete("x") // ensure cache-only key
	cache.PutWithTTL(ctx, "x", "temp-value", 1*time.Second)
	fmt.Println("CACHE  → PUT x (TTL = 1s)")

	time.Sleep(2 * time.Second)

	fmt.Println("CACHE  → TTL expired for x")
	v, _ = cache.Get(ctx, "x")
	fmt.Println("CACHE  → GET x after TTL =", v)

	// ====================================================
	fmt.Println("\n==================== 4) SINGLEFLIGHT ====================")

	wg := sync.WaitGroup{}
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			val, _ := cache.Get(ctx, "b")
			fmt.Printf("GOROUTINE-%d → GET b = %v\n", id, val)
		}(i)
	}
	wg.Wait()

	// ====================================================
	fmt.Println("\n==================== 5) EVICTION ====================")

	for i := 0; i < 50; i++ {
		cache.Put(ctx, fmt.Sprintf("k%d", i), i)
	}

	v, _ = cache.Get(ctx, "a")
	fmt.Println("CACHE  → GET a after eviction =", v)

	// ====================================================
	fmt.Println("\n==================== 6) REMOVE ====================")

	cache.Remove("b")
	store.Delete("b")
	fmt.Println("CACHE  → REMOVE b")

	v, _ = cache.Get(ctx, "b")
	fmt.Println("CACHE  → GET b after remove =", v)

	// ====================================================
	metrics.Print()

	// ====================================================
	fmt.Println("\n==================== SHUTDOWN ====================")
	cache.Close()
	fmt.Println("SYSTEM → cache closed cleanly")
}
