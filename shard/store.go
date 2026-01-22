package shard

import (
	"sync/atomic"

	"github.com/krisalay/in-memory-cache/types"
)

/*
This file defines how data is actually stored inside a shard. This is NOT a normal map.
- Reads should be very fast
- Reads should NOT require locks
- Writes are less frequent and can afford extra work

To achieve this, we use a technique called: "Copy-On-Write" (COW)
*/

// ShardStore is the interface used by a shard to store and retrieve cache entries.
type ShardStore interface {

	// Get retrieves an entry by key.
	Get(string) (*types.CacheEntry, bool)

	// Put inserts or replaces an entry.
	Put(string, *types.CacheEntry)

	// Delete removes an entry.
	Delete(string)

	// Size returns how many entries are stored.
	Size() int64
}

/*
cowStore is a Copy-On-Write implementation of ShardStore.

What "copy-on-write" means:
---------------------------
- Readers always see an immutable snapshot
- Writers create a NEW copy of the map
- The new map replaces the old one atomically

This gives us:
--------------
- Lock-free reads
- Very simple concurrency model
- Predictable performance for reads
*/
type cowStore struct {

	// data holds the actual map[string]*CacheEntry.
	// atomic.Value allows us to: Swap the entire map atomically and let readers safely access it without locks
	data atomic.Value // stores map[string]*CacheEntry

	// size tracks the number of entries. We keep this separate so we don't need to count map entries every time.
	size atomic.Int64
}

func NewCOWStore() *cowStore {
	s := &cowStore{}
	s.data.Store(make(map[string]*types.CacheEntry))
	return s
}

// Get retrieves an entry from the store.
func (s *cowStore) Get(key string) (*types.CacheEntry, bool) {
	m := s.data.Load().(map[string]*types.CacheEntry)
	ent, ok := m[key]
	return ent, ok
}

/*
Put inserts or updates an entry in the store. This is where copy-on-write happens.

1. Load the current map
2. Create a NEW map
3. Copy all existing entries
4. Add the new entry
5. Atomically replace the old map
6. Update the size

- Reads are cheap and frequent
- Writes are slower but less frequent
*/
func (s *cowStore) Put(key string, ent *types.CacheEntry) {
	old := s.data.Load().(map[string]*types.CacheEntry)

	// Create a new map with extra capacity
	n := make(map[string]*types.CacheEntry, len(old)+1)

	// Copy existing entries
	for k, v := range old {
		n[k] = v
	}

	// Add / replace entry
	n[key] = ent

	// Atomically swap the map
	s.data.Store(n)

	// Update size
	s.size.Store(int64(len(n)))
}

// Delete removes an entry from the store. Just like Put, this uses copy-on-write.
func (s *cowStore) Delete(key string) {
	old := s.data.Load().(map[string]*types.CacheEntry)

	// Create a new map without the deleted key
	n := make(map[string]*types.CacheEntry)

	for k, v := range old {
		if k != key {
			n[k] = v
		}
	}

	// Atomically replace map
	s.data.Store(n)

	// Update size
	s.size.Store(int64(len(n)))
}

// Size returns how many entries are in the store.
func (s *cowStore) Size() int64 {
	return s.size.Load()
}
