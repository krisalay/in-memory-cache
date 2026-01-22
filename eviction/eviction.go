package eviction

/*
This file defines how the cache decides what to remove when it runs out of space.
*/

/*
Policy is the interface that all eviction strategies must follow.

This is a set of rules that any eviction algorithm (LRU, LFU, FIFO, etc.) must obey
so the rest of the cache can interact with it in a uniform way.

The cache does NOT care how eviction works internally.
It only calls these methods.
*/
type Policy interface {

	// OnGet is called whenever a key is read from the cache.
	//
	// Some eviction strategies care about reads.
	// For example:
	// - LRU needs to know what was accessed recently
	// - LFU may want to count accesses
	//
	// FIFO usually ignores this.
	OnGet(string)

	// OnPut is called whenever a key is added to the cache.
	//
	// This lets the eviction policy:
	// - Track insertion order
	// - Initialize counters or metadata
	OnPut(string)

	// Remove is called when a key is explicitly removed
	// from the cache (not evicted).
	//
	// This allows the eviction policy to clean up
	// any internal bookkeeping for that key.
	Remove(string)

	// Evict is called when the cache is FULL and needs space.
	//
	// The policy must decide:
	// - Which key should be removed?
	//
	// It returns the key that should be evicted.
	// The cache will then actually remove it from storage.
	Evict() string
}

// PolicyType is a simple identifier for supported eviction strategies.
type PolicyType string

const (
	// LRU (Least Recently Used): Evicts the key that has NOT been accessed for the longest time.
	LRU PolicyType = "LRU"

	// LFU (Least Frequently Used): Evicts the key that has been accessed the fewest times.
	// This works well when:
	// - Some keys are consistently hot
	// - Some keys are rarely used
	LFU PolicyType = "LFU"

	// FIFO (First In First Out): Evicts the oldest inserted key, regardless of access.
	FIFO PolicyType = "FIFO"
)

// NewEvictionPolicy is a small factory function.
// Given a PolicyType, it creates the correct eviction policy.
func NewEvictionPolicy(t PolicyType) Policy {
	switch t {
	case LRU:
		return newLRU()
	case LFU:
		return newLFU()
	case FIFO:
		return newFIFO()
	default:
		panic("unknown eviction policy")
	}
}
