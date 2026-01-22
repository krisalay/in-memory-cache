package engine

import (
	"context"
	"time"

	"github.com/krisalay/in-memory-cache/expiration"
	"github.com/krisalay/in-memory-cache/refresh"
	"github.com/krisalay/in-memory-cache/types"
	"github.com/krisalay/in-memory-cache/writepolicy"
)

/*
CacheEngine is the "brain" of the cache system.
It is responsible for the "behavior" of the cache, NOT storage.
This acts as the policy layer.

It decides:
- When data is expired
- How TTL is updated on reads/writes
- When refresh hooks are triggered
- How data is loaded on cache miss
- How writes are propagated to backing store
- How metrics are recorded

It does NOT:
- Store data
- Handle sharding
- Handle locking
- Decide eviction order
*/
type CacheEngine struct {

	// Expiration controls when a cache entry should be considered “too old”.
	// Example: expire data 5 seconds after last access.
	// If this is nil, entries never expire based on time.
	Expiration expiration.Strategy

	// Refresh is an optional hook that runs when data is read.
	// This is used when we want to refresh data in the background
	// without blocking the current request.
	// If nil, no refresh logic is executed.
	Refresh refresh.Hook

	// Loader is how the cache talks to the outside world when it does NOT have the data.
	// This can be a database call, an API call, or any external call
	// This enables “read-through caching”.
	Loader types.Loader

	// WritePolicy decides what happens when data is written to the cache.
	// Examples:
	// - Write-through: write to DB immediately
	// - Write-back: write to DB asynchronously later
	//
	// If nil, cache writes stay only in memory.
	WritePolicy writepolicy.WritePolicy

	// Metrics is how we keep track of what the cache is doing.
	// Hits, misses, evictions, expirations, refreshes, etc.
	Metrics types.Metrics
}

/*
NewCacheEngine creates a CacheEngine.
*/
func NewCacheEngine(
	exp expiration.Strategy,
	refresh refresh.Hook,
	loader types.Loader,
	writePolicy writepolicy.WritePolicy,
	metrics types.Metrics,
) *CacheEngine {

	// Ensure metrics is always non-nil
	// This avoids defensive nil checks throughout the codebase
	if metrics == nil {
		metrics = types.NoopMetrics{}
	}

	return &CacheEngine{
		Expiration:  exp,
		Refresh:     refresh,
		Loader:      loader,
		WritePolicy: writePolicy,
		Metrics:     metrics,
	}
}

/*
IsExpired checks whether a cache entry is expired.

BEHAVIOR:
---------
- Delegates the decision to the configured Expiration strategy
- Uses current wall-clock time
- Returns false if no expiration strategy is configured
*/
func (e *CacheEngine) IsExpired(ent *types.CacheEntry) bool {
	return e.Expiration != nil &&
		e.Expiration.IsExpired(ent, time.Now())
}

/*
OnRead is called every time the cache successfully returns a value.

This is where read-related behavior lives.

Typical things that happen here:
- Update TTL for expire-after-access strategies
- Trigger a background refresh
- Record refresh metrics
*/
func (e *CacheEngine) OnRead(key string, ent *types.CacheEntry) {
	now := time.Now()

	// Some expiration strategies (like sliding TTL) care about reads
	if e.Expiration != nil {
		e.Expiration.OnAccess(ent, now)
	}

	// Refresh is optional and best-effort.
	// It should never slow down the read path.
	if e.Refresh != nil {
		e.Metrics.Refresh()
		e.Refresh.OnRead(key, ent)
	}
}

/*
OnWrite is called whenever something is written to the cache.

This is where we:
- Apply expiration rules related to writes
- Decide whether to push data to the backing store

Write propagation depends entirely on the configured WritePolicy.
*/
func (e *CacheEngine) OnWrite(ctx context.Context, ent *types.CacheEntry) {
	now := time.Now()

	// Some expiration strategies care about writes.
	if e.Expiration != nil {
		e.Expiration.OnWrite(ent, now)
	}

	if !ent.ExpireAt.IsZero() {
		return
	}

	// Forward the write if a write policy is configured.
	if e.WritePolicy != nil {
		e.WritePolicy.OnWrite(ctx, ent.Key, ent.Value)
	}
}

/*
Load is used when the cache does NOT have the data.

This usually means:
- A database call
- A network request
*/
func (e *CacheEngine) Load(ctx context.Context, key string) (any, error) {
	return e.Loader.Load(ctx, key)
}
