package cache

import (
	"context"
	"time"

	"github.com/krisalay/in-memory-cache/engine"
	evict "github.com/krisalay/in-memory-cache/eviction"
	"github.com/krisalay/in-memory-cache/shard"
	"github.com/krisalay/in-memory-cache/types"
	"golang.org/x/sync/singleflight"
)

/*
ShardedCache is the main cache implementation.
This struct is the orchestrator that connects:
- shards
- eviction
- expiration
- loading
- write policies
- metrics
*/
type ShardedCache struct {
	// shards are the actual storage units. Each shard is an independent mini-cache.
	shards []*shard.Shard

	// engine contains the "rules" of the cache: TTL, refresh, loader, write policy, metrics, etc.
	engine *engine.CacheEngine

	// selector decides which shard a key should go to.
	selector shard.Selector

	// capacity is the maximum number of entries in the cache. This is divided across shards.
	capacity int

	// singleflight prevents multiple goroutines from loading the same key from the backing store simultaneously.
	sf singleflight.Group
}

func NewShardedCache(
	shards int,
	capacity int,
	eviction evict.PolicyType,
	engine *engine.CacheEngine,
) *ShardedCache {

	// Create shards
	s := make([]*shard.Shard, shards)
	for i := range s {
		// Each shard gets its own eviction policy instance
		s[i] = shard.NewShard(evict.NewEvictionPolicy(eviction))
	}

	return &ShardedCache{
		shards:   s,
		engine:   engine,
		selector: &shard.PowerOfTwoSelector{}, // smart shard selection
		capacity: capacity,
	}
}

/*
Get retrieves a value from the cache.
*/
func (c *ShardedCache) Get(ctx context.Context, key string) (any, error) {

	// Decide which shard should handle this key
	sh := c.selector.Select(key, c.shards)

	// Try to read from shard storage
	if ent, ok := sh.Store.Get(key); ok {

		// Check if entry is expired
		if c.engine.IsExpired(ent) {
			c.engine.Metrics.Expire()
			c.Remove(key) // remove expired entry
		} else {
			// Cache hit
			c.engine.Metrics.Hit()

			// Update TTL / refresh logic
			c.engine.OnRead(key, ent)

			// Update eviction metadata
			sh.Eviction.OnGet(key)

			return ent.Value, nil
		}
	}

	// Cache miss
	c.engine.Metrics.Miss()

	/*
		singleflight ensures that:
		- If 100 goroutines request the same missing key,
		  only ONE of them loads it from the backing store.
		- Others wait for the result.
	*/
	val, err, _ := c.sf.Do(key, func() (any, error) {
		return c.engine.Load(ctx, key)
	})
	if err != nil || val == nil {
		return nil, err
	}

	// Store loaded value in cache
	_ = c.Put(ctx, key, val)

	return val, nil
}

/*
Put stores a value in the cache without explicit TTL.
*/
func (c *ShardedCache) Put(ctx context.Context, key string, value any) error {
	return c.PutWithTTL(ctx, key, value, 0)
}

/*
PutWithTTL stores a value with an explicit TTL.
*/
func (c *ShardedCache) PutWithTTL(
	ctx context.Context,
	key string,
	value any,
	ttl time.Duration,
) error {

	// Select shard
	sh := c.selector.Select(key, c.shards)

	// Lock shard for safe writes
	sh.EvictMu.Lock()
	defer sh.EvictMu.Unlock()

	/*
		Check capacity of this shard.
		Total capacity is divided across shards.
	*/
	if sh.Store.Size() >= int64(c.capacity/len(c.shards)) {

		// Evict one key using eviction policy
		evicted := sh.Eviction.Evict()
		if evicted != "" {
			c.engine.Metrics.Eviction()
			sh.Store.Delete(evicted)
		}
	}

	// Create cache entry
	now := time.Now()
	ent := &types.CacheEntry{
		Key:            key,
		Value:          value,
		CreatedAt:      now,
		LastAccessedAt: now,
	}

	// If TTL is provided, set expiration time
	if ttl > 0 {
		ent.ExpireAt = now.Add(ttl)
	}

	// Apply write policy + expiration logic
	c.engine.OnWrite(ctx, ent)

	// Store entry in shard
	sh.Store.Put(key, ent)

	// Update eviction metadata
	sh.Eviction.OnPut(key)

	return nil
}

/*
Remove deletes a key from the cache immediately.
*/
func (c *ShardedCache) Remove(key string) {
	sh := c.selector.Select(key, c.shards)

	sh.EvictMu.Lock()
	defer sh.EvictMu.Unlock()

	sh.Store.Delete(key)
	sh.Eviction.Remove(key)
}

/*
Expire updates TTL of an existing key.
*/
func (c *ShardedCache) Expire(key string, ttl time.Duration) bool {
	sh := c.selector.Select(key, c.shards)

	sh.EvictMu.Lock()
	defer sh.EvictMu.Unlock()

	ent, ok := sh.Store.Get(key)
	if !ok {
		return false
	}

	ent.ExpireAt = time.Now().Add(ttl)
	return true
}

/*
TTL returns remaining time-to-live of a key.
*/
func (c *ShardedCache) TTL(key string) time.Duration {
	sh := c.selector.Select(key, c.shards)

	ent, ok := sh.Store.Get(key)
	if !ok || ent.ExpireAt.IsZero() {
		return -1
	}

	d := time.Until(ent.ExpireAt)
	if d < 0 {
		return -2
	}
	return d
}

/*
Close gracefully shuts down the cache.
This is important for write-back policies,so pending writes are flushed.
*/
func (c *ShardedCache) Close() {
	if c.engine.WritePolicy != nil {
		c.engine.WritePolicy.Close()
	}
}
