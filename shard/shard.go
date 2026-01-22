package shard

import (
	"sync"

	"github.com/krisalay/in-memory-cache/eviction"
)

/*
This file defines what a "Shard" is. A shard is a small, independent piece of the cache.
Instead of having: One big cache and one big lock
We split the cache into many shards. Each shard:
- Holds some portion of the data
- Has its own eviction logic
- Has its own lock for writes

This dramatically improves concurrency and scalability.
*/

type Shard struct {

	// Store holds the actual key → value data for this shard. This is NOT a regular map.
	// It is a copy-on-write store that allows lock-free reads.
	Store ShardStore

	// Eviction controls which key should be removed when this shard runs out of space.
	// Each shard has its OWN eviction policy instance. This avoids shared state and reduces contention.
	Eviction eviction.Policy

	// EvictMu is a mutex used to protect write operations on this shard.
	// - Reads are lock-free
	// - Writes are protected by this mutex
	//
	// This is a deliberate design choice: reads are much more frequent than writes.
	EvictMu sync.Mutex
}

func NewShard(ev eviction.Policy) *Shard {
	return &Shard{
		Store:    NewCOWStore(),
		Eviction: ev,
	}
}
